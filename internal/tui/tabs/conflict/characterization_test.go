package conflict

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/state"
)

// This file characterizes the bookmark-conflict modal's current behavior BEFORE
// the P2.3 Tab-interface refactor. model.go opens/closes this modal and reacts
// to the NavigateTarget commands its Update emits; the tests below lock the
// key/zone routing and the accessor surface model.go relies on.
//
// PLAN(P2.3): keep this green across the interface migration.

// navKind runs a tea.Cmd and extracts the NavigateKind it carries (via
// state.NavigateMsg). Fails the test if the cmd is nil or not a navigation.
func navKind(t *testing.T, cmd tea.Cmd) (state.NavigateKind, state.NavigateTarget) {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a non-nil tea.Cmd carrying a NavigateMsg")
	}
	msg := cmd()
	nav, ok := msg.(state.NavigateMsg)
	if !ok {
		t.Fatalf("expected state.NavigateMsg, got %T", msg)
	}
	return nav.Target.Kind, nav.Target
}

// TestUpdateNoOpWhenHidden locks that a hidden modal ignores all input so keys
// pass through to the underlay tab (model.go only routes to this modal when the
// ViewBookmarkConflict view is active).
func TestUpdateNoOpWhenHidden(t *testing.T) {
	m := NewModel(nil)
	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("hidden modal should emit no cmd, got %v", cmd)
	}
	if got.IsShown() {
		t.Fatal("hidden modal should stay hidden")
	}
}

// TestShowResetsSelection locks that Show(...) always starts on option 0
// (Keep local) regardless of the previous selection.
func TestShowResetsSelection(t *testing.T) {
	m := NewModel(nil)
	m.SetSelectedOption(1)
	m.Show("main", "abc", "def", "local sum", "remote sum", "now", "later")
	if !m.IsShown() {
		t.Fatal("Show should mark the modal shown")
	}
	if m.GetSelectedOption() != "keep_local" {
		t.Fatalf("Show should reset selection to keep_local, got %q", m.GetSelectedOption())
	}
	if m.GetBookmarkName() != "main" {
		t.Fatalf("GetBookmarkName = %q, want main", m.GetBookmarkName())
	}
}

// TestKeyNavigationBoundsAndSides locks j/k clamping and the h/l/left/right
// jump-to-side behavior.
func TestKeyNavigationBoundsAndSides(t *testing.T) {
	m := NewModel(nil)
	m.Show("main", "a", "b", "", "", "", "")

	// k at the top is a no-op (stays keep_local).
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.GetSelectedOption() != "keep_local" {
		t.Fatalf("k at top should stay keep_local, got %q", m.GetSelectedOption())
	}
	// j moves to reset_remote and clamps there.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.GetSelectedOption() != "reset_remote" {
		t.Fatalf("j should move to reset_remote, got %q", m.GetSelectedOption())
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.GetSelectedOption() != "reset_remote" {
		t.Fatalf("j past bottom should clamp at reset_remote, got %q", m.GetSelectedOption())
	}
	// h jumps back to keep_local, l jumps to reset_remote, r forces reset.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	if m.GetSelectedOption() != "keep_local" {
		t.Fatalf("h should select keep_local, got %q", m.GetSelectedOption())
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	if m.GetSelectedOption() != "reset_remote" {
		t.Fatalf("l should select reset_remote, got %q", m.GetSelectedOption())
	}
	m.SetSelectedOption(0)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if m.GetSelectedOption() != "reset_remote" {
		t.Fatalf("r should select reset_remote, got %q", m.GetSelectedOption())
	}
}

// TestEnterEmitsResolveWithSelection locks that Enter emits NavigateResolveConflict
// carrying the current selection and bookmark name.
func TestEnterEmitsResolveWithSelection(t *testing.T) {
	m := NewModel(nil)
	m.Show("feature", "a", "b", "", "", "", "")
	m.SetSelectedOption(1)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	kind, target := navKind(t, cmd)
	if kind != state.NavigateResolveConflict {
		t.Fatalf("Enter should emit NavigateResolveConflict, got %v", kind)
	}
	if target.ConflictBookmarkName != "feature" {
		t.Fatalf("ConflictBookmarkName = %q, want feature", target.ConflictBookmarkName)
	}
	if target.ConflictResolution != "reset_remote" {
		t.Fatalf("ConflictResolution = %q, want reset_remote", target.ConflictResolution)
	}
}

// TestEscEmitsCloseAndHides locks that Esc hides the modal and emits the close
// navigation (so model.go restores the underlay tab).
func TestEscEmitsCloseAndHides(t *testing.T) {
	m := NewModel(nil)
	m.Show("main", "a", "b", "", "", "", "")
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.IsShown() {
		t.Fatal("Esc should hide the modal")
	}
	kind, _ := navKind(t, cmd)
	if kind != state.NavigateCloseBookmarkConflict {
		t.Fatalf("Esc should emit NavigateCloseBookmarkConflict, got %v", kind)
	}
}

// TestViewEmptyWhenHidden locks the render gate: hidden modals paint nothing so
// they never leak into the underlay frame.
func TestViewEmptyWhenHidden(t *testing.T) {
	m := NewModel(nil)
	if strings.TrimSpace(m.View()) != "" {
		t.Fatal("hidden modal View() should be empty")
	}
	m.Show("main", "abc123", "def456", "local", "remote", "", "")
	if strings.TrimSpace(m.View()) == "" {
		t.Fatal("shown modal View() should render non-empty")
	}
}

// TestSetSelectedOptionClampsRange locks that out-of-range option indices are
// ignored (so a stale caller can't push selection into an undefined side).
func TestSetSelectedOptionClampsRange(t *testing.T) {
	m := NewModel(nil)
	m.SetSelectedOption(1)
	m.SetSelectedOption(5) // ignored
	if m.GetSelectedOption() != "reset_remote" {
		t.Fatalf("out-of-range option should be ignored, got %q", m.GetSelectedOption())
	}
	m.SetSelectedOption(-1) // ignored
	if m.GetSelectedOption() != "reset_remote" {
		t.Fatalf("negative option should be ignored, got %q", m.GetSelectedOption())
	}
}
