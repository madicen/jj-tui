package jj

import (
	"fmt"
	"strconv"
	"strings"
)

// UnifiedHunk is one @@ … @@ block from a git unified diff (one file section).
type UnifiedHunk struct {
	OldStart int // 1-based line in old file
	OldCount int // 0 means empty span
	NewStart int
	NewCount int
	Lines    []string // including prefix + - space or "\ No newline..."
}

// ParseGitUnifiedHunksPerPath parses a git unified diff into ordered @@ hunks per b/ path, plus
// binaryPaths for "Binary files … differ" sections (no @@). Hunk split only allows k=0 for those
// (parent snapshot on the peeled side; delta stays on @); use jj split -- paths / files_first_commit
// to move a whole binary file into the child.
func ParseGitUnifiedHunksPerPath(gitDiff string) (map[string][]UnifiedHunk, map[string]struct{}, error) {
	lines := strings.Split(strings.ReplaceAll(gitDiff, "\r\n", "\n"), "\n")
	out := make(map[string][]UnifiedHunk)
	binaryPaths := make(map[string]struct{})
	var curPath string
	var cur []UnifiedHunk
	var curH *UnifiedHunk
	flushFile := func() {
		if curPath != "" {
			// Finish the in-progress @@ hunk before switching files or at EOF; otherwise the
			// previous file's last hunk is dropped when the next diff --git line appears.
			if curH != nil {
				cur = append(cur, *curH)
				curH = nil
			}
			if len(cur) > 0 {
				out[curPath] = append(out[curPath], cur...)
			}
		}
		curPath = ""
		cur = nil
		curH = nil
	}
	for _, raw := range lines {
		line := raw
		if strings.HasPrefix(line, "diff --git ") {
			flushFile()
			p, ok := parseDiffGitBPath(line)
			if !ok {
				continue
			}
			curPath = p
			cur = nil
			curH = nil
			continue
		}
		if curPath == "" {
			continue
		}
		if strings.HasPrefix(line, "Binary files ") && strings.Contains(line, " differ") {
			if curPath != "" {
				binaryPaths[curPath] = struct{}{}
			}
			continue
		}
		if strings.HasPrefix(line, "@@ ") {
			if curH != nil {
				cur = append(cur, *curH)
			}
			h, err := parseUnifiedHunkHeader(line)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %w", curPath, err)
			}
			curH = &h
			continue
		}
		if curH == nil {
			continue
		}
		if len(line) > 0 {
			switch line[0] {
			case '+', '-', ' ', '\\':
				curH.Lines = append(curH.Lines, line)
			}
		}
	}
	flushFile()
	return out, binaryPaths, nil
}

func parseUnifiedHunkHeader(line string) (UnifiedHunk, error) {
	// @@ -l[,n] +l[,m] @@ optional tail
	const p = "@@ "
	if !strings.HasPrefix(line, p) {
		return UnifiedHunk{}, fmt.Errorf("bad hunk header: %q", line)
	}
	rest := strings.TrimSpace(line[len(p):])
	end := strings.Index(rest, " @@")
	if end < 0 {
		return UnifiedHunk{}, fmt.Errorf("bad hunk header: %q", line)
	}
	mid := rest[:end]
	parts := strings.Fields(mid)
	if len(parts) != 2 {
		return UnifiedHunk{}, fmt.Errorf("bad hunk header ranges: %q", line)
	}
	oldPart := strings.TrimPrefix(parts[0], "-")
	newPart := strings.TrimPrefix(parts[1], "+")
	os, oc, err := parseHunkRange(oldPart)
	if err != nil {
		return UnifiedHunk{}, err
	}
	ns, nc, err := parseHunkRange(newPart)
	if err != nil {
		return UnifiedHunk{}, err
	}
	return UnifiedHunk{OldStart: os, OldCount: oc, NewStart: ns, NewCount: nc}, nil
}

func parseHunkRange(s string) (start, count int, err error) {
	if s == "" {
		return 0, 0, fmt.Errorf("empty range")
	}
	comma := strings.IndexByte(s, ',')
	if comma < 0 {
		start, err = strconv.Atoi(s)
		if err != nil {
			return 0, 0, err
		}
		if start == 0 {
			return 0, 0, nil
		}
		return start, 1, nil
	}
	start, err = strconv.Atoi(s[:comma])
	if err != nil {
		return 0, 0, err
	}
	count, err = strconv.Atoi(s[comma+1:])
	if err != nil {
		return 0, 0, err
	}
	return start, count, nil
}

// splitPatchLines splits file text into lines for patching (no trailing empty string after final newline).
func splitPatchLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if s == "" {
		return nil
	}
	if strings.HasSuffix(s, "\n") {
		s = strings.TrimSuffix(s, "\n")
		if s == "" {
			return nil
		}
		return strings.Split(s, "\n")
	}
	return strings.Split(s, "\n")
}

// ApplyUnifiedHunkPrefix returns new content = old content with the first prefix hunks applied
// (hunks must be ordered by OldStart as in git diff). prefix 0 yields orig unchanged.
func ApplyUnifiedHunkPrefix(orig string, hunks []UnifiedHunk, prefix int) (string, error) {
	if prefix < 0 || prefix > len(hunks) {
		return "", fmt.Errorf("hunk prefix %d out of range (0..%d)", prefix, len(hunks))
	}
	if prefix == 0 {
		return orig, nil
	}
	ol := splitPatchLines(orig)
	nl, err := applyHunksToOriginalLines(ol, hunks[:prefix])
	if err != nil {
		return "", err
	}
	return joinPatchLines(nl), nil
}

func joinPatchLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

func applyHunksToOriginalLines(orig []string, hunks []UnifiedHunk) ([]string, error) {
	idx := 0
	var out []string
	for hi := range hunks {
		h := hunks[hi]
		wantStart := h.OldStart - 1
		if wantStart < idx {
			return nil, fmt.Errorf("hunk %d: overlapping or out-of-order patch (cursor %d, want %d)", hi, idx, wantStart)
		}
		if wantStart > len(orig) {
			return nil, fmt.Errorf("hunk %d: old start past EOF", hi)
		}
		out = append(out, orig[idx:wantStart]...)
		idx = wantStart
		for _, ln := range h.Lines {
			if ln == "\\ No newline at end of file" {
				continue
			}
			if len(ln) == 0 {
				continue
			}
			switch ln[0] {
			case ' ':
				want := ln[1:]
				if idx >= len(orig) {
					return nil, fmt.Errorf("hunk %d: context past EOF", hi)
				}
				if orig[idx] != want {
					return nil, fmt.Errorf("hunk %d: context mismatch at line %d", hi, idx+1)
				}
				out = append(out, orig[idx])
				idx++
			case '-':
				want := ln[1:]
				if idx >= len(orig) {
					return nil, fmt.Errorf("hunk %d: delete past EOF", hi)
				}
				if orig[idx] != want {
					return nil, fmt.Errorf("hunk %d: delete mismatch at line %d", hi, idx+1)
				}
				idx++
			case '+':
				out = append(out, ln[1:])
			default:
				return nil, fmt.Errorf("hunk %d: bad line %q", hi, ln)
			}
		}
	}
	out = append(out, orig[idx:]...)
	return out, nil
}

// VerifyUnifiedHunksReconstructRight checks that applying every hunk to left yields right (same
// coordinate system as git unified diff). Call this once per file when validating an LLM prefix.
func VerifyUnifiedHunksReconstructRight(left, right string, hunks []UnifiedHunk) error {
	got, err := ApplyUnifiedHunkPrefix(left, hunks, len(hunks))
	if err != nil {
		return err
	}
	if normalizePatchText(got) != normalizePatchText(right) {
		return fmt.Errorf("applying all hunks does not reconstruct right-hand file")
	}
	return nil
}

func normalizePatchText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.TrimSuffix(s, "\n")
}

// ValidateHunkPrefixPlan checks prefix counts against a git unified diff: each path must exist when
// k > 0, k must be <= number of hunks, and k == len(hunks) is forbidden (would leave nothing on @).
// There must be at least one changed path that still has hunks not fully assigned to the first commit.
func ValidateHunkPrefixPlan(gitDiff string, prefixByPath map[string]int) error {
	if strings.TrimSpace(gitDiff) == "" {
		return fmt.Errorf("empty diff for hunk validation")
	}
	hunksPerPath, binaryPaths, err := ParseGitUnifiedHunksPerPath(gitDiff)
	if err != nil {
		return err
	}
	for p, k := range prefixByPath {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, bin := binaryPaths[p]; bin {
			if k < 0 {
				return fmt.Errorf("negative hunk prefix for %s", p)
			}
			if k != 0 {
				return fmt.Errorf("binary file %s: @@-level split cannot apply (use k=0 here and files_first_commit to peel the whole path, or split the binary outside hunk mode)", p)
			}
			continue
		}
		hunks := hunksPerPath[p]
		if k < 0 {
			return fmt.Errorf("negative hunk prefix for %s", p)
		}
		if k > len(hunks) {
			return fmt.Errorf("hunk prefix k=%d for %q exceeds %d @@ hunks in the current diff (k = count of leading hunks to peel, not a line number — re-count @@ for this path on @ vs @-)", k, p, len(hunks))
		}
		if k > 0 && len(hunks) == 0 {
			return fmt.Errorf("unknown path %s in hunk prefix (not in diff)", p)
		}
		if len(hunks) == 1 && k > 0 {
			return fmt.Errorf("%q: only 1 @@ hunk — hunk peel cannot split this path (need at least 2 hunks); use files_first_commit for the whole file or omit this path from hunk_prefix / hunk_peel_rounds", p)
		}
		if len(hunks) > 0 && k == len(hunks) {
			return fmt.Errorf("cannot assign every @@ hunk of %q to the first commit (k must be < hunk count)", p)
		}
	}
	remainder := false
	for p, hunks := range hunksPerPath {
		k := prefixByPath[p]
		if k < len(hunks) {
			remainder = true
			break
		}
	}
	if !remainder {
		for p := range binaryPaths {
			k := 0
			if kv, ok := prefixByPath[p]; ok {
				k = kv
			}
			if k == 0 {
				remainder = true
				break
			}
		}
	}
	if (len(hunksPerPath) > 0 || len(binaryPaths) > 0) && !remainder {
		return fmt.Errorf("hunk selection would leave no change on @")
	}
	return nil
}
