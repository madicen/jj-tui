package graph

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/mouse"
)

// HandleMessage processes messages for the graph tab
// Returns the updated model, any command to run, and a message to propagate to parent
func (m GraphModel) handleKeyMsg(msg tea.KeyMsg) (GraphModel, tea.Cmd) {
	switch msg.String() {
	// Navigation keys
	case "j", "down":
		if !m.graphFocused {
			// Navigate files in files pane
			if len(m.changedFiles) > 0 && m.selectedFile < len(m.changedFiles)-1 {
				m.selectedFile++
				// Note: viewport scrolling is handled in parent model's ensureFileVisible
			}
		} else {
			// Navigate commits in graph pane
			if m.repository != nil && m.selectedCommit < len(m.repository.Graph.Commits)-1 {
				m.selectedCommit++
				// Note: viewport scrolling and commit loading handled in parent
			}
		}
		return m, nil

	case "k", "up":
		if !m.graphFocused {
			// Navigate files in files pane
			if len(m.changedFiles) > 0 && m.selectedFile > 0 {
				m.selectedFile--
			}
		} else {
			// Navigate commits in graph pane
			if m.selectedCommit > 0 {
				m.selectedCommit--
			}
		}
		return m, nil

	// Focus switching
	case "tab":
		m.graphFocused = !m.graphFocused
		return m, nil

	// Viewport scrolling in graph pane
	case "pgup", "pgdown", "ctrl+u", "ctrl+d", "home", "end", "ctrl+f", "ctrl+b":
		if m.graphFocused {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		} else {
			var cmd tea.Cmd
			m.filesViewport, cmd = m.filesViewport.Update(msg)
			return m, cmd
		}

	// Rebase mode: cancel with esc/q so parent sync gets reset state
	case "esc", "q":
		if m.selectionMode == SelectionRebaseDestination {
			m.selectionMode = SelectionNormal
			m.rebaseSourceCommit = -1
		}
		return m, nil

	// Rebase mode navigation
	case "r":
		if m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			m.selectionMode = SelectionRebaseDestination
			m.rebaseSourceCommit = m.selectedCommit
		}
		return m, nil

		// Selection/action keys handled by parent - just focus changes here
	}

	return m, nil
}

func (m GraphModel) handleZoneClick(zone *zone.ZoneInfo) (GraphModel, tea.Cmd) {
	if zone == nil {
		return m, nil
	}

	// Check for pane focus zones by comparing zone pointers
	if m.zoneManager.Get(mouse.ZoneGraphPane) == zone {
		if !m.graphFocused {
			m.graphFocused = true
		}
		return m, nil
	}

	if m.zoneManager.Get(mouse.ZoneFilesPane) == zone {
		if m.graphFocused {
			m.graphFocused = false
		}
		return m, nil
	}

	// Check for commit selection zones
	if m.repository != nil {
		for commitIndex := range m.repository.Graph.Commits {
			if m.zoneManager.Get(mouse.ZoneCommit(commitIndex)) == zone {
				// If in rebase mode, clicking a commit selects it as destination
				if m.selectionMode == SelectionRebaseDestination {
					// Rebase action is handled by parent model
					return m, nil
				}
				// Normal selection
				m.selectedCommit = commitIndex
				return m, nil
			}
		}
	}

	// Check for changed file zones
	for i := range m.changedFiles {
		if m.zoneManager.Get(mouse.ZoneChangedFile(i)) == zone {
			m.selectedFile = i
			// When clicking a file, switch focus to files pane
			m.graphFocused = false
			return m, nil
		}
	}

	return m, nil
}

// UpdateRepository updates the graph model with new repository data.
// Preserves selection by ChangeID so it survives refresh (no reset).
func (m *GraphModel) UpdateRepository(repo *internal.Repository) {
	if repo == nil {
		return
	}
	oldCommitID := m.changedFilesCommitID
	m.repository = repo
	commits := repo.Graph.Commits
	// Re-resolve selection by ChangeID so refresh doesn't reset it
	if oldCommitID != "" && len(commits) > 0 {
		found := false
		for i, c := range commits {
			if c.ChangeID == oldCommitID {
				m.selectedCommit = i
				found = true
				break
			}
		}
		if !found {
			m.selectedCommit = 0
			m.changedFilesCommitID = ""
			m.changedFiles = nil
		}
	}
	// Clamp selection if repo shrunk
	if m.selectedCommit >= len(commits) {
		m.selectedCommit = max(0, len(commits)-1)
	}
}

// SetDimensions sets the width and height of the graph model
func (m *GraphModel) SetDimensions(width, height int) {
	m.width = width
	m.height = height
}

// SetChangedFiles updates the changed files for the selected commit (only if commitID matches current selection)
func (m *GraphModel) SetChangedFiles(files []jj.ChangedFile, commitID string) {
	if commitID != m.changedFilesCommitID {
		return // Stale load (user selected another commit before this finished)
	}
	m.changedFiles = files
	m.selectedFile = 0
}

// SelectCommit selects a commit by index and sets changedFilesCommitID so refresh can re-resolve selection
func (m *GraphModel) SelectCommit(idx int) {
	if m.repository != nil && idx >= 0 && idx < len(m.repository.Graph.Commits) {
		m.selectedCommit = idx
		m.changedFilesCommitID = m.repository.Graph.Commits[idx].ChangeID
	}
}

// SetSelectionMode sets the selection mode (e.g. rebase destination)
func (m *GraphModel) SetSelectionMode(mode SelectionMode) {
	m.selectionMode = mode
}

// SetRebaseSourceCommit sets the commit index being rebased
func (m *GraphModel) SetRebaseSourceCommit(idx int) {
	m.rebaseSourceCommit = idx
}

// GetSelectionMode returns the current selection mode
func (m *GraphModel) GetSelectionMode() SelectionMode {
	return m.selectionMode
}

// GetRebaseSourceCommit returns the commit index being rebased
func (m *GraphModel) GetRebaseSourceCommit() int {
	return m.rebaseSourceCommit
}

// GetSelectedCommit returns the index of the selected commit
func (m *GraphModel) GetSelectedCommit() int {
	return m.selectedCommit
}

// GetSelectedFile returns the index of the selected file
func (m *GraphModel) GetSelectedFile() int {
	return m.selectedFile
}

// SetSelectedFile sets the selected file index (e.g. from main model mouse handler)
func (m *GraphModel) SetSelectedFile(idx int) {
	if idx >= -1 && idx < len(m.changedFiles) {
		m.selectedFile = idx
	}
}

// IsGraphFocused returns whether the graph pane has focus
func (m *GraphModel) IsGraphFocused() bool {
	return m.graphFocused
}

// SetGraphFocused sets whether the graph pane has focus (e.g. from main model pane click)
func (m *GraphModel) SetGraphFocused(focused bool) {
	m.graphFocused = focused
}

// GetChangedFiles returns the changed files for the selected commit
func (m *GraphModel) GetChangedFiles() []jj.ChangedFile {
	return m.changedFiles
}

// GetChangedFilesCommitID returns the ChangeID for which changed files are loaded (for reload after file ops)
func (m *GraphModel) GetChangedFilesCommitID() string {
	return m.changedFilesCommitID
}

// SetViewport sets the graph viewport (for scrolling support)
func (m *GraphModel) SetViewport(vp viewport.Model) {
	m.viewport = vp
}

// SetFilesViewport sets the files viewport
func (m *GraphModel) SetFilesViewport(vp viewport.Model) {
	m.filesViewport = vp
}

// GetViewport returns the graph viewport
func (m *GraphModel) GetViewport() viewport.Model {
	return m.viewport
}

// GetFilesViewport returns the files viewport
func (m *GraphModel) GetFilesViewport() viewport.Model {
	return m.filesViewport
}

// StartRebaseMode starts rebase mode
func (m *GraphModel) StartRebaseMode(sourceCommitIdx int) {
	m.selectionMode = SelectionRebaseDestination
	m.rebaseSourceCommit = sourceCommitIdx
}

// CancelRebaseMode cancels rebase mode
func (m *GraphModel) CancelRebaseMode() {
	m.selectionMode = SelectionNormal
	m.rebaseSourceCommit = -1
}

// IsInRebaseMode returns whether the graph is in rebase mode
func (m *GraphModel) IsInRebaseMode() bool {
	return m.selectionMode == SelectionRebaseDestination
}
