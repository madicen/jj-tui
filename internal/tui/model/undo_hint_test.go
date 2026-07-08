package model

import (
	"strings"
	"testing"

	"github.com/madicen/jj-tui/internal/tui/state"
)

// TestUndoHintReadySetsAndRenders verifies a ready hint is stored, rendered in
// the status bar, and schedules an expiry tick.
func TestUndoHintReadySetsAndRenders(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()

	_, cmd := m.handleUndoHintReady(undoHintReadyMsg{Desc: "describe commit abcd1234"})
	if m.undoHint != "Ctrl+z undoes: describe commit abcd1234" {
		t.Fatalf("undoHint = %q", m.undoHint)
	}
	if cmd == nil {
		t.Fatal("expected an expiry tick command")
	}
	if !strings.Contains(m.renderStatusBar(), "Ctrl+z undoes: describe commit") {
		t.Fatalf("status bar should show the undo hint; got %q", m.renderStatusBar())
	}
}

// TestUndoHintExpirySeqGuard verifies only the current-sequence expiry clears the
// hint, so a stale tick from an earlier hint can't clear a fresher one.
func TestUndoHintExpirySeqGuard(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()

	m.handleUndoHintReady(undoHintReadyMsg{Desc: "op one"})
	staleSeq := m.undoHintSeq
	// A newer hint arrives, bumping the sequence.
	m.handleUndoHintReady(undoHintReadyMsg{Desc: "op two"})

	// Stale expiry must NOT clear the fresher hint.
	m.handleUndoHintExpired(undoHintExpiredMsg{Seq: staleSeq})
	if m.undoHint == "" {
		t.Fatal("stale expiry should not clear the current hint")
	}
	// Current expiry clears it.
	m.handleUndoHintExpired(undoHintExpiredMsg{Seq: m.undoHintSeq})
	if m.undoHint != "" {
		t.Fatalf("current expiry should clear the hint, got %q", m.undoHint)
	}
}

// TestUndoHintEmptyDescIgnored verifies a blank op description doesn't set a hint.
func TestUndoHintEmptyDescIgnored(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()
	_, cmd := m.handleUndoHintReady(undoHintReadyMsg{Desc: ""})
	if m.undoHint != "" || cmd != nil {
		t.Fatalf("empty description should not set a hint (hint=%q, cmd!=nil=%v)", m.undoHint, cmd != nil)
	}
}

// TestUndoHintPendingClearedOnReload verifies applyRepositoryLoaded consumes the
// pending-hint flag (so it fires exactly once per mutation, not on plain refresh).
func TestUndoHintPendingClearedOnReload(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()
	m.pendingUndoHint = true
	m.applyRepositoryLoaded(m.appState.Repository)
	if m.pendingUndoHint {
		t.Fatal("applyRepositoryLoaded should clear pendingUndoHint")
	}
}

// TestMutatingNavigateSetsPendingHint verifies mutating jj navigations arm the
// hint while non-jj navigations (e.g. tab switches) do not.
func TestMutatingNavigateSetsPendingHint(t *testing.T) {
	if !isMutatingJJNavigate(state.NavigateSaveDescription) {
		t.Error("SaveDescription should be a mutating jj navigate")
	}
	if !isMutatingJJNavigate(state.NavigateRestoreOperation) {
		t.Error("RestoreOperation should be a mutating jj navigate")
	}
	if isMutatingJJNavigate(state.NavigateSubmitPR) {
		t.Error("SubmitPR is a GitHub API call, not a jj operation")
	}
	if isMutatingJJNavigate(state.NavigateBackToGraph) {
		t.Error("BackToGraph is not a mutation")
	}
}
