package model

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/actions"
)

// handleKeyMsg handles keyboard input
func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle error state first - error acts as a modal, blocking all other input
	if m.err != nil {
		switch msg.String() {
		case "ctrl+q", "ctrl+c":
			return m, tea.Quit
		case "ctrl+r":
			m.err = nil
			m.errorCopied = false
			m.viewMode = ViewCommitGraph
			return m, m.refreshRepository()
		case "esc":
			// Clear error and go back to graph, restart auto-refresh
			m.err = nil
			m.errorCopied = false
			m.viewMode = ViewCommitGraph
			m.statusMessage = "Error dismissed"
			return m, m.tickCmd()
		case "c":
			// Copy error to clipboard (but not for "not a jj repo" welcome screen)
			if !m.notJJRepo && m.err != nil {
				return m, actions.CopyToClipboard(m.err.Error())
			}
		case "i":
			// Initialize jj repo if not already one
			if m.notJJRepo {
				m.statusMessage = "Initializing repository..."
				return m, m.runJJInit()
			}
		}
		// Ignore all other keys when in error state - error is modal
		return m, nil
	}

	// Handle warning modal (e.g., empty commit descriptions)
	if m.showWarningModal {
		switch msg.String() {
		case "esc":
			m.showWarningModal = false
			m.warningCommits = nil
			m.statusMessage = "Cancelled"
			return m, nil
		case "enter":
			// Go to the selected commit and start editing its description
			if len(m.warningCommits) > 0 && m.warningSelectedIdx < len(m.warningCommits) {
				selectedCommit := m.warningCommits[m.warningSelectedIdx]
				// Find this commit in the graph and select it
				for i, c := range m.repository.Graph.Commits {
					if c.ChangeID == selectedCommit.ChangeID {
						m.graphTabModel.SelectCommit(i)
						m.showWarningModal = false
						m.warningCommits = nil
						// Start editing description
						return m.handleDescribeCommit()
					}
				}
			}
			m.showWarningModal = false
			m.warningCommits = nil
			return m, nil
		case "up", "k":
			if m.warningSelectedIdx > 0 {
				m.warningSelectedIdx--
			}
			return m, nil
		case "down", "j":
			if m.warningSelectedIdx < len(m.warningCommits)-1 {
				m.warningSelectedIdx++
			}
			return m, nil
		case "ctrl+q", "ctrl+c":
			return m, tea.Quit
		}
		// Ignore all other keys when warning modal is shown
		return m, nil
	}

	// Special handling for edit description view
	if m.viewMode == ViewEditDescription {
		return m.handleDescriptionEditKeyMsg(msg)
	}

	// Special handling for settings view
	if m.viewMode == ViewSettings {
		return m.handleSettingsKeyMsg(msg)
	}

	// Special handling for PR creation view
	if m.viewMode == ViewCreatePR {
		return m.handleCreatePRKeyMsg(msg)
	}

	// Special handling for bookmark creation view
	if m.viewMode == ViewCreateBookmark {
		return m.handleCreateBookmarkKeyMsg(msg)
	}

	// Special handling for bookmark conflict resolution view
	if m.viewMode == ViewBookmarkConflict {
		return m.handleBookmarkConflictKeyMsg(msg)
	}

	// Special handling for divergent commit resolution view
	if m.viewMode == ViewDivergentCommit {
		return m.handleDivergentCommitKeyMsg(msg)
	}

	// Special handling for GitHub login view
	if m.viewMode == ViewGitHubLogin {
		switch msg.String() {
		case "esc":
			m.settingsTabModel.SetGitHubLoginPolling(false)
			m.settingsTabModel.SetGitHubDeviceCode("")
			m.settingsTabModel.SetGitHubUserCode("")
			m.viewMode = ViewSettings
			m.statusMessage = "GitHub login cancelled"
			return m, nil
		case "c":
			if code := m.settingsTabModel.GetGitHubUserCode(); code != "" {
				m.statusMessage = "Copying code to clipboard..."
				return m, actions.CopyToClipboard(code)
			}
			return m, nil
		}
		return m, nil
	}

	// Scroll keys are handled by the active tab (graph, PR, tickets, branches) via PropagateUpdate

	switch msg.String() {
	case "ctrl+q", "ctrl+c":
		return m, tea.Quit
	case "g":
		return m.handleNavigateToGraphTab()
	case "p":
		return m.handleNavigateToPRTab()
	case "t":
		return m.handleNavigateToTicketsTab()
	case "b":
		return m.handleNavigateToBranchesTab()
	case ",":
		return m.handleNavigateToSettingsTab()
	case "h", "?":
		return m.handleNavigateToHelpTab()
	case "ctrl+r":
		return m, m.refreshRepository()
	case "ctrl+z":
		return m.handleUndo()
	case "ctrl+y":
		return m.handleRedo()
	case "esc":
		// Cancel status change mode in Tickets view
		if m.viewMode == ViewTickets && m.statusChangeMode {
			m.statusChangeMode = false
			m.ticketsTabModel.SetStatusChangeMode(false)
			m.statusMessage = "Ready"
			return m, nil
		}
		if m.viewMode != ViewCommitGraph {
			m.viewMode = ViewCommitGraph
		}
	case "tab":
		// Tab switching within delegated views is now handled by tab models
		// This only handles tab switching for non-delegated views
		if m.viewMode != ViewCommitGraph {
			m.viewMode = ViewCommitGraph
		}
	case "j", "down":
		// Tab-specific navigation is now handled by tab models via delegation
		// This switch is kept only for view modes that don't have delegated models
		switch m.viewMode {
		default:
			// For non-delegated views, this will fall through to the next case
		}
		// c, i, D, B, N, o (Tickets), T, U, L, P, F (Branches), y (Help) handled by tabs via request
	case "M":
		// Merge PR handled by PRs tab via request
	case "X":
		// Close PR handled by PRs tab via request
	case "enter", "e":
		// PR open-in-browser, Tickets start bookmark, Graph checkout/rebase handled by tabs via request
	case "x":
		// Delete branch bookmark handled by Branches tab via request
	}
	return m, nil
}

// handleCreateBookmarkKeyMsg handles keyboard input in bookmark creation mode
func (m *Model) handleCreateBookmarkKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle universal keys first
	switch msg.String() {
	case "esc":
		return m.handleBookmarkCancel()
	case "enter", "ctrl+s":
		return m.handleBookmarkSubmit()
	case "tab":
		existing := m.bookmarkModal.GetExistingBookmarks()
		sel := m.bookmarkModal.GetSelectedBookmarkIdx()
		if sel == -1 && len(existing) > 0 {
			m.bookmarkModal.SetSelectedBookmarkIdx(0)
			m.bookmarkModal.GetNameInput().Blur()
		} else {
			m.bookmarkModal.SetSelectedBookmarkIdx(-1)
			m.bookmarkModal.GetNameInput().Focus()
		}
		return m, nil
	}

	if m.bookmarkModal.GetSelectedBookmarkIdx() == -1 {
		var cmd tea.Cmd
		ni := m.bookmarkModal.GetNameInput()
		*ni, cmd = ni.Update(msg)
		m.updateBookmarkNameExists()
		return m, cmd
	}

	switch msg.String() {
	case "j", "down":
		existing := m.bookmarkModal.GetExistingBookmarks()
		if len(existing) > 0 {
			sel := m.bookmarkModal.GetSelectedBookmarkIdx()
			if sel < len(existing)-1 {
				m.bookmarkModal.SetSelectedBookmarkIdx(sel + 1)
			}
		}
		return m, nil
	case "k", "up":
		sel := m.bookmarkModal.GetSelectedBookmarkIdx()
		if sel > -1 {
			m.bookmarkModal.SetSelectedBookmarkIdx(sel - 1)
			if sel == 0 {
				m.bookmarkModal.GetNameInput().Focus()
			}
		}
		return m, nil
	}

	return m, nil
}

// handleBookmarkConflictKeyMsg handles keyboard input in bookmark conflict resolution mode
func (m *Model) handleBookmarkConflictKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel and return to branches view
		m.viewMode = ViewBranches
		m.statusMessage = "Conflict resolution cancelled"
		return m, nil
	case "enter":
		m.statusMessage = "Resolving bookmark conflict..."
		return m, m.resolveBookmarkConflict(m.conflictModal.GetBookmarkName(), m.conflictModal.GetSelectedOption())
	case "j", "down":
		if m.conflictModal.GetSelectedOption() != "reset_remote" {
			m.conflictModal.SetSelectedOption(1)
		}
		return m, nil
	case "k", "up":
		if m.conflictModal.GetSelectedOption() != "keep_local" {
			m.conflictModal.SetSelectedOption(0)
		}
		return m, nil
	case "l", "L":
		m.conflictModal.SetSelectedOption(0)
		return m, nil
	case "r", "R":
		m.conflictModal.SetSelectedOption(1)
		return m, nil
	}
	return m, nil
}

// handleDivergentCommitKeyMsg handles keyboard input in divergent commit resolution mode
func (m *Model) handleDivergentCommitKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel and return to graph view
		m.viewMode = ViewCommitGraph
		m.statusMessage = "Divergent commit resolution cancelled"
		return m, nil
	case "enter":
		keepCommitID := m.divergentModal.GetSelectedCommitID()
		if keepCommitID != "" {
			m.statusMessage = "Resolving divergent commit..."
			return m, m.resolveDivergentCommit(m.divergentModal.GetChangeID(), keepCommitID)
		}
		return m, nil
	case "j", "down":
		cur := m.divergentModal.GetSelectedIdx()
		n := m.divergentModal.GetCommitCount()
		if n > 0 && cur < n-1 {
			m.divergentModal.SetSelectedIdx(cur + 1)
		}
		return m, nil
	case "k", "up":
		cur := m.divergentModal.GetSelectedIdx()
		if cur > 0 {
			m.divergentModal.SetSelectedIdx(cur - 1)
		}
		return m, nil
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		idx := int(msg.String()[0] - '1')
		if idx >= 0 && idx < m.divergentModal.GetCommitCount() {
			m.divergentModal.SetSelectedIdx(idx)
		}
		return m, nil
	}
	return m, nil
}

// handleCreatePRKeyMsg delegates to the PR form modal
func (m *Model) handleCreatePRKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m.handlePRCancel()
	case "ctrl+s":
		return m.handlePRSubmit()
	}
	var cmd tea.Cmd
	m.prFormModal, cmd = m.prFormModal.Update(msg)
	return m, cmd
}

// handleRebaseModeKeyMsg handles keyboard input during rebase destination selection
func (m *Model) handleRebaseModeKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		// Cancel rebase mode
		m.cancelRebaseMode()
		return m, nil
	case "j", "down":
		cur := m.graphTabModel.GetSelectedCommit()
		if m.repository != nil && cur < len(m.repository.Graph.Commits)-1 {
			m.graphTabModel.SelectCommit(cur + 1)
		}
		return m, nil
	case "k", "up":
		cur := m.graphTabModel.GetSelectedCommit()
		if cur > 0 {
			m.graphTabModel.SelectCommit(cur - 1)
		}
		return m, nil
	case "enter":
		cur := m.graphTabModel.GetSelectedCommit()
		if cur >= 0 && cur < len(m.repository.Graph.Commits) {
			return m, m.performRebase(cur)
		}
		return m, nil
	}
	return m, nil
}

// handleDescriptionEditKeyMsg handles keys while editing description
func (m *Model) handleDescriptionEditKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m.handleDescriptionCancel()
	case "ctrl+s":
		return m.handleDescriptionSave()
	}

	var cmd tea.Cmd
	descInput := m.graphTabModel.GetDescriptionInput()
	updated, cmd := descInput.Update(msg)
	m.graphTabModel.SetDescriptionInput(updated)
	return m, cmd
}

// handleSettingsKeyMsg handles keys while in settings view
func (m *Model) handleSettingsKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle cleanup confirmation dialog
	if m.settingsTabModel.GetConfirmingCleanup() != "" {
		switch msg.String() {
		case "y", "Y":
			return m, m.confirmCleanup()
		case "n", "N", "esc":
			m.cancelCleanup()
			return m, nil
		}
		return m, nil
	}

	switch msg.String() {
	case "esc":
		return m.handleSettingsCancel()
	case "ctrl+j":
		tab := m.settingsTabModel.GetSettingsTab()
		tab--
		if tab < 0 {
			tab = 5
		}
		m.settingsTabModel.SetSettingsTab(tab)
		return m, nil
	case "ctrl+k":
		m.settingsTabModel.SetSettingsTab((m.settingsTabModel.GetSettingsTab() + 1) % 6)
		return m, nil
	case "ctrl+s", "enter":
		if m.settingsTabModel.GetSettingsTab() == 5 {
			return m, nil
		}
		inputs := m.settingsTabModel.GetSettingsInputs()
		focused := m.settingsTabModel.GetFocusedField()
		if msg.String() == "enter" && focused < len(inputs)-1 {
			m.settingsTabModel.SetFocusedField(focused + 1)
			for i := range inputs {
				if i == focused+1 {
					inputs[i].Focus()
				} else {
					inputs[i].Blur()
				}
			}
			return m, nil
		}
		return m, m.saveSettings()
	case "tab", "down":
		if m.settingsTabModel.GetSettingsTab() == 4 || m.settingsTabModel.GetSettingsTab() == 5 {
			return m, nil
		}
		inputs := m.settingsTabModel.GetSettingsInputs()
		n := len(inputs)
		if n == 0 {
			return m, nil
		}
		focused := m.settingsTabModel.GetFocusedField()
		next := (focused + 1) % n
		m.settingsTabModel.SetFocusedField(next)
		for i := range inputs {
			if i == next {
				inputs[i].Focus()
			} else {
				inputs[i].Blur()
			}
		}
		return m, nil
	case "shift+tab", "up":
		if m.settingsTabModel.GetSettingsTab() == 4 || m.settingsTabModel.GetSettingsTab() == 5 {
			return m, nil
		}
		inputs := m.settingsTabModel.GetSettingsInputs()
		n := len(inputs)
		if n == 0 {
			return m, nil
		}
		focused := m.settingsTabModel.GetFocusedField()
		next := focused - 1
		if next < 0 {
			next = n - 1
		}
		m.settingsTabModel.SetFocusedField(next)
		for i := range inputs {
			if i == next {
				inputs[i].Focus()
			} else {
				inputs[i].Blur()
			}
		}
		return m, nil
	}

	if m.settingsTabModel.GetSettingsTab() == 4 || m.settingsTabModel.GetSettingsTab() == 5 {
		return m, nil
	}
	inputs := m.settingsTabModel.GetSettingsInputs()
	focused := m.settingsTabModel.GetFocusedField()
	if focused < 0 || focused >= len(inputs) {
		return m, nil
	}
	var cmd tea.Cmd
	inputs[focused], cmd = inputs[focused].Update(msg)
	return m, cmd
}
