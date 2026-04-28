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

// EvologDescribeSplitDoneMsg is sent after optional AI-generated descriptions for @- and @.
type EvologDescribeSplitDoneMsg struct {
	Repository *internal.Repository
	Err        error
}

// EvologDescribeSplitPreviewMsg carries LLM-proposed descriptions before the user applies them.
type EvologDescribeSplitPreviewMsg struct {
	ParentDescription string
	ChildDescription  string
	Err               error
}

type evologDescribeSplitJSON struct {
	ParentDescription string `json:"parent_description"`
	ChildDescription  string `json:"child_description"`
}

func describeSplitImmutableCheck(ctx context.Context, svc *jj.Service) error {
	for _, rev := range []string{"@-", "@"} {
		im, err := svc.RevisionImmutable(ctx, rev)
		if err != nil {
			return fmt.Errorf("immutable check %s: %w", rev, err)
		}
		if im {
			return fmt.Errorf("cannot describe immutable revision %s", rev)
		}
	}
	return nil
}

func buildDescribeSplitUserContent(ctx context.Context, svc *jj.Service) (string, error) {
	if err := describeSplitImmutableCheck(ctx, svc); err != nil {
		return "", err
	}
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

// SuggestEvologSplitDescriptionsCmd runs the LLM only; user confirms in the UI before apply.
func SuggestEvologSplitDescriptionsCmd(reqID int, svc *jj.Service, cfg *config.Config) tea.Cmd {
	_ = reqID
	return func() tea.Msg {
		if svc == nil || cfg == nil || !cfg.AIConfiguredForGeneration() {
			return EvologDescribeSplitPreviewMsg{Err: fmt.Errorf("AI is disabled or no API key")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), cfg.AITimeout())
		defer cancel()
		user, err := buildDescribeSplitUserContent(ctx, svc)
		if err != nil {
			return EvologDescribeSplitPreviewMsg{Err: err}
		}
		provider, err := llm.NewProviderForConfig(cfg)
		if err != nil {
			return EvologDescribeSplitPreviewMsg{Err: err}
		}
		raw, err := provider.Complete(ctx, evologDescribeSplitSystem, user)
		if err != nil {
			return EvologDescribeSplitPreviewMsg{Err: err}
		}
		pd, cd, err := parseEvologDescribeSplitJSON(raw)
		if err != nil {
			return EvologDescribeSplitPreviewMsg{Err: err}
		}
		return EvologDescribeSplitPreviewMsg{ParentDescription: pd, ChildDescription: cd}
	}
}

// ApplyEvologSplitDescriptionsCmd runs jj describe for @- and @ with the given messages.
func ApplyEvologSplitDescriptionsCmd(reqID int, svc *jj.Service, cfg *config.Config, parentDesc, childDesc string) tea.Cmd {
	_ = reqID
	return func() tea.Msg {
		if svc == nil || cfg == nil {
			return EvologDescribeSplitDoneMsg{Err: fmt.Errorf("jj service not available")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), cfg.AITimeout())
		defer cancel()
		if err := describeSplitImmutableCheck(ctx, svc); err != nil {
			return EvologDescribeSplitDoneMsg{Err: err}
		}
		if err := svc.DescribeCommit(ctx, "@-", parentDesc); err != nil {
			return EvologDescribeSplitDoneMsg{Err: fmt.Errorf("describe @-: %w", err)}
		}
		if err := svc.DescribeCommit(ctx, "@", childDesc); err != nil {
			return EvologDescribeSplitDoneMsg{Err: fmt.Errorf("describe @: %w", err)}
		}
		repo, err := svc.GetRepository(ctx, "")
		if err != nil {
			return EvologDescribeSplitDoneMsg{Err: err}
		}
		return EvologDescribeSplitDoneMsg{Repository: repo}
	}
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
		return ApplyEvologSplitDescriptionsCmd(0, svc, cfg, p.ParentDescription, p.ChildDescription)()
	}
}

func parseEvologDescribeSplitJSON(raw string) (parentDesc, childDesc string, err error) {
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
	pd := strings.TrimSpace(v.ParentDescription)
	cd := strings.TrimSpace(v.ChildDescription)
	if pd == "" || cd == "" {
		return "", "", fmt.Errorf("empty parent_description or child_description")
	}
	return pd, cd, nil
}
