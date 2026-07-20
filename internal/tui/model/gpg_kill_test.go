package model

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/util"
)

func TestKillGPGAgentSuccessAutoRetries(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()

	ran := false
	retry := func() tea.Msg {
		ran = true
		return nil
	}
	signingErr := fmt.Errorf("failed to create commit: Signing error\ngpg: signing failed: No pinentry")
	m.applyEffect(effShowRetryableError{err: signingErr, retry: retry})
	if !m.errorModal.HasKillGPG() {
		t.Fatal("signing error should arm Kill gpg-agent on the modal")
	}

	_, cmd, handled := m.handleNavigateError(state.NavigateTarget{Kind: state.NavigateKillGPGAgent})
	if !handled {
		t.Fatal("NavigateKillGPGAgent should be handled")
	}
	if cmd == nil {
		t.Fatal("expected KillGPGAgentCmd")
	}
	// Simulate successful kill without running gpgconf.
	newModel, follow := m.Update(util.KillGPGAgentResultMsg{})
	m = newModel.(*Model)
	if m.errorModal.GetError() != nil {
		t.Fatal("successful kill with pending retry should clear the error modal")
	}
	if m.pendingRetryCmd != nil {
		t.Fatal("auto-retry should consume pendingRetryCmd")
	}
	if follow == nil {
		t.Fatal("expected auto-retry command")
	}
	_ = follow()
	if !ran {
		t.Fatal("auto-retry should re-run the stashed command")
	}
	if m.appState.StatusMessage != "gpg-agent killed" {
		t.Fatalf("status = %q, want gpg-agent killed", m.appState.StatusMessage)
	}
}

func TestKillGPGAgentSuccessWithoutRetryLeavesModal(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()

	signingErr := fmt.Errorf("Signing error: No pinentry")
	m.applyEffect(effShowError{err: signingErr})
	if !m.errorModal.HasKillGPG() {
		t.Fatal("precondition: kill should be armed")
	}
	if m.pendingRetryCmd != nil {
		t.Fatal("precondition: no pending retry")
	}

	newModel, follow := m.Update(util.KillGPGAgentResultMsg{})
	m = newModel.(*Model)
	if follow != nil {
		t.Fatal("without pending retry, kill success should not schedule a cmd")
	}
	if m.errorModal.GetError() == nil {
		t.Fatal("without pending retry, modal should stay open")
	}
	if m.appState.StatusMessage != "gpg-agent killed" {
		t.Fatalf("status = %q, want gpg-agent killed", m.appState.StatusMessage)
	}
}

func TestKillGPGAgentFailureDoesNotAutoRetry(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()

	ran := false
	retry := func() tea.Msg {
		ran = true
		return nil
	}
	m.applyEffect(effShowRetryableError{
		err:   fmt.Errorf("Signing error: No pinentry"),
		retry: retry,
	})

	newModel, follow := m.Update(util.KillGPGAgentResultMsg{Err: fmt.Errorf("kill gpg-agent: boom")})
	m = newModel.(*Model)
	if follow != nil {
		t.Fatal("kill failure must not schedule retry")
	}
	if m.errorModal.GetError() == nil {
		t.Fatal("kill failure should leave the error modal open")
	}
	if m.pendingRetryCmd == nil {
		t.Fatal("kill failure should keep pendingRetryCmd for a later Retry")
	}
	if ran {
		t.Fatal("stashed retry must not have run")
	}
	if m.appState.StatusMessage == "" || m.appState.StatusMessage == "gpg-agent killed" {
		t.Fatalf("expected kill failure status, got %q", m.appState.StatusMessage)
	}
}

func TestErrorMsgWithRetryArmsRetryable(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()

	retry := func() tea.Msg { return nil }
	newModel, _ := m.Update(util.ErrorMsg{
		Err:   fmt.Errorf("Signing error: No pinentry"),
		Retry: retry,
	})
	m = newModel.(*Model)
	if m.errorModal.GetError() == nil {
		t.Fatal("ErrorMsg should open the error modal")
	}
	if !m.errorModal.HasRetry() {
		t.Fatal("ErrorMsg with Retry should arm Retry")
	}
	if !m.errorModal.HasKillGPG() {
		t.Fatal("signing ErrorMsg should arm Kill gpg-agent")
	}
	if m.pendingRetryCmd == nil {
		t.Fatal("ErrorMsg with Retry should stash pendingRetryCmd")
	}
}
