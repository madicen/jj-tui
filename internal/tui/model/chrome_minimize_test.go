package model

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	overlay "github.com/madicen/bubble-overlay"
	"github.com/madicen/jj-tui/internal/tui/state"
)

// chromeOpenDescedit primes the model with the description editor active and
// drives one View() pass so chrome's LastChromeLayout is populated. Tests that
// poke chrome.State directly need this layout for hit-testing helpers
// (mouseTransparent) to return meaningful answers.
func chromeOpenDescedit(t *testing.T) *Model {
	t.Helper()
	m := newTestModel()
	m.appState.ViewMode = state.ViewEditDescription
	m.desceditModal.Show("abc1", "abc1")
	_ = m.View() // populate m.chrome.LastChromeLayout()
	if _, ok := m.chrome.LastChromeLayout(); !ok {
		t.Fatal("chrome.LastChromeLayout() should be populated after View() on a chromed modal")
	}
	return m
}

// TestChromeShowMinimizeButtonConfigured verifies the init wiring opted into
// bubble-overlay's minimize button. Without ShowMinimizeButton the chrome
// wouldn't render the [-] / [+] toggle, the user couldn't collapse the
// window, and the mouseTransparent pass-through has nothing to gate on.
func TestChromeShowMinimizeButtonConfigured(t *testing.T) {
	m := chromeOpenDescedit(t)
	defer m.Close()

	cfg := overlay.DefaultOverlayConfig()
	if m.chrome.Configure == nil {
		t.Fatal("chrome.Configure must be set so ShowMinimizeButton is enabled across all chromed modals")
	}
	m.chrome.Configure(&cfg)
	if !cfg.WindowChrome.ShowMinimizeButton {
		t.Fatal("expected WindowChrome.ShowMinimizeButton = true after Configure")
	}
}

// TestMouseTransparentGating walks the three relevant chrome states:
//
//  1. expanded — focus-trap is in effect, helper returns false everywhere.
//  2. minimized + click on the tab strip — helper returns false so the
//     chrome's own mouse handling (drag, [+] restore, [x]) keeps running.
//  3. minimized + click outside the tab strip — helper returns true so the
//     caller routes the event to the underlay tab.
func TestMouseTransparentGating(t *testing.T) {
	m := chromeOpenDescedit(t)
	defer m.Close()

	outsideX, outsideY := 1, m.height-1 // bottom-left corner is well clear of any centered modal

	if m.mouseTransparent(outsideX, outsideY) {
		t.Fatal("expanded chrome must keep focus trap: mouseTransparent should be false")
	}

	m.chrome.State.Minimized = true
	_ = m.View() // re-render so LastChromeLayout reflects the collapsed tab strip
	layout, _ := m.chrome.LastChromeLayout()
	insideX, insideY := layout.Left+1, layout.Top

	if m.mouseTransparent(insideX, insideY) {
		t.Fatalf("minimized chrome must keep its own clicks: mouseTransparent at (%d,%d) inside tab strip rect should be false", insideX, insideY)
	}
	if !m.mouseTransparent(outsideX, outsideY) {
		t.Fatalf("minimized chrome must pass clicks through outside its rect: mouseTransparent at (%d,%d) should be true", outsideX, outsideY)
	}
}

// TestMinimizedZoneInBoundsRoutesToUnderlay verifies the MsgZoneInBounds
// case forwards to the underlay tab when the chrome is collapsed and the
// click landed outside the tab strip. Without this branch the event keys
// off m.appState.ViewMode and lands on the dormant descedit modal, which
// silently drops anything that isn't its own zone.
//
// We stand in for "click on a graph commit" by feeding routeMouseToUnderlay
// a representative MouseMsg and asserting the helper claims to have handled
// it via the graph branch — that's all our caller needs to know to take the
// pass-through path.
func TestMinimizedZoneInBoundsRoutesToUnderlay(t *testing.T) {
	m := chromeOpenDescedit(t)
	defer m.Close()
	m.chrome.State.Minimized = true
	_ = m.View()

	if got := m.layoutContentMode(); got != state.ViewCommitGraph {
		t.Fatalf("descedit's underlay should default to ViewCommitGraph; got %v", got)
	}

	_, handled := m.routeMouseToUnderlay(tea.MouseMsg{Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft, X: 5, Y: 5})
	if !handled {
		t.Fatal("routeMouseToUnderlay should claim the event when layoutContentMode resolves to a tab view (graph)")
	}
}
