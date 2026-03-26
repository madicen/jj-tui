package util

import (
	"strings"
)

// JJExactBookmarkPattern returns a jj string pattern that matches exactly this bookmark name.
// jj bookmark delete/set and git push --bookmark interpret bare names as globs; see:
// https://jj-vcs.github.io/jj/latest/revsets/#string-patterns
func JJExactBookmarkPattern(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return name
	}
	if strings.HasPrefix(name, "exact:") {
		return name
	}
	return "exact:" + name
}

// LocalBookmarkName returns the jj bookmark name for revsets and commands like name@origin
// (strips one @remote suffix, e.g. feature@origin -> feature).
func LocalBookmarkName(b string) string {
	b = strings.TrimSpace(b)
	if i := strings.Index(b, "@"); i > 0 {
		return b[:i]
	}
	return b
}

// NormalizeBookmarkListToken normalizes a bookmark token as emitted by jj graph templates
// or `jj bookmark list`: trims space, removes trailing * (current-bookmark) and ?
// (diverged / conflicted) markers—possibly interleaved—and reports whether any ? was present
// in the raw token.
func NormalizeBookmarkListToken(token string) (name string, hadConflictQuestionMark bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", false
	}
	hadConflictQuestionMark = strings.Contains(token, "?")
	name = token
	for strings.HasSuffix(name, "?") || strings.HasSuffix(name, "*") {
		if strings.HasSuffix(name, "?") {
			name = strings.TrimSuffix(name, "?")
			continue
		}
		name = strings.TrimSuffix(name, "*")
	}
	return name, hadConflictQuestionMark
}

// FirstOperableBookmarkName returns a name suitable for jj bookmark delete/set/move.
// Prefers a branch token without @remote; otherwise strips the remote suffix from the first entry.
func FirstOperableBookmarkName(branches []string) string {
	for _, b := range branches {
		b = strings.TrimSpace(b)
		if b == "" {
			continue
		}
		name, _ := NormalizeBookmarkListToken(b)
		if name == "" || strings.Contains(name, "@") {
			continue
		}
		return name
	}
	if len(branches) > 0 {
		name, _ := NormalizeBookmarkListToken(strings.TrimSpace(branches[0]))
		return LocalBookmarkName(name)
	}
	return ""
}
