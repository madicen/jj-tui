package githublogin

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/state"
)

// This file characterizes the GitHub login modal's current behavior BEFORE the
// P2.3 Tab-interface refactor. model.go drives this modal (device flow polling,
// gh CLI mode) and reacts to the commands its Update emits. The tests lock the
// key routing, mode switching, and the accessor surface model.go depends on.
//
// PLAN(P2.3): keep this green across the interface migration.

// TestDefaultModeIsDevice locks that a fresh model starts in device-flow mode
// (model.go picks the chrome title from Mode()).
func TestDefaultModeIsDevice(t *testing.T) {
	m := NewModel(nil)
	if m.Mode() != LoginModeDevice {
		t.Fatalf("default Mode() = %v, want LoginModeDevice", m.Mode())
	}
	if m.GetPolling() {
		t.Fatal("fresh model should not be polling")
	}
}

// TestEscEmitsCancelNavigation locks that Esc emits NavigateGitHubLoginCancel in
// both modes so model.go tears the modal down consistently.
func TestEscEmitsCancelNavigation(t *testing.T) {
	for _, setup := range []struct {
		name string
		mode func(*Model)
	}{
		{"device", func(m *Model) { m.SetDeviceFlow("dc", "USER-CODE", "https://x", 5) }},
		{"ghcli", func(m *Model) { m.SetGhCLILoginMode() }},
	} {
		t.Run(setup.name, func(t *testing.T) {
			m := NewModel(nil)
			setup.mode(&m)
			_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
			if cmd == nil {
				t.Fatal("Esc should emit a cmd")
			}
			nav, ok := cmd().(state.NavigateMsg)
			if !ok {
				t.Fatalf("expected NavigateMsg, got %T", cmd())
			}
			if nav.Target.Kind != state.NavigateGitHubLoginCancel {
				t.Fatalf("Esc kind = %v, want NavigateGitHubLoginCancel", nav.Target.Kind)
			}
		})
	}
}

// TestEnterInGhCLIModeRunsGhAuth locks that Enter in gh-CLI mode triggers the
// gh auth login command rather than the device-flow copy/open behavior.
func TestEnterInGhCLIModeRunsGhAuth(t *testing.T) {
	m := NewModel(nil)
	m.SetGhCLILoginMode()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter in gh CLI mode should emit a cmd")
	}
	// GhAuthLoginCmd emits an internal message (not a NavigateMsg); assert it is
	// NOT a navigation, which distinguishes it from the cancel/device paths.
	if _, isNav := cmd().(state.NavigateMsg); isNav {
		t.Fatal("Enter in gh CLI mode should not emit a NavigateMsg")
	}
}

// TestEnterInDeviceModeNoCodeIsNoOp locks that Enter before a user code arrives
// does nothing (device flow hasn't started yet).
func TestEnterInDeviceModeNoCodeIsNoOp(t *testing.T) {
	m := NewModel(nil)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("Enter with no user code should be a no-op, got cmd %v", cmd)
	}
}

// TestSetDeviceFlowPopulatesStateAndPolls locks the device-flow accessors
// model.go reads to drive polling.
func TestSetDeviceFlowPopulatesStateAndPolls(t *testing.T) {
	m := NewModel(nil)
	m.SetDeviceFlow("device-code", "USER-CODE", "https://github.com/login/device", 7)
	if m.Mode() != LoginModeDevice {
		t.Fatalf("SetDeviceFlow should set device mode, got %v", m.Mode())
	}
	if m.GetDeviceCode() != "device-code" {
		t.Fatalf("GetDeviceCode = %q, want device-code", m.GetDeviceCode())
	}
	if m.GetPollInterval() != 7 {
		t.Fatalf("GetPollInterval = %d, want 7", m.GetPollInterval())
	}
	if !m.GetPolling() {
		t.Fatal("SetDeviceFlow should set polling=true")
	}
}

// TestClearFlowResetsToDeviceDefaults locks that ClearFlow returns the model to
// the fresh device-mode defaults (called on cancel/success/error).
func TestClearFlowResetsToDeviceDefaults(t *testing.T) {
	m := NewModel(nil)
	m.SetGhCLILoginMode()
	m.SetDeviceFlow("dc", "uc", "url", 9)
	m.ClearFlow()
	if m.Mode() != LoginModeDevice {
		t.Fatalf("ClearFlow Mode() = %v, want LoginModeDevice", m.Mode())
	}
	if m.GetDeviceCode() != "" || m.GetPollInterval() != 0 || m.GetPolling() {
		t.Fatalf("ClearFlow should zero device state: code=%q interval=%d polling=%v",
			m.GetDeviceCode(), m.GetPollInterval(), m.GetPolling())
	}
}

// TestViewRendersBothModes locks that View() renders non-empty output in both
// modes so the modal is never blank.
func TestViewRendersBothModes(t *testing.T) {
	m := NewModel(nil)
	m.SetDeviceFlow("dc", "USER-CODE", "https://github.com/login/device", 5)
	if !strings.Contains(m.View(), "USER-CODE") {
		t.Fatal("device-flow View() should show the user code")
	}
	m.SetGhCLILoginMode()
	if !strings.Contains(m.View(), "gh auth login") {
		t.Fatal("gh CLI View() should mention gh auth login")
	}
}

// TestZoneIDsTrackMode locks the zone set per mode; model.go uses these to route
// mouse clicks to the modal.
func TestZoneIDsTrackMode(t *testing.T) {
	m := NewModel(nil)
	if len(m.ZoneIDs()) != 2 {
		t.Fatalf("device-mode ZoneIDs should have 2 entries, got %d", len(m.ZoneIDs()))
	}
	m.SetGhCLILoginMode()
	if len(m.ZoneIDs()) != 2 {
		t.Fatalf("gh CLI ZoneIDs should have 2 entries, got %d", len(m.ZoneIDs()))
	}
}
