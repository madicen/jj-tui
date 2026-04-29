package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/integrations/llm"
)

const evologDescribeSplitMaxDiff = 60_000

const evologDescribeSplitSystem = `You write two jj change descriptions after an evolog-based split.
Reply with a single JSON object only (no markdown fences):
{"parent_description":"...","child_description":"..."}

Rules:
- parent_description is for the immediate parent of the working copy (@-): one short title line, optional blank line and bullets.
- child_description is for the working copy (@): same style.
- Plain text only; no markdown code fences in the values.
`

const evologDescribeSplitSystemChildOnly = `You write one jj change description for the working copy (@) after an evolog-based split.
The parent of @ (@-) is immutable (e.g. protected bookmark like main); do not describe it.

Reply with a single JSON object only (no markdown fences):
{"child_description":"..."}

Rules:
- child_description is for @ only: one short title line, optional blank line and bullets.
- Plain text only; no markdown code fences in the value.
`

// EvologDescribeSplitDoneMsg is sent after optional AI-generated descriptions for @- and/or @.
type EvologDescribeSplitDoneMsg struct {
	Repository *internal.Repository
	OnlyChild  bool // true when @- was skipped (immutable parent)
	Err        error
}

// EvologDescribeSplitPreviewMsg carries LLM-proposed descriptions before the user applies them.
type EvologDescribeSplitPreviewMsg struct {
	ParentDescription  string
	ChildDescription   string
	SkipParentDescribe bool // when true, @- is immutable; apply only runs jj describe on @
	Err                error
}

type evologDescribeSplitJSON struct {
	ParentDescription string `json:"parent_description"`
	ChildDescription  string `json:"child_description"`
}

// DescribeSplitParentWritable reports whether jj describe may run on @- (parent of working copy @).
func DescribeSplitParentWritable(ctx context.Context, svc *jj.Service) (parentWritable bool, err error) {
	imAt, err := svc.RevisionImmutable(ctx, "@")
	if err != nil {
		return false, fmt.Errorf("immutable check @: %w", err)
	}
	if imAt {
		return false, fmt.Errorf("cannot describe immutable revision @")
	}
	imParent, err := svc.RevisionImmutable(ctx, "@-")
	if err != nil {
		return false, fmt.Errorf("immutable check @-: %w", err)
	}
	return !imParent, nil
}

func buildDescribeSplitUserContentDual(ctx context.Context, svc *jj.Service) (string, error) {
	parentDiff, err := svc.GitFormatDiffForRevision(ctx, "@-", evologDescribeSplitMaxDiff)
	if err != nil {
		return "", fmt.Errorf("diff @-: %w", err)
	}
	childDiff, err := svc.GitFormatDiffForRevision(ctx, "@", evologDescribeSplitMaxDiff)
	if err != nil {
		return "", fmt.Errorf("diff @: %w", err)
	}
	parentLine, _ := svc.GetCommitDescription(ctx, "@-")
	childLine, _ := svc.GetCommitDescription(ctx, "@")
	var ub strings.Builder
	fmt.Fprintf(&ub, "Current parent (@-) description:\n%s\n\nCurrent child (@) description:\n%s\n\nUnified diff for @- (vs its parents):\n%s\n\nUnified diff for @ (vs its parents):\n%s\n\nWrite JSON parent_description and child_description.",
		strings.TrimSpace(parentLine), strings.TrimSpace(childLine), strings.TrimSpace(parentDiff), strings.TrimSpace(childDiff))
	return ub.String(), nil
}

func buildDescribeSplitUserContentChildOnly(ctx context.Context, svc *jj.Service) (string, error) {
	childDiff, err := svc.GitFormatDiffForRevision(ctx, "@", evologDescribeSplitMaxDiff)
	if err != nil {
		return "", fmt.Errorf("diff @: %w", err)
	}
	childLine, _ := svc.GetCommitDescription(ctx, "@")
	var ub strings.Builder
	fmt.Fprintf(&ub, "The parent of @ (@-) is immutable; propose a description for @ only.\n\nCurrent @ description:\n%s\n\nUnified diff for @ (vs its parents):\n%s\n\nWrite JSON child_description only.",
		strings.TrimSpace(childLine), strings.TrimSpace(childDiff))
	return ub.String(), nil
}

func buildDescribeSplitLLMPayload(ctx context.Context, svc *jj.Service) (system, user string, parentWritable bool, err error) {
	parentWritable, err = DescribeSplitParentWritable(ctx, svc)
	if err != nil {
		return "", "", false, err
	}
	if parentWritable {
		user, err = buildDescribeSplitUserContentDual(ctx, svc)
		return evologDescribeSplitSystem, user, true, err
	}
	user, err = buildDescribeSplitUserContentChildOnly(ctx, svc)
	return evologDescribeSplitSystemChildOnly, user, false, err
}

// SuggestEvologSplitDescriptionsCmd runs the LLM only; user confirms in the UI before apply.
func SuggestEvologSplitDescriptionsCmd(reqID int, svc *jj.Service, cfg *config.Config) tea.Cmd {
	_ = reqID
	return func() tea.Msg {
		if svc == nil || cfg == nil || !cfg.AIConfiguredForGeneration() {
			return EvologDescribeSplitPreviewMsg{Err: fmt.Errorf("AI is disabled or no API key")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), cfg.AITimeout())
		defer cancel()
		system, user, parentWritable, err := buildDescribeSplitLLMPayload(ctx, svc)
		if err != nil {
			return EvologDescribeSplitPreviewMsg{Err: err}
		}
		provider, err := llm.NewProviderForConfig(cfg)
		if err != nil {
			return EvologDescribeSplitPreviewMsg{Err: err}
		}
		raw, err := provider.Complete(ctx, system, user)
		if err != nil {
			return EvologDescribeSplitPreviewMsg{Err: err}
		}
		pd, cd, err := parseEvologDescribeSplitJSON(raw, parentWritable)
		if err != nil {
			return EvologDescribeSplitPreviewMsg{Err: err}
		}
		return EvologDescribeSplitPreviewMsg{
			ParentDescription:  pd,
			ChildDescription:   cd,
			SkipParentDescribe: !parentWritable,
		}
	}
}

// ApplyEvologSplitDescriptionsCmd runs jj describe for @- (unless skipParentDescribe) and @.
func ApplyEvologSplitDescriptionsCmd(reqID int, svc *jj.Service, cfg *config.Config, parentDesc, childDesc string, skipParentDescribe bool) tea.Cmd {
	_ = reqID
	return func() tea.Msg {
		if svc == nil || cfg == nil {
			return EvologDescribeSplitDoneMsg{Err: fmt.Errorf("jj service not available")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), cfg.AITimeout())
		defer cancel()
		imAt, err := svc.RevisionImmutable(ctx, "@")
		if err != nil {
			return EvologDescribeSplitDoneMsg{Err: fmt.Errorf("immutable check @: %w", err)}
		}
		if imAt {
			return EvologDescribeSplitDoneMsg{Err: fmt.Errorf("cannot describe immutable revision @")}
		}
		if !skipParentDescribe {
			if err := describeDualParentMutable(ctx, svc); err != nil {
				return EvologDescribeSplitDoneMsg{Err: err}
			}
			if err := svc.DescribeCommit(ctx, "@-", parentDesc); err != nil {
				return EvologDescribeSplitDoneMsg{Err: fmt.Errorf("describe @-: %w", err)}
			}
		}
		if err := svc.DescribeCommit(ctx, "@", childDesc); err != nil {
			return EvologDescribeSplitDoneMsg{Err: fmt.Errorf("describe @: %w", err)}
		}
		repo, err := svc.GetRepository(ctx, "")
		if err != nil {
			return EvologDescribeSplitDoneMsg{Err: err}
		}
		return EvologDescribeSplitDoneMsg{Repository: repo, OnlyChild: skipParentDescribe}
	}
}

func describeDualParentMutable(ctx context.Context, svc *jj.Service) error {
	im, err := svc.RevisionImmutable(ctx, "@-")
	if err != nil {
		return fmt.Errorf("immutable check @-: %w", err)
	}
	if im {
		return fmt.Errorf("cannot describe immutable revision @-")
	}
	return nil
}

// DescribeEvologSplitCommitsCmd runs suggest + apply in one message (used by tests).
func DescribeEvologSplitCommitsCmd(reqID int, svc *jj.Service, cfg *config.Config) tea.Cmd {
	_ = reqID
	return func() tea.Msg {
		prev := SuggestEvologSplitDescriptionsCmd(0, svc, cfg)()
		p, ok := prev.(EvologDescribeSplitPreviewMsg)
		if !ok {
			return EvologDescribeSplitDoneMsg{Err: fmt.Errorf("internal: expected preview msg")}
		}
		if p.Err != nil {
			return EvologDescribeSplitDoneMsg{Err: p.Err}
		}
		return ApplyEvologSplitDescriptionsCmd(0, svc, cfg, p.ParentDescription, p.ChildDescription, p.SkipParentDescribe)()
	}
}

func parseEvologDescribeSplitJSON(raw string, requireParent bool) (parentDesc, childDesc string, err error) {
	s := strings.TrimSpace(raw)
	if i := strings.Index(s, "{"); i >= 0 {
		if j := strings.LastIndex(s, "}"); j > i {
			s = s[i : j+1]
		}
	}
	var v evologDescribeSplitJSON
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return "", "", fmt.Errorf("parse describe JSON: %w", err)
	}
	cd := strings.TrimSpace(v.ChildDescription)
	if cd == "" {
		return "", "", fmt.Errorf("empty child_description")
	}
	pd := strings.TrimSpace(v.ParentDescription)
	if requireParent {
		if pd == "" {
			return "", "", fmt.Errorf("empty parent_description or child_description")
		}
	}
	return pd, cd, nil
}
