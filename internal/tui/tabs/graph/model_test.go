package graph

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestGraphModel_Update_HandlesMouseWheelScroll(t *testing.T) {
	m := NewGraphModel(nil)
	// Init viewports via SetDimensions (same path as main view before first WindowSizeMsg)
	m.SetDimensions(80, 24)
	if m.GetViewport().Width == 0 {
		t.Fatal("SetDimensions should init viewports")
	}

	// Set content longer than viewport height so we can scroll
	lines := make([]string, 50)
	for i := range lines {
		lines[i] = "commit " + string(rune('a'+i%26)) + " line"
	}
	content := strings.Join(lines, "\n")
	vp := m.GetViewport()
	vp.SetContent(content)
	m.SetViewport(vp)

	// Ensure we have scrollable content
	if m.GetViewport().TotalLineCount() <= m.GetViewport().Height {
		t.Skipf("need more lines than height: total=%d height=%d",
			m.GetViewport().TotalLineCount(), m.GetViewport().Height)
	}

	y0 := m.GetViewport().YOffset
	wheelDown := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelDown,
		X:      40,
		Y:      10,
	}
	updated, cmd := m.Update(wheelDown)
	if cmd != nil {
		t.Logf("Update returned non-nil cmd (ignored)")
	}
	m2, ok := updated.(*GraphModel)
	if !ok {
		t.Fatalf("Update returned %T, want *GraphModel", updated)
	}
	y1 := m2.GetViewport().YOffset
	if y1 <= y0 {
		t.Errorf("wheel down should increase YOffset: was %d, got %d", y0, y1)
	}

	// Wheel up should decrease
	wheelUp := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelUp,
		X:      40,
		Y:      10,
	}
	updated, _ = m2.Update(wheelUp)
	m3 := updated.(*GraphModel)
	y2 := m3.GetViewport().YOffset
	if y2 >= y1 {
		t.Errorf("wheel up should decrease YOffset: was %d, got %d", y1, y2)
	}
}

func TestGraphModel_SetDimensions_InitsViewports(t *testing.T) {
	m := NewGraphModel(nil)
	// Viewports are pre-initialized in NewGraphModel so mouse wheel works without clicking first
	if m.GetViewport().Width == 0 {
		t.Errorf("new model viewport should be pre-initialized for wheel scroll, got width 0")
	}
	m.SetDimensions(80, 24)
	if m.GetViewport().Width != 80 {
		t.Errorf("after SetDimensions viewport width want 80, got %d", m.GetViewport().Width)
	}
	if m.GetFilesViewport().Width != 80 {
		t.Errorf("after SetDimensions files viewport width want 80, got %d", m.GetFilesViewport().Width)
	}
}
