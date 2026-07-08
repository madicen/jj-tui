package model

// P5.5 retry-coverage tests. These lock the generic (non-AI) Retry path on the error modal:
//   - effShowRetryableError populates the modal AND arms the Retry button + replay command,
//   - a plain effShowError clears any stale replay target so Retry never fires against an
//     unrelated failure,
//   - NavigateRetryError re-runs the stashed command verbatim and dismisses the modal,
//   - NavigateDismissError forgets the replay target.

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/state"
)

func TestRetryableErrorArmsRetry(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()

	ran := false
	retry := func() tea.Msg {
		ran = true
		return nil
	}
	boom := fmt.Errorf("network down")
	m.applyEffect(effShowRetryableError{err: boom, retry: retry})

	if m.errorModal.GetError() == nil {
		t.Fatal("effShowRetryableError should populate the error modal")
	}
	if !m.errorModal.HasRetry() {
		t.Fatal("effShowRetryableError with a non-nil retry should arm the Retry button")
	}
	if m.pendingRetryCmd == nil {
		t.Fatal("effShowRetryableError should stash the replay command")
	}

	_, cmd, handled := m.handleNavigateError(state.NavigateTarget{Kind: state.NavigateRetryError})
	if !handled {
		t.Fatal("NavigateRetryError should be handled")
	}
	if m.errorModal.GetError() != nil {
		t.Fatal("retry should dismiss the error modal")
	}
	if m.pendingRetryCmd != nil {
		t.Fatal("retry should consume (clear) the replay command")
	}
	if cmd == nil {
		t.Fatal("retry should return the replay command")
	}
	_ = cmd()
	if !ran {
		t.Fatal("retry should re-run the exact stashed command")
	}
}

func TestRetryableErrorNilRetryDegradesToPlain(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()

	m.applyEffect(effShowRetryableError{err: fmt.Errorf("boom"), retry: nil})
	if m.errorModal.HasRetry() {
		t.Fatal("a nil retry must not arm the Retry button")
	}
	if m.pendingRetryCmd != nil {
		t.Fatal("a nil retry must not leave a replay command")
	}
}

func TestPlainErrorClearsStaleRetry(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()

	m.applyEffect(effShowRetryableError{err: fmt.Errorf("first"), retry: func() tea.Msg { return nil }})
	if m.pendingRetryCmd == nil {
		t.Fatal("precondition: retry should be armed")
	}
	// A subsequent, unrelated non-retryable error must clear the stale replay target.
	m.applyEffect(effShowError{err: fmt.Errorf("second")})
	if m.pendingRetryCmd != nil {
		t.Fatal("effShowError should clear any stale replay command")
	}
	if m.errorModal.HasRetry() {
		t.Fatal("effShowError should not offer Retry")
	}
}

func TestDismissClearsRetry(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()

	m.applyEffect(effShowRetryableError{err: fmt.Errorf("boom"), retry: func() tea.Msg { return nil }})
	_, _, handled := m.handleNavigateError(state.NavigateTarget{Kind: state.NavigateDismissError})
	if !handled {
		t.Fatal("NavigateDismissError should be handled")
	}
	if m.pendingRetryCmd != nil {
		t.Fatal("dismiss should forget the replay command")
	}
	if m.errorModal.GetError() != nil {
		t.Fatal("dismiss should clear the error modal")
	}
}
