package model

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/tui/genmenu"
	"github.com/madicen/jj-tui/internal/tui/state"
)

func testPressMouse() tea.MouseMsg {
	return tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
}

func testTick(pressID int) genmenu.TickMsg {
	return genmenu.TickMsg{Owner: "zone:t", PressID: pressID}
}

// TestResolveAIOverride covers the lookup path used by handleNavigate when a
// NavigateGenerate* target carries an AIOverrideProfile name.
func TestResolveAIOverride(t *testing.T) {
	t.Run("empty returns nil", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()
		m.appState.Config = &config.Config{
			AIProfiles:      []config.AIProfile{{Name: "fast", Provider: "openai_compatible"}},
			AIActiveProfile: "fast",
		}
		if got := m.resolveAIOverride(state.NavigateTarget{}); got != nil {
			t.Fatalf("empty AIOverrideProfile should yield nil; got %+v", got)
		}
	})

	t.Run("known profile returns matching pointer", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()
		m.appState.Config = &config.Config{
			AIProfiles: []config.AIProfile{
				{Name: "fast", Provider: "openai_compatible", Model: "gpt-4o-mini"},
				{Name: "smart", Provider: "openai_compatible", Model: "gpt-4o"},
			},
			AIActiveProfile: "fast",
		}
		got := m.resolveAIOverride(state.NavigateTarget{AIOverrideProfile: "smart"})
		if got == nil || got.Model != "gpt-4o" {
			t.Fatalf("expected smart profile; got %+v", got)
		}
	})

	t.Run("unknown profile falls back to nil", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()
		m.appState.Config = &config.Config{
			AIProfiles:      []config.AIProfile{{Name: "fast", Provider: "openai_compatible"}},
			AIActiveProfile: "fast",
		}
		if got := m.resolveAIOverride(state.NavigateTarget{AIOverrideProfile: "ghost"}); got != nil {
			t.Fatalf("unknown profile should yield nil; got %+v", got)
		}
	})
}

// TestPushAIProfilesToFormModals verifies the modal-side profile list is
// kept in sync with the config's AIProfileList(). Used when modals open and
// when settings save reloads config.
func TestPushAIProfilesToFormModals(t *testing.T) {
	m := newTestModel()
	defer m.Close()
	m.appState.Config = &config.Config{
		AIProfiles: []config.AIProfile{
			{Name: "a", Provider: "openai_compatible"},
			{Name: "b", Provider: "openai_compatible", Model: "gpt-4o"},
		},
		AIActiveProfile: "b",
	}
	m.pushAIProfilesToFormModals()

	// Manually open each modal's menu to confirm the profile list propagated.
	desc := m.desceditModal.MenuState()
	if desc == nil {
		t.Fatal("desceditModal.MenuState() should not be nil")
	}
	// Show the modal so MenuOverlay works.
	m.desceditModal.Show("abc1", "abc1")
	desc.Reset()
	// Beginpress + tick to flip shown:
	_ = desc.BeginPress("zone:t", testPressMouse())
	desc.OpenIfMatches(testTick(1))
	v, _, _ := m.desceditModal.MenuOverlay()
	if v == "" {
		t.Fatal("desceditModal.MenuOverlay should not be empty after profile push + open")
	}
}
