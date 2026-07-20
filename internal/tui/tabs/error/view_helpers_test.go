package error

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/state"
)

func TestRenderModalCapsHeight(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("word ", 200)
	modal := renderModal(nil, 100, 24, long, false, true, false)
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
	modal := renderModal(nil, 100, 24, msg, false, true, false)
	if strings.Contains(modal, "truncated") {
		t.Fatalf("did not expect truncation hint for short message")
	}
	if !strings.Contains(modal, msg) {
		t.Fatalf("expected original message in modal")
	}
}

func TestRenderModalHidesRetryWhenNotApplicable(t *testing.T) {
	t.Parallel()
	msg := "non-retryable failure"
	withRetry := renderModal(nil, 100, 24, msg, false, true, false)
	withoutRetry := renderModal(nil, 100, 24, msg, false, false, false)
	if !strings.Contains(withRetry, "Retry") {
		t.Fatalf("expected Retry button when hasRetry=true")
	}
	if strings.Contains(withoutRetry, "Retry") {
		t.Fatalf("did not expect Retry button when hasRetry=false")
	}
}

func TestRenderModalKillGPGOnlyWhenArmed(t *testing.T) {
	t.Parallel()
	msg := "signing error: No pinentry"
	with := renderModal(nil, 100, 24, msg, false, false, true)
	without := renderModal(nil, 100, 24, msg, false, false, false)
	if !strings.Contains(with, "Kill gpg-agent") {
		t.Fatalf("expected Kill gpg-agent when hasKillGPG=true")
	}
	if strings.Contains(without, "Kill gpg-agent") {
		t.Fatalf("did not expect Kill gpg-agent when hasKillGPG=false")
	}
}

func TestSetErrorArmsKillGPGForSigningFailure(t *testing.T) {
	t.Parallel()
	m := NewModel()
	m.SetError(fmt.Errorf("failed to create commit: Internal error\n2: Signing error\ngpg: signing failed: No pinentry"), false, "")
	if !m.HasKillGPG() {
		t.Fatal("signing/pinentry error should arm Kill gpg-agent")
	}
	m.SetError(fmt.Errorf("failed to push: remote rejected"), false, "")
	if m.HasKillGPG() {
		t.Fatal("unrelated error must not arm Kill gpg-agent")
	}
}

func TestKillGPGKeyEmitsNavigate(t *testing.T) {
	t.Parallel()
	m := NewModel()
	m.SetError(fmt.Errorf("Signing error: No pinentry"), false, "")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if cmd == nil {
		t.Fatal("k should emit a navigate cmd when Kill gpg-agent is armed")
	}
	msg := cmd()
	navMsg, ok := msg.(state.NavigateMsg)
	if !ok {
		t.Fatalf("got %T, want NavigateMsg", msg)
	}
	if navMsg.Target.Kind != state.NavigateKillGPGAgent {
		t.Fatalf("Kind = %v, want NavigateKillGPGAgent", navMsg.Target.Kind)
	}
}
