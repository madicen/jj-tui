package descedit

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/tui/genmenu"
)

// TestGenMenuLifecycle confirms the descedit modal exposes the genmenu state,
// renders an overlay only while the menu is shown, and resets the menu on
// cancel. The full long-press detection (which needs a live zone manager
// scan over a rendered view) is covered indirectly by genmenu_test.go.
func TestGenMenuLifecycle(t *testing.T) {
	m := NewModel(zone.New())
	m.Show("abc1", "abc1")

	m.SetAIProfiles([]config.AIProfile{
		{Name: "fast", Provider: "openai_compatible", Model: "gpt-4o-mini"},
		{Name: "smart", Provider: "openai_compatible", Model: "gpt-4o"},
	}, "fast")

	view, _, _ := m.MenuOverlay()
	if view != "" {
		t.Fatal("MenuOverlay should be empty when the menu is not shown")
	}

	st := m.MenuState()
	if st == nil {
		t.Fatal("MenuState should return a non-nil pointer")
	}
	// Manually advance state to "shown" — sidesteps the live zone-bounds path
	// since we're isolating the descedit overlay wiring here.
	_ = st.BeginPress("zone:test", tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 0, Y: 0})
	if !st.OpenIfMatches(genmenu.TickMsg{Owner: "zone:test", PressID: 1}) {
		t.Fatal("OpenIfMatches should open the menu on a matching tick")
	}

	view, _, _ = m.MenuOverlay()
	if view == "" {
		t.Fatal("MenuOverlay should be non-empty while shown")
	}
	if !strings.Contains(view, "fast") || !strings.Contains(view, "smart") {
		t.Fatalf("overlay should list both profiles:\n%s", view)
	}

	// Cancel flow resets the menu (and the modal).
	updated, _ := m.Update(CancelRequestedMsg{})
	m = updated
	if m.MenuState().IsShown() {
		t.Fatal("Cancel should reset genmenu state")
	}
}
