package error

import (
	"strings"
	"testing"
)

func TestRenderModalCapsHeight(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("word ", 200)
	modal := renderModal(nil, 100, 24, long, false)
	lines := strings.Split(modal, "\n")
	if len(lines) > 24 {
		t.Fatalf("modal has %d lines, want at most 24 (terminal height budget)", len(lines))
	}
	if !strings.Contains(modal, "truncated") || !strings.Contains(modal, "copy the full message") {
		t.Fatalf("expected truncation hint in modal output")
	}
}

func TestRenderModalNoTruncateWhenShort(t *testing.T) {
	t.Parallel()
	msg := "short error"
	modal := renderModal(nil, 100, 24, msg, false)
	if strings.Contains(modal, "truncated") {
		t.Fatalf("did not expect truncation hint for short message")
	}
	if !strings.Contains(modal, msg) {
		t.Fatalf("expected original message in modal")
	}
}
