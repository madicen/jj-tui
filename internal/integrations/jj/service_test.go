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
	if !strings.Contains(rs, "parents(bookmarks() & mutable())") {
		t.Errorf("DefaultGraphRevset should union parents(bookmarks() & mutable()) for fork display; got %q", rs)
	}
	if !strings.Contains(rs, "bookmarks()") {
		t.Errorf("DefaultGraphRevset should include bookmarks(); got %q", rs)
	}
	if !strings.Contains(rs, "main@origin") {
		t.Errorf("DefaultGraphRevset should include main@origin; got %q", rs)
	}
	if !strings.Contains(rs, "ancestors(@)") || !strings.Contains(rs, "descendants(@)") {
		t.Errorf("DefaultGraphRevset should tie mutable cone to @; got %q", rs)
	}
	if !strings.Contains(rs, "heads(::(bookmarks() & mutable()) & immutable())") || !strings.Contains(rs, "heads(::(@) & immutable())") {
		t.Errorf("DefaultGraphRevset should add per-anchor immutable bases (bookmarks vs @); got %q", rs)
	}
	if !strings.Contains(rs, "immutable())::(bookmarks() & mutable())") || !strings.Contains(rs, "immutable())::@") {
		t.Errorf("DefaultGraphRevset should add immutable-to-tip and immutable-to-@ ranges; got %q", rs)
	}
	if !strings.Contains(rs, "(::(bookmarks() & mutable()) & mutable())") {
		t.Errorf("DefaultGraphRevset should include full mutable ancestor chain of bookmark tips; got %q", rs)
	}
	if !strings.Contains(rs, "(bookmarks() & mutable())") {
		t.Errorf("DefaultGraphRevset should seed mutable bookmark tips; got %q", rs)
	}
}
