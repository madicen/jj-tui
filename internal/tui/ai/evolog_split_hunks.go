package ai

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/madicen/jj-tui/internal/integrations/jj"
)

const evologSplitMaxHunkPreviewSteps = 10
const evologSplitMaxHunkPreviewBytesPerStep = 24_000

// buildEvologHunkHintBlock appends git unified diff excerpts so the model can name @@ hunks by path
// and per-file hunk index (0..n-1) for hunk_prefix_first_commit.
func buildEvologHunkHintBlock(ctx context.Context, jjSvc *jj.Service, entries []jj.EvologEntry, stepLimit int) (string, error) {
	if jjSvc == nil || len(entries) < 2 || stepLimit < 1 {
		return "", nil
	}
	n := len(entries)
	maxStep := min(stepLimit, evologSplitMaxHunkPreviewSteps, n-1)
	var b strings.Builder
	b.WriteString("\n## Hunk reference (git unified diff excerpts per step; each @@ block is one hunk in file order)\n")
	b.WriteString("Use \"hunk_peel_rounds\" for multiple sequential jj splits on @ (one map per split). \"hunk_prefix_first_commit\" is a single peel (same as one element of hunk_peel_rounds). path → k: k is how many leading @@ hunks move to the first commit of that split (1 <= k <= H-1 for H hunks on that path; if H is 1 use files_first_commit for the whole file).\n")
	b.WriteString("Sections that say \"Binary files … differ\" have no @@ hunks — do not put those paths in hunk maps with k>0; use files_first_commit in the main JSON to peel the whole file.\n")
	b.WriteString("Use repo-relative paths exactly as in each diff's \"b/…\" side (directory prefixes matter). If a path is wrong for this step, the client drops the hunk peel and still runs the FAQ split.\n")
	for i := 1; i <= maxStep; i++ {
		from := strings.TrimSpace(entries[i].CommitID)
		to := strings.TrimSpace(entries[i-1].CommitID)
		if from == "" || to == "" {
			continue
		}
		diff, err := jjSvc.GitFormatDiffFromTo(ctx, from, to, evologSplitMaxHunkPreviewBytesPerStep)
		if err != nil {
			return "", fmt.Errorf("hunk hint step %d: %w", i, err)
		}
		if strings.TrimSpace(diff) == "" {
			continue
		}
		fmt.Fprintf(&b, "\n### Step %d (row %d → row %d)\n", i, i, i-1)
		b.WriteString(trimHunkHintDiff(diff))
	}
	return b.String(), nil
}

func trimHunkHintDiff(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	var out []string
	for _, ln := range lines {
		if strings.HasPrefix(ln, "index ") {
			continue
		}
		out = append(out, ln)
		if len(out) > 400 {
			out = append(out, "…(truncated for prompt size)")
			break
		}
	}
	return strings.Join(out, "\n")
}

// resolveEvologHunkPathKey maps an LLM path to a key present in hunksPerPath or binaryPaths.
// It tries normalized exact match, then a unique basename match, then a unique suffix match.
func resolveEvologHunkPathKey(norm string, hunksPerPath map[string][]jj.UnifiedHunk, binaryPaths map[string]struct{}) (string, bool) {
	if norm == "" {
		return "", false
	}
	if _, ok := binaryPaths[norm]; ok {
		return norm, true
	}
	if _, ok := hunksPerPath[norm]; ok {
		return norm, true
	}
	base := filepath.Base(norm)
	if base != "." && base != "" && base != "/" {
		candidates := make(map[string]struct{})
		for k := range hunksPerPath {
			if filepath.Base(k) == base {
				candidates[k] = struct{}{}
			}
		}
		for k := range binaryPaths {
			if filepath.Base(k) == base {
				candidates[k] = struct{}{}
			}
		}
		if len(candidates) == 1 {
			for k := range candidates {
				return k, true
			}
		}
	}
	var suff []string
	seen := make(map[string]struct{})
	add := func(k string) {
		if k == "" {
			return
		}
		if k == norm || strings.HasSuffix(k, "/"+norm) {
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				suff = append(suff, k)
			}
		}
	}
	for k := range hunksPerPath {
		add(k)
	}
	for k := range binaryPaths {
		add(k)
	}
	if len(suff) == 1 {
		return suff[0], true
	}
	return "", false
}

// stripSingleHunkPeelPaths removes paths whose unified diff has only one @@ hunk but k>0 (cannot
// hunk-peel). Those paths should be listed in files_first_commit for a whole-file jj split instead.
func stripSingleHunkPeelPaths(diff string, clean map[string]int) (repaired map[string]int, autoFiles []string, extraNote string) {
	if len(clean) == 0 {
		return clean, nil, ""
	}
	hunksPerPath, _, err := jj.ParseGitUnifiedHunksPerPath(diff)
	if err != nil {
		return clean, nil, ""
	}
	repaired = make(map[string]int, len(clean))
	seenFile := map[string]struct{}{}
	for p, k := range clean {
		h := hunksPerPath[p]
		if len(h) == 1 && k > 0 {
			if _, ok := seenFile[p]; !ok {
				seenFile[p] = struct{}{}
				autoFiles = append(autoFiles, p)
			}
			continue
		}
		repaired[p] = k
	}
	if len(autoFiles) == 0 {
		return clean, nil, ""
	}
	var b strings.Builder
	b.WriteString(" single-@@ paths moved to files_first_commit: ")
	b.WriteString(strings.Join(autoFiles, ", "))
	b.WriteString(".")
	return repaired, autoFiles, b.String()
}

const evologHunkSplitValidateMaxBytes = 4 << 20

func validateEvologHunkPrefixAgainstGitDiff(diff string, prefix map[string]int) (map[string]int, []string, string, error) {
	hunksPerPath, binaryPaths, err := jj.ParseGitUnifiedHunksPerPath(diff)
	if err != nil {
		return nil, nil, "", err
	}
	clean := make(map[string]int)
	var note strings.Builder
	var dropped []string
	for orig, k := range prefix {
		norm := normalizeRepoPathForDiff(orig)
		if norm == "" {
			continue
		}
		key, ok := resolveEvologHunkPathKey(norm, hunksPerPath, binaryPaths)
		if !ok {
			if k != 0 {
				dropped = append(dropped, orig)
			}
			continue
		}
		if _, bin := binaryPaths[key]; bin {
			if k != 0 {
				return nil, nil, "", fmt.Errorf("binary path %q cannot use hunk peel (k=%d); use files_first_commit for a whole-file peel or k=0", orig, k)
			}
			clean[key] = 0
			continue
		}
		if _, has := hunksPerPath[key]; !has {
			if k != 0 {
				dropped = append(dropped, orig)
			}
			continue
		}
		clean[key] = k
	}
	if len(dropped) > 0 {
		note.WriteString(" dropped hunk paths not in this step diff;")
	}
	var autoFiles []string
	clean, autoFiles, extra := stripSingleHunkPeelPaths(diff, clean)
	if extra != "" {
		note.WriteString(extra)
	}
	if len(clean) == 0 {
		if len(autoFiles) > 0 {
			return nil, autoFiles, strings.TrimSpace(note.String()), nil
		}
		if len(prefix) > 0 {
			note.WriteString(" no hunk keys matched this step's unified diff; omitting hunk peel (FAQ split still applies).")
			return nil, nil, strings.TrimSpace(note.String()), nil
		}
		return nil, nil, strings.TrimSpace(note.String()), nil
	}
	if err := jj.ValidateHunkPrefixPlan(diff, clean); err != nil {
		return nil, autoFiles, strings.TrimSpace(note.String()), err
	}
	return clean, autoFiles, strings.TrimSpace(note.String()), nil
}

// ValidateEvologHunkPrefixAgainstStep checks prefix counts against the same git diff the client will
// use for hunk peels: `jj diff --from @- --to @` on the working copy. That matches SplitRevisionByHunkPrefix
// and jj split's left/right trees; it is not the evolog row hop at pickIdx (those differ whenever
// recommended_index != 1 or the tip is not a single-parent chain matching the table).
//
// entries and pickIdx are kept for callers; pickIdx must still refer to a valid parent row (1..len-1).
//
// autoFiles lists paths that had only one @@ hunk but k>0 — merge into files_first_commit so the client can whole-file peel them before hunk rounds.
func ValidateEvologHunkPrefixAgainstStep(ctx context.Context, jjSvc *jj.Service, entries []jj.EvologEntry, pickIdx int, prefix map[string]int) (map[string]int, []string, string, error) {
	if jjSvc == nil || len(prefix) == 0 {
		return nil, nil, "", nil
	}
	if len(entries) < 2 {
		return nil, nil, "", fmt.Errorf("need evolog context for hunk validation")
	}
	if pickIdx < 1 || pickIdx >= len(entries) {
		return nil, nil, "", fmt.Errorf("invalid pick index for hunk validation")
	}
	return ValidateEvologHunkPrefixAgainstWorkingCopy(ctx, jjSvc, prefix)
}

// ValidateEvologHunkPrefixAgainstWorkingCopy validates hunk prefix maps against jj diff @- → @ (same
// basis as SplitRevisionByHunkPrefix and the evolog hunk diff-editor spec).
func ValidateEvologHunkPrefixAgainstWorkingCopy(ctx context.Context, jjSvc *jj.Service, prefix map[string]int) (map[string]int, []string, string, error) {
	if jjSvc == nil || len(prefix) == 0 {
		return nil, nil, "", nil
	}
	valCtx, cancel := context.WithTimeout(ctx, evologSplitFileValidateTimeout)
	defer cancel()
	diff, err := jjSvc.GitFormatDiffFromTo(valCtx, "@-", "@", evologHunkSplitValidateMaxBytes)
	if err != nil {
		return nil, nil, "", fmt.Errorf("git diff @- → @ for hunk validation: %w", err)
	}
	return validateEvologHunkPrefixAgainstGitDiff(diff, prefix)
}
