package graph

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/mouse"
)

// handleRebaseDragMouse handles press/motion for click-drag rebase. Release is handled in handleZoneClick.
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
			return
		}
		m.rebaseDragSource = idx
		m.rebaseDragHoverDest = idx
		m.selectedCommit = idx
	case tea.MouseActionMotion:
		if m.rebaseDragSource < 0 {
			return
		}
		m.rebaseDragHoverDest = commitUnder()
	default:
		return
	}
}
