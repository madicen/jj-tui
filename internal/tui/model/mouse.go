package model

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/tui/actions"
	"github.com/madicen/jj-tui/internal/tui/mouse"
)

// handleZoneClick handles clicks detected by bubblezone
func (m *Model) handleZoneClick(clickedZone *zone.ZoneInfo) (tea.Model, tea.Cmd) {
	if clickedZone == nil {
		return m, nil
	}

	userClicked := m.createIsZoneClickedFunc(clickedZone)

	// Handle error state first - error modal blocks most mouse interactions
	if m.err != nil {
		// For "not a jj repo", allow the init button
		if m.notJJRepo && userClicked(mouse.ZoneActionJJInit) {
			m.statusMessage = "Initializing repository..."
			return m, m.runJJInit()
		}
		// For regular errors, only allow copy/dismiss/retry/quit
		if userClicked(mouse.ZoneActionCopyError) {
			return m.handleCopyError()
		}
		if userClicked(mouse.ZoneActionDismissError) {
			return m.handleDismissError()
		}
		if userClicked(mouse.ZoneActionRetry) {
			m.err = nil
			m.errorCopied = false
			m.viewMode = ViewCommitGraph
			return m, m.refreshRepository()
		}
		if userClicked(mouse.ZoneActionQuit) {
			return m, tea.Quit
		}
		// Block all other mouse clicks during error state
		return m, nil
	}

	// Handle warning modal clicks
	if m.showWarningModal {
		if userClicked(mouse.ZoneWarningGoToCommit) {
			// Go to the selected commit and start editing its description
			if len(m.warningCommits) > 0 && m.warningSelectedIdx < len(m.warningCommits) {
				selectedCommit := m.warningCommits[m.warningSelectedIdx]
				// Find this commit in the graph and select it
				for i, c := range m.repository.Graph.Commits {
					if c.ChangeID == selectedCommit.ChangeID {
						m.selectedCommit = i
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
		}
		if userClicked(mouse.ZoneWarningDismiss) {
			m.showWarningModal = false
			m.warningCommits = nil
			m.statusMessage = "Cancelled"
			return m, nil
		}
		// Block all other mouse clicks during warning modal
		return m, nil
	}

	// GitHub login copy code button
	if userClicked(mouse.ZoneGitHubLoginCopyCode) && m.githubUserCode != "" {
		m.statusMessage = "Copying code to clipboard..."
		return m, actions.CopyToClipboard(m.githubUserCode)
	}

	// Check tab zones
	if userClicked(mouse.ZoneTabGraph) {
		return m.handleNavigateToGraphTab()
	}
	if userClicked(mouse.ZoneTabPRs) {
		return m.handleNavigateToPRTab()
	}
	if userClicked(mouse.ZoneTabJira) {
		return m.handleNavigateToTicketsTab()
	}
	if userClicked(mouse.ZoneTabBranches) {
		return m.handleNavigateToBranchesTab()
	}
	if userClicked(mouse.ZoneTabSettings) {
		return m.handleNavigateToSettingsTab()
	}
	if userClicked(mouse.ZoneTabHelp) {
		return m.handleNavigateToHelpTab()
	}

	// Check status bar action zones
	if userClicked(mouse.ZoneActionQuit) {
		return m, tea.Quit
	}
	if userClicked(mouse.ZoneActionRefresh) {
		return m, m.refreshRepository()
	}
	if userClicked(mouse.ZoneActionNewCommit) {
		return m.handleNewCommit()
	}
	if userClicked(mouse.ZoneActionUndo) {
		return m.handleUndo()
	}
	if userClicked(mouse.ZoneActionRedo) {
		return m.handleRedo()
	}

	// Check graph view pane zones for click-to-focus
	if m.viewMode == ViewCommitGraph {
		if userClicked(mouse.ZoneGraphPane) {
			if !m.graphFocused {
				m.graphFocused = true
				m.statusMessage = m.handleGraphFoucsMessage()
			}
			return m, nil
		}
		if userClicked(mouse.ZoneFilesPane) {
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
			if userClicked(mouse.ZoneCommit(commitIndex)) {
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
	if userClicked(mouse.ZoneActionCheckout) {
		return m.handleCheckoutCommit()
	}
	if userClicked(mouse.ZoneActionSquash) {
		return m.handleSquashCommit()
	}
	if userClicked(mouse.ZoneActionDescribe) {
		return m.handleDescribeCommit()
	}
	if userClicked(mouse.ZoneActionAbandon) {
		return m.handleAbandonCommit()
	}
	if userClicked(mouse.ZoneActionResolveDivergent) {
		return m.handleResolveDivergentCommit()
	}
	if userClicked(mouse.ZoneActionRebase) {
		return m.handleRebase()
	}
	if userClicked(mouse.ZoneActionBookmark) {
		return m.handleCreateBookmark()
	}
	if userClicked(mouse.ZoneActionMoveFileUp) {
		return m.handleMoveFileUp()
	}
	if userClicked(mouse.ZoneActionMoveFileDown) {
		return m.handleMoveFileDown()
	}
	if userClicked(mouse.ZoneActionRevertFile) {
		return m.handleRevertFile()
	}
	if userClicked(mouse.ZoneActionCreatePR) {
		return m.handleCreatePR()
	}
	if userClicked(mouse.ZoneActionDelBookmark) {
		return m.handleDeleteBookmark()
	}
	if userClicked(mouse.ZoneActionUpdatePR) {
		return m.handleUpdatePR()
	}

	// Check changed file zones (for clicking to select a file)
	for i := range m.changedFiles {
		if userClicked(mouse.ZoneChangedFile(i)) {
			m.selectedFile = i
			// When clicking a file, switch focus to files pane
			m.graphFocused = false
			return m, nil
		}
	}

	// Check PR zones
	if m.repository != nil {
		for index := range m.repository.PRs {
			if userClicked(mouse.ZonePR(index)) {
				m.selectedPR = index
				return m, nil
			}
		}
	}

	// Check PR open in browser button
	if userClicked(mouse.ZonePROpenBrowser) {
		return m.handleOpenPRInBrowser()
	}

	// Check PR merge button
	if userClicked(mouse.ZonePRMerge) {
		return m.handleMergePR()
	}

	// Check PR close button
	if userClicked(mouse.ZonePRClose) {
		return m.handleClosePR()
	}

	// Description editor zones
	if userClicked(mouse.ZoneDescSave) {
		return m.handleDescriptionSave()
	}
	if userClicked(mouse.ZoneDescCancel) {
		return m.handleDescriptionCancel()
	}

	// Bookmark creation zones
	if m.viewMode == ViewCreateBookmark {
		// Check for clicks on existing bookmarks
		for i := range m.existingBookmarks {
			if userClicked(mouse.ZoneExistingBookmark(i)) {
				m.selectedBookmarkIdx = i
				m.bookmarkNameInput.Blur()
				return m, nil
			}
		}

		if userClicked(mouse.ZoneBookmarkName) {
			m.selectedBookmarkIdx = -1 // Switch to new bookmark mode
			m.bookmarkNameInput.Focus()
			return m, nil
		}
		if userClicked(mouse.ZoneBookmarkSubmit) {
			return m.handleBookmarkSubmit()
		}
		if userClicked(mouse.ZoneBookmarkCancel) {
			return m.handleBookmarkCancel()
		}
	}

	// Bookmark conflict resolution zones
	if m.viewMode == ViewBookmarkConflict {
		if userClicked(mouse.ZoneConflictKeepLocal) {
			m.conflictSelectedOption = 0 // mouse.ZoneConflictKeepLocal
			return m, nil
		}
		if userClicked(mouse.ZoneConflictResetRemote) {
			m.conflictSelectedOption = 1
			return m, nil
		}
		if userClicked(mouse.ZoneConflictConfirm) {
			resolution := "keep_local"
			if m.conflictSelectedOption == 1 {
				resolution = "reset_remote"
			}
			m.statusMessage = "Resolving bookmark conflict..."
			return m, m.resolveBookmarkConflict(m.conflictBookmarkName, resolution)
		}
		if userClicked(mouse.ZoneConflictCancel) {
			m.viewMode = ViewBranches
			m.statusMessage = "Conflict resolution cancelled"
			return m, nil
		}
	}

	// Divergent commit resolution zones
	if m.viewMode == ViewDivergentCommit {
		// Check for clicks on divergent commit options
		for i := range m.divergentCommitIDs {
			if userClicked(mouse.ZoneDivergentCommit(i)) {
				m.divergentSelectedIdx = i
				return m, nil
			}
		}
		if userClicked(mouse.ZoneDivergentConfirm) {
			if len(m.divergentCommitIDs) > 0 && m.divergentSelectedIdx < len(m.divergentCommitIDs) {
				keepCommitID := m.divergentCommitIDs[m.divergentSelectedIdx]
				m.statusMessage = "Resolving divergent commit..."
				return m, m.resolveDivergentCommit(m.divergentChangeID, keepCommitID)
			}
		}
		if userClicked(mouse.ZoneDivergentCancel) {
			m.viewMode = ViewCommitGraph
			m.statusMessage = "Divergent commit resolution cancelled"
			return m, nil
		}
	}

	// PR creation zones
	if m.viewMode == ViewCreatePR {
		if userClicked(mouse.ZonePRTitle) {
			m.prFocusedField = 0
			m.prTitleInput.Focus()
			m.prBodyInput.Blur()
			return m, nil
		}
		if userClicked(mouse.ZonePRBody) {
			m.prFocusedField = 1
			m.prTitleInput.Blur()
			m.prBodyInput.Focus()
			return m, nil
		}
		if userClicked(mouse.ZonePRSubmit) {
			return m.handlePRSubmit()
		}
		if userClicked(mouse.ZonePRCancel) {
			return m.handlePRCancel()
		}
	}

	// Check ticket zones
	for i := range m.ticketList {
		if userClicked(mouse.ZoneJiraTicket(i)) {
			m.selectedTicket = i
			return m, nil
		}
	}

	// Check ticket create branch button
	if userClicked(mouse.ZoneJiraCreateBranch) {
		return m.handleStartBookmarkFromTicket()
	}

	// Check "Change Status" button to toggle status change mode
	if userClicked(mouse.ZoneJiraChangeStatus) {
		return m.handleToggleStatusChangeMode()
	}

	// Check ticket transition buttons (only when status change mode is active)
	if m.viewMode == ViewTickets && m.ticketService != nil && !m.transitionInProgress && m.statusChangeMode {
		for i, t := range m.availableTransitions {
			zoneID := mouse.ZoneJiraTransition + fmt.Sprintf("%d", i)
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
	if userClicked(mouse.ZoneTicketOpenBrowser) {
		if m.viewMode == ViewTickets {
			return m.handleOpenTicketInBrowser()
		}
	}

	// Check branch zones
	for i := range m.branchList {
		if userClicked(mouse.ZoneBranch(i)) {
			m.selectedBranch = i
			return m, nil
		}
	}

	// Check branch action buttons
	if m.viewMode == ViewBranches {
		if userClicked(mouse.ZoneBranchTrack) {
			return m.handleTrackBranch()
		}
		if userClicked(mouse.ZoneBranchUntrack) {
			return m.handleUntrackBranch()
		}
		if userClicked(mouse.ZoneBranchRestore) {
			return m.handleRestoreLocalBranch()
		}
		if userClicked(mouse.ZoneBranchDelete) {
			return m.handleDeleteBranchBookmark()
		}
		if userClicked(mouse.ZoneBranchPush) {
			return m.handlePushBranch()
		}
		if userClicked(mouse.ZoneBranchFetch) {
			return m.handleFetchAll()
		}
		if userClicked(mouse.ZoneBranchResolveConflict) {
			return m.handleResolveBookmarkConflict()
		}
	}

	// Settings input field clicks
	if m.viewMode == ViewSettings {
		// Settings sub-tabs
		if userClicked(mouse.ZoneSettingsTabGitHub) {
			m.settingsTab = 0
			return m, nil
		}
		if userClicked(mouse.ZoneSettingsTabJira) {
			m.settingsTab = 1
			return m, nil
		}
		if userClicked(mouse.ZoneSettingsTabCodecks) {
			m.settingsTab = 2
			return m, nil
		}
		if userClicked(mouse.ZoneSettingsTabTickets) {
			m.settingsTab = 3
			return m, nil
		}
		if userClicked(mouse.ZoneSettingsTabBranches) {
			m.settingsTab = 4
			return m, nil
		}
		if userClicked(mouse.ZoneSettingsTabAdvanced) {
			m.settingsTab = 5
			return m, nil
		}

		// Handle Tickets tab operations
		if m.settingsTab == 3 {
			// Ticket provider selection
			if userClicked(mouse.ZoneSettingsTicketProviderNone) {
				m.settingsTicketProvider = ""
				return m, nil
			}
			if userClicked(mouse.ZoneSettingsTicketProviderJira) {
				m.settingsTicketProvider = "jira"
				return m, nil
			}
			if userClicked(mouse.ZoneSettingsTicketProviderCodecks) {
				m.settingsTicketProvider = "codecks"
				return m, nil
			}
			if userClicked(mouse.ZoneSettingsTicketProviderGitHubIssues) {
				m.settingsTicketProvider = "github_issues"
				return m, nil
			}
			// Auto-status toggle (now in Tickets tab)
			if userClicked(mouse.ZoneSettingsAutoInProgress) {
				m.settingsAutoInProgress = !m.settingsAutoInProgress
				return m, nil
			}
		}

		// Handle Advanced tab operations
		if m.settingsTab == 5 {
			// Cleanup confirmation buttons
			if m.confirmingCleanup != "" {
				if userClicked(mouse.ZoneSettingsAdvancedConfirmYes) {
					return m, m.confirmCleanup()
				}
				if userClicked(mouse.ZoneSettingsAdvancedConfirmNo) {
					m.cancelCleanup()
					return m, nil
				}
				return m, nil
			}

			// Advanced tab action buttons
			if userClicked(mouse.ZoneSettingsAdvancedDeleteBookmarks) {
				m.startDeleteBookmarks()
				return m, nil
			}
			if userClicked(mouse.ZoneSettingsAdvancedAbandonOldCommits) {
				m.startAbandonOldCommits()
				return m, nil
			}
			// Sanitize bookmarks toggle
			if userClicked(mouse.ZoneSettingsSanitizeBookmarks) {
				m.settingsSanitizeBookmarks = !m.settingsSanitizeBookmarks
				return m, nil
			}
			return m, nil
		}

		// GitHub login button
		if userClicked(mouse.ZoneSettingsGitHubLogin) {
			m.statusMessage = "Starting GitHub login..."
			return m, m.startGitHubLogin()
		}

		// GitHub filter toggles
		if userClicked(mouse.ZoneSettingsGitHubOnlyMine) {
			m.settingsOnlyMine = !m.settingsOnlyMine
			return m, nil
		}
		if userClicked(mouse.ZoneSettingsGitHubShowMerged) {
			m.settingsShowMerged = !m.settingsShowMerged
			return m, nil
		}
		if userClicked(mouse.ZoneSettingsGitHubShowClosed) {
			m.settingsShowClosed = !m.settingsShowClosed
			return m, nil
		}
		if userClicked(mouse.ZoneSettingsGitHubPRLimitDecrease) {
			if m.settingsPRLimit > 25 {
				m.settingsPRLimit -= 25
			}
			return m, nil
		}
		if userClicked(mouse.ZoneSettingsGitHubPRLimitIncrease) {
			if m.settingsPRLimit < 500 {
				m.settingsPRLimit += 25
			}
			return m, nil
		}

		// PR Refresh Interval controls
		if userClicked(mouse.ZoneSettingsGitHubRefreshDecrease) {
			if m.settingsPRRefreshInterval > 30 {
				m.settingsPRRefreshInterval -= 30 // Decrease by 30 seconds
			} else if m.settingsPRRefreshInterval > 0 {
				m.settingsPRRefreshInterval = 0 // Go to disabled
			}
			return m, nil
		}
		if userClicked(mouse.ZoneSettingsGitHubRefreshIncrease) {
			if m.settingsPRRefreshInterval == 0 {
				m.settingsPRRefreshInterval = 30 // Enable at 30 seconds
			} else if m.settingsPRRefreshInterval < 600 {
				m.settingsPRRefreshInterval += 30 // Increase by 30 seconds (max 10 min)
			}
			return m, nil
		}
		if userClicked(mouse.ZoneSettingsGitHubRefreshToggle) {
			if m.settingsPRRefreshInterval == 0 {
				m.settingsPRRefreshInterval = 120 // Enable at 2 minutes (default)
			} else {
				m.settingsPRRefreshInterval = 0 // Disable
			}
			return m, nil
		}

		// Branch Limit controls
		if userClicked(mouse.ZoneSettingsBranchLimitDecrease) {
			if m.settingsBranchLimit > 10 {
				m.settingsBranchLimit -= 10
			} else if m.settingsBranchLimit > 0 {
				m.settingsBranchLimit = 0 // 0 means unlimited
			}
			return m, nil
		}
		if userClicked(mouse.ZoneSettingsBranchLimitIncrease) {
			if m.settingsBranchLimit == 0 {
				m.settingsBranchLimit = 10 // Start at 10
			} else if m.settingsBranchLimit < 200 {
				m.settingsBranchLimit += 10 // Increase by 10 (max 200)
			}
			return m, nil
		}

		// Clear buttons for each field (in order of input indices)
		// 0=GitHub Token, 1=Jira URL, 2=Jira User, 3=Jira Token, 4=Jira Project, 5=Jira JQL, 6=Jira Excluded,
		// 7=Codecks Subdomain, 8=Codecks Token, 9=Codecks Project, 10=Codecks Excluded, 11=GitHub Issues Excluded
		clearZones := []string{
			mouse.ZoneSettingsGitHubTokenClear,          // 0
			mouse.ZoneSettingsJiraURLClear,              // 1
			mouse.ZoneSettingsJiraUserClear,             // 2
			mouse.ZoneSettingsJiraTokenClear,            // 3
			mouse.ZoneSettingsJiraProjectClear,          // 4
			mouse.ZoneSettingsJiraJQLClear,              // 5
			mouse.ZoneSettingsJiraExcludedClear,         // 6
			mouse.ZoneSettingsCodecksSubdomainClear,     // 7
			mouse.ZoneSettingsCodecksTokenClear,         // 8
			mouse.ZoneSettingsCodecksProjectClear,       // 9
			mouse.ZoneSettingsCodecksExcludedClear,      // 10
			mouse.ZoneSettingsGitHubIssuesExcludedClear, // 11
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
			mouse.ZoneSettingsGitHubToken,          // 0
			mouse.ZoneSettingsJiraURL,              // 1
			mouse.ZoneSettingsJiraUser,             // 2
			mouse.ZoneSettingsJiraToken,            // 3
			mouse.ZoneSettingsJiraProject,          // 4
			mouse.ZoneSettingsJiraJQL,              // 5
			mouse.ZoneSettingsJiraExcluded,         // 6
			mouse.ZoneSettingsCodecksSubdomain,     // 7
			mouse.ZoneSettingsCodecksToken,         // 8
			mouse.ZoneSettingsCodecksProject,       // 9
			mouse.ZoneSettingsCodecksExcluded,      // 10
			mouse.ZoneSettingsGitHubIssuesExcluded, // 11
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
		if userClicked(mouse.ZoneSettingsSave) {
			return m, m.saveSettings()
		}

		// Save Local button
		if userClicked(mouse.ZoneSettingsSaveLocal) {
			return m, m.saveSettingsLocal()
		}

		// Cancel button
		if userClicked(mouse.ZoneSettingsCancel) {
			return m.handleSettingsCancel()
		}
	}

	// Help view tab clicks
	if m.viewMode == ViewHelp {
		if userClicked(mouse.ZoneHelpTabShortcuts) {
			m.helpTab = 0
			m.helpSelectedCommand = 0
			return m, nil
		}
		if userClicked(mouse.ZoneHelpTabCommands) {
			m.helpTab = 1
			m.helpSelectedCommand = 0
			return m, nil
		}

		// Command copy buttons (only on Commands tab)
		if m.helpTab == 1 {
			history := m.getFilteredCommandHistory()
			for i := 0; i < len(history) && i < 50; i++ {
				if userClicked(fmt.Sprintf("%s%d", mouse.ZoneHelpCommandCopy, i)) {
					cmd := history[i].Command
					m.statusMessage = "Copied: " + cmd
					return m, actions.CopyToClipboard(cmd)
				}
			}
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
