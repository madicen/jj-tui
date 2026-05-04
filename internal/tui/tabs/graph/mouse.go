package graph

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/mousedouble"
)

// handleZoneClick handles zone click messages; returns (updated model, optional request, direct cmd).
func (m GraphModel) handleZoneClick(msg zone.MsgZoneInBounds) (GraphModel, *Request, tea.Cmd) {
	event := msg.Event
	inBounds := func(id string) bool {
		zm := m.zoneManager.Get(id)
		return zm != nil && zm.InBounds(event)
	}

	// Context menu: if visible, check menu item zones first. Any other click dismisses.
	if m.contextMenu != nil {
		items := contextMenuItems()
		for i, item := range items {
			if inBounds(mouse.ZoneCtxMenuItem(i)) {
				fileIdx := m.contextMenu.FileIndex
				m.contextMenu = nil
				m.graphFocused = false
				m.selectedFile = fileIdx
				req := item.Request
				return m, &req, nil
			}
		}
		m.contextMenu = nil
		return m, nil, nil
	}

	// Commit context menu: same pattern as file context menu.
	if m.commitContextMenu != nil {
		firstParentImm := m.commitMenuFirstParentImmutable()
		items := m.commitContextMenuRows(m.commitContextMenu.CommitIndex, firstParentImm)
		for i, item := range items {
			if inBounds(mouse.ZoneCommitCtxMenuItem(i)) {
				ci := m.commitContextMenu.CommitIndex
				m.commitContextMenu = nil
				m.graphFocused = true
				m.selectedCommit = ci
				req := item.Request
				return m, &req, nil
			}
		}
		m.commitContextMenu = nil
		return m, nil, nil
	}

	if msg.Zone == nil {
		return m, nil, nil
	}
	z := msg.Zone

	// Click-drag rebase: resolve on mouse-up before other targets (source = active drag or press-only anchor).
	dragSrc := -1
	switch {
	case m.rebaseDragSource >= 0:
		dragSrc = m.rebaseDragSource
	case m.rebasePressAnchor >= 0:
		dragSrc = m.rebasePressAnchor
	}
	if dragSrc >= 0 && m.selectionMode == SelectionNormal && m.repository != nil {
		dest := -1
		for i := range m.repository.Graph.Commits {
			if inBounds(mouse.ZoneCommit(i)) {
				dest = i
				break
			}
		}
		if dest < 0 {
			m.rebasePressAnchor = -1
			m.rebaseDragSource = -1
			m.rebaseDragHoverDest = -1
		} else if dest != dragSrc {
			src := dragSrc
			m.rebasePressAnchor = -1
			m.rebaseDragSource = -1
			m.rebaseDragHoverDest = -1
			m.graphFocused = true
			m.selectedCommit = dest
			return m, &Request{DragRebase: true, DragRebaseFrom: src, DragRebaseTo: dest}, nil
		} else {
			m.rebasePressAnchor = -1
			m.rebaseDragSource = -1
			m.rebaseDragHoverDest = -1
		}
	}

	if m.zoneOverlap.ShouldSkipOverlappingRelease(event, m.mousePressGen) {
		return m, nil, nil
	}

	if m.repository != nil {
		for commitIndex := range m.repository.Graph.Commits {
			if m.zoneManager.Get(mouse.ZoneCommit(commitIndex)) == z {
				return applyCommitRowMouseSelection(m, commitIndex, event)
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
			return applyChangedFileRowMouseSelection(m, i, event)
		}
	}

	if inBounds(mouse.ZoneGraphPane) {
		if m.repository != nil {
			for commitIndex := range m.repository.Graph.Commits {
				if inBounds(mouse.ZoneCommit(commitIndex)) {
					return applyCommitRowMouseSelection(m, commitIndex, event)
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
				return applyChangedFileRowMouseSelection(m, i, event)
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
	if inBounds(mouse.ZoneActionViewFileDiff) {
		return m, &Request{ViewFileDiff: true}, nil
	}
	if inBounds(mouse.ZoneActionOpenInExternalEditor) {
		return m, &Request{OpenInExternalEditor: true}, nil
	}

	return m, nil, nil
}

func applyCommitRowMouseSelection(m GraphModel, commitIndex int, event tea.MouseMsg) (GraphModel, *Request, tea.Cmd) {
	m.graphFocused = true
	if m.selectionMode == SelectionRebaseDestination {
		return m, &Request{PerformRebase: true, RebaseDestIndex: commitIndex}, nil
	}
	if m.repository == nil || commitIndex < 0 || commitIndex >= len(m.repository.Graph.Commits) {
		return m, nil, nil
	}
	key := fmt.Sprintf("graph:commit:%d", commitIndex)
	if m.rowDoubleClick.ObserveLeftRelease(key, event, time.Now(), mousedouble.DefaultDoubleClickWindow) {
		c := m.repository.Graph.Commits[commitIndex]
		if !c.Immutable && !c.IsWorking {
			m.selectedCommit = commitIndex
			return m, &Request{Checkout: true}, nil
		}
	}
	m.selectedCommit = commitIndex
	m.changedFilesCommitID = ""
	m.changedFiles = nil
	commitID := m.repository.Graph.Commits[commitIndex].ChangeID
	return m, &Request{LoadChangedFiles: &commitID}, nil
}

func applyChangedFileRowMouseSelection(m GraphModel, fileIndex int, event tea.MouseMsg) (GraphModel, *Request, tea.Cmd) {
	m.selectedFile = fileIndex
	m.graphFocused = false
	if fileIndex < 0 || fileIndex >= len(m.changedFiles) {
		return m, nil, nil
	}
	key := fmt.Sprintf("graph:file:%d", fileIndex)
	if m.rowDoubleClick.ObserveLeftRelease(key, event, time.Now(), mousedouble.DefaultDoubleClickWindow) {
		return m, &Request{OpenInExternalEditor: true}, nil
	}
	return m, nil, nil
}
