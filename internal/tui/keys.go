package tui

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
			m.statusMessage = "Refreshing..."
			m.loading = true
			return m, m.loadRepository()
		case "esc":
			// Clear error and go back to graph
			m.err = nil
			m.viewMode = ViewCommitGraph
			m.statusMessage = "Error dismissed"
			return m, nil
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
		m.viewMode = ViewCommitGraph
	case "p":
		m.viewMode = ViewPullRequests
		// Load PRs when switching to PR view
		if m.githubService != nil {
			m.statusMessage = "Loading PRs..."
			return m, m.loadPRs()
		} else {
			m.statusMessage = "GitHub service not initialized"
		}
	case "t": // 't' for tickets
		m.viewMode = ViewJira
		// Load tickets when switching to Tickets view
		if m.jiraService != nil {
			m.statusMessage = "Loading tickets..."
			return m, m.loadJiraTickets()
		}
	case ",": // ',' for settings (like many apps use comma for settings)
		m.viewMode = ViewSettings
		// Focus first input when entering settings
		m.settingsFocusedField = 0
		for i := range m.settingsInputs {
			if i == 0 {
				m.settingsInputs[i].Focus()
			} else {
				m.settingsInputs[i].Blur()
			}
		}
	case "h", "?":
		m.viewMode = ViewHelp
	case "n":
		if m.viewMode == ViewCommitGraph && m.jjService != nil {
			// Create a new commit
			m.statusMessage = "Creating new commit..."
			return m, m.createNewCommit()
		}
	case "d":
		// Edit description of selected commit
		if m.viewMode == ViewCommitGraph && m.selectedCommit >= 0 && m.jjService != nil && m.repository != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if commit.Immutable {
				m.statusMessage = "Cannot edit description: commit is immutable"
				return m, nil
			}
			return m.startEditingDescription(commit)
		}
	case "ctrl+r":
		m.statusMessage = "Refreshing..."
		m.loading = true
		// Always refresh PRs too if GitHub is connected (needed for Update PR button on graph)
		if m.githubService != nil {
			return m, tea.Batch(m.loadRepository(), m.loadPRs())
		}
		return m, m.loadRepository()
	case "esc":
		if m.viewMode != ViewCommitGraph {
			m.viewMode = ViewCommitGraph
		}
	case "j", "down":
		if m.viewMode == ViewPullRequests {
			if m.repository != nil && m.selectedPR < len(m.repository.PRs)-1 {
				m.selectedPR++
			}
		} else if m.viewMode == ViewJira {
			if m.selectedTicket < len(m.jiraTickets)-1 {
				m.selectedTicket++
			}
		} else {
			if m.repository != nil && m.selectedCommit < len(m.repository.Graph.Commits)-1 {
				m.selectedCommit++
				// Load changed files for the newly selected commit
				commit := m.repository.Graph.Commits[m.selectedCommit]
				m.changedFilesCommitID = commit.ChangeID
				m.changedFiles = nil
				return m, m.loadChangedFiles(commit.ChangeID)
			}
		}
	case "k", "up":
		if m.viewMode == ViewPullRequests {
			if m.selectedPR > 0 {
				m.selectedPR--
			}
		} else if m.viewMode == ViewJira {
			if m.selectedTicket > 0 {
				m.selectedTicket--
			}
		} else {
			if m.selectedCommit > 0 && m.repository != nil {
				m.selectedCommit--
				// Load changed files for the newly selected commit
				commit := m.repository.Graph.Commits[m.selectedCommit]
				m.changedFilesCommitID = commit.ChangeID
				m.changedFiles = nil
				return m, m.loadChangedFiles(commit.ChangeID)
			}
		}
	case "o":
		// Open PR URL in browser (PR view only)
		if m.viewMode == ViewPullRequests && m.repository != nil && m.selectedPR >= 0 && m.selectedPR < len(m.repository.PRs) {
			pr := m.repository.PRs[m.selectedPR]
			if pr.URL != "" {
				m.statusMessage = fmt.Sprintf("Opening PR #%d...", pr.Number)
				return m, openURL(pr.URL)
			}
		}
		// Open Jira ticket URL in browser (Jira view only)
		if m.viewMode == ViewJira && m.jiraService != nil && m.selectedTicket >= 0 && m.selectedTicket < len(m.jiraTickets) {
			ticket := m.jiraTickets[m.selectedTicket]
			ticketURL := m.jiraService.GetTicketURL(ticket.Key)
			m.statusMessage = fmt.Sprintf("Opening %s...", ticket.Key)
			return m, openURL(ticketURL)
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
		// In Jira view, start bookmark creation from ticket
		if m.viewMode == ViewJira && m.selectedTicket >= 0 && m.selectedTicket < len(m.jiraTickets) && m.jjService != nil {
			ticket := m.jiraTickets[m.selectedTicket]
			m.startBookmarkFromJiraTicket(ticket)
			return m, nil
		}
		// In commit view, edit selected commit (jj edit)
		if m.viewMode == ViewCommitGraph && m.selectedCommit >= 0 && m.jjService != nil && m.repository != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if commit.Immutable {
				m.statusMessage = "Cannot edit: commit is immutable"
				return m, nil
			}
			return m, m.checkoutCommit()
		}
	case "s":
		// Squash selected commit
		if m.selectedCommit >= 0 && m.jjService != nil && m.repository != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if commit.Immutable {
				m.statusMessage = "Cannot squash: commit is immutable"
				return m, nil
			}
			return m, m.squashCommit()
		}
	case "a":
		// Abandon selected commit
		if m.viewMode == ViewCommitGraph && m.selectedCommit >= 0 && m.jjService != nil && m.repository != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if commit.Immutable {
				m.statusMessage = "Cannot abandon: commit is immutable"
				return m, nil
			}
			return m, m.abandonCommit()
		}
	case "r":
		// Start rebase mode
		if m.viewMode == ViewCommitGraph && m.selectedCommit >= 0 && m.jjService != nil && m.repository != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if commit.Immutable {
				m.statusMessage = "Cannot rebase: commit is immutable"
				return m, nil
			}
			m.startRebaseMode()
			return m, nil
		}
	case "c":
		// Create PR from selected commit
		if m.viewMode == ViewCommitGraph && m.selectedCommit >= 0 && m.jjService != nil && m.githubService != nil && m.repository != nil {
			m.startCreatePR()
			return m, nil
		} else if m.githubService == nil {
			m.statusMessage = "GitHub not connected. Configure in Settings (,)"
			return m, nil
		}
	case "b":
		// Create bookmark on selected commit
		if m.viewMode == ViewCommitGraph && m.selectedCommit >= 0 && m.jjService != nil && m.repository != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if commit.Immutable {
				m.statusMessage = "Cannot create bookmark: commit is immutable"
				return m, nil
			}
			m.startCreateBookmark()
			return m, nil
		}
	case "x":
		// Delete bookmark from selected commit
		if m.viewMode == ViewCommitGraph && m.selectedCommit >= 0 && m.jjService != nil && m.repository != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if len(commit.Branches) == 0 {
				m.statusMessage = "No bookmark on this commit to delete"
				return m, nil
			}
			return m, m.deleteBookmark()
		}
	case "u":
		// Push updates to PR (for commits with PR branches or their descendants)
		if m.viewMode == ViewCommitGraph && m.selectedCommit >= 0 && m.jjService != nil && m.repository != nil {
			// Find the PR branch for this commit (could be on this commit or an ancestor)
			prBranch := m.findPRBranchForCommit(m.selectedCommit)
			if prBranch == "" {
				m.statusMessage = "No open PR found for this commit or its ancestors"
				return m, nil
			}
			commit := m.repository.Graph.Commits[m.selectedCommit]
			// Check if we need to move the bookmark (commit doesn't have it directly)
			needsMoveBookmark := true
			for _, branch := range commit.Branches {
				if branch == prBranch {
					needsMoveBookmark = false
					break
				}
			}
			return m, m.pushToPR(prBranch, commit.ChangeID, needsMoveBookmark)
		}
	}
	return m, nil
}

// handleCreateBookmarkKeyMsg handles keyboard input in bookmark creation mode
func (m *Model) handleCreateBookmarkKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel bookmark creation
		m.viewMode = ViewCommitGraph
		m.statusMessage = "Bookmark creation cancelled"
		return m, nil
	case "enter", "ctrl+s":
		// Submit bookmark
		if m.jjService != nil {
			return m, m.submitBookmark()
		}
		return m, nil
	case "j", "down":
		// Navigate down in existing bookmarks list
		if len(m.existingBookmarks) > 0 {
			if m.selectedBookmarkIdx < len(m.existingBookmarks)-1 {
				m.selectedBookmarkIdx++
				m.bookmarkNameInput.Blur()
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

	// Only pass keys to input if we're in "new bookmark" mode
	if m.selectedBookmarkIdx == -1 {
		var cmd tea.Cmd
		m.bookmarkNameInput, cmd = m.bookmarkNameInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleCreatePRKeyMsg handles keyboard input in PR creation mode
func (m *Model) handleCreatePRKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel PR creation
		m.viewMode = ViewCommitGraph
		m.statusMessage = "PR creation cancelled"
		return m, nil
	case "ctrl+s":
		// Submit PR
		if m.githubService != nil && m.jjService != nil {
			return m, m.submitPR()
		}
		return m, nil
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
		// Cancel editing
		m.viewMode = ViewCommitGraph
		m.editingCommitID = ""
		m.statusMessage = "Description edit cancelled"
		return m, nil
	case "ctrl+s":
		// Save the description
		return m, m.saveDescription()
	}

	// Pass other keys to the textarea
	var cmd tea.Cmd
	m.descriptionInput, cmd = m.descriptionInput.Update(msg)
	return m, cmd
}

// handleSettingsKeyMsg handles keys while in settings view
func (m *Model) handleSettingsKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel and go back
		m.viewMode = ViewCommitGraph
		m.statusMessage = "Settings cancelled"
		return m, nil
	case "ctrl+s", "enter":
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
		// Save settings
		return m, m.saveSettings()
	case "tab", "down":
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

	// Pass other keys to the focused input
	var cmd tea.Cmd
	m.settingsInputs[m.settingsFocusedField], cmd = m.settingsInputs[m.settingsFocusedField].Update(msg)
	return m, cmd
}

