// Package listnav holds the shared list-tab navigation, scrolling, and
// long-press gesture state used by the branches, PRs, and tickets tabs.
//
// Historically each of those tabs re-implemented the same list scroll offset
// clamping, wheel-scroll handling, "scroll selection into view" logic, and the
// press/motion/release long-press arming state machine. Model centralises that
// mechanical plumbing while leaving each tab in control of its own rendering,
// context-menu contents, and request/action dispatch.
package listnav

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/tui/longpress"
)

// wheelStep is the number of list rows a single mouse-wheel notch scrolls.
const wheelStep = 3

// Model holds the shared list scroll offset and long-press arming state. It is
// meant to be embedded (anonymously) in a tab's own Model so the fields and
// methods promote onto the tab.
type Model struct {
	// YOffset is the vertical scroll offset of the list region (details stay
	// fixed above it).
	YOffset int

	// Long-press arming state for row context menus.
	LongPressItemIndex int
	LongPressPressID   int
	LongPressMouseX    int
	LongPressMouseY    int
}

// New returns a Model with the long-press index reset to the "not armed"
// sentinel (-1).
func New() Model {
	return Model{LongPressItemIndex: -1}
}

// WheelScroll adjusts YOffset for mouse-wheel events, returning true when the
// event was a wheel notch (and therefore fully handled). Non-wheel mouse events
// return false so the caller can route them to long-press handling.
func (m *Model) WheelScroll(msg tea.MouseMsg) bool {
	isWheel := tea.MouseEvent(msg).IsWheel() ||
		msg.Button == tea.MouseButtonWheelUp ||
		msg.Button == tea.MouseButtonWheelDown
	if !isWheel {
		return false
	}
	isUp := msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelLeft
	if isUp {
		m.YOffset -= wheelStep
		if m.YOffset < 0 {
			m.YOffset = 0
		}
	} else {
		m.YOffset += wheelStep
	}
	return true
}

// LongPressConfig parameterises ArmLongPress for a specific tab.
type LongPressConfig struct {
	// MenuOpen reports whether a context menu (or submenu) is already open. When
	// true, motion never disarms via the slack box and a fresh press does not
	// start a new long-press.
	MenuOpen bool
	// ItemCount is the number of selectable rows in the list.
	ItemCount int
	// RowZoneID maps a row index to its bubblezone id.
	RowZoneID func(i int) string
	// Threshold is how long the press must be held before MakeTick fires.
	Threshold time.Duration
	// MakeTick builds the tea.Msg emitted after Threshold, carrying the press id
	// so a stale tick (from an earlier, since-cancelled press) can be ignored.
	MakeTick func(pressID int) tea.Msg
}

// ArmLongPress runs the shared press/motion/release arming state machine and
// returns the tick command to schedule when a press begins over a row. Callers
// remain responsible for any menu hover-tracking, which is tab-specific.
func (m *Model) ArmLongPress(zm *zone.Manager, msg tea.MouseMsg, cfg LongPressConfig) tea.Cmd {
	switch msg.Action {
	case tea.MouseActionMotion:
		// Stay armed while the cursor remains over the originating row or within
		// the small slack box around the anchor. Doesn't apply once a menu is
		// already shown — those have their own hover tracking.
		if !cfg.MenuOpen && m.LongPressItemIndex >= 0 {
			origin := cfg.RowZoneID(m.LongPressItemIndex)
			if !longpress.StillArmed(zm, origin, m.LongPressMouseX, m.LongPressMouseY, msg) {
				m.LongPressItemIndex = -1
			}
		}

	case tea.MouseActionPress:
		if msg.Button != tea.MouseButtonLeft {
			return nil
		}
		if cfg.MenuOpen {
			return nil
		}
		for i := 0; i < cfg.ItemCount; i++ {
			z := zm.Get(cfg.RowZoneID(i))
			if z != nil && z.InBounds(msg) {
				m.LongPressPressID++
				m.LongPressItemIndex = i
				m.LongPressMouseX = msg.X
				m.LongPressMouseY = msg.Y
				pressID := m.LongPressPressID
				threshold := cfg.Threshold
				makeTick := cfg.MakeTick
				return tea.Tick(threshold, func(time.Time) tea.Msg {
					return makeTick(pressID)
				})
			}
		}

	case tea.MouseActionRelease:
		m.LongPressItemIndex = -1
	}
	return nil
}

// ScrollToSelected nudges YOffset so that the row at index selected is visible
// within a viewport of listHeight rows. It is a no-op when selected is out of
// range (< 0 or >= total).
func (m *Model) ScrollToSelected(selected, total, listHeight int) {
	if selected < 0 || selected >= total {
		return
	}
	if selected < m.YOffset {
		m.YOffset = selected
	} else if selected >= m.YOffset+listHeight {
		m.YOffset = selected - listHeight + 1
	}
}

// VisibleRange clamps YOffset to a valid range for total lines shown in a
// viewport of listHeight rows and returns the [start, end) slice bounds of the
// visible window.
func (m *Model) VisibleRange(total, listHeight int) (start, end int) {
	maxOffset := 0
	if total > listHeight {
		maxOffset = total - listHeight
	}
	if m.YOffset > maxOffset {
		m.YOffset = maxOffset
	}
	if m.YOffset < 0 {
		m.YOffset = 0
	}
	start = m.YOffset
	end = start + listHeight
	if end > total {
		end = total
	}
	return start, end
}
