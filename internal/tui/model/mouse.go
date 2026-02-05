package model

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

// handleZoneClick handles clicks detected by bubblezone
func (m *Model) handleZoneClick(zoneInfo *zone.ZoneInfo) (tea.Model, tea.Cmd) {
	if zoneInfo == nil {
		return m, nil
	}

	// Check for JJ init button (shown when not in a jj repo)
	if m.zone.Get(ZoneActionJJInit) == zoneInfo && m.notJJRepo {
		m.statusMessage = "Initializing repository..."
		return m, m.runJJInit()
	}

	// Check tab zones
	if m.zone.Get(ZoneTabGraph) == zoneInfo {
		return m, func() tea.Msg { return TabSelectedMsg{Tab: ViewCommitGraph} }
	}
	if m.zone.Get(ZoneTabPRs) == zoneInfo {
		return m, func() tea.Msg { return TabSelectedMsg{Tab: ViewPullRequests} }
	}
	if m.zone.Get(ZoneTabJira) == zoneInfo {
		m.viewMode = ViewJira
		if m.ticketService != nil {
			m.statusMessage = "Loading tickets..."
			return m, m.loadTickets()
		}
		return m, nil
	}
	if m.zone.Get(ZoneTabSettings) == zoneInfo {
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
		return m, nil
	}
	if m.zone.Get(ZoneTabHelp) == zoneInfo {
		return m, func() tea.Msg { return TabSelectedMsg{Tab: ViewHelp} }
	}

	// Check status bar action zones
	if m.zone.Get(ZoneActionQuit) == zoneInfo {
		return m, tea.Batch(
			func() tea.Msg { return ActionMsg{Action: ActionQuit} },
			tea.Quit,
		)
	}
	if m.zone.Get(ZoneActionRefresh) == zoneInfo {
		return m, m.refreshRepository()
	}
	if m.zone.Get(ZoneActionNewCommit) == zoneInfo {
		// Create a new commit (same as pressing 'n')
		if m.jjService != nil {
			m.statusMessage = "Creating new commit..."
			return m, m.createNewCommit()
		}
	}
	if m.zone.Get(ZoneActionCopyError) == zoneInfo {
		// Copy error to clipboard (works with m.err or status message errors)
		m.statusMessage = "Copying error to clipboard..."
		return m, m.copyErrorToClipboard()
	}
	if m.zone.Get(ZoneActionDismissError) == zoneInfo {
		// Dismiss/clear the error and restart auto-refresh
		m.err = nil
		m.statusMessage = "Ready"
		return m, m.tickCmd()
	}

	// Check graph view pane zones for click-to-focus
	if m.viewMode == ViewCommitGraph {
		if m.zone.Get(ZoneGraphPane) == zoneInfo {
			if !m.graphFocused {
				m.graphFocused = true
				m.statusMessage = "Graph pane focused"
			}
			return m, nil
		}
		if m.zone.Get(ZoneFilesPane) == zoneInfo {
			if m.graphFocused {
				m.graphFocused = false
				m.statusMessage = "Files pane focused"
			}
			return m, nil
		}
	}

	// Check commit zones
	if m.repository != nil {
		for i := range m.repository.Graph.Commits {
			if m.zone.Get(ZoneCommit(i)) == zoneInfo {
				// If in rebase mode, clicking a commit selects it as destination
				if m.selectionMode == SelectionRebaseDestination {
					return m, m.performRebase(i)
				}
				// Normal selection
				return m, func() tea.Msg {
					return CommitSelectedMsg{
						Index:    i,
						CommitID: m.repository.Graph.Commits[i].ID,
					}
				}
			}
		}
	}

	// Check commit action zones (only for mutable commits)
	if m.zone.Get(ZoneActionCheckout) == zoneInfo {
		if m.isSelectedCommitValid() && m.jjService != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if commit.Immutable {
				m.statusMessage = "Cannot edit: commit is immutable"
				return m, nil
			}
			return m, m.checkoutCommit()
		}
	}
	if m.zone.Get(ZoneActionSquash) == zoneInfo {
		if m.isSelectedCommitValid() && m.jjService != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if commit.Immutable {
				m.statusMessage = "Cannot squash: commit is immutable"
				return m, nil
			}
			return m, m.squashCommit()
		}
	}
	if m.zone.Get(ZoneActionDescribe) == zoneInfo {
		if m.isSelectedCommitValid() && m.jjService != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if commit.Immutable {
				m.statusMessage = "Cannot edit description: commit is immutable"
				return m, nil
			}
			return m.startEditingDescription(commit)
		}
	}
	if m.zone.Get(ZoneActionAbandon) == zoneInfo {
		if m.isSelectedCommitValid() && m.jjService != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if commit.Immutable {
				m.statusMessage = "Cannot abandon: commit is immutable"
				return m, nil
			}
			return m, m.abandonCommit()
		}
	}
	if m.zone.Get(ZoneActionRebase) == zoneInfo {
		if m.isSelectedCommitValid() && m.jjService != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if commit.Immutable {
				m.statusMessage = "Cannot rebase: commit is immutable"
				return m, nil
			}
			m.startRebaseMode()
			return m, nil
		}
	}
	if m.zone.Get(ZoneActionCreatePR) == zoneInfo {
		if m.isSelectedCommitValid() && m.jjService != nil && m.githubService != nil {
			m.startCreatePR()
			return m, nil
		} else if m.githubService == nil {
			m.statusMessage = "GitHub not connected. Configure in Settings (,)"
			return m, nil
		}
	}
	if m.zone.Get(ZoneActionBookmark) == zoneInfo {
		if m.isSelectedCommitValid() && m.jjService != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if commit.Immutable {
				m.statusMessage = "Cannot create bookmark: commit is immutable"
				return m, nil
			}
			m.startCreateBookmark()
			return m, nil
		}
	}
	if m.zone.Get(ZoneActionDelBookmark) == zoneInfo {
		if m.isSelectedCommitValid() && m.jjService != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if len(commit.Branches) == 0 {
				m.statusMessage = "No bookmark on this commit to delete"
				return m, nil
			}
			return m, m.deleteBookmark()
		}
	}
	if m.zone.Get(ZoneActionPush) == zoneInfo {
		if m.isSelectedCommitValid() && m.jjService != nil {
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

	// Check PR zones
	if m.repository != nil {
		for i := range m.repository.PRs {
			if m.zone.Get(ZonePR(i)) == zoneInfo {
				m.selectedPR = i
				return m, nil
			}
		}
	}

	// Description editor zones
	if m.zone.Get(ZoneDescSave) == zoneInfo {
		if m.viewMode == ViewEditDescription {
			return m, m.saveDescription()
		}
	}
	if m.zone.Get(ZoneDescCancel) == zoneInfo {
		if m.viewMode == ViewEditDescription {
			m.viewMode = ViewCommitGraph
			m.editingCommitID = ""
			m.statusMessage = "Description edit cancelled"
			return m, nil
		}
	}

	// Bookmark creation zones
	if m.viewMode == ViewCreateBookmark {
		// Check for clicks on existing bookmarks
		for i := range m.existingBookmarks {
			if m.zone.Get(ZoneExistingBookmark(i)) == zoneInfo {
				m.selectedBookmarkIdx = i
				m.bookmarkNameInput.Blur()
				return m, nil
			}
		}

		if m.zone.Get(ZoneBookmarkName) == zoneInfo {
			m.selectedBookmarkIdx = -1 // Switch to new bookmark mode
			m.bookmarkNameInput.Focus()
			return m, nil
		}
		if m.zone.Get(ZoneBookmarkSubmit) == zoneInfo {
			if m.jjService != nil {
				return m, m.submitBookmark()
			}
			return m, nil
		}
		if m.zone.Get(ZoneBookmarkCancel) == zoneInfo {
			m.viewMode = ViewCommitGraph
			m.statusMessage = "Bookmark creation cancelled"
			return m, nil
		}
	}

	// PR creation zones
	if m.viewMode == ViewCreatePR {
		if m.zone.Get(ZonePRTitle) == zoneInfo {
			m.prFocusedField = 0
			m.prTitleInput.Focus()
			m.prBodyInput.Blur()
			return m, nil
		}
		if m.zone.Get(ZonePRBody) == zoneInfo {
			m.prFocusedField = 1
			m.prTitleInput.Blur()
			m.prBodyInput.Focus()
			return m, nil
		}
		if m.zone.Get(ZonePRSubmit) == zoneInfo {
			if m.githubService != nil && m.jjService != nil {
				return m, m.submitPR()
			}
			return m, nil
		}
		if m.zone.Get(ZonePRCancel) == zoneInfo {
			m.viewMode = ViewCommitGraph
			m.statusMessage = "PR creation cancelled"
			return m, nil
		}
	}

	// Check ticket zones
	for i := range m.ticketList {
		if m.zone.Get(ZoneJiraTicket(i)) == zoneInfo {
			m.selectedTicket = i
			return m, nil
		}
	}

	// Check ticket create branch button
	if m.zone.Get(ZoneJiraCreateBranch) == zoneInfo {
		if m.viewMode == ViewJira && m.selectedTicket >= 0 && m.selectedTicket < len(m.ticketList) && m.jjService != nil {
			ticket := m.ticketList[m.selectedTicket]
			m.startBookmarkFromTicket(ticket)
			return m, nil
		}
	}

	// Check ticket open in browser button
	if m.zone.Get(ZoneJiraOpenBrowser) == zoneInfo {
		if m.viewMode == ViewJira && m.ticketService != nil && m.selectedTicket >= 0 && m.selectedTicket < len(m.ticketList) {
			ticket := m.ticketList[m.selectedTicket]
			ticketURL := m.ticketService.GetTicketURL(ticket)
			m.statusMessage = fmt.Sprintf("Opening %s...", ticket.Key)
			return m, openURL(ticketURL)
		}
	}

	// Settings input field clicks
	if m.viewMode == ViewSettings {
		// Settings sub-tabs
		if m.zone.Get(ZoneSettingsTabGitHub) == zoneInfo {
			m.settingsTab = 0
			return m, nil
		}
		if m.zone.Get(ZoneSettingsTabJira) == zoneInfo {
			m.settingsTab = 1
			return m, nil
		}
		if m.zone.Get(ZoneSettingsTabCodecks) == zoneInfo {
			m.settingsTab = 2
			return m, nil
		}

		// GitHub login button
		if m.zone.Get(ZoneSettingsGitHubLogin) == zoneInfo {
			m.statusMessage = "Starting GitHub login..."
			return m, m.startGitHubLogin()
		}

		// GitHub filter toggles
		if m.zone.Get(ZoneSettingsGitHubShowMerged) == zoneInfo {
			m.settingsShowMerged = !m.settingsShowMerged
			return m, nil
		}
		if m.zone.Get(ZoneSettingsGitHubShowClosed) == zoneInfo {
			m.settingsShowClosed = !m.settingsShowClosed
			return m, nil
		}

		// Clear buttons for each field (in order of input indices)
		// 0=GitHub Token, 1=Jira URL, 2=Jira User, 3=Jira Token, 4=Jira Excluded,
		// 5=Codecks Subdomain, 6=Codecks Token, 7=Codecks Project, 8=Codecks Excluded
		clearZones := []string{
			ZoneSettingsGitHubTokenClear,      // 0
			ZoneSettingsJiraURLClear,          // 1
			ZoneSettingsJiraUserClear,         // 2
			ZoneSettingsJiraTokenClear,        // 3
			ZoneSettingsJiraExcludedClear,     // 4
			ZoneSettingsCodecksSubdomainClear, // 5
			ZoneSettingsCodecksTokenClear,     // 6
			ZoneSettingsCodecksProjectClear,   // 7
			ZoneSettingsCodecksExcludedClear,  // 8
		}
		for i, zoneID := range clearZones {
			if m.zone.Get(zoneID) == zoneInfo {
				if i < len(m.settingsInputs) {
					m.settingsInputs[i].SetValue("")
					m.settingsInputs[i].Focus()
					m.settingsFocusedField = i
					// Blur other inputs
					for j := range m.settingsInputs {
						if j != i {
							m.settingsInputs[j].Blur()
						}
					}
				}
				return m, nil
			}
		}

		// Input field zones (in order of input indices)
		settingsZones := []string{
			ZoneSettingsGitHubToken,      // 0
			ZoneSettingsJiraURL,          // 1
			ZoneSettingsJiraUser,         // 2
			ZoneSettingsJiraToken,        // 3
			ZoneSettingsJiraExcluded,     // 4
			ZoneSettingsCodecksSubdomain, // 5
			ZoneSettingsCodecksToken,     // 6
			ZoneSettingsCodecksProject,   // 7
			ZoneSettingsCodecksExcluded,  // 8
		}
		for i, zoneID := range settingsZones {
			if m.zone.Get(zoneID) == zoneInfo {
				m.settingsFocusedField = i
				for j := range m.settingsInputs {
					if j == i {
						m.settingsInputs[j].Focus()
					} else {
						m.settingsInputs[j].Blur()
					}
				}
				return m, nil
			}
		}

		// Save button (global)
		if m.zone.Get(ZoneSettingsSave) == zoneInfo {
			return m, m.saveSettings()
		}

		// Save Local button
		if m.zone.Get(ZoneSettingsSaveLocal) == zoneInfo {
			return m, m.saveSettingsLocal()
		}

		// Cancel button
		if m.zone.Get(ZoneSettingsCancel) == zoneInfo {
			m.viewMode = ViewCommitGraph
			m.statusMessage = "Settings cancelled"
			return m, nil
		}
	}

	return m, nil
}

// handleAction handles action messages
func (m *Model) handleAction(action ActionType) (tea.Model, tea.Cmd) {
	switch action {
	case ActionQuit:
		return m, tea.Quit
	case ActionRefresh:
		return m, m.refreshRepository()
	case ActionNewPR:
		m.viewMode = ViewCreatePR
	case ActionCheckout, ActionEdit:
		if m.isSelectedCommitValid() && m.jjService != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if commit.Immutable {
				m.statusMessage = "Cannot edit: commit is immutable"
				return m, nil
			}
			return m, m.checkoutCommit()
		}
	case ActionSquash:
		if m.isSelectedCommitValid() && m.jjService != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if commit.Immutable {
				m.statusMessage = "Cannot squash: commit is immutable"
				return m, nil
			}
			return m, m.squashCommit()
		}
	case ActionHelp:
		m.viewMode = ViewHelp
	}
	return m, nil
}

