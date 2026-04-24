package ai

import (
	"context"
	"fmt"
	"path/filepath"
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

// pathsFromJJSummaryLines extracts repo-relative paths from `jj diff --summary` lines (e.g. "M foo.go").
func pathsFromJJSummaryLines(lines []string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		p := normalizeRepoPathForDiff(parts[1])
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
	for _, f := range suggested {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		key := normalizeRepoPathForDiff(f)
		if key == "" {
			continue
		}
		if _, ok := allowed[key]; ok {
			kept = append(kept, key)
		}
	}
	if len(suggested) > 0 && len(kept) == 0 {
		return nil, false, fmt.Errorf("no suggested file paths appear in the diff for this split boundary")
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
		if _, ok := allowed[key]; !ok {
			note = note + " dropped unknown path " + f + ";"
		}
	}
	return kept, strings.TrimSpace(note), nil
}
