package graph

import (
	"github.com/charmbracelet/bubbles/textarea"
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
				m.scrollToSelectedFile = true
			}
		} else {
			// Navigate commits in graph pane
			if m.repository != nil && m.selectedCommit < len(m.repository.Graph.Commits)-1 {
				m.selectedCommit++
				m.changedFilesCommitID = m.repository.Graph.Commits[m.selectedCommit].ChangeID
				m.scrollToSelectedCommit = true
				return m, Request{LoadChangedFiles: &m.changedFilesCommitID}.Cmd()
			}
		}
		return m, nil

	case "k", "up":
		if !m.graphFocused {
			// Navigate files in files pane
			if len(m.changedFiles) > 0 && m.selectedFile > 0 {
				m.selectedFile--
				m.scrollToSelectedFile = true
			}
		} else {
			// Navigate commits in graph pane
			if m.selectedCommit > 0 {
				m.selectedCommit--
				m.changedFilesCommitID = m.repository.Graph.Commits[m.selectedCommit].ChangeID
				m.scrollToSelectedCommit = true
				return m, Request{LoadChangedFiles: &m.changedFilesCommitID}.Cmd()
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

	// Rebase mode: start (r) → main checks immutable; confirm destination (enter)
	case "r":
		if m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			return m, Request{StartRebaseMode: true}.Cmd()
		}
		return m, nil

	case "enter", "e":
		if m.graphFocused && m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			if m.selectionMode == SelectionRebaseDestination {
				return m, Request{PerformRebase: true, RebaseDestIndex: m.selectedCommit}.Cmd()
			}
			return m, Request{Checkout: true}.Cmd()
		}
		return m, nil

	case "n":
		if m.repository != nil {
			return m, Request{NewCommit: true}.Cmd()
		}
	case "d":
		if m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			return m, Request{StartEditDescription: true}.Cmd()
		}
	case "s":
		if m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			return m, Request{Squash: true}.Cmd()
		}
	case "a":
		if m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			c := m.repository.Graph.Commits[m.selectedCommit]
			if c.Divergent {
				changeID := c.ChangeID
				return m, Request{ResolveDivergent: &changeID}.Cmd()
			}
			return m, Request{Abandon: true}.Cmd()
		}
	case "m":
		if m.repository != nil {
			return m, Request{CreateBookmark: true}.Cmd()
		}
	case "x":
		if m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			return m, Request{DeleteBookmark: true}.Cmd()
		}
	case "u":
		if m.repository != nil {
			return m, Request{UpdatePR: true}.Cmd()
		}
	case "c":
		if m.repository != nil {
			return m, Request{CreatePR: true}.Cmd()
		}
	case "[":
		if !m.graphFocused {
			return m, Request{MoveFileUp: true}.Cmd()
		}
	case "]":
		if !m.graphFocused {
			return m, Request{MoveFileDown: true}.Cmd()
		}
	case "v":
		if !m.graphFocused {
			return m, Request{RevertFile: true}.Cmd()
		}
	}

	return m, nil
}

func (m GraphModel) handleZoneClick(msg zone.MsgZoneInBounds) (GraphModel, tea.Cmd) {
	if msg.Zone == nil {
		return m, nil
	}
	zone := msg.Zone
	event := msg.Event
	inBounds := func(id string) bool {
		z := m.zoneManager.Get(id)
		return z != nil && z.InBounds(event)
	}

	// Check commit and file zones first (the zone that was actually hit).
	// Pane zones wrap the whole graph/files area, so we must not check them first or they'd swallow commit clicks.
	if m.repository != nil {
		for commitIndex := range m.repository.Graph.Commits {
			if m.zoneManager.Get(mouse.ZoneCommit(commitIndex)) == zone {
				m.graphFocused = true // clicking a commit focuses the graph pane
				if m.selectionMode == SelectionRebaseDestination {
					return m, Request{PerformRebase: true, RebaseDestIndex: commitIndex}.Cmd()
				}
				m.selectedCommit = commitIndex
				m.changedFilesCommitID = m.repository.Graph.Commits[commitIndex].ChangeID
				return m, Request{LoadChangedFiles: &m.changedFilesCommitID}.Cmd()
			}
		}
	}
	for i := range m.changedFiles {
		if m.zoneManager.Get(mouse.ZoneChangedFile(i)) == zone {
			m.selectedFile = i
			m.graphFocused = false
			return m, nil
		}
	}

	// Pane focus: only when click wasn't on a commit or file (e.g. connector lines or empty area).
	// If zone was the pane (not a commit), check by position whether we're inside a commit zone so commit clicks still work.
	if inBounds(mouse.ZoneGraphPane) {
		if m.repository != nil {
			for commitIndex := range m.repository.Graph.Commits {
				if inBounds(mouse.ZoneCommit(commitIndex)) {
					m.graphFocused = true // clicking a commit focuses the graph pane
					if m.selectionMode == SelectionRebaseDestination {
						return m, Request{PerformRebase: true, RebaseDestIndex: commitIndex}.Cmd()
					}
					m.selectedCommit = commitIndex
					m.changedFilesCommitID = m.repository.Graph.Commits[commitIndex].ChangeID
					return m, Request{LoadChangedFiles: &m.changedFilesCommitID}.Cmd()
				}
			}
		}
		if !m.graphFocused {
			m.graphFocused = true
		}
		return m, nil
	}
	if inBounds(mouse.ZoneFilesPane) {
		for i := range m.changedFiles {
			if inBounds(mouse.ZoneChangedFile(i)) {
				m.selectedFile = i
				m.graphFocused = false
				return m, nil
			}
		}
		if m.graphFocused {
			m.graphFocused = false
		}
		return m, nil
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

// SetDimensions sets the width and height of the graph model and lazy-inits viewports if needed
// so that mouse wheel scrolling works even when the graph never received a WindowSizeMsg.
func (m *GraphModel) SetDimensions(width, height int) {
	m.width = width
	m.height = height
	if m.viewport.Width == 0 && width > 0 && height > 0 {
		h := max(1, height/2)
		m.viewport = viewport.New(max(1, width), h)
		m.viewport.MouseWheelEnabled = true
		m.filesViewport = viewport.New(max(1, width), h)
		m.filesViewport.MouseWheelEnabled = true
	}
}

// SetChangedFiles updates the changed files for the selected commit.
// Accepts when commitID matches changedFilesCommitID, or when it matches the currently selected commit's ChangeID
// (so initial load after repository load is applied even if changedFilesCommitID wasn't set before the async load).
func (m *GraphModel) SetChangedFiles(files []jj.ChangedFile, commitID string) {
	accept := (commitID == m.changedFilesCommitID) ||
		(m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) &&
			commitID == m.repository.Graph.Commits[m.selectedCommit].ChangeID)
	if !accept {
		return
	}
	if m.changedFilesCommitID != commitID {
		m.changedFilesCommitID = commitID
	}
	m.changedFiles = files
	m.selectedFile = 0
	m.scrollToSelectedFile = true
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
		m.scrollToSelectedFile = true
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

// Description editing (ViewEditDescription)
func (m *GraphModel) GetDescriptionInput() *textarea.Model {
	return &m.descriptionInput
}

func (m *GraphModel) SetDescriptionInput(ta textarea.Model) {
	m.descriptionInput = ta
}

func (m *GraphModel) GetEditingCommitID() string {
	return m.editingCommitID
}

func (m *GraphModel) SetEditingCommitID(id string) {
	m.editingCommitID = id
}
