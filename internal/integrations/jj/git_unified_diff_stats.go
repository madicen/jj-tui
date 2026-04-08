package jj

import (
	"strconv"
	"strings"
)

// gitLineCounts holds per-file + / − line counts from a git-format unified diff.
type gitLineCounts struct {
	added, removed int
}

// parseGitUnifiedDiffStats maps repo-relative paths (the "b/" side of each diff --git hunk)
// to inserted/deleted line counts. Used to enrich DiffChangedFilesFromTo when jj only
// provides a path list via --summary.
func parseGitUnifiedDiffStats(gitDiff string) map[string]gitLineCounts {
	out := make(map[string]gitLineCounts)
	lines := strings.Split(gitDiff, "\n")
	var cur string
	var acc gitLineCounts
	flush := func() {
		if cur != "" {
			out[cur] = acc
		}
		cur = ""
		acc = gitLineCounts{}
	}
	for _, raw := range lines {
		line := strings.TrimSuffix(raw, "\r")
		if strings.HasPrefix(line, "diff --git ") {
			flush()
			if np, ok := parseDiffGitBPath(line); ok {
				cur = np
			}
			continue
		}
		if cur == "" {
			continue
		}
		if strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- ") {
			continue
		}
		if len(line) > 0 && line[0] == '+' {
			acc.added++
			continue
		}
		if len(line) > 0 && line[0] == '-' {
			acc.removed++
			continue
		}
	}
	flush()
	return out
}

// parseDiffGitBPath extracts the new path from a "diff --git a/… b/…" line.
func parseDiffGitBPath(line string) (string, bool) {
	const prefix = "diff --git "
	if !strings.HasPrefix(line, prefix) {
		return "", false
	}
	rest := strings.TrimPrefix(line, prefix)
	var rawB string
	if len(rest) > 0 && rest[0] == '"' {
		var ok bool
		rawB, ok = bPathFromQuotedDiffGit(rest)
		if !ok {
			return "", false
		}
	} else {
		idx := strings.Index(rest, " b/")
		if idx < 0 {
			return "", false
		}
		rawB = strings.TrimSpace(rest[idx+len(" b/"):])
	}
	pathB := unquoteGitPath(rawB)
	if pathB == "" {
		return "", false
	}
	return pathB, true
}

// bPathFromQuotedDiffGit handles `diff --git "a/…" "b/…"` (paths with spaces).
func bPathFromQuotedDiffGit(rest string) (string, bool) {
	rest = strings.TrimSpace(rest)
	if len(rest) < 2 || rest[0] != '"' {
		return "", false
	}
	end1 := endOfGitQuotedPath(rest, 0)
	if end1 < 0 {
		return "", false
	}
	rest2 := strings.TrimSpace(rest[end1:])
	if len(rest2) < 2 || rest2[0] != '"' {
		return "", false
	}
	end2 := endOfGitQuotedPath(rest2, 0)
	if end2 < 0 {
		return "", false
	}
	secondField := rest2[:end2]
	u, err := strconv.Unquote(secondField)
	if err != nil {
		return "", false
	}
	u = strings.TrimPrefix(u, "b/")
	return u, true
}

// endOfGitQuotedPath returns the index just past the closing double-quote of a git-quoted path
// starting at start (must point at opening '"').
func endOfGitQuotedPath(s string, start int) int {
	if start >= len(s) || s[start] != '"' {
		return -1
	}
	i := start + 1
	for i < len(s) {
		switch s[i] {
		case '\\':
			if i+1 >= len(s) {
				return -1
			}
			i += 2
		case '"':
			return i + 1
		default:
			i++
		}
	}
	return -1
}

func unquoteGitPath(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' {
		if u, err := strconv.Unquote(s); err == nil {
			return u
		}
	}
	return s
}
