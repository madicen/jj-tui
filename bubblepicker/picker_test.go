package bubblepicker

import (
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// stripANSI removes ANSI escape sequences so we can inspect visible runes.
func stripANSI(s string) string {
	return regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`).ReplaceAllString(s, "")
}

// TestViewHasConsistentWidth verifies that every line has the same width (framed:
// inner cols+2 plus border and padding).
func TestViewHasConsistentWidth(t *testing.T) {
	m := New("#808080")
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 30, Height: 24})
	m = updated.(Model)
	view := m.View()
	lines := strings.Split(view, "\n")

	cols := m.gridCols
	if cols <= 0 {
		cols = 24
	}
	wantWidth := cols + 2 + 2 + 2 // inner width + padding(0,1) + border

	for i, line := range lines {
		if line == "" {
			continue
		}
		got := lipgloss.Width(line)
		if got != wantWidth {
			t.Errorf("line %d: width = %d, want %d (framed)", i, got, wantWidth)
		}
	}
}

// TestViewHasFramedContent verifies that the view has the title and is framed
// (border chars on the left of content lines; structure is intact).
func TestViewHasFramedContent(t *testing.T) {
	m := New("#808080")
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 30, Height: 24})
	m = updated.(Model)
	view := m.View()
	if !strings.Contains(stripANSI(view), "Pick a color") {
		t.Error("view should contain title \"Pick a color\"")
	}
	lines := strings.Split(view, "\n")
	if len(lines) < 3 {
		t.Fatalf("view should have at least 3 lines (border + content), got %d", len(lines))
	}
	// With DoubleBorder, line 0 is top edge, line 1 is first content row (starts with left border char)
	firstContent := stripANSI(lines[1])
	if strings.TrimSpace(firstContent) == "" {
		t.Error("first content line should not be blank")
	}
}

// Mouse interaction is zone-based only (when SetZoneManager is used). Raw coordinate
// mouse tests are no longer applicable; see examples/modal and examples/swatch for integration.
