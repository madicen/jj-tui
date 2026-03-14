// Package bubblepicker provides SwatchPicker: a color square that opens the full
// modal picker on click, with overlay positioning and mouse offset handled internally.

package bubblepicker

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

// SwatchPicker is a color swatch (square + optional label + hex) that opens the
// modal picker when clicked. Embed it in your model, set bounds, forward messages,
// and use ViewWithOverlay(mainView, width, height) for the view. On ColorChosenMsg,
// update the swatch with SetColor(msg.Color) (or store the color in your app).
// SetZoneManager so the picker uses zone-based mouse interaction (host must Scan the view).
type SwatchPicker struct {
	color   string
	label   string
	row     int
	col     int
	w       int
	h       int
	picker  Model
	open    bool
	focused bool // When true, arrow is highlighted (for keyboard-only clients)

	// Optional zone manager: when set, picker uses zones and receives raw screen mouse events.
	zoneManager *zone.Manager

	// Set by ViewWithOverlay for in-modal bounds check when forwarding mouse
	lastOverlayLeft    int
	lastOverlayTop     int
	lastModalW         int
	lastOverlayHeight  int
	lastViewWidth      int
	lastViewHeight     int
}

// Picker symbol shown to the right of the color square (indicates "click to open picker").
const swatchPickerSymbol = "▼"

// NewSwatchPicker returns a swatch that shows color and opens the modal picker on click.
// initialColor is hex (e.g. "#7E00AF"); label is optional (not shown in the minimal UI).
func NewSwatchPicker(initialColor, label string) *SwatchPicker {
	if initialColor == "" {
		initialColor = "#7E00AF"
	}
	return &SwatchPicker{
		color: initialColor,
		label: label,
	}
}

// SwatchView returns the swatch as a single line: one cell of color plus the picker symbol (▼).
// No border. When focused (e.g. for keyboard nav), the arrow is highlighted.
func (s *SwatchPicker) SwatchView() string {
	colorBlock := lipgloss.NewStyle().Background(lipgloss.Color(s.color)).Render(" ")
	symbol := swatchPickerSymbol
	if s.focused {
		symbol = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true).Render(symbol)
	}
	return colorBlock + symbol
}

// SetFocused sets whether the swatch is focused (e.g. for keyboard-only navigation).
// When true, the picker arrow (▼) is rendered in a brighter color.
func (s *SwatchPicker) SetFocused(f bool) {
	s.focused = f
}

// Focused returns whether the swatch is currently focused.
func (s *SwatchPicker) Focused() bool {
	return s.focused
}

// Open returns whether the swatch's color picker modal is currently open.
// Use this so your app can route input only to the open swatch when a modal is active.
func (s *SwatchPicker) Open() bool {
	return s.open
}

// SetZoneManager sets the zone manager for the picker. When set, the picker uses
// zone-based mouse interaction (click hue bar / grid / Accept) and receives raw
// screen mouse events; the host must run zone.Scan() on the view that contains the overlay.
func (s *SwatchPicker) SetZoneManager(zm *zone.Manager) {
	s.zoneManager = zm
}

// Size returns the display size (width, height in cells) of the swatch: 2 wide (color + symbol), 1 high.
// Use this when building your layout and when calling SetBounds.
func (s *SwatchPicker) Size() (width, height int) {
	return 2, 1
}

// SetBounds sets where the swatch is drawn (0-based row, col) and its size (w, h).
// If w or h is 0, Size() is used for that dimension. Call this before ViewWithOverlay
// so the modal is centered on the swatch and clicks are detected correctly.
func (s *SwatchPicker) SetBounds(row, col, w, h int) {
	s.row = row
	s.col = col
	if w <= 0 || h <= 0 {
		pw, ph := s.Size()
		if w <= 0 {
			s.w = pw
		} else {
			s.w = w
		}
		if h <= 0 {
			s.h = ph
		} else {
			s.h = h
		}
	} else {
		s.w = w
		s.h = h
	}
}

// SetColor sets the current color (hex). Call this when you receive ColorChosenMsg.
func (s *SwatchPicker) SetColor(c string) {
	s.color = c
}

// Color returns the current color (hex).
func (s *SwatchPicker) Color() string {
	return s.color
}

// ViewWithOverlay returns the view to display. If the picker is open, it overlays
// the modal on mainView. It stores overlay position and dimensions on the receiver
// so Update can use them for mouse offset—no need to reassign the return value.
//
//   return app.swatch.ViewWithOverlay(mainView, width, height)
func (s *SwatchPicker) ViewWithOverlay(mainView string, viewWidth, viewHeight int) string {
	if !s.open {
		return mainView
	}
	modalContent := s.picker.View()
	modalLines := strings.Split(modalContent, "\n")
	overlayHeight := len(modalLines)
	modalW := 0
	for _, l := range modalLines {
		if w := lipgloss.Width(l); w > modalW {
			modalW = w
		}
	}
	centerRow := s.row + s.h/2
	centerCol := s.col + s.w/2
	leftPad := centerCol - modalW/2
	topPad := centerRow - overlayHeight/2
	leftPad = max(leftPad, 0)
	if leftPad+modalW > viewWidth {
		leftPad = max(viewWidth-modalW, 0)
	}
	topPad = max(topPad, 0)
	if topPad+overlayHeight > viewHeight {
		topPad = max(viewHeight-overlayHeight, 0)
	}
	s.lastOverlayLeft = leftPad
	s.lastOverlayTop = topPad
	s.lastModalW = modalW
	s.lastOverlayHeight = overlayHeight
	s.lastViewWidth = viewWidth
	s.lastViewHeight = viewHeight
	return OverlayView(mainView, modalContent, viewWidth, viewHeight, topPad, leftPad)
}

// Update handles messages. Forward all tea.Msg to it. When the user picks a color,
// you'll receive ColorChosenMsg; call SetColor(msg.Color) and assign the returned
// model back. When the picker is open, Update forwards to the picker with the
// correct mouse offset.
func (s *SwatchPicker) Update(msg tea.Msg) (*SwatchPicker, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		s.lastViewWidth = m.Width
		s.lastViewHeight = m.Height
		if s.open {
			picker, cmd := s.picker.Update(tea.WindowSizeMsg{Width: 42, Height: 22})
			s.picker = picker.(Model)
			// Recompute overlay position so mouse coords stay correct after resize
			if s.lastOverlayHeight > 0 && s.lastModalW > 0 {
				centerRow := s.row + s.h/2
				centerCol := s.col + s.w/2
				leftPad := centerCol - s.lastModalW/2
				topPad := centerRow - s.lastOverlayHeight/2
				leftPad = max(leftPad, 0)
				if s.lastViewWidth > 0 && leftPad+s.lastModalW > s.lastViewWidth {
					leftPad = max(s.lastViewWidth-s.lastModalW, 0)
				}
				topPad = max(topPad, 0)
				if s.lastViewHeight > 0 && topPad+s.lastOverlayHeight > s.lastViewHeight {
					topPad = max(s.lastViewHeight-s.lastOverlayHeight, 0)
				}
				s.lastOverlayLeft = leftPad
				s.lastOverlayTop = topPad
			}
			return s, cmd
		}
		return s, nil

	case tea.KeyMsg:
		if s.open {
			updated, cmd := s.picker.Update(m)
			s.picker = updated.(Model)
			return s, cmd
		}
		return s, nil

	case tea.MouseMsg:
		if s.open {
			leftPad := s.lastOverlayLeft
			topPad := s.lastOverlayTop
			if s.lastModalW <= 0 && s.lastViewWidth > 0 {
				leftPad = max((s.lastViewWidth-44)/2, 0)
			}
			if s.lastOverlayHeight <= 0 && s.lastViewHeight > 0 {
				topPad = max((s.lastViewHeight-22)/2, 0)
			}
			// Only forward to picker when click is inside the modal rect (X 0-based, Y 1-based).
			inModal := m.X >= leftPad && m.X < leftPad+s.lastModalW &&
				m.Y >= topPad+1 && m.Y <= topPad+s.lastOverlayHeight
			if !inModal {
				return s, nil
			}
			// When using zones, picker gets raw screen coords (zones are registered from full view).
			if s.zoneManager != nil {
				updated, cmd := s.picker.Update(m)
				s.picker = updated.(Model)
				return s, cmd
			}
			relX, relY := MouseToModalCoords(m.X, m.Y, leftPad, topPad)
			relMsg := tea.MouseMsg{
				X: relX, Y: relY,
				Button: m.Button, Action: m.Action, Alt: m.Alt, Ctrl: m.Ctrl, Shift: m.Shift,
			}
			updated, cmd := s.picker.Update(relMsg)
			s.picker = updated.(Model)
			return s, cmd
		}
		if m.Action == tea.MouseActionPress && m.Button == tea.MouseButtonLeft {
			// When zoneManager is set, the app only forwards to us when the zone was in bounds,
			// so we must not re-check bounds (zone covers e.g. "Color 1: ■▼", not just the 2-cell swatch).
			// When zoneManager is nil, use 0-based X and 1-based Y bounds.
			inBounds := s.zoneManager != nil ||
				(m.X >= s.col && m.X < s.col+s.w && m.Y >= s.row+1 && m.Y <= s.row+s.h)
			if inBounds {
				next := *s
				next.picker = New(s.color)
				if s.zoneManager != nil {
					next.picker.SetZoneManager(s.zoneManager)
				}
				picker, cmd := next.picker.Update(tea.WindowSizeMsg{Width: 42, Height: 22})
				next.picker = picker.(Model)
				next.open = true
				// Compute overlay position so first mouse event uses correct offset (no fallback)
				modalW, overlayHeight := next.picker.ViewSize()
				centerRow := next.row + next.h/2
				centerCol := next.col + next.w/2
				leftPad := centerCol - modalW/2
				topPad := centerRow - overlayHeight/2
				leftPad = max(leftPad, 0)
				if next.lastViewWidth > 0 && leftPad+modalW > next.lastViewWidth {
					leftPad = max(next.lastViewWidth-modalW, 0)
				}
				topPad = max(topPad, 0)
				if next.lastViewHeight > 0 && topPad+overlayHeight > next.lastViewHeight {
					topPad = max(next.lastViewHeight-overlayHeight, 0)
				}
				next.lastOverlayLeft = leftPad
				next.lastOverlayTop = topPad
				next.lastModalW = modalW
				next.lastOverlayHeight = overlayHeight
				return &next, cmd
			}
		}
		return s, nil

	case ColorChosenMsg:
		if !s.open {
			return s, nil
		}
		next := *s
		next.color = m.Color
		next.open = false
		return &next, nil

	case ColorCanceledMsg:
		if !s.open {
			return s, nil
		}
		next := *s
		next.open = false
		return &next, nil
	}

	if s.open {
		updated, cmd := s.picker.Update(msg)
		s.picker = updated.(Model)
		return s, cmd
	}
	return s, nil
}

// MouseToModalCoords converts screen (x, y) from Bubble Tea to modal-relative (relX, relY)
// for the picker. X is 0-based, Y is 1-based. The picker expects Y=1 for the first row and
// X=2 for the first content column (col 0 = padding; it does col-- then contentCol = col-1).
func MouseToModalCoords(screenX, screenY, overlayLeft, overlayTop int) (relX, relY int) {
	// Y 1-based: first overlay line is at screen Y = overlayTop+1 -> pass relY=1
	relY = screenY - overlayTop
	// X 0-based: first overlay column is at screen X = overlayLeft -> picker expects relX=2 for first content column
	relX = screenX - overlayLeft + 2
	return relX, relY
}
