package util

import "testing"

func TestNormalizeBookmarkListToken(t *testing.T) {
	tests := []struct {
		token       string
		wantName    string
		wantQMarker bool
	}{
		{"", "", false},
		{"  ", "", false},
		{"feature", "feature", false},
		{"feature*", "feature", false},
		{"feature?", "feature", true},
		{"feature?*", "feature", true},
		{"feature*?", "feature", true},
		{"feature??", "feature", true},
		{"my/feature-1", "my/feature-1", false},
		// jj bookmark list may append display-only " (conflicted)" before ':' — not part of the revset name.
		{"madicen/APP-429 (conflicted)", "madicen/APP-429", true},
		{"feat (diverged)", "feat", true},
		{"wip (CONFLICTED)*", "wip", true},
	}
	for _, tt := range tests {
		name, q := NormalizeBookmarkListToken(tt.token)
		if name != tt.wantName || q != tt.wantQMarker {
			t.Errorf("NormalizeBookmarkListToken(%q) = (%q, %v); want (%q, %v)", tt.token, name, q, tt.wantName, tt.wantQMarker)
		}
	}
}

func TestFirstOperableBookmarkName_stripsConflictMarker(t *testing.T) {
	got := FirstOperableBookmarkName([]string{"wip?"})
	if got != "wip" {
		t.Fatalf("FirstOperableBookmarkName = %q; want wip", got)
	}
}

func TestBookmarkNameForRevset_stripsJJListLabel(t *testing.T) {
	got := BookmarkNameForRevset("madicen/APP-429-svc-github-improvments (conflicted)")
	want := "madicen/APP-429-svc-github-improvments"
	if got != want {
		t.Fatalf("BookmarkNameForRevset = %q; want %q", got, want)
	}
}

func TestRevsetQuotedSymbol(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", `""`},
		{`simple`, `"simple"`},
		{`madicen/APP-429`, `"madicen/APP-429"`},
		{`a"b`, `"a\"b"`},
		{`a\b`, `"a\\b"`},
		{`foo@origin`, `"foo@origin"`},
	}
	for _, tt := range tests {
		if got := RevsetQuotedSymbol(tt.in); got != tt.want {
			t.Errorf("RevsetQuotedSymbol(%q) = %q; want %q", tt.in, got, tt.want)
		}
	}
}
func TestRevsetExactPattern(t *testing.T) {
	if got, want := RevsetExactPattern("madicen/x"), `exact:"madicen/x"`; got != want {
		t.Fatalf("RevsetExactPattern = %q; want %q", got, want)
	}
}
