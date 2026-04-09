package graph

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/mouse"
)

// handleRebaseDragMouse handles press/motion for click-drag rebase. Release is handled in handleZoneClick.
// Rebase row styling only appears after the pointer moves to a different commit than the press target,
// so a simple click to select does not flash rebase highlights.
func (m *GraphModel) handleRebaseDragMouse(msg tea.MouseMsg) {
	if tea.MouseEvent(msg).IsWheel() {
		return
	}
	if m.selectionMode == SelectionRebaseDestination || !m.graphFocused {
		return
	}
	if m.repository == nil {
		return
	}
	inBounds := func(id string) bool {
		z := m.zoneManager.Get(id)
		return z != nil && z.InBounds(msg)
	}
	commitUnder := func() int {
		for i := range m.repository.Graph.Commits {
			if inBounds(mouse.ZoneCommit(i)) {
				return i
			}
		}
		return -1
	}
	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button != tea.MouseButtonLeft {
			return
		}
		idx := commitUnder()
		if idx < 0 {
			m.rebasePressAnchor = -1
			m.rebaseDragSource = -1
			m.rebaseDragHoverDest = -1
			return
		}
		m.rebasePressAnchor = idx
		m.rebaseDragSource = -1
		m.rebaseDragHoverDest = -1
		m.selectedCommit = idx
	case tea.MouseActionMotion:
		if m.rebasePressAnchor < 0 {
			return
		}
		hover := commitUnder()
		if m.rebaseDragSource < 0 {
			// Only start drag styling after the pointer leaves the pressed row while left is held.
			if msg.Button == tea.MouseButtonLeft && hover >= 0 && hover != m.rebasePressAnchor {
				m.rebaseDragSource = m.rebasePressAnchor
			}
		} else if msg.Button == tea.MouseButtonLeft || msg.Button == tea.MouseButtonNone {
			m.rebaseDragHoverDest = hover
		}
	default:
		return
	}
}
