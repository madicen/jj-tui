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
