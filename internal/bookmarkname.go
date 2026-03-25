package internal

import "strings"

// LocalBookmarkName returns the jj bookmark name for revsets and commands like name@origin
// (strips one @remote suffix, e.g. feature@origin -> feature).
func LocalBookmarkName(b string) string {
	b = strings.TrimSpace(b)
	if i := strings.Index(b, "@"); i > 0 {
		return b[:i]
	}
	return b
}

// FirstOperableBookmarkName returns a name suitable for jj bookmark delete/set/move.
// Prefers a branch token without @remote; otherwise strips the remote suffix from the first entry.
func FirstOperableBookmarkName(branches []string) string {
	for _, b := range branches {
		b = strings.TrimSpace(b)
		if b == "" || strings.Contains(b, "@") {
			continue
		}
		return b
	}
	if len(branches) > 0 {
		return LocalBookmarkName(strings.TrimSpace(branches[0]))
	}
	return ""
}
