package bookmark

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/tui/genmenu"
)

// TestGenMenuLifecycle mirrors descedit's lifecycle test for the bookmark form.
func TestGenMenuLifecycle(t *testing.T) {
	m := NewModel(zone.New())
	m.Show(0, []string{})
	m.SetAIProfiles([]config.AIProfile{
		{Name: "fast", Provider: "openai_compatible"},
		{Name: "local", Provider: "ollama"},
	}, "fast")

	if v, _, _ := m.MenuOverlay(); v != "" {
		t.Fatal("MenuOverlay should be empty before the menu is shown")
	}
	st := m.MenuState()
	_ = st.BeginPress("zone:test", tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	st.OpenIfMatches(genmenu.TickMsg{Owner: "zone:test", PressID: 1})

	v, _, _ := m.MenuOverlay()
	if !strings.Contains(v, "local") {
		t.Fatalf("overlay should include the local profile:\n%s", v)
	}

	updated, _ := m.Update(CancelRequestedMsg{})
	m = updated
	if m.MenuState().IsShown() {
		t.Fatal("Cancel should reset genmenu state")
	}
}
