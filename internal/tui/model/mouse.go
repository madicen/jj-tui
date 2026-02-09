package model

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

// handleZoneClick handles clicks detected by bubblezone
func (m *Model) handleZoneClick(clickedZone *zone.ZoneInfo) (tea.Model, tea.Cmd) {
	if clickedZone == nil {
		return m, nil
	}

	userClicked := m.createIsZoneClickedFunc(clickedZone)

	// Check for JJ init button (shown when not in a jj repo)
	if userClicked(ZoneActionJJInit) && m.notJJRepo {
		m.statusMessage = "Initializing repository..."
		return m, m.runJJInit()
	}

	// Check tab zones
	if userClicked(ZoneTabGraph) {
		return m.handleNavigateToGraphTab()
	}
	if userClicked(ZoneTabPRs) {
		return m.handleNavigateToPRTab()
	}
	if userClicked(ZoneTabJira) {
		return m.handleNavigateToTicketsTab()
	}
	if userClicked(ZoneTabBranches) {
		return m.handleNavigateToBranchesTab()
	}
	if userClicked(ZoneTabSettings) {
		return m.handleNavigateToSettingsTab()
	}
	if userClicked(ZoneTabHelp) {
		return m.handleNavigateToHelpTab()
	}

	// Check status bar action zones
	if userClicked(ZoneActionQuit) {
		return m, tea.Quit
	}
	if userClicked(ZoneActionRefresh) {
		return m, m.refreshRepository()
	}
	if userClicked(ZoneActionNewCommit) {
		return m.handleNewCommit()
	}
	if userClicked(ZoneActionCopyError) {
		return m.handleCopyError()
	}
	if userClicked(ZoneActionDismissError) {
		return m.handleDismissError()
	}
	if userClicked(ZoneActionUndo) {
		return m.handleUndo()
	}
	if userClicked(ZoneActionRedo) {
		return m.handleRedo()
	}

	// Check graph view pane zones for click-to-focus
	if m.viewMode == ViewCommitGraph {
		if userClicked(ZoneGraphPane) {
			if !m.graphFocused {
				m.graphFocused = true
				m.statusMessage = m.handleGraphFoucsMessage()
			}
			return m, nil
		}
		if userClicked(ZoneFilesPane) {
			if m.graphFocused {
				m.graphFocused = false
				m.statusMessage = m.handleGraphFoucsMessage()
			}
			return m, nil
		}
	}

	// Check commit zones
	if m.repository != nil {
		for commitIndex := range m.repository.Graph.Commits {
			if userClicked(ZoneCommit(commitIndex)) {
				// If in rebase mode, clicking a commit selects it as destination
				if m.selectionMode == SelectionRebaseDestination {
					return m, m.performRebase(commitIndex)
				}
				// Normal selection
				return m.handleSelectCommit(commitIndex)
			}
		}
	}

	// Check commit action zones (only for mutable commits)
	if userClicked(ZoneActionCheckout) {
		return m.handleCheckoutCommit()
	}
	if userClicked(ZoneActionSquash) {
		return m.handleSquashCommit()
	}
	if userClicked(ZoneActionDescribe) {
		return m.handleDescribeCommit()
	}
	if userClicked(ZoneActionAbandon) {
		return m.handleAbandonCommit()
	}
	if userClicked(ZoneActionRebase) {
		return m.handleRebase()
	}
	if userClicked(ZoneActionBookmark) {
		return m.handleCreateBookmark()
	}
	if userClicked(ZoneActionMoveFileUp) {
		return m.handleMoveFileUp()
	}
	if userClicked(ZoneActionMoveFileDown) {
		return m.handleMoveFileDown()
	}
	if userClicked(ZoneActionCreatePR) {
		return m.handleCreatePR()
	}
	if userClicked(ZoneActionDelBookmark) {
		return m.handleDeleteBookmark()
	}
	if userClicked(ZoneActionUpdatePR) {
		return m.handleUpdatePR()
	}

	// Check changed file zones (for clicking to select a file)
	for i := range m.changedFiles {
		if userClicked(ZoneChangedFile(i)) {
			m.selectedFile = i
			// When clicking a file, switch focus to files pane
			m.graphFocused = false
			return m, nil
		}
	}

	// Check PR zones
	if m.repository != nil {
		for index := range m.repository.PRs {
			if userClicked(ZonePR(index)) {
				m.selectedPR = index
				return m, nil
			}
		}
	}

	// Check PR open in browser button
	if userClicked(ZonePROpenBrowser) {
		return m.handleOpenPRInBrowser()
	}

	// Check PR merge button
	if userClicked(ZonePRMerge) {
		return m.handleMergePR()
	}

	// Check PR close button
	if userClicked(ZonePRClose) {
		return m.handleClosePR()
	}

	// Description editor zones
	if userClicked(ZoneDescSave) {
		return m.handleDescriptionSave()
	}
	if userClicked(ZoneDescCancel) {
		return m.handleDescriptionCancel()
	}

	// Bookmark creation zones
	if m.viewMode == ViewCreateBookmark {
		// Check for clicks on existing bookmarks
		for i := range m.existingBookmarks {
			if userClicked(ZoneExistingBookmark(i)) {
				m.selectedBookmarkIdx = i
				m.bookmarkNameInput.Blur()
				return m, nil
			}
		}

		if userClicked(ZoneBookmarkName) {
			m.selectedBookmarkIdx = -1 // Switch to new bookmark mode
			m.bookmarkNameInput.Focus()
			return m, nil
		}
		if userClicked(ZoneBookmarkSubmit) {
			return m.handleBookmarkSubmit()
		}
		if userClicked(ZoneBookmarkCancel) {
			return m.handleBookmarkCancel()
		}
	}

	// PR creation zones
	if m.viewMode == ViewCreatePR {
		if userClicked(ZonePRTitle) {
			m.prFocusedField = 0
			m.prTitleInput.Focus()
			m.prBodyInput.Blur()
			return m, nil
		}
		if userClicked(ZonePRBody) {
			m.prFocusedField = 1
			m.prTitleInput.Blur()
			m.prBodyInput.Focus()
			return m, nil
		}
		if userClicked(ZonePRSubmit) {
			return m.handlePRSubmit()
		}
		if userClicked(ZonePRCancel) {
			return m.handlePRCancel()
		}
	}

	// Check ticket zones
	for i := range m.ticketList {
		if userClicked(ZoneJiraTicket(i)) {
			m.selectedTicket = i
			return m, nil
		}
	}

	// Check ticket create branch button
	if userClicked(ZoneJiraCreateBranch) {
		return m.handleStartBookmarkFromTicket()
	}

	// Check "Change Status" button to toggle status change mode
	if userClicked(ZoneJiraChangeStatus) {
		return m.handleToggleStatusChangeMode()
	}

	// Check ticket transition buttons (only when status change mode is active)
	if m.viewMode == ViewTickets && m.ticketService != nil && !m.transitionInProgress && m.statusChangeMode {
		for i, t := range m.availableTransitions {
			zoneID := ZoneJiraTransition + fmt.Sprintf("%d", i)
			if userClicked(zoneID) {
				if m.selectedTicket >= 0 && m.selectedTicket < len(m.ticketList) {
					m.transitionInProgress = true
					ticket := m.ticketList[m.selectedTicket]
					m.statusMessage = fmt.Sprintf("Setting %s to %s...", ticket.DisplayKey, t.Name)
					return m, m.transitionTicket(t.ID)
				}
			}
		}
	}

	// Check ticket open in browser button
	if userClicked(ZoneTicketOpenBrowser) {
		if m.viewMode == ViewTickets {
			return m.handleOpenTicketInBrowser()
		}
	}

	// Check branch zones
	for i := range m.branchList {
		if userClicked(ZoneBranch(i)) {
			m.selectedBranch = i
			return m, nil
		}
	}

	// Check branch action buttons
	if m.viewMode == ViewBranches {
		if userClicked(ZoneBranchTrack) {
			return m.handleTrackBranch()
		}
		if userClicked(ZoneBranchUntrack) {
			return m.handleUntrackBranch()
		}
		if userClicked(ZoneBranchRestore) {
			return m.handleRestoreLocalBranch()
		}
		if userClicked(ZoneBranchDelete) {
			return m.handleDeleteBranchBookmark()
		}
		if userClicked(ZoneBranchPush) {
			return m.handlePushBranch()
		}
		if userClicked(ZoneBranchFetch) {
			return m.handleFetchAll()
		}
	}

	// Settings input field clicks
	if m.viewMode == ViewSettings {
		// Settings sub-tabs
		if userClicked(ZoneSettingsTabGitHub) {
			m.settingsTab = 0
			return m, nil
		}
		if userClicked(ZoneSettingsTabJira) {
			m.settingsTab = 1
			return m, nil
		}
		if userClicked(ZoneSettingsTabCodecks) {
			m.settingsTab = 2
			return m, nil
		}
		if userClicked(ZoneSettingsTabAdvanced) {
			m.settingsTab = 3
			return m, nil
		}

		// Handle Advanced tab operations
		if m.settingsTab == 3 {
			// Cleanup confirmation buttons
			if m.confirmingCleanup != "" {
				if userClicked(ZoneSettingsAdvancedConfirmYes) {
					return m, m.confirmCleanup()
				}
				if userClicked(ZoneSettingsAdvancedConfirmNo) {
					m.cancelCleanup()
					return m, nil
				}
				return m, nil
			}

			// Advanced tab action buttons
			if userClicked(ZoneSettingsAdvancedDeleteBookmarks) {
				m.startDeleteBookmarks()
				return m, nil
			}
			if userClicked(ZoneSettingsAdvancedAbandonOldCommits) {
				m.startAbandonOldCommits()
				return m, nil
			}
			if userClicked(ZoneSettingsAdvancedTrackOriginMain) {
				m.startTrackOriginMain()
				return m, m.trackOriginMain()
			}
			// Auto-status toggle
			if userClicked(ZoneSettingsAutoInProgress) {
				m.settingsAutoInProgress = !m.settingsAutoInProgress
				return m, nil
			}
			return m, nil
		}

		// GitHub login button
		if userClicked(ZoneSettingsGitHubLogin) {
			m.statusMessage = "Starting GitHub login..."
			return m, m.startGitHubLogin()
		}

		// GitHub filter toggles
		if userClicked(ZoneSettingsGitHubOnlyMine) {
			m.settingsOnlyMine = !m.settingsOnlyMine
			return m, nil
		}
		if userClicked(ZoneSettingsGitHubShowMerged) {
			m.settingsShowMerged = !m.settingsShowMerged
			return m, nil
		}
		if userClicked(ZoneSettingsGitHubShowClosed) {
			m.settingsShowClosed = !m.settingsShowClosed
			return m, nil
		}
		if userClicked(ZoneSettingsGitHubPRLimitDecrease) {
			if m.settingsPRLimit > 25 {
				m.settingsPRLimit -= 25
			}
			return m, nil
		}
		if userClicked(ZoneSettingsGitHubPRLimitIncrease) {
			if m.settingsPRLimit < 500 {
				m.settingsPRLimit += 25
			}
			return m, nil
		}

		// PR Refresh Interval controls
		if userClicked(ZoneSettingsGitHubRefreshDecrease) {
			if m.settingsPRRefreshInterval > 30 {
				m.settingsPRRefreshInterval -= 30 // Decrease by 30 seconds
			} else if m.settingsPRRefreshInterval > 0 {
				m.settingsPRRefreshInterval = 0 // Go to disabled
			}
			return m, nil
		}
		if userClicked(ZoneSettingsGitHubRefreshIncrease) {
			if m.settingsPRRefreshInterval == 0 {
				m.settingsPRRefreshInterval = 30 // Enable at 30 seconds
			} else if m.settingsPRRefreshInterval < 600 {
				m.settingsPRRefreshInterval += 30 // Increase by 30 seconds (max 10 min)
			}
			return m, nil
		}
		if userClicked(ZoneSettingsGitHubRefreshToggle) {
			if m.settingsPRRefreshInterval == 0 {
				m.settingsPRRefreshInterval = 120 // Enable at 2 minutes (default)
			} else {
				m.settingsPRRefreshInterval = 0 // Disable
			}
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
			if userClicked(zoneID) {
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
			if userClicked(zoneID) {
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
		if userClicked(ZoneSettingsSave) {
			return m, m.saveSettings()
		}

		// Save Local button
		if userClicked(ZoneSettingsSaveLocal) {
			return m, m.saveSettingsLocal()
		}

		// Cancel button
		if userClicked(ZoneSettingsCancel) {
			return m.handleSettingsCancel()
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
