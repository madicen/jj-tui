package model

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// handleKeyMsg handles keyboard input
func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle error state first - always allow quit and retry
	if m.err != nil {
		switch msg.String() {
		case "ctrl+q", "ctrl+c":
			return m, tea.Quit
		case "ctrl+r":
			m.err = nil
			m.viewMode = ViewCommitGraph
			return m, m.refreshRepository()
		case "esc":
			// Clear error and go back to graph, restart auto-refresh
			m.err = nil
			m.viewMode = ViewCommitGraph
			m.statusMessage = "Error dismissed"
			return m, m.tickCmd()
		case "i":
			// Initialize jj repo if not already one
			if m.notJJRepo {
				m.statusMessage = "Initializing repository..."
				return m, m.runJJInit()
			}
		}
		// Ignore other keys when in error state
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

	// Special handling for GitHub login view
	if m.viewMode == ViewGitHubLogin {
		if msg.String() == "esc" {
			m.githubLoginPolling = false
			m.githubDeviceCode = ""
			m.githubUserCode = ""
			m.viewMode = ViewSettings
			m.statusMessage = "GitHub login cancelled"
			return m, nil
		}
		return m, nil
	}

	// Special handling for rebase mode
	if m.selectionMode == SelectionRebaseDestination {
		return m.handleRebaseModeKeyMsg(msg)
	}

	// Handle scroll keys - pass to viewport
	switch msg.String() {
	case "pgup", "pgdown", "ctrl+u", "ctrl+d", "home", "end", "ctrl+f", "ctrl+b":
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "ctrl+q", "ctrl+c":
		return m, tea.Quit
	case "g":
		return m.handleNavigateToGraphTab()
	case "p":
		return m.handleNavigateToPRTab()
	case "t": // 't' for tickets
		return m.handleNavigateToTicketsTab()
	case ",": // ',' for settings (like many apps use comma for settings)
		return m.handleNavigateToSettingsTab()
	case "h", "?":
		return m.handleNavigateToHelpTab()
	case "n":
		if m.viewMode == ViewCommitGraph {
			return m.handleNewCommit()
		}
	case "d":
		// Edit description of selected commit
		if m.viewMode == ViewCommitGraph {
			return m.handleDescribeCommit()
		}
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
			m.statusMessage = "Ready"
			return m, nil
		}
		if m.viewMode != ViewCommitGraph {
			m.viewMode = ViewCommitGraph
		}
	case "tab":
		// Switch focus between graph and files panes in graph view
		if m.viewMode == ViewCommitGraph {
			m.graphFocused = !m.graphFocused
			m.statusMessage = m.handleGraphFoucsMessage()
		}
	case "j", "down":
		switch m.viewMode {
		case ViewPullRequests:
			if m.repository != nil {
				m.selectedPR = min(m.selectedPR+1, len(m.repository.PRs)-1)
				m.ensureGraphCommitVisible(m.selectedPR)
			}
		case ViewTickets:
			if m.selectedTicket < len(m.ticketList)-1 {
				m.selectedTicket++
				// Scroll viewport to keep selection visible
				m.ensureSelectionVisible(m.selectedTicket)
				// Load transitions for newly selected ticket
				m.availableTransitions = nil
				m.loadingTransitions = true
				return m, m.loadTransitions()
			}
		case ViewCommitGraph:
			if !m.graphFocused {
				// Scroll files pane down (if there are files)
				if len(m.changedFiles) > 0 {
					m.filesViewport.ScrollDown(1)
				}
			} else {
				// Navigate commits in graph pane
				if m.repository != nil && m.selectedCommit < len(m.repository.Graph.Commits)-1 {
					m.selectedCommit++
					// Scroll viewport to keep selection visible
					m.ensureGraphCommitVisible(m.selectedCommit)
					return m.handleSelectCommit(m.selectedCommit)
				}
			}
		}
	case "k", "up":
		switch m.viewMode {
		case ViewPullRequests:
			if m.selectedPR > 0 {
				m.selectedPR--
				// Scroll viewport to keep selection visible
				m.ensureSelectionVisible(m.selectedPR)
			}
		case ViewTickets:
			if m.selectedTicket > 0 {
				m.selectedTicket--
				// Scroll viewport to keep selection visible
				m.ensureSelectionVisible(m.selectedTicket)
				// Load transitions for newly selected ticket
				m.availableTransitions = nil
				m.loadingTransitions = true
				return m, m.loadTransitions()
			}
		case ViewCommitGraph:
			if !m.graphFocused {
				// Scroll files pane up (if there are files)
				if len(m.changedFiles) > 0 {
					m.filesViewport.ScrollUp(1)
				}
			} else {
				// Navigate commits in graph pane
				if m.selectedCommit > 0 && m.repository != nil {
					m.selectedCommit--
					// Scroll viewport to keep selection visible
					m.ensureGraphCommitVisible(m.selectedCommit)
					return m.handleSelectCommit(m.selectedCommit)
				}
			}
		}
	case "c":
		// Toggle status change mode (Tickets view only)
		if m.viewMode == ViewTickets {
			return m.handleToggleStatusChangeMode()
		}
		if m.viewMode == ViewCommitGraph {
			return m.handleCreatePR()
		}
	case "i":
		// Set ticket to "In Progress" (Tickets view only, requires status change mode)
		if m.viewMode == ViewTickets && m.statusChangeMode {
			return m.handleTransitionToInProgress()
		}
	case "D":
		// Set ticket to "Done" (Tickets view only, requires status change mode)
		if m.viewMode == ViewTickets && m.statusChangeMode {
			return m.handleTransitionToDone()
		}
	case "B":
		// Set ticket to "Blocked" (Tickets view only, requires status change mode)
		if m.viewMode == ViewTickets && m.statusChangeMode {
			return m.handleTransitionToBlocked()
		}
	case "N":
		// Set ticket to "Not Started" (Tickets view only, requires status change mode)
		if m.viewMode == ViewTickets && m.statusChangeMode {
			return m.handleTransitionToNotStarted()
		}
	case "o":
		// Open PR URL in browser (PR view only)
		if m.viewMode == ViewPullRequests {
			return m.handleOpenPRInBrowser()
		}
		// Open ticket URL in browser (Tickets view only)
		if m.viewMode == ViewTickets {
			return m.handleOpenTicketInBrowser()
		}
	case "M":
		// Merge PR (PR view only, open PRs only)
		if m.viewMode == ViewPullRequests {
			return m.handleMergePR()
		}
	case "X":
		// Close PR (PR view only, open PRs only)
		if m.viewMode == ViewPullRequests {
			return m.handleClosePR()
		}
	case "enter", "e":
		// In PR view, open the PR in browser
		if m.viewMode == ViewPullRequests && m.repository != nil && m.selectedPR >= 0 && m.selectedPR < len(m.repository.PRs) {
			pr := m.repository.PRs[m.selectedPR]
			if pr.URL != "" {
				m.statusMessage = fmt.Sprintf("Opening PR #%d...", pr.Number)
				return m, openURL(pr.URL)
			}
		}
		// In Tickets view, start bookmark creation from ticket
		if m.viewMode == ViewTickets && m.selectedTicket >= 0 && m.selectedTicket < len(m.ticketList) && m.jjService != nil {
			ticket := m.ticketList[m.selectedTicket]
			m.startBookmarkFromTicket(ticket)
			return m, nil
		}
		// In commit view, edit selected commit (jj edit)
		if m.viewMode == ViewCommitGraph {
			return m.handleCheckoutCommit()
		}
	case "s":
		if m.viewMode == ViewCommitGraph {
			return m.handleSquashCommit()
		}
	case "a":
		if m.viewMode == ViewCommitGraph {
			return m.handleAbandonCommit()
		}
	case "r":
		if m.viewMode == ViewCommitGraph {
			return m.handleRebase()
		}
	case "b":
		if m.viewMode == ViewCommitGraph {
			return m.handleCreateBookmark()
		}
	case "x":
		// Delete bookmark from selected commit
		if m.viewMode == ViewCommitGraph && m.isSelectedCommitValid() && m.jjService != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if len(commit.Branches) == 0 {
				m.statusMessage = "No bookmark on this commit to delete"
				return m, nil
			}
			return m, m.deleteBookmark()
		}
	case "u":
		// Push updates to PR (for commits with PR branches or their descendants)
		if m.viewMode == ViewCommitGraph {
			return m.handleUpdatePR()
		}
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
		// Toggle between new bookmark input and existing bookmarks list
		if m.selectedBookmarkIdx == -1 && len(m.existingBookmarks) > 0 {
			m.selectedBookmarkIdx = 0
			m.bookmarkNameInput.Blur()
		} else {
			m.selectedBookmarkIdx = -1
			m.bookmarkNameInput.Focus()
		}
		return m, nil
	}

	// If we're in "new bookmark" mode (input focused), pass all other keys to input
	if m.selectedBookmarkIdx == -1 {
		var cmd tea.Cmd
		m.bookmarkNameInput, cmd = m.bookmarkNameInput.Update(msg)
		return m, cmd
	}

	// Navigation only applies when in existing bookmarks list mode
	switch msg.String() {
	case "j", "down":
		// Navigate down in existing bookmarks list
		if len(m.existingBookmarks) > 0 {
			if m.selectedBookmarkIdx < len(m.existingBookmarks)-1 {
				m.selectedBookmarkIdx++
			}
		}
		return m, nil
	case "k", "up":
		// Navigate up in existing bookmarks list (or to new bookmark input)
		if m.selectedBookmarkIdx > -1 {
			m.selectedBookmarkIdx--
			if m.selectedBookmarkIdx == -1 {
				m.bookmarkNameInput.Focus()
			}
		}
		return m, nil
	}

	return m, nil
}

// handleCreatePRKeyMsg handles keyboard input in PR creation mode
func (m *Model) handleCreatePRKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m.handlePRCancel()
	case "ctrl+s":
		return m.handlePRSubmit()
	case "tab", "down":
		// Move to next field
		if m.prFocusedField == 0 {
			m.prFocusedField = 1
			m.prTitleInput.Blur()
			m.prBodyInput.Focus()
		}
		return m, nil
	case "shift+tab", "up":
		// Move to previous field
		if m.prFocusedField == 1 {
			m.prFocusedField = 0
			m.prBodyInput.Blur()
			m.prTitleInput.Focus()
		}
		return m, nil
	}

	// Pass other keys to the focused input
	var cmd tea.Cmd
	if m.prFocusedField == 0 {
		m.prTitleInput, cmd = m.prTitleInput.Update(msg)
	} else {
		m.prBodyInput, cmd = m.prBodyInput.Update(msg)
	}
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
		// Move down in commit list
		if m.repository != nil && m.selectedCommit < len(m.repository.Graph.Commits)-1 {
			m.selectedCommit++
		}
		return m, nil
	case "k", "up":
		// Move up in commit list
		if m.selectedCommit > 0 {
			m.selectedCommit--
		}
		return m, nil
	case "enter":
		// Confirm rebase destination
		if m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			return m, m.performRebase(m.selectedCommit)
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

	// Pass other keys to the textarea
	var cmd tea.Cmd
	m.descriptionInput, cmd = m.descriptionInput.Update(msg)
	return m, cmd
}

// handleSettingsKeyMsg handles keys while in settings view
func (m *Model) handleSettingsKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle cleanup confirmation dialog
	if m.confirmingCleanup != "" {
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
		// Previous sub-tab
		m.settingsTab--
		if m.settingsTab < 0 {
			m.settingsTab = 3
		}
		return m, nil
	case "ctrl+k":
		// Next sub-tab
		m.settingsTab = (m.settingsTab + 1) % 4
		return m, nil
	case "ctrl+s", "enter":
		// Handle Advanced tab specially
		if m.settingsTab == 3 {
			// Advanced tab - these don't use settings saving, they use direct actions
			return m, nil
		}
		// If on a field and press enter, move to next field
		// If on last field, save
		if msg.String() == "enter" && m.settingsFocusedField < len(m.settingsInputs)-1 {
			m.settingsFocusedField++
			for i := range m.settingsInputs {
				if i == m.settingsFocusedField {
					m.settingsInputs[i].Focus()
				} else {
					m.settingsInputs[i].Blur()
				}
			}
			return m, nil
		}
		// Save settings to global config
		return m, m.saveSettings()
	case "ctrl+l":
		// Save settings to local .jj-tui.json
		return m, m.saveSettingsLocal()
	case "tab", "down":
		// Skip tab for Advanced (no input fields)
		if m.settingsTab == 3 {
			return m, nil
		}
		// Move to next field
		m.settingsFocusedField = (m.settingsFocusedField + 1) % len(m.settingsInputs)
		for i := range m.settingsInputs {
			if i == m.settingsFocusedField {
				m.settingsInputs[i].Focus()
			} else {
				m.settingsInputs[i].Blur()
			}
		}
		return m, nil
	case "shift+tab", "up":
		// Skip tab for Advanced (no input fields)
		if m.settingsTab == 3 {
			return m, nil
		}
		// Move to previous field
		m.settingsFocusedField--
		if m.settingsFocusedField < 0 {
			m.settingsFocusedField = len(m.settingsInputs) - 1
		}
		for i := range m.settingsInputs {
			if i == m.settingsFocusedField {
				m.settingsInputs[i].Focus()
			} else {
				m.settingsInputs[i].Blur()
			}
		}
		return m, nil
	}

	// Skip input handling for Advanced tab
	if m.settingsTab == 3 {
		return m, nil
	}

	// Pass other keys to the focused input
	var cmd tea.Cmd
	m.settingsInputs[m.settingsFocusedField], cmd = m.settingsInputs[m.settingsFocusedField].Update(msg)
	return m, cmd
}
