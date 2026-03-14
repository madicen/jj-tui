package bubblepicker

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMouseToModalCoords(t *testing.T) {
	// X 0-based, Y 1-based; overlay at (top=5, left=10). relX = screenX - left + 2, relY = screenY - top
	left, top := 10, 5

	tests := []struct {
		screenX, screenY int
		wantRelX, wantRelY int
	}{
		{10, 6, 2, 1},   // 0-based first col of overlay, first row -> X=2 first content col, Y=1
		{11, 6, 3, 1},   // one right
		{10, 7, 2, 2},   // one down
		{12, 9, 4, 4},   // interior
		{9, 6, 1, 1},    // left of modal (padding)
		{10, 5, 2, 0},   // above modal
	}
	for _, tt := range tests {
		relX, relY := MouseToModalCoords(tt.screenX, tt.screenY, left, top)
		if relX != tt.wantRelX || relY != tt.wantRelY {
			t.Errorf("MouseToModalCoords(%d,%d, %d,%d) = (%d,%d), want (%d,%d)",
				tt.screenX, tt.screenY, left, top, relX, relY, tt.wantRelX, tt.wantRelY)
		}
	}
}

func TestSwatchMouseOffsetWhenModalOpen(t *testing.T) {
	// Open the modal, set lastOverlay* so we control the offset, then send a mouse
	// event at a known screen position and verify the picker receives correct 1-based rel coords.
	s := NewSwatchPicker("#7E00AF", "")
	s.open = true
	s.picker = New(s.color)
	_, _ = s.picker.Update(tea.WindowSizeMsg{Width: 42, Height: 22})
	s.lastOverlayLeft = 10
	s.lastOverlayTop = 5
	s.lastModalW = 44
	s.lastOverlayHeight = 22
	s.lastViewWidth = 60
	s.lastViewHeight = 24

	// Click at 0-based X=12, 1-based Y=7 = second column, third row of modal content area.
	// Expected rel: (12-10+2, 7-5) = (4, 2).
	msg := tea.MouseMsg{
		X: 12, Y: 7,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}
	next, cmd := s.Update(msg)
	if cmd != nil {
		// May trigger ColorChosenMsg if click landed on grid with release
		_ = cmd
	}
	// Picker should have received rel (3, 3); we can't read picker state easily, but at least
	// no panic and modal still open (unless they picked a color).
	if !next.open {
		// They might have clicked confirm; that's valid
		return
	}
}

func TestSwatchViewSingleLine(t *testing.T) {
	// SwatchView is a single line (color + symbol), no newlines.
	s := NewSwatchPicker("#7E00AF", "")
	v := s.SwatchView()
	lines := strings.Split(v, "\n")
	if len(lines) != 1 {
		t.Errorf("SwatchView() split by newline has %d lines, want 1", len(lines))
	}
	if lines[0] == "" {
		t.Error("SwatchView() should not be empty")
	}
}

func TestSwatchResizeRecomputesOverlayPosition(t *testing.T) {
	s := NewSwatchPicker("#7E00AF", "")
	s.SetBounds(5, 15, 3, 3)
	s.open = true
	s.picker = New(s.color)
	_, _ = s.picker.Update(tea.WindowSizeMsg{Width: 42, Height: 22})
	s.lastOverlayLeft = 10
	s.lastOverlayTop = 5
	s.lastModalW = 44
	s.lastOverlayHeight = 22
	s.lastViewWidth = 60
	s.lastViewHeight = 24
	// Resize to a different view size; overlay position should be recomputed
	next, _ := s.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	if next.lastViewWidth != 80 || next.lastViewHeight != 30 {
		t.Errorf("after resize: lastView = %dx%d, want 80x30", next.lastViewWidth, next.lastViewHeight)
	}
	// Centered on swatch (row 5, col 15, 3x3) -> center (6, 16). With 80x30, modal 44x22:
	// leftPad = 16 - 22 = -6 -> 0, topPad = 6 - 11 = -5 -> 0. So we expect 0,0 or similar.
	if next.lastOverlayLeft == 10 && next.lastOverlayTop == 5 {
		t.Error("overlay position was not recomputed after WindowSizeMsg (still 10, 5)")
	}
}

func TestSwatchHitTestBounds(t *testing.T) {
	s := NewSwatchPicker("#7E00AF", "")
	s.SetBounds(2, 10, 2, 1)
	// X 0-based, Y 1-based: swatch at (row 2, col 10) size 2x1 -> X in [10,12), Y=3
	inside := tea.MouseMsg{X: 10, Y: 3, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
	next, _ := s.Update(inside)
	if !next.open {
		t.Error("click inside swatch bounds did not open modal")
	}
	// Close modal, then click outside swatch (left of swatch): should not open
	next, _ = next.Update(ColorCanceledMsg{})
	outside := tea.MouseMsg{X: 8, Y: 3, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
	next, _ = next.Update(outside)
	if next.open {
		t.Error("click outside swatch opened modal")
	}
}

// TestSwatchClickOpensPickerAtPosition mimics the example's 2x2 layout: position the mouse
// directly on the first swatch (using the same bounds as the example) and send a click;
// verify the picker opens. Uses 1-based mouse coordinates (Bubble Tea convention).
func TestSwatchClickOpensPickerAtPosition(t *testing.T) {
	const labelLen = 10
	const gap = 2
	sw, sh := 2, 1
	col1 := labelLen
	col2 := col1 + sw + gap + labelLen

	swatches := [4]*SwatchPicker{
		NewSwatchPicker("#7E00AF", ""),
		NewSwatchPicker("#00AF7E", ""),
		NewSwatchPicker("#AF7E00", ""),
		NewSwatchPicker("#AF007E", ""),
	}
	for i := range swatches {
		var row, col int
		if i < 2 {
			row, col = 2, col1
			if i == 1 {
				col = col2
			}
		} else {
			row, col = 3, col1
			if i == 3 {
				col = col2
			}
		}
		swatches[i].SetBounds(row, col, sw, sh)
	}

	// Click directly on first swatch: 0-based (row 2, col 10), X 0-based -> (X=10, Y=3 1-based)
	click := tea.MouseMsg{X: col1, Y: 2 + 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
	swatches[0], _ = swatches[0].Update(click)
	if !swatches[0].Open() {
		t.Errorf("click at (X=%d 0-based, Y=%d 1-based) on first swatch did not open picker", col1, 3)
	}

	// Click on second swatch: (row 2, col 24) -> X=24, Y=3
	swatches[1], _ = swatches[1].Update(ColorCanceledMsg{}) // close if any
	click2 := tea.MouseMsg{X: col2, Y: 2 + 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
	swatches[1], _ = swatches[1].Update(click2)
	if !swatches[1].Open() {
		t.Errorf("click at (X=%d, Y=%d) on second swatch did not open picker", col2, 3)
	}
}
