package genmenu

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func mouseMsg(action tea.MouseAction, x, y int) tea.MouseMsg {
	return tea.MouseMsg{
		Button: tea.MouseButtonLeft,
		Action: action,
		X:      x,
		Y:      y,
	}
}

func TestBeginPress_SchedulesMatchingTick(t *testing.T) {
	var s State
	cmd := s.BeginPress("zone:test", mouseMsg(tea.MouseActionPress, 10, 5))
	if cmd == nil {
		t.Fatal("BeginPress should return a tea.Cmd")
	}
	if !s.IsArmed() {
		t.Fatal("press should be armed after BeginPress")
	}
	if s.IsShown() {
		t.Fatal("menu should not be visible before tick fires")
	}
	if x, y := s.MouseAnchor(); x != 10 || y != 5 {
		t.Fatalf("anchor: got (%d,%d)", x, y)
	}
}

func TestOpenIfMatches_OpensOnMatchingTick(t *testing.T) {
	var s State
	_ = s.BeginPress("zone:test", mouseMsg(tea.MouseActionPress, 4, 7))
	tick := TickMsg{Owner: "zone:test", PressID: 1}
	if !s.OpenIfMatches(tick) {
		t.Fatal("tick matching latest press should open the menu")
	}
	if !s.IsShown() {
		t.Fatal("menu should be shown after matching tick")
	}
	// A second matching tick should NOT re-open or re-trigger.
	if s.OpenIfMatches(tick) {
		t.Fatal("a tick must not re-open an already-shown menu")
	}
}

func TestOpenIfMatches_IgnoresStaleTick(t *testing.T) {
	var s State
	_ = s.BeginPress("zone:test", mouseMsg(tea.MouseActionPress, 0, 0))
	// User released before threshold.
	s.CancelPress()
	tick := TickMsg{Owner: "zone:test", PressID: 1}
	if s.OpenIfMatches(tick) {
		t.Fatal("stale tick after CancelPress should not open menu")
	}
}

func TestOpenIfMatches_IgnoresDifferentOwner(t *testing.T) {
	var s State
	_ = s.BeginPress("zone:a", mouseMsg(tea.MouseActionPress, 0, 0))
	tick := TickMsg{Owner: "zone:other", PressID: 1}
	if s.OpenIfMatches(tick) {
		t.Fatal("tick from different owner must not open this state's menu")
	}
}

func TestOpenIfMatches_IgnoresOlderPressID(t *testing.T) {
	var s State
	_ = s.BeginPress("zone:test", mouseMsg(tea.MouseActionPress, 0, 0))
	_ = s.BeginPress("zone:test", mouseMsg(tea.MouseActionPress, 0, 0))
	stale := TickMsg{Owner: "zone:test", PressID: 1}
	if s.OpenIfMatches(stale) {
		t.Fatal("tick with stale PressID must not open menu")
	}
}

func TestClose_DismissesMenu(t *testing.T) {
	var s State
	_ = s.BeginPress("zone:test", mouseMsg(tea.MouseActionPress, 0, 0))
	s.OpenIfMatches(TickMsg{Owner: "zone:test", PressID: 1})
	if !s.Close() {
		t.Fatal("Close should return true when menu was open")
	}
	if s.IsShown() {
		t.Fatal("Close should hide menu")
	}
}

func TestReset_ClearsAll(t *testing.T) {
	var s State
	_ = s.BeginPress("zone:test", mouseMsg(tea.MouseActionPress, 5, 5))
	s.OpenIfMatches(TickMsg{Owner: "zone:test", PressID: 1})
	s.Reset()
	if s.IsShown() || s.IsArmed() {
		t.Fatal("Reset must clear shown/armed")
	}
}

// TestOnMotion_KeepsArmedWithinSlack verifies trackpad jitter or single-cell
// drift does not cancel the long-press. The user explicitly asked for a couple
// pixels of tolerance; in a TUI that translates to a 1-2 cell slack box.
func TestOnMotion_KeepsArmedWithinSlack(t *testing.T) {
	var s State
	_ = s.BeginPress("zone:test", mouseMsg(tea.MouseActionPress, 10, 5))
	s.OnMotion(nil, mouseMsg(tea.MouseActionMotion, 10+MotionSlack, 5+MotionSlack))
	if !s.IsArmed() {
		t.Fatal("motion within MotionSlack of anchor must not cancel the press")
	}
	if x, y := s.MouseAnchor(); x != 10 || y != 5 {
		t.Fatalf("anchor must not move on tolerated motion: got (%d,%d)", x, y)
	}
}

// TestOnMotion_CancelsBeyondSlackWithNoZone confirms the slack is bounded:
// once the cursor drifts further than MotionSlack and we have no zone manager
// to fall back on, the press cancels so a real drag isn't accidentally held.
func TestOnMotion_CancelsBeyondSlackWithNoZone(t *testing.T) {
	var s State
	_ = s.BeginPress("zone:test", mouseMsg(tea.MouseActionPress, 10, 5))
	s.OnMotion(nil, mouseMsg(tea.MouseActionMotion, 10+MotionSlack+1, 5))
	if s.IsArmed() {
		t.Fatal("motion beyond MotionSlack without a zone match must cancel")
	}
}

// TestOnMotion_NoOpWhenShown ensures that once the menu is up, motion events
// flow through to the hover logic instead of accidentally tearing down the
// press state via OnMotion.
func TestOnMotion_NoOpWhenShown(t *testing.T) {
	var s State
	_ = s.BeginPress("zone:test", mouseMsg(tea.MouseActionPress, 10, 5))
	s.OpenIfMatches(TickMsg{Owner: "zone:test", PressID: 1})
	s.OnMotion(nil, mouseMsg(tea.MouseActionMotion, 9999, 9999))
	if !s.IsShown() {
		t.Fatal("OnMotion must not close an already-shown menu")
	}
}
