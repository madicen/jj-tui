package graph

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/tui/state"
)

func boolPtr(b bool) *bool { return &b }

// TestConfirmDestructive_DefaultOnGatesAbandon verifies that with the default (nil) toggle an
// abandon request is intercepted by a y/n confirmation instead of running immediately.
func TestConfirmDestructive_DefaultOnGatesAbandon(t *testing.T) {
	m := newTestGraphModel()
	m.selectedCommit = 0
	app := &state.AppState{Repository: m.repository, Config: &config.Config{}}

	if !m.maybeConfirmDestructive(Request{Abandon: true}, app) {
		t.Fatal("abandon should be gated by a confirmation when toggle defaults on")
	}
	if m.confirm == nil {
		t.Fatal("m.confirm should be set after gating an abandon")
	}
	if m.confirm.prompt == "" {
		t.Error("confirm prompt should be a non-empty one-line consequence")
	}
}

// TestConfirmDestructive_ToggleOffRunsImmediately verifies that when ui.confirm_destructive is
// false the request is not gated (caller runs it immediately, preserving old behavior).
func TestConfirmDestructive_ToggleOffRunsImmediately(t *testing.T) {
	m := newTestGraphModel()
	m.selectedCommit = 0
	app := &state.AppState{
		Repository: m.repository,
		Config:     &config.Config{UIConfig: config.UIConfig{ConfirmDestructive: boolPtr(false)}},
	}

	if m.maybeConfirmDestructive(Request{Abandon: true}, app) {
		t.Fatal("abandon should NOT be gated when toggle is off")
	}
	if m.confirm != nil {
		t.Error("m.confirm should stay nil when toggle is off")
	}
}

// TestConfirmDestructive_NonDestructiveNotGated verifies non-destructive requests pass through.
func TestConfirmDestructive_NonDestructiveNotGated(t *testing.T) {
	m := newTestGraphModel()
	app := &state.AppState{Repository: m.repository, Config: &config.Config{}}

	if m.maybeConfirmDestructive(Request{CreatePR: true}, app) {
		t.Error("non-destructive request should not be gated")
	}
}

// TestConfirmDestructive_ResolveConfirm verifies y confirms (returns the request), n cancels,
// and any other key is swallowed while the prompt stays up.
func TestConfirmDestructive_ResolveConfirm(t *testing.T) {
	m := newTestGraphModel()
	m.selectedCommit = 0
	app := &state.AppState{Repository: m.repository, Config: &config.Config{}}

	// Other keys are swallowed and keep the prompt up.
	m.confirm = &destructiveConfirm{req: Request{Abandon: true}, prompt: "confirm?"}
	if _, run := m.resolveConfirm(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}, app); run {
		t.Error("unrelated key should not confirm")
	}
	if m.confirm == nil {
		t.Error("unrelated key should keep the confirmation pending")
	}

	// n cancels.
	if _, run := m.resolveConfirm(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}, app); run {
		t.Error("n should not confirm")
	}
	if m.confirm != nil {
		t.Error("n should clear the pending confirmation")
	}

	// y confirms and returns the original request.
	m.confirm = &destructiveConfirm{req: Request{Backout: true}, prompt: "confirm?"}
	req, run := m.resolveConfirm(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}, app)
	if !run {
		t.Fatal("y should confirm")
	}
	if !req.Backout {
		t.Error("confirmed request should be the original backout request")
	}
	if m.confirm != nil {
		t.Error("y should clear the pending confirmation")
	}
}
