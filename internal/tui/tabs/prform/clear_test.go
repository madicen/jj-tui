package prform

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/mouse"
)

// TestClearMarksEmptyOnlyTheirField verifies each red ✗ clears its own field and focuses it.
func TestClearMarksEmptyOnlyTheirField(t *testing.T) {
	m := NewModel(nil)
	m.Show(0, "main", "feature-x")
	m.SetTitle("title")
	m.SetBody("body")

	m, _ = m.handleZoneClick(mouse.ZonePRBodyClear)
	if m.GetBody() != "" || m.GetTitle() != "title" || m.GetFocusedField() != 1 {
		t.Fatalf("body clear: title=%q body=%q focus=%d", m.GetTitle(), m.GetBody(), m.GetFocusedField())
	}

	m.SetBody("body")
	m, _ = m.handleZoneClick(mouse.ZonePRTitleClear)
	if m.GetTitle() != "" || m.GetBody() != "body" || m.GetFocusedField() != 0 {
		t.Fatalf("title clear: title=%q body=%q focus=%d", m.GetTitle(), m.GetBody(), m.GetFocusedField())
	}

	m.SetTitle("title")
	m.SetFocusedField(1)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlX})
	if m.GetBody() != "" || m.GetTitle() != "title" {
		t.Fatalf("ctrl+x on body: title=%q body=%q", m.GetTitle(), m.GetBody())
	}
}
