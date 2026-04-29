package jj

import (
	"strings"
	"testing"
)

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
