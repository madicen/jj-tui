package graph

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// handleKeyMsg handles keyboard input; returns (updated model, optional request, direct cmd).
func (m GraphModel) handleKeyMsg(msg tea.KeyMsg) (GraphModel, *Request, tea.Cmd) {
	// The blame overlay owns the keyboard while shown.
	if m.annotate != nil && m.annotate.shown {
		return m.handleAnnotateKey(msg)
	}
	switch {
	// Navigation keys
	case key.Matches(msg, m.keys.MoveDown):
		if !m.graphFocused {
			if len(m.changedFiles) > 0 && m.selectedFile < len(m.changedFiles)-1 {
				m.selectedFile++
				m.scrollToSelectedFile = true
			}
		} else {
			if m.repository != nil && m.selectedCommit < len(m.repository.Graph.Commits)-1 {
				m.selectedCommit++
				m.changedFilesCommitID = ""
				m.changedFiles = nil
				m.scrollToSelectedCommit = true
				commitID := m.repository.Graph.Commits[m.selectedCommit].ChangeID
				return m, &Request{LoadChangedFiles: &commitID}, nil
			}
		}
		return m, nil, nil

	case key.Matches(msg, m.keys.MoveUp):
		if !m.graphFocused {
			if len(m.changedFiles) > 0 && m.selectedFile > 0 {
				m.selectedFile--
				m.scrollToSelectedFile = true
			}
		} else {
			if m.selectedCommit > 0 {
				m.selectedCommit--
				m.changedFilesCommitID = ""
				m.changedFiles = nil
				m.scrollToSelectedCommit = true
				commitID := m.repository.Graph.Commits[m.selectedCommit].ChangeID
				return m, &Request{LoadChangedFiles: &commitID}, nil
			}
		}
		return m, nil, nil

	case key.Matches(msg, m.keys.ToggleFocus):
		m.graphFocused = !m.graphFocused
		return m, nil, nil

	case key.Matches(msg, m.keys.Scroll):
		if m.graphFocused {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, nil, cmd
		}
		var cmd tea.Cmd
		m.filesViewport, cmd = m.filesViewport.Update(msg)
		return m, nil, cmd

	case key.Matches(msg, m.keys.CancelSelection):
		if m.contextMenu != nil {
			m.contextMenu = nil
			return m, nil, nil
		}
		if m.commitContextMenu != nil {
			m.commitContextMenu = nil
			return m, nil, nil
		}
		if m.selectionMode == SelectionRebaseDestination {
			m.selectionMode = SelectionNormal
			m.rebaseSourceCommit = -1
			m.duplicateMode = false
		}
		if m.selectionMode == SelectionMergeSource {
			m.selectionMode = SelectionNormal
			m.mergeTargetCommit = -1
		}
		m.rebasePressAnchor = -1
		m.rebaseDragSource = -1
		m.rebaseDragHoverDest = -1
		return m, nil, nil

	case key.Matches(msg, m.keys.Rebase):
		if m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			return m, &Request{StartRebaseMode: true}, nil
		}
		return m, nil, nil

	case key.Matches(msg, m.keys.Merge):
		if m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			return m, &Request{StartMergeMode: true}, nil
		}
		return m, nil, nil

	case key.Matches(msg, m.keys.Checkout):
		if m.graphFocused && m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			if m.selectionMode == SelectionRebaseDestination {
				return m, &Request{PerformRebase: true, RebaseDestIndex: m.selectedCommit}, nil
			}
			if m.selectionMode == SelectionMergeSource {
				return m, &Request{PerformMerge: true, MergeSourceIndex: m.selectedCommit}, nil
			}
			return m, &Request{Checkout: true}, nil
		}
		return m, nil, nil

	case key.Matches(msg, m.keys.NewCommit):
		if m.repository != nil {
			return m, &Request{NewCommit: true}, nil
		}
	case key.Matches(msg, m.keys.EditDescription):
		if m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			c := m.repository.Graph.Commits[m.selectedCommit]
			if c.Divergent {
				changeID := c.ChangeID
				return m, &Request{ResolveDivergent: &changeID}, nil
			}
			return m, &Request{StartEditDescription: true}, nil
		}
	case key.Matches(msg, m.keys.Squash):
		if m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			return m, &Request{Squash: true}, nil
		}
	case key.Matches(msg, m.keys.Abandon):
		if m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			return m, &Request{Abandon: true}, nil
		}
	case key.Matches(msg, m.keys.Absorb):
		// Absorb always operates on the working copy (@), so it doesn't depend on
		// the current graph selection — only on having a loaded repository.
		if m.repository != nil {
			return m, &Request{StartAbsorb: true}, nil
		}
	case key.Matches(msg, m.keys.Duplicate):
		if m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			return m, &Request{Duplicate: true}, nil
		}
	case key.Matches(msg, m.keys.CreateBookmark):
		if m.repository != nil {
			return m, &Request{CreateBookmark: true}, nil
		}
	case key.Matches(msg, m.keys.DeleteBookmark):
		if m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			return m, &Request{DeleteBookmark: true}, nil
		}
	case key.Matches(msg, m.keys.UpdatePR):
		if m.repository != nil {
			return m, &Request{UpdatePR: true}, nil
		}
	case key.Matches(msg, m.keys.CreatePR):
		// Match Branches tab: resolve diverged bookmark with lowercase c. (Create PR only when not conflicted.)
		if m.repository != nil && m.graphFocused && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			c := m.repository.Graph.Commits[m.selectedCommit]
			if len(c.ConflictedBranches) > 0 {
				return m, &Request{ResolveBookmarkConflict: true}, nil
			}
		}
		if m.repository != nil {
			return m, &Request{CreatePR: true}, nil
		}
	case key.Matches(msg, m.keys.ResolveConflict):
		if m.graphFocused && m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			c := m.repository.Graph.Commits[m.selectedCommit]
			if len(c.ConflictedBranches) > 0 {
				return m, &Request{ResolveBookmarkConflict: true}, nil
			}
		}
	case key.Matches(msg, m.keys.MoveDelta):
		if m.graphFocused && m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			c := m.repository.Graph.Commits[m.selectedCommit]
			if c.HasDeltaVsBookmarkOrigin {
				return m, &Request{MoveDeltaOntoOrigin: true}, nil
			}
		}
	case key.Matches(msg, m.keys.EvologSplit):
		if m.graphFocused && m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			c := m.repository.Graph.Commits[m.selectedCommit]
			if c.EvologSplitViable {
				return m, &Request{StartEvologSplit: true}, nil
			}
		}
	case key.Matches(msg, m.keys.MoveFileUp):
		if !m.graphFocused {
			return m, &Request{MoveFileUp: true}, nil
		}
	case key.Matches(msg, m.keys.MoveFileDown):
		if !m.graphFocused {
			return m, &Request{MoveFileDown: true}, nil
		}
	case key.Matches(msg, m.keys.RevertFile):
		if !m.graphFocused {
			return m, &Request{RevertFile: true}, nil
		}
	case key.Matches(msg, m.keys.ViewFileDiff):
		if !m.graphFocused {
			return m, &Request{ViewFileDiff: true}, nil
		}
	case key.Matches(msg, m.keys.Annotate):
		if !m.graphFocused {
			return m, &Request{Annotate: true}, nil
		}
	case key.Matches(msg, m.keys.OpenExternal):
		if !m.graphFocused {
			return m, &Request{OpenInExternalEditor: true}, nil
		}
	}

	return m, nil, nil
}
