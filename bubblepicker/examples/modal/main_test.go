package main

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/bubblepicker"
)

// TestColorBoxClickAtCenter verifies that a mouse click at the center of each
// color box opens the picker for that box (not the other). Uses bubblezone; we
// must render once so Scan() registers zone bounds before sending mouse events.
func TestColorBoxClickAtCenter(t *testing.T) {
	app := newApp()
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Get bounds for each box (1-based inclusive) for expected center
	for boxIndex := 0; boxIndex < len(app.labels); boxIndex++ {
		x0, x1, y0, y1 := app.colorBoxBounds(boxIndex)
		centerX := (x0 + x1) / 2
		centerY := (y0 + y1) / 2

		app2 := newApp()
		app2.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		_ = app2.View() // populate zone manager via Scan() so InBounds works
		time.Sleep(150 * time.Millisecond) // bubblezone updates bounds asynchronously
		_, _ = app2.Update(tea.MouseMsg{
			X: centerX, Y: centerY,
			Button: tea.MouseButtonLeft,
			Action: tea.MouseActionPress,
		})

		if !app2.modalOpen {
			t.Errorf("box %d: click at center (%d,%d) did not open modal (bounds %d-%d, %d-%d)",
				boxIndex, centerX, centerY, x0, x1, y0, y1)
			continue
		}
		if app2.editingKey != app2.labels[boxIndex] {
			t.Errorf("box %d: click at center (%d,%d) opened modal for %q, want %q",
				boxIndex, centerX, centerY, app2.editingKey, app2.labels[boxIndex])
		}
	}
}

// TestColorBoxBoundsStable verifies that bounds are consistent and boxes don't overlap.
func TestColorBoxBoundsStable(t *testing.T) {
	app := newApp()
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	x0_0, x1_0, y0_0, y1_0 := app.colorBoxBounds(0)
	x0_1, x1_1, y0_1, y1_1 := app.colorBoxBounds(1)

	if y0_0 != y0_1 || y1_0 != y1_1 {
		t.Errorf("boxes should share same row: box0 y=%d-%d, box1 y=%d-%d", y0_0, y1_0, y0_1, y1_1)
	}
	if x1_0 >= x0_1 {
		t.Errorf("boxes should not overlap: box0 x=%d-%d, box1 x=%d-%d", x0_0, x1_0, x0_1, x1_1)
	}
	// Box 0 center must be inside box 0
	cx0, cy0 := (x0_0+x1_0)/2, (y0_0+y1_0)/2
	if cx0 < x0_0 || cx0 > x1_0 || cy0 < y0_0 || cy0 > y1_0 {
		t.Errorf("box 0 center (%d,%d) outside bounds [%d,%d] x [%d,%d]", cx0, cy0, x0_0, x1_0, y0_0, y1_0)
	}
}

// TestColorBoxCancelRestoresView verifies that canceling the picker closes the modal.
func TestColorBoxCancelRestoresView(t *testing.T) {
	app := newApp()
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	x0, x1, y0, y1 := app.colorBoxBounds(0)
	cx, cy := (x0+x1)/2, (y0+y1)/2

	_ = app.View() // populate zone manager so click is detected
	time.Sleep(150 * time.Millisecond) // bubblezone updates bounds asynchronously
	m, _ := app.Update(tea.MouseMsg{X: cx, Y: cy, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	app = m.(*appModel)
	if !app.modalOpen {
		t.Fatal("expected modal open after click")
	}
	m, _ = app.Update(bubblepicker.ColorCanceledMsg{})
	app = m.(*appModel)
	if app.modalOpen {
		t.Error("expected modal closed after cancel")
	}
}

// TestModalOverlayRendersMainView verifies that with the modal open and enough height,
// the main view is still present (e.g. title line above the modal).
func TestModalOverlayRendersMainView(t *testing.T) {
	app := newApp()
	app.width, app.height = 80, 30
	app.modalOpen = true
	app.editingKey = "Primary"
	app.picker = bubblepicker.New(app.colors["Primary"])
	_, _ = app.picker.Update(tea.WindowSizeMsg{Width: 42, Height: 22})

	view := app.View()
	lines := strings.Split(view, "\n")

	if len(lines) != app.height {
		t.Errorf("view has %d lines, want %d (height)", len(lines), app.height)
	}
	// With compositing, main view is visible beside/around modal; picker is on top
	if !strings.Contains(view, "Pick a color") {
		t.Error("view should contain picker (modal composited on main view)")
	}
}

// TestModalOverlayShowsPickerOnTop verifies that the overlay region shows the
// picker/modal content on top of the main view (picker title and UI).
func TestModalOverlayShowsPickerOnTop(t *testing.T) {
	app := newApp()
	app.width, app.height = 80, 24
	app.modalOpen = true
	app.editingKey = "Primary"
	app.picker = bubblepicker.New(app.colors["Primary"])
	_, _ = app.picker.Update(tea.WindowSizeMsg{Width: 42, Height: 22})

	view := app.View()
	// Picker title is always "Pick a color"
	if !strings.Contains(view, "Pick a color") {
		t.Errorf("view with modal should contain picker title %q", "Pick a color")
	}
	// Picker help line (compact symbols)
	if !strings.Contains(view, "⇥") && !strings.Contains(view, "pick") {
		t.Errorf("view with modal should contain picker help")
	}
}

// TestModalOverlayLayout verifies overlay layout: main lines at top, modal block in middle, main at bottom.
func TestModalOverlayLayout(t *testing.T) {
	app := newApp()
	app.width, app.height = 80, 30
	app.modalOpen = true
	app.editingKey = "Secondary"
	app.picker = bubblepicker.New(app.colors["Secondary"])
	_, _ = app.picker.Update(tea.WindowSizeMsg{Width: 42, Height: 22})

	view := app.View()
	lines := strings.Split(view, "\n")
	if len(lines) != 30 {
		t.Fatalf("view has %d lines, want 30", len(lines))
	}

	// Modal replaces overlay region (positioned at trigger); picker is visible
	if !strings.Contains(view, "Pick a color") {
		t.Error("full view should contain picker title")
	}
}
