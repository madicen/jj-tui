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

// TestFileDiffMinimizeButtonClickToggles guards against regressions where the
// chrome state nudges (added so the file diff's auto-sized box stays in sync
// with bubble-overlay's ContentSize cache) accidentally clobber the [-]
// button's effect. We open the file diff, render once so chrome publishes its
// layout, then send a left-press at the cached MinimizeX/MinimizeY cell and
// assert that LayerState.Minimized flipped to true and survives the next
// View(). If anything reintroduces an unconditional State reset, or breaks
// the chrome Configure wiring, this test catches it.
func TestFileDiffMinimizeButtonClickToggles(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	m.fileDiffModal = m.fileDiffModal.SetDimensions(m.width, m.height)
	m.fileDiffModal = m.fileDiffModal.ShowPreloadedStyledDiff(
		"File diff",
		"sample @ abc1",
		"diff --git a/x b/x\nindex 1..2 100644\n--- a/x\n+++ b/x\n@@ -1,1 +1,1 @@\n-old\n+new\n",
	)
	m.appState.ViewMode = state.ViewFileDiff

	_ = m.View() // populate chrome layout for hit-testing
	layout, ok := m.chrome.LastChromeLayout()
	if !ok {
		t.Fatal("chrome.LastChromeLayout() should be populated after View() on a file diff modal")
	}
	if layout.Regions.MinimizeW == 0 || layout.Regions.MinimizeH == 0 {
		t.Fatalf("minimize region missing from chrome layout — ShowMinimizeButton must be enabled for file diff; got regions: %+v", layout.Regions)
	}

	minX := layout.Left + layout.Regions.MinimizeX
	minY := layout.Top + layout.Regions.MinimizeY

	// Sanity-check that our Configure callback actually flips ShowMinimizeButton
	// — if this fires, the rest of the test is meaningless and we should look
	// at init.go's chrome wiring instead of the press handling.
	probe := overlay.DefaultOverlayConfig()
	if m.chrome.Configure != nil {
		m.chrome.Configure(&probe)
	}
	if !probe.WindowChrome.ShowMinimizeButton {
		t.Fatalf("chrome.Configure did not enable ShowMinimizeButton; got %+v", probe.WindowChrome)
	}

	press := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: minX, Y: minY}
	newM, _ := m.Update(press)
	updated, ok := newM.(*Model)
	if !ok {
		t.Fatalf("Update returned %T, expected *Model", newM)
	}
	if !updated.chrome.State.Minimized {
		t.Fatalf("expected chrome.State.Minimized=true after pressing [-] at (%d,%d); state=%+v; regions=%+v", minX, minY, updated.chrome.State, layout.Regions)
	}

	// Quiet frames (no dimension changes) must leave LayerState alone so user
	// interactions stick — the seq-gated reset in view_helpers.go regresses
	// here if it ever fires unconditionally.
	_ = updated.View()
	if !updated.chrome.State.Minimized {
		t.Fatal("chrome.State.Minimized should survive a subsequent View() — the per-frame reset must be gated on dimension changes only")
	}
}

// TestChromeConsumedPressSwallowsMatchingRelease guards the [x] close-leak
// fix: when window chrome consumes a press (e.g. the user clicks the [x] tab
// button), the matching release must NOT fall through to the underlay zone
// dispatcher — otherwise the release lands on whatever graph zone (e.g. the
// "split" button) happens to sit beneath the modal's close button and fires
// a stray action right after closing the modal. The MouseMsg handler tracks
// this via the chromeConsumedPress flag; the matching release is dropped and
// the flag cleared.
func TestChromeConsumedPressSwallowsMatchingRelease(t *testing.T) {
	m := chromeOpenDescedit(t)
	defer m.Close()

	// Stand in for "chrome just consumed a press on [x]" — the press path in
	// Update() sets this flag whenever chrome.Update returned consumed=true on
	// a MouseActionPress. We don't drive the actual press here because that
	// requires hit-testing the close-button cell, which is unrelated to the
	// swallow logic we want to lock in.
	m.chromeConsumedPress = true

	releaseMsg := tea.MouseMsg{Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft, X: 5, Y: 5}
	newM, cmd := m.Update(releaseMsg)
	updated, ok := newM.(*Model)
	if !ok {
		t.Fatalf("Update returned %T, expected *Model", newM)
	}

	if updated.chromeConsumedPress {
		t.Fatal("chromeConsumedPress should be cleared after the matching release is swallowed")
	}
	if cmd != nil {
		t.Fatal("expected nil cmd from swallowed release; non-nil means the release leaked to underlay zones (e.g. the graph split button)")
	}
}
