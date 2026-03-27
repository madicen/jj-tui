package jj

import (
	"strings"
	"testing"
)

func TestBookmarkListParseOriginDivergence_aheadBehindCandidate(t *testing.T) {
	sample := `main: abc 11111111 root
  @git: abc 11111111 root
  @origin: abc 11111111 root
vhs/conflict-feature: def 22222222 feat amended
  @git: def 22222222 feat amended
  @origin (ahead by 1 commits, behind by 1 commits): def/1 33333333 (hidden) feat
`
	stated, ahBoth := bookmarkListParseOriginDivergence(sample)
	if len(stated) != 0 {
		t.Fatalf("unexpected conflicted stated: %v", stated)
	}
	if !ahBoth["vhs/conflict-feature"] {
		t.Fatalf("expected vhs/conflict-feature in aheadBehindBothNonZero, got ahBoth=%v", ahBoth)
	}
	if ahBoth["main"] {
		t.Fatalf("did not expect main in ahBoth: %v", ahBoth)
	}
}

func TestBookmarkListParseOriginDivergence_conflictedWord(t *testing.T) {
	sample := `topic: x y msg
  @origin (conflicted): a b other
`
	stated, ahBoth := bookmarkListParseOriginDivergence(sample)
	if !stated["topic"] {
		t.Fatalf("expected topic in conflictedStated: %v", stated)
	}
	if len(ahBoth) != 0 {
		t.Fatalf("unexpected ahBoth: %v", ahBoth)
	}
}

func TestBookmarkListParseOriginDivergence_aheadZeroNotCandidate(t *testing.T) {
	// jj prints both "ahead" and "behind" even when ahead is 0 (merged / behind-only vs remembered tip).
	sample := `feature: abc 11111111 tip
  @git: abc 11111111 tip
  @origin (ahead by 0 commits, behind by 14 commits): abc 11111111 tip
`
	stated, ahBoth := bookmarkListParseOriginDivergence(sample)
	if len(stated) != 0 || len(ahBoth) != 0 {
		t.Fatalf("expected no candidates, stated=%v ahBoth=%v", stated, ahBoth)
	}
}

func TestParseBookmarkListRemoteLine_qualifiedOrigin(t *testing.T) {
	r, info, ok := parseBookmarkListRemoteLine("@origin (ahead by 1 commits, behind by 1 commits): foo bar baz")
	if !ok || r != "origin" || !strings.Contains(info, "foo") {
		t.Fatalf("got %q %q %v", r, info, ok)
	}
}

func TestChangeIDRootKey(t *testing.T) {
	if got, want := changeIDRootKey("olxoxuzz/11"), "olxoxuzz"; got != want {
		t.Fatalf("changeIDRootKey = %q, want %q", got, want)
	}
	if got, want := changeIDRootKey("  AbCd "), "abcd"; got != want {
		t.Fatalf("changeIDRootKey = %q, want %q", got, want)
	}
}
