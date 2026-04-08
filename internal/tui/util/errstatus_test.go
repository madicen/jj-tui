package util

import (
	"errors"
	"testing"
)

func TestStatusStringFromError(t *testing.T) {
	if got := StatusStringFromError(nil, 50); got != "" {
		t.Errorf("nil: got %q", got)
	}
	long := errors.New("abcdefghijklmnopqrst")
	// min width is 8; use 12 so we truncate after clamp
	got := StatusStringFromError(long, 12)
	want := "abcdefghijk…"
	if got != want {
		t.Errorf("truncate: got %q want %q", got, want)
	}
	multi := errors.New("line1\nline2")
	got2 := StatusStringFromError(multi, 100)
	if got2 != "line1 line2" {
		t.Errorf("newlines: got %q", got2)
	}
}
