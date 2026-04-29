package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/integrations/llm"
)

// EvologSplitExpectedChainStep is one narrative step in the AI-predicted outcome after automation.
type EvologSplitExpectedChainStep struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

const evologChainPreviewMaxSummary = 16_000

const evologChainPreviewSystem = `You predict how the jj repository will look after an "evolog split" automation plan runs (FAQ-style parent moves, optional file-level jj split, optional hunk-level jj splits).

Reply with a single JSON object only (no markdown fences):
{"steps":[{"label":"short phase name","description":"one or two sentences"},...],"final_parent_description":"...","final_child_description":"..."}

Rules:
- "steps": ordered oldest-first / bottom-of-stack toward the tip: mention FAQ rewiring (one step per FAQ base if multiple), optional file peel, each hunk peel round, then the tip. Use 0–12 steps; keep each description concise.
- "final_child_description": required. Full jj description for @ (working copy) after ALL automation — first line is the title; optional further lines.
- "final_parent_description": full jj description for @- (immediate parent of @) after all steps when that revision is expected to be mutable and user-describable; if the plan leaves @- as an immutable remote-tracking parent, still provide a short placeholder line (the client may skip applying it).
- Plain text in string values only; no markdown code fences inside values.
`

type evologChainPreviewJSON struct {
	Steps                  []EvologSplitExpectedChainStep `json:"steps"`
	FinalParentDescription string                         `json:"final_parent_description"`
	FinalChildDescription  string                         `json:"final_child_description"`
}

func buildEvologChainPreviewUserPrompt(entries []jj.EvologEntry, msg *EvologSplitSuggestMsg) (string, error) {
	if msg == nil {
		return "", fmt.Errorf("nil suggest message")
	}
	var b strings.Builder
	b.WriteString("## Automation plan (after client runs this)\n")
	fmt.Fprintf(&b, "- recommended_index (evolog row): %d\n", msg.PickIndex)
	if len(msg.MultiSplitBaseCommitIDs) > 0 {
		b.WriteString("- FAQ bases (deepest-first commit ids): ")
		b.WriteString(strings.Join(msg.MultiSplitBaseCommitIDs, ", "))
		b.WriteByte('\n')
	} else {
		b.WriteString("- FAQ bases: (default single base from recommended_index)\n")
	}
	if len(msg.FilesForFirstCommit) > 0 {
		fmt.Fprintf(&b, "- files_first_commit (jj split -r @): %s\n", strings.Join(msg.FilesForFirstCommit, ", "))
	} else {
		b.WriteString("- files_first_commit: (none)\n")
	}
	if len(msg.HunkPeelRounds) > 0 {
		for i, hm := range msg.HunkPeelRounds {
			var keys []string
			for p, k := range hm {
				keys = append(keys, fmt.Sprintf("%s:%d", p, k))
			}
			sort.Strings(keys)
			fmt.Fprintf(&b, "- hunk peel round %d: %s\n", i+1, strings.Join(keys, ", "))
		}
	} else {
		b.WriteString("- hunk_peel_rounds: (none)\n")
	}
	b.WriteString("\n## AI rationale for this plan\n")
	b.WriteString(strings.TrimSpace(msg.Rationale))
	b.WriteString("\n\n## Evolog rows (newest first, truncated)\n")
	maxRows := min(14, len(entries))
	for i := 0; i < maxRows; i++ {
		e := entries[i]
		sum := strings.TrimSpace(e.Summary)
		sum = strings.ReplaceAll(sum, "\n", " ")
		if len(sum) > 120 {
			sum = sum[:117] + "..."
		}
		short := strings.TrimSpace(e.CommitIDShort)
		if short == "" && len(e.CommitID) >= 8 {
			short = e.CommitID[:8]
		}
		fmt.Fprintf(&b, "- %d %s — %s\n", i, short, sum)
	}
	if len(entries) > maxRows {
		fmt.Fprintf(&b, "- … +%d more rows\n", len(entries)-maxRows)
	}
	return b.String(), nil
}

func parseEvologChainPreviewJSON(raw string) ([]EvologSplitExpectedChainStep, string, string, error) {
	s := strings.TrimSpace(raw)
	if i := strings.Index(s, "{"); i >= 0 {
		if j := strings.LastIndex(s, "}"); j > i {
			s = s[i : j+1]
		}
	}
	var v evologChainPreviewJSON
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return nil, "", "", fmt.Errorf("parse chain preview JSON: %w", err)
	}
	cd := strings.TrimSpace(v.FinalChildDescription)
	if cd == "" {
		return nil, "", "", fmt.Errorf("empty final_child_description")
	}
	pd := strings.TrimSpace(v.FinalParentDescription)
	var steps []EvologSplitExpectedChainStep
	for _, st := range v.Steps {
		l := strings.TrimSpace(st.Label)
		d := strings.TrimSpace(st.Description)
		if l == "" && d == "" {
			continue
		}
		steps = append(steps, EvologSplitExpectedChainStep{Label: l, Description: d})
	}
	if len(steps) > 24 {
		steps = steps[:24]
	}
	return steps, pd, cd, nil
}

// runEvologChainPreviewLLM fills ExpectedOutcomeChain and precomputed describe fields on msg when the split plan is non-empty.
func runEvologChainPreviewLLM(ctx context.Context, jjSvc *jj.Service, cfg *config.Config, entries []jj.EvologEntry, msg *EvologSplitSuggestMsg) {
	if msg == nil || jjSvc == nil || cfg == nil || !cfg.AIConfiguredForGeneration() {
		return
	}
	if msg.NoSplit || len(entries) < 2 {
		return
	}
	user, err := buildEvologChainPreviewUserPrompt(entries, msg)
	if err != nil {
		return
	}
	summaryCtx, cancelSum := context.WithTimeout(ctx, evologSplitFileValidateTimeout)
	defer cancelSum()
	lines, derr := jjSvc.DiffSummaryLinesFromTo(summaryCtx, "@-", "@")
	var diffSummary string
	if derr != nil {
		diffSummary = "(jj diff @- @ --summary failed: " + derr.Error() + ")"
	} else {
		diffSummary = strings.TrimSpace(strings.Join(lines, "\n"))
	}
	if len(diffSummary) > evologChainPreviewMaxSummary {
		diffSummary = diffSummary[:evologChainPreviewMaxSummary] + "\n…(truncated)"
	}
	parentLine, _ := jjSvc.GetCommitDescription(ctx, "@-")
	childLine, _ := jjSvc.GetCommitDescription(ctx, "@")
	var ub strings.Builder
	ub.WriteString(user)
	fmt.Fprintf(&ub, "\n\n## Current @- description (first line)\n%s\n\n## Current @ description (first line)\n%s\n",
		strings.TrimSpace(parentLine), strings.TrimSpace(childLine))
	ub.WriteString("\n## Current working-copy change (jj diff --summary @- → @)\n")
	ub.WriteString(diffSummary)
	ub.WriteString("\n")

	provider, err := llm.NewProviderForConfig(cfg)
	if err != nil {
		return
	}
	raw, err := provider.Complete(ctx, evologChainPreviewSystem, ub.String())
	if err != nil {
		return
	}
	steps, pd, cd, err := parseEvologChainPreviewJSON(raw)
	if err != nil {
		return
	}
	msg.ExpectedOutcomeChain = steps
	msg.PrecomputedDescribeParent = pd
	msg.PrecomputedDescribeChild = cd
}
