package graph

import "testing"

func TestBookmarkNameForOriginSplit(t *testing.T) {
	t.Parallel()
	if got := bookmarkNameForOriginSplit([]string{"main", "feat-x"}); got != "feat-x" {
		t.Fatalf("got %q, want feat-x", got)
	}
	if got := bookmarkNameForOriginSplit([]string{"main"}); got != "" {
		t.Fatalf("main only: got %q, want empty", got)
	}
	if got := bookmarkNameForOriginSplit([]string{"fix", "other"}); got != "fix" {
		t.Fatalf("got %q, want fix", got)
	}
}
