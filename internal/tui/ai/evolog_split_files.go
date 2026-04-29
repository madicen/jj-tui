package ai

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/madicen/jj-tui/internal/integrations/jj"
)

// normalizeRepoPathForDiff canonicalizes repo-relative paths for comparison with LLM output.
func normalizeRepoPathForDiff(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = filepath.Clean(p)
	p = filepath.ToSlash(p)
	p = strings.TrimPrefix(p, "./")
	// LLM output sometimes uses leading slash; jj summary/name-only are repo-relative.
	for strings.HasPrefix(p, "/") {
		p = strings.TrimPrefix(p, "/")
	}
	return p
}

// matchAllowedPath maps a normalized model path to a key in allowed when the model echoes git-style
// a/… or b/… prefixes, uses different casing than jj, etc.
func matchAllowedPath(allowed map[string]struct{}, key string) (canonical string, ok bool) {
	if key == "" || len(allowed) == 0 {
		return "", false
	}
	if _, ok := allowed[key]; ok {
		return key, true
	}
	for _, alt := range []string{
		strings.TrimPrefix(key, "a/"),
		strings.TrimPrefix(key, "b/"),
	} {
		if alt != "" && alt != key {
			if _, ok := allowed[alt]; ok {
				return alt, true
			}
		}
	}
	for cand := range allowed {
		if strings.EqualFold(cand, key) {
			return cand, true
		}
		for _, alt := range []string{strings.TrimPrefix(key, "a/"), strings.TrimPrefix(key, "b/")} {
			if alt != "" && strings.EqualFold(cand, alt) {
				return cand, true
			}
		}
	}
	return "", false
}

func formatSuggestedOriginalSample(suggested []string, max int) string {
	var parts []string
	for _, f := range suggested {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		parts = append(parts, f)
		if len(parts) >= max {
			break
		}
	}
	if len(parts) == 0 {
		return "(empty)"
	}
	sort.Strings(parts)
	s := strings.Join(parts, ", ")
	if nonEmptySuggestedCount(suggested) > len(parts) {
		return s + " …"
	}
	return s
}

func formatPathSetSample(m map[string]struct{}, max int) string {
	if len(m) == 0 {
		return "(none)"
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) > max {
		return strings.Join(keys[:max], ", ") + fmt.Sprintf(" … +%d", len(keys)-max)
	}
	return strings.Join(keys, ", ")
}

func nonEmptySuggestedCount(suggested []string) int {
	n := 0
	for _, f := range suggested {
		if strings.TrimSpace(f) != "" {
			n++
		}
	}
	return n
}

// pathsFromJJSummaryLines extracts repo-relative paths from `jj diff --summary` lines (e.g. "M foo.go").
func pathsFromJJSummaryLines(lines []string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		// Status is first token (A/M/D/…); path may contain spaces.
		p := normalizeRepoPathForDiff(strings.Join(fields[1:], " "))
		if p != "" {
			out[p] = struct{}{}
		}
	}
	return out
}

// filterEvologSplitFilePaths keeps only paths that appear in allowed (from jj diff --summary for one evolog step).
// If the model lists every changed path, fullPartition is true and kept is nil (caller should skip file-phase split).
// Returns an error only when the model suggested paths but none match allowed.
func filterEvologSplitFilePaths(allowed map[string]struct{}, suggested []string) (kept []string, fullPartition bool, err error) {
	seen := make(map[string]struct{})
	for _, f := range suggested {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		key := normalizeRepoPathForDiff(f)
		if key == "" {
			continue
		}
		canonical, ok := matchAllowedPath(allowed, key)
		if !ok {
			continue
		}
		if _, dup := seen[canonical]; dup {
			continue
		}
		seen[canonical] = struct{}{}
		kept = append(kept, canonical)
	}
	if nonEmptySuggestedCount(suggested) > 0 && len(kept) == 0 {
		return nil, false, fmt.Errorf("no suggested file paths appear in the diff for this split boundary (model: %s; jj diff for this evolog step — from newer row to row above: %s)",
			formatSuggestedOriginalSample(suggested, 8),
			formatPathSetSample(allowed, 12))
	}
	if len(allowed) > 0 && len(kept) == len(allowed) && len(kept) > 0 {
		// jj split with filesets cannot move every path into the peeled commit — @ would have no tree delta.
		return nil, true, nil
	}
	return kept, false, nil
}

// ValidateAndFilterEvologSplitFiles loads jj diff --summary from the chosen base row to the row above
// (same hop as the "## Steps" section in the evolog-split prompt) and filters LLM paths.
func ValidateAndFilterEvologSplitFiles(ctx context.Context, jjSvc *jj.Service, entries []jj.EvologEntry, pickIdx int, files []string) ([]string, string, error) {
	if len(files) == 0 || pickIdx < 1 || pickIdx >= len(entries) {
		return files, "", nil
	}
	from := strings.TrimSpace(entries[pickIdx].CommitID)
	to := strings.TrimSpace(entries[pickIdx-1].CommitID)
	if from == "" || to == "" {
		return nil, "", fmt.Errorf("missing commit ids for file validation")
	}
	lines, err := jjSvc.DiffSummaryLinesFromTo(ctx, from, to)
	if err != nil {
		return nil, "", fmt.Errorf("diff summary for file validation: %w", err)
	}
	allowed := pathsFromJJSummaryLines(lines)
	// Also accept paths from `jj diff --name-only` for the same hop. Some jj versions or change shapes
	// can make --summary lines parse poorly while --name-only still lists the correct set; this avoids
	// rejecting valid model paths that match the real diff.
	if namePaths, nerr := jjSvc.DiffNameOnlyLinesFromTo(ctx, from, to); nerr == nil {
		for _, p := range namePaths {
			p = normalizeRepoPathForDiff(p)
			if p != "" {
				allowed[p] = struct{}{}
			}
		}
	}
	kept, fullPartition, err := filterEvologSplitFilePaths(allowed, files)
	if err != nil {
		return nil, "", err
	}
	if fullPartition {
		return nil, "file-level split skipped: every changed path was listed (jj needs at least one path to stay on @) — use a subset, or use multiple FAQ peels via split_base_commit_ids", nil
	}
	var note string
	for _, f := range files {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		key := normalizeRepoPathForDiff(f)
		if key == "" {
			continue
		}
		if _, ok := matchAllowedPath(allowed, key); !ok {
			note = note + " dropped unknown path " + f + ";"
		}
	}
	return kept, strings.TrimSpace(note), nil
}
