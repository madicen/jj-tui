package jj

import (
	"strings"
	"testing"
)

func TestSanitizeBookmarkName(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Implement auth", "Implement_auth"},
		{"Foo, bar", "Foo_bar"},
		{"a---b", "a-b"},
		{"a___b", "a_b"},
		{"  trim  ", "trim"},
		{"-__-x-_", "x"},
		{"café-123", "café-123"},
		{"", ""},
		{"!!!", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := SanitizeBookmarkName(tt.in)
			if got != tt.want {
				t.Errorf("SanitizeBookmarkName(%q) = %q; want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestTruncateBookmarkNameTo(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		max   int
		want  string
	}{
		// Short / boundary inputs pass through verbatim.
		{"empty", "", 50, ""},
		{"under max", "feature-auth", 50, "feature-auth"},
		{"exactly max", "abcde", 5, "abcde"},
		// Hard truncate when there's no help from word boundaries.
		{"hard truncate", "implementauthentication", 10, "implementa"},
		// Truncation must not leave a dangling separator at the end.
		{"trim trailing dash", "feature-auth-system", 8, "feature"},
		{"trim trailing underscore", "feature_auth_system", 8, "feature"},
		{"trim trailing slash", "team/feature/sub", 13, "team/feature"},
		// Realistic AI-generated name that exceeds 50 cols. The first 50 runes happen
		// to land on a '-' boundary ("…crawler-with-"), so the trim-trailing-separator
		// rule removes that dash and we keep 49 visible runes.
		{"long ai name 50",
			"implement-hybrid-backend-for-product-crawler-with-llm-extraction-and-json-ld-fallback",
			50,
			"implement-hybrid-backend-for-product-crawler-with",
		},
		// Multi-byte unicode must be cut on rune boundaries, never mid-codepoint.
		{"unicode", "café-test-with-some-extras", 6, "café-t"},
		{"all unicode", strings.Repeat("é", 100), 5, strings.Repeat("é", 5)},
		// Degenerate caps.
		{"max zero", "anything", 0, ""},
		{"max negative", "anything", -1, ""},
		// Pure separators get hollowed out by the trim, which is correct: the input
		// wasn't a usable bookmark name to begin with.
		{"only separators", "----", 3, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateBookmarkNameTo(tt.in, tt.max)
			if got != tt.want {
				t.Errorf("TruncateBookmarkNameTo(%q, %d) = %q; want %q", tt.in, tt.max, got, tt.want)
			}
			// Output must never end with a separator (the rule the trim enforces).
			if got != "" {
				switch got[len(got)-1] {
				case '-', '_', '/':
					t.Errorf("output ends with a separator: %q", got)
				}
			}
		})
	}
}

func TestTruncateBookmarkName_UsesPackageCap(t *testing.T) {
	// 100 a's should collapse to MaxBookmarkNameLen a's (no separators to trim).
	in := strings.Repeat("a", 100)
	got := TruncateBookmarkName(in)
	if len(got) != MaxBookmarkNameLen {
		t.Fatalf("TruncateBookmarkName length = %d; want %d", len(got), MaxBookmarkNameLen)
	}
	for _, r := range got {
		if r != 'a' {
			t.Fatalf("unexpected rune in output: %q (full %q)", r, got)
		}
	}
}

func TestDefaultGraphRevset(t *testing.T) {
	rs := DefaultGraphRevset
	if !strings.Contains(rs, "mutable()") {
		t.Errorf("DefaultGraphRevset should constrain rows via mutable(); got %q", rs)
	}
	if !strings.Contains(rs, "ancestors(@)") || !strings.Contains(rs, "descendants(@)") {
		t.Errorf("DefaultGraphRevset should tie mutable cone to @; got %q", rs)
	}
	if !strings.Contains(rs, "(parents(@)+)::") {
		t.Errorf("DefaultGraphRevset should include sibling subtree (parents(@)+):: for split branches; got %q", rs)
	}
	if !strings.Contains(rs, "bookmarks() & mine()") {
		t.Errorf("DefaultGraphRevset should pin (bookmarks() & mine()) so other contributors' branch tips don't clutter the view; got %q", rs)
	}
	if !strings.Contains(rs, "trunk()") {
		t.Errorf("DefaultGraphRevset should anchor on trunk(); got %q", rs)
	}
}

func TestApplyMineFilterToRevsetEmpty(t *testing.T) {
	out := ApplyMineFilterToRevset("")
	if !strings.Contains(out, DefaultGraphRevset) {
		t.Errorf("ApplyMineFilterToRevset(\"\") should fall back to DefaultGraphRevset; got %q", out)
	}
	if !strings.Contains(out, "ancestors(mine() | trunk() | @, 2)") {
		t.Errorf("expected mine|trunk|@ ancestor pin in output; got %q", out)
	}
	// Pin terms after the intersection guarantee @ and trunk() are always visible
	// even if the base revset is narrow and excludes them.
	if !strings.HasSuffix(out, "| trunk() | @") {
		t.Errorf("expected output to end with `| trunk() | @` pins; got %q", out)
	}
}

func TestApplyMineFilterToRevsetCustomBase(t *testing.T) {
	base := "all()"
	out := ApplyMineFilterToRevset(base)
	if !strings.Contains(out, "(all())") {
		t.Errorf("custom base should be wrapped in parens to preserve precedence; got %q", out)
	}
	if strings.Contains(out, DefaultGraphRevset) {
		t.Errorf("custom base should not pull in DefaultGraphRevset; got %q", out)
	}
}

func TestServiceBookmarkListRemoteFlag(t *testing.T) {
	s := &Service{}
	if got := s.BookmarkListRemoteFlag(); got != "--all-remotes" {
		t.Errorf("zero-value Service should default to --all-remotes (legacy); got %q", got)
	}
	s.BookmarkListPreferTracked = true
	if got := s.BookmarkListRemoteFlag(); got != "--tracked" {
		t.Errorf("BookmarkListPreferTracked=true should yield --tracked; got %q", got)
	}
}
