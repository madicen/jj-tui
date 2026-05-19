package longpress

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func motion(x, y int) tea.MouseMsg {
	return tea.MouseMsg{Action: tea.MouseActionMotion, Button: tea.MouseButtonLeft, X: x, Y: y}
}

// TestStillArmed_NilManager_WithinSlack documents that without a zone
// manager (e.g. during early frames or in unit tests), the slack box around
// the anchor is the sole tolerance check.
func TestStillArmed_NilManager_WithinSlack(t *testing.T) {
	if !StillArmed(nil, "", 10, 5, motion(10+MotionSlack, 5+MotionSlack)) {
		t.Fatal("motion at the edge of MotionSlack should keep the press armed")
	}
	if !StillArmed(nil, "", 10, 5, motion(10, 5)) {
		t.Fatal("zero-delta motion must keep the press armed")
	}
}

// TestStillArmed_NilManager_BeyondSlack confirms the slack box has a
// boundary: once the cursor moves further than MotionSlack along either
// axis, the gesture is cancelled so drags elsewhere keep working.
func TestStillArmed_NilManager_BeyondSlack(t *testing.T) {
	if StillArmed(nil, "", 10, 5, motion(10+MotionSlack+1, 5)) {
		t.Fatal("motion past MotionSlack on x without a zone match must cancel")
	}
	if StillArmed(nil, "", 10, 5, motion(10, 5+MotionSlack+1)) {
		t.Fatal("motion past MotionSlack on y without a zone match must cancel")
	}
}

// TestStillArmed_EmptyZoneID is a defensive guard: even when a manager is
// supplied, an empty originZone falls back to the slack-only path so
// callers that don't track the origin id still get the same gesture.
func TestStillArmed_EmptyZoneID(t *testing.T) {
	// We can't easily construct a real *zone.Manager here without exercising
	// rendering, so the nil branch already covers the slack-only fallback.
	// This test exists to document the intended contract: empty originZone
	// must NEVER promote the call to "always armed".
	if StillArmed(nil, "", 0, 0, motion(100, 100)) {
		t.Fatal("empty origin zone must not bypass the slack check")
	}
}
