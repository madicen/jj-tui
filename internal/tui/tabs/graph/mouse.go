package graph

import (
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/tui/mouse"
)

// handleZoneClick handles zone click messages; returns (updated model, optional request, direct cmd).
func (m GraphModel) handleZoneClick(msg zone.MsgZoneInBounds) (GraphModel, *Request, tea.Cmd) {
	if msg.Zone == nil {
		return m, nil, nil
	}
	z := msg.Zone
	event := msg.Event
	inBounds := func(id string) bool {
		zm := m.zoneManager.Get(id)
		return zm != nil && zm.InBounds(event)
	}

	if m.repository != nil {
		for commitIndex := range m.repository.Graph.Commits {
			if m.zoneManager.Get(mouse.ZoneCommit(commitIndex)) == z {
				m.graphFocused = true
				if m.selectionMode == SelectionRebaseDestination {
					return m, &Request{PerformRebase: true, RebaseDestIndex: commitIndex}, nil
				}
				m.selectedCommit = commitIndex
				m.changedFilesCommitID = ""
				m.changedFiles = nil
				commitID := m.repository.Graph.Commits[commitIndex].ChangeID
				return m, &Request{LoadChangedFiles: &commitID}, nil
			}
		}
		for commitIndex := range m.repository.Graph.Commits {
			if m.zoneManager.Get(mouse.ZoneActionMoveOntoOriginAt(commitIndex)) == z {
				m.graphFocused = true
				m.selectedCommit = commitIndex
				m.changedFilesCommitID = ""
				m.changedFiles = nil
				return m, &Request{MoveDeltaOntoOrigin: true}, nil
			}
		}
		for commitIndex := range m.repository.Graph.Commits {
			if m.zoneManager.Get(mouse.ZoneActionEvologSplitAt(commitIndex)) == z {
				m.graphFocused = true
				m.selectedCommit = commitIndex
				m.changedFilesCommitID = ""
				m.changedFiles = nil
				return m, &Request{StartEvologSplit: true}, nil
			}
		}
		for commitIndex := range m.repository.Graph.Commits {
			if m.zoneManager.Get(mouse.ZoneActionResolveBookmarkConflictAt(commitIndex)) == z {
				m.graphFocused = true
				m.selectedCommit = commitIndex
				m.changedFilesCommitID = ""
				m.changedFiles = nil
				return m, &Request{ResolveBookmarkConflict: true}, nil
			}
		}
	}
	for i := range m.changedFiles {
		if m.zoneManager.Get(mouse.ZoneChangedFile(i)) == z {
			m.selectedFile = i
			m.graphFocused = false
			return m, nil, nil
		}
	}

	if inBounds(mouse.ZoneGraphPane) {
		if m.repository != nil {
			for commitIndex := range m.repository.Graph.Commits {
				if inBounds(mouse.ZoneCommit(commitIndex)) {
					m.graphFocused = true
					if m.selectionMode == SelectionRebaseDestination {
						return m, &Request{PerformRebase: true, RebaseDestIndex: commitIndex}, nil
					}
					m.selectedCommit = commitIndex
					m.changedFilesCommitID = ""
					m.changedFiles = nil
					commitID := m.repository.Graph.Commits[commitIndex].ChangeID
					return m, &Request{LoadChangedFiles: &commitID}, nil
				}
			}
			for commitIndex := range m.repository.Graph.Commits {
				if inBounds(mouse.ZoneActionMoveOntoOriginAt(commitIndex)) {
					m.graphFocused = true
					m.selectedCommit = commitIndex
					m.changedFilesCommitID = ""
					m.changedFiles = nil
					return m, &Request{MoveDeltaOntoOrigin: true}, nil
				}
			}
			for commitIndex := range m.repository.Graph.Commits {
				if inBounds(mouse.ZoneActionEvologSplitAt(commitIndex)) {
					m.graphFocused = true
					m.selectedCommit = commitIndex
					m.changedFilesCommitID = ""
					m.changedFiles = nil
					return m, &Request{StartEvologSplit: true}, nil
				}
			}
		}
		if !m.graphFocused {
			m.graphFocused = true
		}
		return m, nil, nil
	}
	if inBounds(mouse.ZoneFilesPane) {
		for i := range m.changedFiles {
			if inBounds(mouse.ZoneChangedFile(i)) {
				m.selectedFile = i
				m.graphFocused = false
				return m, nil, nil
			}
		}
		if m.graphFocused {
			m.graphFocused = false
		}
		return m, nil, nil
	}

	if inBounds(mouse.ZoneActionNewCommit) {
		return m, &Request{NewCommit: true}, nil
	}
	if inBounds(mouse.ZoneActionCheckout) {
		return m, &Request{Checkout: true}, nil
	}
	if inBounds(mouse.ZoneActionDescribe) {
		return m, &Request{StartEditDescription: true}, nil
	}
	if inBounds(mouse.ZoneActionSquash) {
		return m, &Request{Squash: true}, nil
	}
	if inBounds(mouse.ZoneActionRebase) {
		return m, &Request{StartRebaseMode: true}, nil
	}
	if inBounds(mouse.ZoneActionAbandon) {
		return m, &Request{Abandon: true}, nil
	}
	if inBounds(mouse.ZoneActionBookmark) {
		return m, &Request{CreateBookmark: true}, nil
	}
	if inBounds(mouse.ZoneActionDelBookmark) {
		return m, &Request{DeleteBookmark: true}, nil
	}
	if inBounds(mouse.ZoneActionResolveDivergent) {
		if m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			c := m.repository.Graph.Commits[m.selectedCommit]
			if c.Divergent {
				changeID := c.ChangeID
				return m, &Request{ResolveDivergent: &changeID}, nil
			}
		}
		return m, nil, nil
	}
	if inBounds(mouse.ZoneActionResolveBookmarkConflict) {
		if m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			c := m.repository.Graph.Commits[m.selectedCommit]
			if len(c.ConflictedBranches) > 0 {
				return m, &Request{ResolveBookmarkConflict: true}, nil
			}
		}
		return m, nil, nil
	}
	if inBounds(mouse.ZoneActionUpdatePR) {
		return m, &Request{UpdatePR: true}, nil
	}
	if inBounds(mouse.ZoneActionCreatePR) {
		return m, &Request{CreatePR: true}, nil
	}
	if inBounds(mouse.ZoneActionMoveFileUp) {
		return m, &Request{MoveFileUp: true}, nil
	}
	if inBounds(mouse.ZoneActionMoveFileDown) {
		return m, &Request{MoveFileDown: true}, nil
	}
	if inBounds(mouse.ZoneActionRevertFile) {
		return m, &Request{RevertFile: true}, nil
	}

	return m, nil, nil
}
