package jj

import (
	"context"
	"strings"
)

// AbsorbTarget is a mutable ancestor revision that receives (or would receive)
// changes from `jj absorb`.
type AbsorbTarget struct {
	ChangeID    string // change_id shown by jj (usually 8 chars)
	CommitID    string // commit_id shown by jj (usually 8 chars)
	Description string // first line of the target's description
}

// AbsorbPreview summarizes what `jj absorb` would do, derived from a non-mutating
// dry run. When Nothing is true, no changes can be absorbed.
type AbsorbPreview struct {
	Targets []AbsorbTarget
	Summary string // human-readable summary lines, suitable for a preview modal
	Nothing bool
}

// AbsorbDryRun previews `jj absorb` without mutating the repository.
//
// jj 0.43.0 has no `--dry-run` flag for absorb, so we use
// `--no-integrate-operation`: jj computes and reports the absorption but leaves
// the resulting operations uncommitted, so the repo state (op log, working copy)
// is unchanged. The reported target revisions match those of a subsequent real
// `jj absorb`, because absorb is deterministic for a given working-copy state.
func (s *Service) AbsorbDryRun(ctx context.Context) (*AbsorbPreview, error) {
	out, err := s.runJJCombined(ctx, "absorb", "--no-integrate-operation")
	if err != nil {
		return nil, err
	}
	return parseAbsorbOutput(out), nil
}

// Absorb runs `jj absorb`, moving working-copy changes into the closest mutable
// ancestors where the corresponding lines were last modified. The returned
// preview reflects what jj reported it actually did.
func (s *Service) Absorb(ctx context.Context) (*AbsorbPreview, error) {
	out, err := s.runJJCombined(ctx, "absorb")
	if err != nil {
		return nil, err
	}
	return parseAbsorbOutput(out), nil
}

// parseAbsorbOutput extracts the absorb summary from jj's (mostly stderr) output.
//
// A successful absorb prints:
//
//	Absorbed changes into N revisions:
//	  <change_id> <commit_id> <description>
//	  …
//	Rebased M descendant commits.
//
// When nothing can be absorbed jj prints "Nothing changed." Lines emitted by
// --no-integrate-operation ("… left uncommitted because …"), warnings, and hints
// are ignored.
func parseAbsorbOutput(out string) *AbsorbPreview {
	preview := &AbsorbPreview{}
	var summaryLines []string
	inTargets := false
	for _, raw := range strings.Split(out, "\n") {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if trimmed == "Nothing changed." {
			preview.Nothing = true
			summaryLines = append(summaryLines, trimmed)
			continue
		}
		if strings.HasPrefix(trimmed, "Absorbed changes into") {
			inTargets = true
			summaryLines = append(summaryLines, trimmed)
			continue
		}
		// Indented entries under the "Absorbed changes into" header list one
		// target revision each: "<change_id> <commit_id> <description...>".
		if inTargets && (strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t")) {
			if t, ok := parseAbsorbTarget(trimmed); ok {
				preview.Targets = append(preview.Targets, t)
				summaryLines = append(summaryLines, "  "+trimmed)
				continue
			}
		}
		inTargets = false
		if strings.HasPrefix(trimmed, "Rebased ") {
			summaryLines = append(summaryLines, trimmed)
		}
	}
	preview.Summary = strings.Join(summaryLines, "\n")
	if preview.Summary == "" {
		preview.Nothing = true
		preview.Summary = "Nothing changed."
	}
	return preview
}

// parseAbsorbTarget parses one "<change_id> <commit_id> <description...>" entry.
func parseAbsorbTarget(s string) (AbsorbTarget, bool) {
	fields := strings.Fields(s)
	if len(fields) < 2 {
		return AbsorbTarget{}, false
	}
	t := AbsorbTarget{ChangeID: fields[0], CommitID: fields[1]}
	if len(fields) > 2 {
		// Rejoin the remainder using single spaces; the description here is only
		// the first line and used for display, so exact whitespace is not needed.
		t.Description = strings.Join(fields[2:], " ")
	}
	return t, true
}
