package ai

import (
	"context"
	"fmt"
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
	b.WriteString("\n## Hunk reference (git unified diff excerpts per step; for each path, hunks are numbered 0,1,… in file order)\n")
	b.WriteString("Use \"hunk_peel_rounds\" for multiple sequential jj splits on @ (one map per split). \"hunk_prefix_first_commit\" is a single peel (same as one element of hunk_peel_rounds). path → k: first k hunks go into the child commit; k < total hunks per path; at least one path must keep a remainder after each peel.\n")
	b.WriteString("Sections that say \"Binary files … differ\" have no @@ hunks — do not put those paths in hunk maps with k>0; use files_first_commit in the main JSON to peel the whole file.\n")
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

// ValidateEvologHunkPrefixAgainstStep checks prefix counts against the git diff for the evolog hop at pickIdx.
func ValidateEvologHunkPrefixAgainstStep(ctx context.Context, jjSvc *jj.Service, entries []jj.EvologEntry, pickIdx int, prefix map[string]int) (map[string]int, string, error) {
	if jjSvc == nil || len(prefix) == 0 {
		return nil, "", nil
	}
	if pickIdx < 1 || pickIdx >= len(entries) {
		return nil, "", fmt.Errorf("invalid pick index for hunk validation")
	}
	from := strings.TrimSpace(entries[pickIdx].CommitID)
	to := strings.TrimSpace(entries[pickIdx-1].CommitID)
	if from == "" || to == "" {
		return nil, "", fmt.Errorf("missing commit ids for hunk validation")
	}
	valCtx, cancel := context.WithTimeout(ctx, evologSplitFileValidateTimeout)
	defer cancel()
	diff, err := jjSvc.GitFormatDiffFromTo(valCtx, from, to, evologHunkSplitValidateMaxBytes)
	if err != nil {
		return nil, "", fmt.Errorf("git diff for hunk validation: %w", err)
	}
	hunksPerPath, binaryPaths, err := jj.ParseGitUnifiedHunksPerPath(diff)
	if err != nil {
		return nil, "", err
	}
	clean := make(map[string]int)
	var note strings.Builder
	var dropped []string
	for orig, k := range prefix {
		key := normalizeRepoPathForDiff(orig)
		if key == "" {
			continue
		}
		if _, bin := binaryPaths[key]; bin {
			if k != 0 {
				return nil, "", fmt.Errorf("binary path %q cannot use hunk peel (k=%d); use files_first_commit for a whole-file peel or k=0", orig, k)
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
	if len(clean) == 0 {
		if len(prefix) > 0 {
			return nil, strings.TrimSpace(note.String()), fmt.Errorf("no valid hunk_prefix_first_commit paths in this step diff")
		}
		return nil, strings.TrimSpace(note.String()), nil
	}
	if err := jj.ValidateHunkPrefixPlan(diff, clean); err != nil {
		return nil, strings.TrimSpace(note.String()), err
	}
	return clean, strings.TrimSpace(note.String()), nil
}

const evologHunkSplitValidateMaxBytes = 4 << 20
