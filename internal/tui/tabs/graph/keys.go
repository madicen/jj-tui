package graph

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleKeyMsg handles keyboard input; returns (updated model, optional request, direct cmd).
func (m GraphModel) handleKeyMsg(msg tea.KeyMsg) (GraphModel, *Request, tea.Cmd) {
	switch msg.String() {
	// Navigation keys
	case "j", "down":
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

	case "k", "up":
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

	case "tab":
		m.graphFocused = !m.graphFocused
		return m, nil, nil

	case "pgup", "pgdown", "ctrl+u", "ctrl+d", "home", "end", "ctrl+f", "ctrl+b":
		if m.graphFocused {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, nil, cmd
		} else {
			var cmd tea.Cmd
			m.filesViewport, cmd = m.filesViewport.Update(msg)
			return m, nil, cmd
		}

	case "esc", "q":
		if m.selectionMode == SelectionRebaseDestination {
			m.selectionMode = SelectionNormal
			m.rebaseSourceCommit = -1
		}
		return m, nil, nil

	case "r":
		if m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			return m, &Request{StartRebaseMode: true}, nil
		}
		return m, nil, nil

	case "enter", "e":
		if m.graphFocused && m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			if m.selectionMode == SelectionRebaseDestination {
				return m, &Request{PerformRebase: true, RebaseDestIndex: m.selectedCommit}, nil
			}
			return m, &Request{Checkout: true}, nil
		}
		return m, nil, nil

	case "n":
		if m.repository != nil {
			return m, &Request{NewCommit: true}, nil
		}
	case "d":
		if m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			c := m.repository.Graph.Commits[m.selectedCommit]
			if c.Divergent {
				changeID := c.ChangeID
				return m, &Request{ResolveDivergent: &changeID}, nil
			}
			return m, &Request{StartEditDescription: true}, nil
		}
	case "s":
		if m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			return m, &Request{Squash: true}, nil
		}
	case "a":
		if m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			return m, &Request{Abandon: true}, nil
		}
	case "m":
		if m.repository != nil {
			return m, &Request{CreateBookmark: true}, nil
		}
	case "x":
		if m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			return m, &Request{DeleteBookmark: true}, nil
		}
	case "u":
		if m.repository != nil {
			return m, &Request{UpdatePR: true}, nil
		}
	case "c":
		if m.repository != nil {
			return m, &Request{CreatePR: true}, nil
		}
	case "G":
		if m.graphFocused && m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			return m, &Request{MoveDeltaOntoOrigin: true}, nil
		}
	case "[":
		if !m.graphFocused {
			return m, &Request{MoveFileUp: true}, nil
		}
	case "]":
		if !m.graphFocused {
			return m, &Request{MoveFileDown: true}, nil
		}
	case "v":
		if !m.graphFocused {
			return m, &Request{RevertFile: true}, nil
		}
	}

	return m, nil, nil
}
