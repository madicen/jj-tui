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

	if userClicked(mouse.ZoneGitHubLoginCopyCode) && m.settingsTabModel.GetGitHubUserCode() != "" {
		m.statusMessage = "Copying code to clipboard..."
		return m, actions.CopyToClipboard(m.settingsTabModel.GetGitHubUserCode())
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
				m.graphTabModel.SetGraphFocused(true)
				m.statusMessage = m.handleGraphFoucsMessage()
			}
			return m, nil
		}
		if userClicked(mouse.ZoneFilesPane) {
			if m.graphFocused {
				m.graphFocused = false
				m.graphTabModel.SetGraphFocused(false)
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
	changedFiles := m.graphTabModel.GetChangedFiles()
	for i := range changedFiles {
		if userClicked(mouse.ZoneChangedFile(i)) {
			m.graphTabModel.SetSelectedFile(i)
			m.graphTabModel.SetGraphFocused(false)
			m.graphFocused = false
			return m, nil
		}
	}

	// PR zones (list, open, merge, close) delegated to PRs tab when ViewPullRequests

	// Description editor zones
	if userClicked(mouse.ZoneDescSave) {
		return m.handleDescriptionSave()
	}
	if userClicked(mouse.ZoneDescCancel) {
		return m.handleDescriptionCancel()
	}

	// Bookmark creation zones
	if m.viewMode == ViewCreateBookmark {
		existing := m.bookmarkModal.GetExistingBookmarks()
		for i := range existing {
			if userClicked(mouse.ZoneExistingBookmark(i)) {
				m.bookmarkModal.SetSelectedBookmarkIdx(i)
				m.bookmarkModal.GetNameInput().Blur()
				return m, nil
			}
		}

		if userClicked(mouse.ZoneBookmarkName) {
			m.bookmarkModal.SetSelectedBookmarkIdx(-1)
			m.bookmarkModal.GetNameInput().Focus()
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
			m.conflictModal.SetSelectedOption(0)
			return m, nil
		}
		if userClicked(mouse.ZoneConflictResetRemote) {
			m.conflictModal.SetSelectedOption(1)
			return m, nil
		}
		if userClicked(mouse.ZoneConflictConfirm) {
			m.statusMessage = "Resolving bookmark conflict..."
			return m, m.resolveBookmarkConflict(m.conflictModal.GetBookmarkName(), m.conflictModal.GetSelectedOption())
		}
		if userClicked(mouse.ZoneConflictCancel) {
			m.viewMode = ViewBranches
			m.statusMessage = "Conflict resolution cancelled"
			return m, nil
		}
	}

	// Divergent commit resolution zones
	if m.viewMode == ViewDivergentCommit {
		n := m.divergentModal.GetCommitCount()
		for i := 0; i < n; i++ {
			if userClicked(mouse.ZoneDivergentCommit(i)) {
				m.divergentModal.SetSelectedIdx(i)
				return m, nil
			}
		}
		if userClicked(mouse.ZoneDivergentConfirm) {
			keepCommitID := m.divergentModal.GetSelectedCommitID()
			if keepCommitID != "" {
				m.statusMessage = "Resolving divergent commit..."
				return m, m.resolveDivergentCommit(m.divergentModal.GetChangeID(), keepCommitID)
			}
		}
		if userClicked(mouse.ZoneDivergentCancel) {
			m.viewMode = ViewCommitGraph
			m.statusMessage = "Divergent commit resolution cancelled"
			return m, nil
		}
	}

	if m.viewMode == ViewCreatePR {
		if userClicked(mouse.ZonePRTitle) {
			m.prFormModal.SetFocusedField(0)
			return m, nil
		}
		if userClicked(mouse.ZonePRBody) {
			m.prFormModal.SetFocusedField(1)
			return m, nil
		}
		if userClicked(mouse.ZonePRSubmit) {
			return m.handlePRSubmit()
		}
		if userClicked(mouse.ZonePRCancel) {
			return m.handlePRCancel()
		}
	}

	// Ticket zones (list, create branch, change status, transitions, open browser) delegated to Tickets tab when ViewTickets

	// Branch zones (list + action buttons) delegated to Branches tab when ViewBranches

	// Settings input field clicks
	if m.viewMode == ViewSettings {
		inputs := m.settingsTabModel.GetSettingsInputs()
		// Settings sub-tabs
		if userClicked(mouse.ZoneSettingsTabGitHub) {
			m.settingsTabModel.SetSettingsTab(0)
			return m, nil
		}
		if userClicked(mouse.ZoneSettingsTabJira) {
			m.settingsTabModel.SetSettingsTab(1)
			return m, nil
		}
		if userClicked(mouse.ZoneSettingsTabCodecks) {
			m.settingsTabModel.SetSettingsTab(2)
			return m, nil
		}
		if userClicked(mouse.ZoneSettingsTabTickets) {
			m.settingsTabModel.SetSettingsTab(3)
			return m, nil
		}
		if userClicked(mouse.ZoneSettingsTabBranches) {
			m.settingsTabModel.SetSettingsTab(4)
			return m, nil
		}
		if userClicked(mouse.ZoneSettingsTabAdvanced) {
			m.settingsTabModel.SetSettingsTab(5)
			return m, nil
		}

		// Handle Tickets tab operations
		if m.settingsTabModel.GetSettingsTab() == 3 {
			// Ticket provider selection
			if userClicked(mouse.ZoneSettingsTicketProviderNone) {
				m.settingsTabModel.SetSettingsTicketProvider("")
				return m, nil
			}
			if userClicked(mouse.ZoneSettingsTicketProviderJira) {
				m.settingsTabModel.SetSettingsTicketProvider("jira")
				return m, nil
			}
			if userClicked(mouse.ZoneSettingsTicketProviderCodecks) {
				m.settingsTabModel.SetSettingsTicketProvider("codecks")
				return m, nil
			}
			if userClicked(mouse.ZoneSettingsTicketProviderGitHubIssues) {
				m.settingsTabModel.SetSettingsTicketProvider("github_issues")
				return m, nil
			}
			// Auto-status toggle (now in Tickets tab)
			if userClicked(mouse.ZoneSettingsAutoInProgress) {
				m.settingsTabModel.SetSettingsAutoInProgress(!(m.settingsTabModel.GetSettingsAutoInProgress()))
				return m, nil
			}
		}

		// Handle Advanced tab operations
		if m.settingsTabModel.GetSettingsTab() == 5 {
			// Cleanup confirmation buttons
			if m.settingsTabModel.GetConfirmingCleanup() != "" {
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
				m.settingsTabModel.SetSettingsSanitizeBookmarks(!(m.settingsTabModel.GetSettingsSanitizeBookmarks()))
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
			m.settingsTabModel.SetSettingsOnlyMine(!(m.settingsTabModel.GetSettingsOnlyMine()))
			return m, nil
		}
		if userClicked(mouse.ZoneSettingsGitHubShowMerged) {
			m.settingsTabModel.SetSettingsShowMerged(!m.settingsTabModel.GetSettingsShowMerged())
			return m, nil
		}
		if userClicked(mouse.ZoneSettingsGitHubShowClosed) {
			m.settingsTabModel.SetSettingsShowClosed(!m.settingsTabModel.GetSettingsShowClosed())
			return m, nil
		}
		if userClicked(mouse.ZoneSettingsGitHubPRLimitDecrease) {
			if m.settingsTabModel.GetSettingsPRLimit() > 25 {
				m.settingsTabModel.SetSettingsPRLimit(m.settingsTabModel.GetSettingsPRLimit() - 25)
			}
			return m, nil
		}
		if userClicked(mouse.ZoneSettingsGitHubPRLimitIncrease) {
			if m.settingsTabModel.GetSettingsPRLimit() < 500 {
				m.settingsTabModel.SetSettingsPRLimit(m.settingsTabModel.GetSettingsPRLimit() + 25)
			}
			return m, nil
		}

		// PR Refresh Interval controls
		if userClicked(mouse.ZoneSettingsGitHubRefreshDecrease) {
			if m.settingsTabModel.GetSettingsPRRefreshInterval() > 30 {
				m.settingsTabModel.SetSettingsPRRefreshInterval(m.settingsTabModel.GetSettingsPRRefreshInterval() - 30) // Decrease by 30 seconds
			} else if m.settingsTabModel.GetSettingsPRRefreshInterval() > 0 {
				m.settingsTabModel.SetSettingsPRRefreshInterval(0) // Go to disabled
			}
			return m, nil
		}
		if userClicked(mouse.ZoneSettingsGitHubRefreshIncrease) {
			if m.settingsTabModel.GetSettingsPRRefreshInterval() == 0 {
				m.settingsTabModel.SetSettingsPRRefreshInterval(30) // Enable at 30 seconds
			} else if m.settingsTabModel.GetSettingsPRRefreshInterval() < 600 {
				m.settingsTabModel.SetSettingsPRRefreshInterval(m.settingsTabModel.GetSettingsPRRefreshInterval() + 30) // Increase by 30 seconds (max 10 min)
			}
			return m, nil
		}
		if userClicked(mouse.ZoneSettingsGitHubRefreshToggle) {
			if m.settingsTabModel.GetSettingsPRRefreshInterval() == 0 {
				m.settingsTabModel.SetSettingsPRRefreshInterval(120) // Enable at 2 minutes (default)
			} else {
				m.settingsTabModel.SetSettingsPRRefreshInterval(0) // Disable
			}
			return m, nil
		}

		// Branch Limit controls
		if userClicked(mouse.ZoneSettingsBranchLimitDecrease) {
			if m.settingsTabModel.GetSettingsBranchLimit() > 10 {
				m.settingsTabModel.SetSettingsBranchLimit(m.settingsTabModel.GetSettingsBranchLimit() - 10)
			} else if m.settingsTabModel.GetSettingsBranchLimit() > 0 {
				m.settingsTabModel.SetSettingsBranchLimit(0) // 0 means unlimited
			}
			return m, nil
		}
		if userClicked(mouse.ZoneSettingsBranchLimitIncrease) {
			if m.settingsTabModel.GetSettingsBranchLimit() == 0 {
				m.settingsTabModel.SetSettingsBranchLimit(10) // Start at 10
			} else if m.settingsTabModel.GetSettingsBranchLimit() < 200 {
				m.settingsTabModel.SetSettingsBranchLimit(m.settingsTabModel.GetSettingsBranchLimit() + 10) // Increase by 10 (max 200)
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
				if i < len(inputs) {
					m.settingsTabModel.SetSettingInputValue(i, "")
					inputs[i].Focus()
					m.settingsTabModel.SetFocusedField(i)
					// Blur other inputs
					for j := range inputs {
						if j != i {
							inputs[j].Blur()
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
				m.settingsTabModel.SetFocusedField(i)
				for j := range inputs {
					if j == i {
						inputs[j].Focus()
					} else {
						inputs[j].Blur()
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
			m.helpTabModel.SetHelpTab(0)
			m.helpTabModel.SetSelectedCommand(0)
			return m, nil
		}
		if userClicked(mouse.ZoneHelpTabCommands) {
			m.helpTabModel.SetHelpTab(1)
			m.helpTabModel.SetSelectedCommand(0)
			return m, nil
		}

		// Command copy buttons (only on Commands tab)
		if m.helpTabModel.GetHelpTab() == 1 {
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
			commit := m.repository.Graph.Commits[m.GetSelectedCommit()]
			if commit.Immutable {
				m.statusMessage = "Cannot edit: commit is immutable"
				return m, nil
			}
			return m, m.checkoutCommit()
		}
	case ActionSquash:
		if m.isSelectedCommitValid() && m.jjService != nil {
			commit := m.repository.Graph.Commits[m.GetSelectedCommit()]
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
