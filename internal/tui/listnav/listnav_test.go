package listnav

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

func TestNewResetsLongPress(t *testing.T) {
	m := New()
	if m.LongPressItemIndex != -1 {
		t.Errorf("New().LongPressItemIndex = %d, want -1", m.LongPressItemIndex)
	}
	if m.YOffset != 0 {
		t.Errorf("New().YOffset = %d, want 0", m.YOffset)
	}
}

func TestWheelScroll(t *testing.T) {
	m := New()
	m.YOffset = 5

	if !m.WheelScroll(tea.MouseMsg{Button: tea.MouseButtonWheelDown}) {
		t.Fatal("wheel down should be handled")
	}
	if m.YOffset != 8 {
		t.Errorf("after wheel down YOffset = %d, want 8", m.YOffset)
	}

	if !m.WheelScroll(tea.MouseMsg{Button: tea.MouseButtonWheelUp}) {
		t.Fatal("wheel up should be handled")
	}
	if m.YOffset != 5 {
		t.Errorf("after wheel up YOffset = %d, want 5", m.YOffset)
	}

	// Clamp at zero.
	m.YOffset = 1
	m.WheelScroll(tea.MouseMsg{Button: tea.MouseButtonWheelUp})
	if m.YOffset != 0 {
		t.Errorf("wheel up should clamp to 0, got %d", m.YOffset)
	}

	// Non-wheel events are not handled.
	m.YOffset = 4
	if m.WheelScroll(tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}) {
		t.Error("non-wheel event should not be handled")
	}
	if m.YOffset != 4 {
		t.Errorf("non-wheel event must not change YOffset, got %d", m.YOffset)
	}
}

func TestVisibleRange(t *testing.T) {
	m := New()

	// Fits entirely: no offset, full range.
	m.YOffset = 0
	start, end := m.VisibleRange(3, 10)
	if start != 0 || end != 3 {
		t.Errorf("VisibleRange(3,10) = (%d,%d), want (0,3)", start, end)
	}

	// Over-scrolled offset clamps to maxOffset = total-listHeight.
	m.YOffset = 500
	start, end = m.VisibleRange(100, 10)
	if start != 90 || end != 100 {
		t.Errorf("clamped VisibleRange = (%d,%d), want (90,100)", start, end)
	}
	if m.YOffset != 90 {
		t.Errorf("YOffset should be clamped to 90, got %d", m.YOffset)
	}

	// Negative offset clamps to 0.
	m.YOffset = -5
	start, end = m.VisibleRange(100, 10)
	if start != 0 || m.YOffset != 0 {
		t.Errorf("negative offset should clamp to 0, got start=%d YOffset=%d", start, m.YOffset)
	}
	_ = end
}

func TestScrollToSelected(t *testing.T) {
	m := New()

	// Selection above the window scrolls up to it.
	m.YOffset = 20
	m.ScrollToSelected(5, 100, 10)
	if m.YOffset != 5 {
		t.Errorf("scroll up: YOffset = %d, want 5", m.YOffset)
	}

	// Selection below the window scrolls down so it's the last visible row.
	m.YOffset = 0
	m.ScrollToSelected(15, 100, 10)
	if m.YOffset != 6 {
		t.Errorf("scroll down: YOffset = %d, want 6", m.YOffset)
	}

	// Selection already visible: no change.
	m.YOffset = 10
	m.ScrollToSelected(12, 100, 10)
	if m.YOffset != 10 {
		t.Errorf("in-view selection should not scroll, got %d", m.YOffset)
	}

	// Out-of-range selection is a no-op.
	m.YOffset = 3
	m.ScrollToSelected(-1, 100, 10)
	m.ScrollToSelected(200, 100, 10)
	if m.YOffset != 3 {
		t.Errorf("out-of-range selection should not scroll, got %d", m.YOffset)
	}
}

func TestArmLongPress_MenuOpenAndButtonGuards(t *testing.T) {
	m := New()
	zm := zone.New()

	cfg := LongPressConfig{
		MenuOpen:  true,
		ItemCount: 3,
		RowZoneID: func(i int) string { return "row" },
		Threshold: time.Millisecond,
		MakeTick:  func(id int) tea.Msg { return id },
	}
	press := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 1, Y: 1}
	if cmd := m.ArmLongPress(zm, press, cfg); cmd != nil {
		t.Error("press while menu open should not arm")
	}

	cfg.MenuOpen = false
	rightPress := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonRight, X: 1, Y: 1}
	if cmd := m.ArmLongPress(zm, rightPress, cfg); cmd != nil {
		t.Error("non-left press should not arm")
	}
}

func TestArmLongPress_ReleaseResets(t *testing.T) {
	m := New()
	m.LongPressItemIndex = 2
	zm := zone.New()
	cfg := LongPressConfig{RowZoneID: func(i int) string { return "row" }}
	m.ArmLongPress(zm, tea.MouseMsg{Action: tea.MouseActionRelease}, cfg)
	if m.LongPressItemIndex != -1 {
		t.Errorf("release should reset LongPressItemIndex, got %d", m.LongPressItemIndex)
	}
}

func TestArmLongPress_MotionDisarmsOutsideSlack(t *testing.T) {
	m := New()
	m.LongPressItemIndex = 1
	m.LongPressMouseX = 10
	m.LongPressMouseY = 10
	cfg := LongPressConfig{
		MenuOpen:  false,
		RowZoneID: func(i int) string { return "" }, // no zone -> slack box only
	}
	// A motion far from the anchor disarms.
	far := tea.MouseMsg{Action: tea.MouseActionMotion, X: 40, Y: 40}
	m.ArmLongPress(nil, far, cfg)
	if m.LongPressItemIndex != -1 {
		t.Errorf("motion outside slack should disarm, got %d", m.LongPressItemIndex)
	}

	// A motion within slack keeps it armed.
	m.LongPressItemIndex = 1
	near := tea.MouseMsg{Action: tea.MouseActionMotion, X: 11, Y: 11}
	m.ArmLongPress(nil, near, cfg)
	if m.LongPressItemIndex != 1 {
		t.Errorf("motion within slack should stay armed, got %d", m.LongPressItemIndex)
	}
}

func TestArmLongPress_ArmsOverRowZone(t *testing.T) {
	m := New()
	zm := zone.New()

	// Register a row zone by scanning marked content.
	const rowID = "listnav-row-0"
	scanned := zm.Scan(zm.Mark(rowID, "the row content"))
	time.Sleep(20 * time.Millisecond)
	_ = scanned

	z := zm.Get(rowID)
	if z == nil {
		t.Skip("row zone not registered after scan")
		return
	}

	cfg := LongPressConfig{
		MenuOpen:  false,
		ItemCount: 1,
		RowZoneID: func(i int) string { return rowID },
		Threshold: time.Millisecond,
		MakeTick:  func(id int) tea.Msg { return id },
	}
	press := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      z.StartX + 1,
		Y:      z.StartY,
	}
	cmd := m.ArmLongPress(zm, press, cfg)
	if cmd == nil {
		t.Fatal("press over row should return a tick command")
	}
	if m.LongPressItemIndex != 0 {
		t.Errorf("LongPressItemIndex = %d, want 0", m.LongPressItemIndex)
	}
	if m.LongPressPressID != 1 {
		t.Errorf("LongPressPressID = %d, want 1", m.LongPressPressID)
	}
	if msg := cmd(); msg != 1 {
		t.Errorf("tick msg = %v, want press id 1", msg)
	}
}
