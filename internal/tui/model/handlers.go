package model

import (
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleCheckoutCommit() (tea.Model, tea.Cmd) {
	if m.isSelectedCommitValid() && m.jjService != nil {
		commit := m.repository.Graph.Commits[m.selectedCommit]
		if commit.Immutable {
			m.statusMessage = "Cannot edit: commit is immutable"
			return m, nil
		}
		return m, m.checkoutCommit()
	}
	return m, nil
}

func (m *Model) handleSquashCommit() (tea.Model, tea.Cmd) {
	if m.isSelectedCommitValid() && m.jjService != nil {
		commit := m.repository.Graph.Commits[m.selectedCommit]
		if commit.Immutable {
			m.statusMessage = "Cannot squash: commit is immutable"
			return m, nil
		}
		return m, m.squashCommit()
	}
	return m, nil
}

func (m *Model) handleAbandonCommit() (tea.Model, tea.Cmd) {
	if m.isSelectedCommitValid() && m.jjService != nil {
		commit := m.repository.Graph.Commits[m.selectedCommit]
		if commit.Immutable {
			m.statusMessage = "Cannot abandon: commit is immutable"
			return m, nil
		}
		return m, m.abandonCommit()
	}
	return m, nil
}

func (m *Model) handleDescribeCommit() (tea.Model, tea.Cmd) {
	if m.isSelectedCommitValid() && m.jjService != nil {
		commit := m.repository.Graph.Commits[m.selectedCommit]
		if commit.Immutable {
			m.statusMessage = "Cannot edit description: commit is immutable"
			return m, nil
		}
		return m.startEditingDescription(commit)
	}
	return m, nil
}

func (m *Model) handleNewCommit() (tea.Model, tea.Cmd) {
	if m.jjService != nil {
		// Create a new commit as a child of the selected commit
		// This is valid even for immutable commits - we're creating a child, not modifying the parent
		if m.isSelectedCommitValid() {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			m.statusMessage = fmt.Sprintf("Creating new commit from %s...", commit.ShortID)
		} else {
			m.statusMessage = "Creating new commit..."
		}
		return m, m.createNewCommit()
	}
	return m, nil
}

func (m *Model) handleRebase() (tea.Model, tea.Cmd) {
	if m.isSelectedCommitValid() && m.jjService != nil {
		commit := m.repository.Graph.Commits[m.selectedCommit]
		if commit.Immutable {
			m.statusMessage = "Cannot rebase: commit is immutable"
			return m, nil
		}
		m.startRebaseMode()
	}
	return m, nil
}

func (m *Model) handleGraphFoucsMessage() string {
	return If(m.graphFocused, "Graph pane focused", "Files pane focused")
}

func (m *Model) handleNavigateToGraphTab() (tea.Model, tea.Cmd) {
	m.viewMode = ViewCommitGraph
	m.statusMessage = "Loading commit graph"
	return m, m.refreshRepository()
}

func (m *Model) handleNavigateToPRTab() (tea.Model, tea.Cmd) {
	m.viewMode = ViewPullRequests
	// Load PRs when switching to PR view
	if m.githubService != nil {
		m.statusMessage = "Loading PRs..."
		return m, m.loadPRs()
	}
	m.statusMessage = "GitHub service not initialized"
	return m, nil
}

func (m *Model) handleNavigateToTicketsTab() (tea.Model, tea.Cmd) {
	m.viewMode = ViewTickets
	if m.ticketService != nil {
		m.statusMessage = "Loading tickets..."
		return m, m.loadTickets()
	}
	return m, nil
}

func (m *Model) handleNavigateToSettingsTab() (tea.Model, tea.Cmd) {
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

func (m *Model) handleNavigateToHelpTab() (tea.Model, tea.Cmd) {
	m.viewMode = ViewHelp
	m.statusMessage = "Loaded Help"
	return m, nil
}

func (m *Model) handleNavigateToBranchesTab() (tea.Model, tea.Cmd) {
	m.viewMode = ViewBranches
	m.statusMessage = "Loading branches..."
	return m, m.loadBranches()
}

// Branch action handlers

func (m *Model) handleTrackBranch() (tea.Model, tea.Cmd) {
	if m.viewMode != ViewBranches || len(m.branchList) == 0 {
		return m, nil
	}
	if m.selectedBranch < 0 || m.selectedBranch >= len(m.branchList) {
		return m, nil
	}
	branch := m.branchList[m.selectedBranch]
	if branch.IsLocal || branch.IsTracked {
		m.statusMessage = "Branch is already tracked"
		return m, nil
	}
	m.statusMessage = fmt.Sprintf("Tracking branch %s...", branch.Name)
	return m, m.trackBranch(branch.Name, branch.Remote)
}

func (m *Model) handleUntrackBranch() (tea.Model, tea.Cmd) {
	if m.viewMode != ViewBranches || len(m.branchList) == 0 {
		return m, nil
	}
	if m.selectedBranch < 0 || m.selectedBranch >= len(m.branchList) {
		return m, nil
	}
	branch := m.branchList[m.selectedBranch]
	if !branch.IsTracked {
		m.statusMessage = "Branch is not tracked"
		return m, nil
	}
	m.statusMessage = fmt.Sprintf("Untracking branch %s...", branch.Name)
	return m, m.untrackBranch(branch.Name, branch.Remote)
}

func (m *Model) handleRestoreLocalBranch() (tea.Model, tea.Cmd) {
	if m.viewMode != ViewBranches || len(m.branchList) == 0 {
		return m, nil
	}
	if m.selectedBranch < 0 || m.selectedBranch >= len(m.branchList) {
		return m, nil
	}
	branch := m.branchList[m.selectedBranch]
	if !branch.LocalDeleted {
		m.statusMessage = "Branch local copy is not deleted"
		return m, nil
	}
	m.statusMessage = fmt.Sprintf("Restoring local branch %s...", branch.Name)
	return m, m.restoreLocalBranch(branch.Name, branch.CommitID)
}

func (m *Model) handleDeleteBranchBookmark() (tea.Model, tea.Cmd) {
	if m.viewMode != ViewBranches || len(m.branchList) == 0 {
		return m, nil
	}
	if m.selectedBranch < 0 || m.selectedBranch >= len(m.branchList) {
		return m, nil
	}
	branch := m.branchList[m.selectedBranch]
	if !branch.IsLocal {
		m.statusMessage = "Can only delete local bookmarks"
		return m, nil
	}
	m.statusMessage = fmt.Sprintf("Deleting bookmark %s...", branch.Name)
	return m, m.deleteBranchBookmark(branch.Name)
}

func (m *Model) handlePushBranch() (tea.Model, tea.Cmd) {
	if m.viewMode != ViewBranches || len(m.branchList) == 0 {
		return m, nil
	}
	if m.selectedBranch < 0 || m.selectedBranch >= len(m.branchList) {
		return m, nil
	}
	branch := m.branchList[m.selectedBranch]
	if !branch.IsLocal {
		m.statusMessage = "Can only push local branches"
		return m, nil
	}
	m.statusMessage = fmt.Sprintf("Pushing branch %s...", branch.Name)
	return m, m.pushBranch(branch.Name)
}

func (m *Model) handleFetchAll() (tea.Model, tea.Cmd) {
	m.statusMessage = "Fetching from all remotes..."
	return m, m.fetchAllRemotes()
}

func (m *Model) handleCopyError() (tea.Model, tea.Cmd) {
	// Copy error to clipboard (works with m.err or status message errors)
	// Important: capture the error BEFORE changing statusMessage
	errMsg := m.getErrorMessage()
	if errMsg != "" {
		m.statusMessage = "Copying error to clipboard..."
		return m, m.copyErrorMessageToClipboard(errMsg)
	}
	return m, nil
}

func (m *Model) handleDismissError() (tea.Model, tea.Cmd) {
	// Dismiss/clear the error and restart auto-refresh
	m.err = nil
	m.statusMessage = "Ready"
	return m, m.tickCmd()
}

func (m *Model) handleUndo() (tea.Model, tea.Cmd) {
	if m.jjService != nil {
		m.statusMessage = "Undoing..."
		return m, m.undoOperation()
	}
	return m, nil
}

func (m *Model) handleRedo() (tea.Model, tea.Cmd) {
	if m.jjService != nil {
		m.statusMessage = "Redoing..."
		return m, m.redoOperation()
	}
	return m, nil
}

func (m *Model) handleSelectCommit(index int) (tea.Model, tea.Cmd) {
	m.selectedCommit = index
	// Load changed files for the selected commit
	if m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
		commit := m.repository.Graph.Commits[m.selectedCommit]
		m.changedFilesCommitID = commit.ChangeID
		m.changedFiles = nil // Clear old files while loading
		m.selectedFile = 0   // Reset file selection
		return m, m.loadChangedFiles(commit.ChangeID)
	}
	return m, nil
}

func (m *Model) handleCreatePR() (tea.Model, tea.Cmd) {
	if m.githubService == nil {
		m.statusMessage = "GitHub not connected. Configure in Settings (,)"
		return m, nil
	}
	if m.isSelectedCommitValid() && m.jjService != nil {
		m.startCreatePR()
	}
	return m, nil
}

func (m *Model) handleCreateBookmark() (tea.Model, tea.Cmd) {
	if m.isSelectedCommitValid() && m.jjService != nil {
		commit := m.repository.Graph.Commits[m.selectedCommit]
		if commit.Immutable {
			m.statusMessage = "Cannot create bookmark: commit is immutable"
			return m, nil
		}
		m.startCreateBookmark()
	}
	return m, nil
}

func (m *Model) handleDeleteBookmark() (tea.Model, tea.Cmd) {
	if m.isSelectedCommitValid() && m.jjService != nil {
		commit := m.repository.Graph.Commits[m.selectedCommit]
		if len(commit.Branches) == 0 {
			m.statusMessage = "No bookmark on this commit to delete"
			return m, nil
		}
		return m, m.deleteBookmark()
	}
	return m, nil
}

func (m *Model) handleUpdatePR() (tea.Model, tea.Cmd) {
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
		if slices.Contains(commit.Branches, prBranch) {
			needsMoveBookmark = false
		}
		return m, m.pushToPR(prBranch, commit.ChangeID, needsMoveBookmark)
	}
	return m, nil
}

func (m *Model) handleOpenPRInBrowser() (tea.Model, tea.Cmd) {
	if m.repository != nil && m.selectedPR >= 0 && m.selectedPR < len(m.repository.PRs) {
		pr := m.repository.PRs[m.selectedPR]
		if pr.URL != "" {
			m.statusMessage = fmt.Sprintf("Opening PR #%d...", pr.Number)
			return m, openURL(pr.URL)
		}
	}
	return m, nil
}

func (m *Model) handleOpenTicketInBrowser() (tea.Model, tea.Cmd) {
	if m.ticketService != nil && m.selectedTicket >= 0 && m.selectedTicket < len(m.ticketList) {
		ticket := m.ticketList[m.selectedTicket]
		ticketURL := m.ticketService.GetTicketURL(ticket)
		m.statusMessage = fmt.Sprintf("Opening %s...", ticket.DisplayKey)
		return m, openURL(ticketURL)
	}
	return m, nil
}

func (m *Model) handleMergePR() (tea.Model, tea.Cmd) {
	if m.viewMode == ViewPullRequests && m.githubService != nil && m.repository != nil && m.selectedPR >= 0 && m.selectedPR < len(m.repository.PRs) {
		pr := m.repository.PRs[m.selectedPR]
		if pr.State != "open" {
			m.statusMessage = "Can only merge open PRs"
			return m, nil
		}
		m.statusMessage = fmt.Sprintf("Merging PR #%d...", pr.Number)
		return m, m.mergePR(pr.Number)
	}
	return m, nil
}

func (m *Model) handleClosePR() (tea.Model, tea.Cmd) {
	if m.viewMode == ViewPullRequests && m.githubService != nil && m.repository != nil && m.selectedPR >= 0 && m.selectedPR < len(m.repository.PRs) {
		pr := m.repository.PRs[m.selectedPR]
		if pr.State != "open" {
			m.statusMessage = "Can only close open PRs"
			return m, nil
		}
		m.statusMessage = fmt.Sprintf("Closing PR #%d...", pr.Number)
		return m, m.closePR(pr.Number)
	}
	return m, nil
}

func (m *Model) handleToggleStatusChangeMode() (tea.Model, tea.Cmd) {
	if m.viewMode == ViewTickets && m.ticketService != nil && !m.transitionInProgress {
		m.statusChangeMode = !m.statusChangeMode
		if m.statusChangeMode {
			m.statusMessage = "Select a status to apply (i/D/B/N or Esc to cancel)"
		} else {
			m.statusMessage = "Ready"
		}
	}
	return m, nil
}

func (m *Model) handleStartBookmarkFromTicket() (tea.Model, tea.Cmd) {
	if m.viewMode == ViewTickets && m.selectedTicket >= 0 && m.selectedTicket < len(m.ticketList) && m.jjService != nil {
		ticket := m.ticketList[m.selectedTicket]
		m.startBookmarkFromTicket(ticket)
	}
	return m, nil
}

func (m *Model) handleTransitionToInProgress() (tea.Model, tea.Cmd) {
	if m.viewMode != ViewTickets || m.ticketService == nil || !m.statusChangeMode || m.transitionInProgress {
		return m, nil
	}
	if m.selectedTicket < 0 || m.selectedTicket >= len(m.ticketList) {
		return m, nil
	}
	// Find "in progress" transition (must contain "progress" or "start" but NOT "not start")
	for _, t := range m.availableTransitions {
		lowerName := strings.ToLower(t.Name)
		isInProgress := strings.Contains(lowerName, "progress") ||
			(strings.Contains(lowerName, "start") && !strings.Contains(lowerName, "not start") && !strings.Contains(lowerName, "not_start"))
		if isInProgress {
			m.transitionInProgress = true
			ticket := m.ticketList[m.selectedTicket]
			m.statusMessage = fmt.Sprintf("Setting %s to %s...", ticket.DisplayKey, t.Name)
			return m, m.transitionTicket(t.ID)
		}
	}
	m.statusMessage = "No 'In Progress' transition available"
	return m, nil
}

func (m *Model) handleTransitionToDone() (tea.Model, tea.Cmd) {
	if m.viewMode != ViewTickets || m.ticketService == nil || !m.statusChangeMode || m.transitionInProgress {
		return m, nil
	}
	if m.selectedTicket < 0 || m.selectedTicket >= len(m.ticketList) {
		return m, nil
	}
	// Find "done" transition
	for _, t := range m.availableTransitions {
		lowerName := strings.ToLower(t.Name)
		if strings.Contains(lowerName, "done") || strings.Contains(lowerName, "complete") || strings.Contains(lowerName, "resolve") {
			m.transitionInProgress = true
			ticket := m.ticketList[m.selectedTicket]
			m.statusMessage = fmt.Sprintf("Setting %s to %s...", ticket.DisplayKey, t.Name)
			return m, m.transitionTicket(t.ID)
		}
	}
	m.statusMessage = "No 'Done' transition available"
	return m, nil
}

func (m *Model) handleTransitionToBlocked() (tea.Model, tea.Cmd) {
	if m.viewMode != ViewTickets || m.ticketService == nil || !m.statusChangeMode || m.transitionInProgress {
		return m, nil
	}
	if m.selectedTicket < 0 || m.selectedTicket >= len(m.ticketList) {
		return m, nil
	}
	// Find "blocked" transition
	for _, t := range m.availableTransitions {
		lowerName := strings.ToLower(t.Name)
		if strings.Contains(lowerName, "block") {
			m.transitionInProgress = true
			ticket := m.ticketList[m.selectedTicket]
			m.statusMessage = fmt.Sprintf("Setting %s to %s...", ticket.DisplayKey, t.Name)
			return m, m.transitionTicket(t.ID)
		}
	}
	m.statusMessage = "No 'Blocked' transition available"
	return m, nil
}

func (m *Model) handleTransitionToNotStarted() (tea.Model, tea.Cmd) {
	if m.viewMode != ViewTickets || m.ticketService == nil || !m.statusChangeMode || m.transitionInProgress {
		return m, nil
	}
	if m.selectedTicket < 0 || m.selectedTicket >= len(m.ticketList) {
		return m, nil
	}
	// Find "not started" transition
	for _, t := range m.availableTransitions {
		lowerName := strings.ToLower(t.Name)
		if strings.Contains(lowerName, "not") && strings.Contains(lowerName, "start") {
			m.transitionInProgress = true
			ticket := m.ticketList[m.selectedTicket]
			m.statusMessage = fmt.Sprintf("Setting %s to %s...", ticket.DisplayKey, t.Name)
			return m, m.transitionTicket(t.ID)
		}
	}
	m.statusMessage = "No 'Not Started' transition available"
	return m, nil
}

func (m *Model) handleDescriptionSave() (tea.Model, tea.Cmd) {
	if m.viewMode == ViewEditDescription {
		return m, m.saveDescription()
	}
	return m, nil
}

func (m *Model) handleDescriptionCancel() (tea.Model, tea.Cmd) {
	if m.viewMode == ViewEditDescription {
		m.viewMode = ViewCommitGraph
		m.editingCommitID = ""
		m.statusMessage = "Description edit cancelled"
	}
	return m, nil
}

func (m *Model) handleBookmarkCancel() (tea.Model, tea.Cmd) {
	if m.viewMode == ViewCreateBookmark {
		m.viewMode = ViewCommitGraph
		m.statusMessage = "Bookmark creation cancelled"
	}
	return m, nil
}

func (m *Model) handleBookmarkSubmit() (tea.Model, tea.Cmd) {
	if m.viewMode == ViewCreateBookmark && m.jjService != nil {
		return m, m.submitBookmark()
	}
	return m, nil
}

func (m *Model) handlePRCancel() (tea.Model, tea.Cmd) {
	if m.viewMode == ViewCreatePR {
		m.viewMode = ViewCommitGraph
		m.statusMessage = "PR creation cancelled"
	}
	return m, nil
}

func (m *Model) handlePRSubmit() (tea.Model, tea.Cmd) {
	if m.viewMode == ViewCreatePR && m.githubService != nil && m.jjService != nil {
		return m, m.submitPR()
	}
	return m, nil
}

func (m *Model) handleSettingsCancel() (tea.Model, tea.Cmd) {
	if m.viewMode == ViewSettings {
		m.viewMode = ViewCommitGraph
		m.statusMessage = "Settings cancelled"
	}
	return m, nil
}

func (m *Model) handleMoveFileUp() (tea.Model, tea.Cmd) {
	if m.viewMode != ViewCommitGraph || m.graphFocused {
		return m, nil
	}
	if m.jjService == nil || len(m.changedFiles) == 0 {
		return m, nil
	}
	if m.selectedFile < 0 || m.selectedFile >= len(m.changedFiles) {
		return m, nil
	}

	// Get the selected commit and verify it's mutable
	if m.repository == nil || m.selectedCommit < 0 || m.selectedCommit >= len(m.repository.Graph.Commits) {
		return m, nil
	}
	commit := m.repository.Graph.Commits[m.selectedCommit]
	if commit.Immutable {
		m.statusMessage = "Cannot move file: commit is immutable"
		return m, nil
	}

	file := m.changedFiles[m.selectedFile]
	// Use the selected commit's ChangeID directly (not the cached changedFilesCommitID)
	commitID := commit.ChangeID
	// "Up" in the graph view means toward newer commits (children)
	m.statusMessage = fmt.Sprintf("Moving %s to new child commit...", file.Path)
	return m, m.moveFileToChild(commitID, file.Path)
}

func (m *Model) handleMoveFileDown() (tea.Model, tea.Cmd) {
	if m.viewMode != ViewCommitGraph || m.graphFocused {
		return m, nil
	}
	if m.jjService == nil || len(m.changedFiles) == 0 {
		return m, nil
	}
	if m.selectedFile < 0 || m.selectedFile >= len(m.changedFiles) {
		return m, nil
	}

	// Get the selected commit and verify it's mutable
	if m.repository == nil || m.selectedCommit < 0 || m.selectedCommit >= len(m.repository.Graph.Commits) {
		return m, nil
	}
	commit := m.repository.Graph.Commits[m.selectedCommit]
	if commit.Immutable {
		m.statusMessage = "Cannot move file: commit is immutable"
		return m, nil
	}

	file := m.changedFiles[m.selectedFile]
	// Use the selected commit's ChangeID directly (not the cached changedFilesCommitID)
	commitID := commit.ChangeID
	// "Down" in the graph view means toward older commits (parents)
	m.statusMessage = fmt.Sprintf("Moving %s to new parent commit...", file.Path)
	return m, m.moveFileToParent(commitID, file.Path)
}
