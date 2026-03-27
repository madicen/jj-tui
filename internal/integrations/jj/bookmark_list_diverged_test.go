package jj

import (
	"strings"
	"testing"
)

func TestBookmarkListMarksOriginDiverged(t *testing.T) {
	sample := `main: abc 11111111 root
  @git: abc 11111111 root
  @origin: abc 11111111 root
vhs/conflict-feature: def 22222222 feat amended
  @git: def 22222222 feat amended
  @origin (ahead by 1 commits, behind by 1 commits): def/1 33333333 (hidden) feat
`
	got := bookmarkListMarksOriginDiverged(sample)
	if !got["vhs/conflict-feature"] {
		t.Fatalf("expected vhs/conflict-feature diverged, got %v", got)
	}
	if got["main"] {
		t.Fatalf("did not expect main: %v", got)
	}
}

func TestBookmarkListMarksOriginDiverged_conflictedWord(t *testing.T) {
	sample := `topic: x y msg
  @origin (conflicted): a b other
`
	got := bookmarkListMarksOriginDiverged(sample)
	if !got["topic"] {
		t.Fatalf("expected topic from conflicted line: %v", got)
	}
}

func TestBookmarkListMarksOriginDiverged_aheadZeroNotDiverged(t *testing.T) {
	// jj prints both "ahead" and "behind" even when ahead is 0 (merged / behind-only vs remembered tip).
	// That is not a bookmark fork; do not block delete or show ⚠ diverged.
	sample := `feature: abc 11111111 tip
  @git: abc 11111111 tip
  @origin (ahead by 0 commits, behind by 14 commits): abc 11111111 tip
`
	got := bookmarkListMarksOriginDiverged(sample)
	if got["feature"] {
		t.Fatalf("expected no divergence for ahead-by-0, got %v", got)
	}
}

func TestParseBookmarkListRemoteLine_qualifiedOrigin(t *testing.T) {
	r, info, ok := parseBookmarkListRemoteLine("@origin (ahead by 1 commits, behind by 1 commits): foo bar baz")
	if !ok || r != "origin" || !strings.Contains(info, "foo") {
		t.Fatalf("got %q %q %v", r, info, ok)
	}
}
