package model

import (
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	branchestab "github.com/madicen/jj-tui/internal/tui/tabs/branches"
	graphtab "github.com/madicen/jj-tui/internal/tui/tabs/graph"
	helptab "github.com/madicen/jj-tui/internal/tui/tabs/help"
	prstab "github.com/madicen/jj-tui/internal/tui/tabs/prs"
	settingstab "github.com/madicen/jj-tui/internal/tui/tabs/settings"
	ticketstab "github.com/madicen/jj-tui/internal/tui/tabs/tickets"
)

func (m *Model) handleCheckoutCommit() (tea.Model, tea.Cmd) {
	if m.isSelectedCommitValid() && m.jjService != nil {
		commit := m.repository.Graph.Commits[m.GetSelectedCommit()]
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
		commit := m.repository.Graph.Commits[m.GetSelectedCommit()]
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
		commit := m.repository.Graph.Commits[m.GetSelectedCommit()]
		if commit.Immutable {
			m.statusMessage = "Cannot abandon: commit is immutable"
			return m, nil
		}
		if commit.Divergent {
			// Show divergent commit resolution dialog
			return m.handleResolveDivergentCommit()
		}
		return m, m.abandonCommit()
	}
	return m, nil
}

func (m *Model) handleResolveDivergentCommit() (tea.Model, tea.Cmd) {
	if m.isSelectedCommitValid() && m.jjService != nil {
		commit := m.repository.Graph.Commits[m.GetSelectedCommit()]
		if !commit.Divergent {
			m.statusMessage = "This commit is not divergent"
			return m, nil
		}
		m.statusMessage = "Loading divergent commit info..."
		return m, m.loadDivergentCommitInfo(commit.ChangeID)
	}
	return m, nil
}

func (m *Model) handleDescribeCommit() (tea.Model, tea.Cmd) {
	if m.isSelectedCommitValid() && m.jjService != nil {
		commit := m.repository.Graph.Commits[m.GetSelectedCommit()]
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
			commit := m.repository.Graph.Commits[m.GetSelectedCommit()]
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
		commit := m.repository.Graph.Commits[m.GetSelectedCommit()]
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

// handleGraphRequest processes requests from the graph tab (keys/zones); main runs jj commands.
func (m *Model) handleGraphRequest(r graphtab.Request) (tea.Model, tea.Cmd) {
	ctx := m.graphRequestContext()
	// Requests executed by the graph tab (returns cmd + optional status message)
	if r.Checkout || r.Squash || r.Abandon || r.PerformRebase || r.DeleteBookmark ||
		r.MoveFileUp || r.MoveFileDown || r.RevertFile || r.NewCommit {
		cmd, statusMsg := graphtab.ExecuteRequest(r, ctx)
		if statusMsg != "" {
			if statusMsg == graphtab.StatusNeedsDivergentDialog {
				return m.handleResolveDivergentCommit()
			}
			m.statusMessage = statusMsg
			return m, nil
		}
		if cmd != nil {
			if r.NewCommit {
				if m.isSelectedCommitValid() {
					commit := m.repository.Graph.Commits[m.GetSelectedCommit()]
					m.statusMessage = fmt.Sprintf("Creating new commit from %s...", commit.ShortID)
				} else {
					m.statusMessage = "Creating new commit..."
				}
			}
			return m, cmd
		}
	}
	// Load changed files (returns model-internal message)
	if r.LoadChangedFiles != nil {
		return m, m.loadChangedFiles(*r.LoadChangedFiles)
	}
	if r.ResolveDivergent != nil {
		m.statusMessage = "Loading divergent commit info..."
		return m, m.loadDivergentCommitInfo(*r.ResolveDivergent)
	}
	if r.StartEditDescription {
		return m.handleDescribeCommit()
	}
	if r.StartRebaseMode {
		return m.handleRebase()
	}
	if r.CreateBookmark {
		return m.handleCreateBookmark()
	}
	if r.CreatePR {
		return m.handleCreatePR()
	}
	if r.UpdatePR {
		return m.handleUpdatePR()
	}
	return m, nil
}

// graphRequestContext builds the context passed to graph.ExecuteRequest.
func (m *Model) graphRequestContext() *graphtab.RequestContext {
	if m.repository == nil {
		return nil
	}
	return &graphtab.RequestContext{
		JJService:            m.jjService,
		Repository:           m.repository,
		SelectedCommit:       m.GetSelectedCommit(),
		RebaseSourceCommit:   m.rebaseSourceCommit,
		ChangedFiles:         m.graphTabModel.GetChangedFiles(),
		ChangedFilesCommitID: m.graphTabModel.GetChangedFilesCommitID(),
		SelectedFile:         m.graphTabModel.GetSelectedFile(),
		GraphFocused:         m.graphFocused,
	}
}

func (m *Model) handlePRsRequest(r prstab.Request) (tea.Model, tea.Cmd) {
	ctx := m.prsRequestContext()
	cb := &prstab.Callbacks{
		MergePR:       m.mergePR,
		ClosePR:       m.closePR,
		OpenInBrowser: m.openPRURL,
	}
	cmd, statusMsg := prstab.ExecuteRequest(r, ctx, cb)
	if statusMsg != "" {
		m.statusMessage = statusMsg
		return m, nil
	}
	if cmd != nil {
		if r.OpenInBrowser && ctx != nil && ctx.SelectedPRValid() {
			pr := ctx.SelectedPRData()
			if pr != nil {
				m.statusMessage = fmt.Sprintf("Opening PR #%d...", pr.Number)
			}
		}
		if r.MergePR && ctx != nil && ctx.SelectedPRValid() {
			pr := ctx.SelectedPRData()
			if pr != nil {
				m.statusMessage = fmt.Sprintf("Merging PR #%d...", pr.Number)
			}
		}
		if r.ClosePR && ctx != nil && ctx.SelectedPRValid() {
			pr := ctx.SelectedPRData()
			if pr != nil {
				m.statusMessage = fmt.Sprintf("Closing PR #%d...", pr.Number)
			}
		}
		return m, cmd
	}
	return m, nil
}

func (m *Model) prsRequestContext() *prstab.RequestContext {
	if m.repository == nil {
		return nil
	}
	return &prstab.RequestContext{
		Repository: m.repository,
		SelectedPR: m.GetSelectedPR(),
		GitHubOK:   m.isGitHubAvailable(),
		DemoMode:   m.demoMode,
	}
}

// openPRURL opens a URL in the browser (used by PRs tab callback).
func (m *Model) openPRURL(url string) tea.Cmd {
	return openURL(url)
}

func (m *Model) handleBranchesRequest(r branchestab.Request) (tea.Model, tea.Cmd) {
	if r.TrackBranch {
		return m.handleTrackBranch()
	}
	if r.UntrackBranch {
		return m.handleUntrackBranch()
	}
	if r.RestoreLocalBranch {
		return m.handleRestoreLocalBranch()
	}
	if r.DeleteBranchBookmark {
		return m.handleDeleteBranchBookmark()
	}
	if r.PushBranch {
		return m.handlePushBranch()
	}
	if r.FetchAll {
		return m.handleFetchAll()
	}
	if r.ResolveBookmarkConflict {
		return m.handleResolveBookmarkConflict()
	}
	return m, nil
}

func (m *Model) handleTicketsRequest(r ticketstab.Request) (tea.Model, tea.Cmd) {
	ctx := m.ticketsRequestContext()
	cb := &ticketstab.Callbacks{
		OpenInBrowser:    m.openTicketURL,
		TransitionTicket: m.transitionTicketWithState,
	}
	res := ticketstab.ExecuteRequest(r, ctx, cb)
	if res.StatusMsg != "" {
		m.statusMessage = res.StatusMsg
		return m, nil
	}
	if res.NeedToggleMode {
		return m.handleToggleStatusChangeMode()
	}
	if res.NeedStartBookmark {
		return m.handleStartBookmarkFromTicket()
	}
	if res.Cmd != nil {
		if res.TransitionStatus != "" {
			m.transitionInProgress = true
			m.ticketsTabModel.SetTransitionInProgress(true)
			m.statusMessage = res.TransitionStatus
		} else if r.OpenInBrowser && ctx.SelectedTicketValid() {
			if t := ctx.SelectedTicketData(); t != nil {
				m.statusMessage = fmt.Sprintf("Opening %s...", t.DisplayKey)
			}
		}
		return m, res.Cmd
	}
	return m, nil
}

func (m *Model) ticketsRequestContext() *ticketstab.RequestContext {
	return &ticketstab.RequestContext{
		TicketList:           m.ticketList,
		SelectedTicket:       m.GetSelectedTicket(),
		AvailableTransitions: m.availableTransitions,
		TransitionInProgress: m.transitionInProgress,
		TicketService:        m.ticketService,
	}
}

// openTicketURL opens a URL in the browser (used by Tickets tab callback).
func (m *Model) openTicketURL(url string) tea.Cmd {
	return openURL(url)
}

// transitionTicketWithState sets transition-in-progress state and returns the transition command (used by Tickets tab callback).
func (m *Model) transitionTicketWithState(transitionID string) tea.Cmd {
	m.transitionInProgress = true
	m.ticketsTabModel.SetTransitionInProgress(true)
	return m.transitionTicket(transitionID)
}

func (m *Model) handleHelpRequest(r helptab.Request) (tea.Model, tea.Cmd) {
	cmd, statusMsg := helptab.ExecuteRequest(r)
	if statusMsg != "" {
		m.statusMessage = statusMsg
		return m, cmd
	}
	return m, nil
}

func (m *Model) handleSettingsRequest(r settingstab.Request) (tea.Model, tea.Cmd) {
	cb := &settingstab.Callbacks{
		SaveSettings:      m.saveSettings,
		SaveSettingsLocal: m.saveSettingsLocal,
	}
	res := settingstab.ExecuteRequest(r, cb)
	if res.StatusMsg != "" {
		m.statusMessage = res.StatusMsg
		return m, nil
	}
	if res.NeedCancel {
		return m.handleSettingsCancel()
	}
	if res.Cmd != nil {
		return m, res.Cmd
	}
	return m, nil
}

func (m *Model) handleNavigateToGraphTab() (tea.Model, tea.Cmd) {
	m.viewMode = ViewCommitGraph
	m.statusMessage = "Loading commit graph"
	return m, m.refreshRepository()
}

func (m *Model) handleNavigateToPRTab() (tea.Model, tea.Cmd) {
	m.viewMode = ViewPullRequests
	// Load PRs when switching to PR view
	if m.isGitHubAvailable() {
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
	m.settingsTabModel.SetFocusedField(0)
	inputs := m.settingsTabModel.GetSettingsInputs()
	for i := range inputs {
		if i == 0 {
			inputs[i].Focus()
		} else {
			inputs[i].Blur()
		}
	}
	return m, nil
}

func (m *Model) handleNavigateToHelpTab() (tea.Model, tea.Cmd) {
	m.viewMode = ViewHelp
	// Keep current help sub-tab (Shortcuts vs History) and selection so returning to Help remembers where you were
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
	if m.GetSelectedBranch() < 0 || m.GetSelectedBranch() >= len(m.branchList) {
		return m, nil
	}
	branch := m.branchList[m.GetSelectedBranch()]
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
	if m.GetSelectedBranch() < 0 || m.GetSelectedBranch() >= len(m.branchList) {
		return m, nil
	}
	branch := m.branchList[m.GetSelectedBranch()]
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
	if m.GetSelectedBranch() < 0 || m.GetSelectedBranch() >= len(m.branchList) {
		return m, nil
	}
	branch := m.branchList[m.GetSelectedBranch()]
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
	if m.GetSelectedBranch() < 0 || m.GetSelectedBranch() >= len(m.branchList) {
		return m, nil
	}
	branch := m.branchList[m.GetSelectedBranch()]
	if !branch.IsLocal {
		m.statusMessage = "Can only delete local bookmarks"
		return m, nil
	}
	// If the branch has a conflict, prompt user to resolve it first
	if branch.HasConflict {
		m.statusMessage = "This bookmark has diverged. Resolve the conflict first (press 'c')."
		return m, nil
	}
	m.statusMessage = fmt.Sprintf("Deleting bookmark %s...", branch.Name)
	return m, m.deleteBranchBookmark(branch.Name)
}

func (m *Model) handleResolveBookmarkConflict() (tea.Model, tea.Cmd) {
	if m.viewMode != ViewBranches || len(m.branchList) == 0 {
		return m, nil
	}
	if m.GetSelectedBranch() < 0 || m.GetSelectedBranch() >= len(m.branchList) {
		return m, nil
	}
	branch := m.branchList[m.GetSelectedBranch()]
	if !branch.HasConflict {
		m.statusMessage = "This bookmark is not conflicted"
		return m, nil
	}
	m.statusMessage = "Loading conflict info..."
	return m, m.loadBookmarkConflictInfo(branch.Name)
}

func (m *Model) handlePushBranch() (tea.Model, tea.Cmd) {
	if m.viewMode != ViewBranches || len(m.branchList) == 0 {
		return m, nil
	}
	if m.GetSelectedBranch() < 0 || m.GetSelectedBranch() >= len(m.branchList) {
		return m, nil
	}
	branch := m.branchList[m.GetSelectedBranch()]
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
	m.errorCopied = false
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
	m.graphTabModel.SelectCommit(index)
	if m.repository != nil && index >= 0 && index < len(m.repository.Graph.Commits) {
		commit := m.repository.Graph.Commits[index]
		return m, m.loadChangedFiles(commit.ChangeID)
	}
	return m, nil
}

func (m *Model) handleCreatePR() (tea.Model, tea.Cmd) {
	if !m.isGitHubAvailable() {
		m.statusMessage = "GitHub not connected. Configure in Settings (,)"
		return m, nil
	}
	if m.isSelectedCommitValid() && m.jjService != nil {
		// Check for commits with empty descriptions
		emptyDescCommits := m.findCommitsWithEmptyDescriptions()
		if len(emptyDescCommits) > 0 {
			m.showWarningModal = true
			m.warningTitle = "Commits Need Descriptions"
			m.warningMessage = "GitHub requires commit descriptions. Please add descriptions before creating a PR."
			m.warningCommits = emptyDescCommits
			m.warningSelectedIdx = 0
			return m, nil
		}
		m.startCreatePR()
	}
	return m, nil
}

func (m *Model) handleCreateBookmark() (tea.Model, tea.Cmd) {
	if m.isSelectedCommitValid() && m.jjService != nil {
		commit := m.repository.Graph.Commits[m.GetSelectedCommit()]
		if commit.Immutable {
			m.statusMessage = "Cannot create bookmark: commit is immutable"
			return m, nil
		}
		m.startCreateBookmark()
		// Load branches in the background to ensure duplicate checking has full data
		return m, m.loadBranches()
	}
	return m, nil
}

func (m *Model) handleDeleteBookmark() (tea.Model, tea.Cmd) {
	if m.isSelectedCommitValid() && m.jjService != nil {
		commit := m.repository.Graph.Commits[m.GetSelectedCommit()]
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
		prBranch := m.findPRBranchForCommit(m.GetSelectedCommit())
		if prBranch == "" {
			m.statusMessage = "No open PR found for this commit or its ancestors"
			return m, nil
		}

		// Check for commits with empty descriptions
		emptyDescCommits := m.findCommitsWithEmptyDescriptions()
		if len(emptyDescCommits) > 0 {
			m.showWarningModal = true
			m.warningTitle = "Commits Need Descriptions"
			m.warningMessage = "GitHub requires commit descriptions. Please add descriptions before updating the PR."
			m.warningCommits = emptyDescCommits
			m.warningSelectedIdx = 0
			return m, nil
		}

		commit := m.repository.Graph.Commits[m.GetSelectedCommit()]
		needsMoveBookmark := true
		if slices.Contains(commit.Branches, prBranch) {
			needsMoveBookmark = false
		}
		return m, m.pushToPR(prBranch, commit.ChangeID, needsMoveBookmark)
	}
	return m, nil
}

func (m *Model) handleOpenPRInBrowser() (tea.Model, tea.Cmd) {
	if m.repository != nil && m.GetSelectedPR() >= 0 && m.GetSelectedPR() < len(m.repository.PRs) {
		pr := m.repository.PRs[m.GetSelectedPR()]
		if pr.URL != "" {
			if m.demoMode {
				m.statusMessage = fmt.Sprintf("PR #%d: %s (demo mode - browser disabled)", pr.Number, pr.URL)
				return m, nil
			}
			m.statusMessage = fmt.Sprintf("Opening PR #%d...", pr.Number)
			return m, openURL(pr.URL)
		}
	}
	return m, nil
}

func (m *Model) handleOpenTicketInBrowser() (tea.Model, tea.Cmd) {
	if m.ticketService != nil && m.GetSelectedTicket() >= 0 && m.GetSelectedTicket() < len(m.ticketList) {
		ticket := m.ticketList[m.GetSelectedTicket()]
		ticketURL := m.ticketService.GetTicketURL(ticket)
		m.statusMessage = fmt.Sprintf("Opening %s...", ticket.DisplayKey)
		return m, openURL(ticketURL)
	}
	return m, nil
}

func (m *Model) handleMergePR() (tea.Model, tea.Cmd) {
	if m.viewMode == ViewPullRequests && m.isGitHubAvailable() && m.repository != nil && m.GetSelectedPR() >= 0 && m.GetSelectedPR() < len(m.repository.PRs) {
		pr := m.repository.PRs[m.GetSelectedPR()]
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
	if m.viewMode == ViewPullRequests && m.isGitHubAvailable() && m.repository != nil && m.GetSelectedPR() >= 0 && m.GetSelectedPR() < len(m.repository.PRs) {
		pr := m.repository.PRs[m.GetSelectedPR()]
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
		m.ticketsTabModel.SetStatusChangeMode(m.statusChangeMode)
		if m.statusChangeMode {
			m.statusMessage = "Select a status to apply (i/D/B/N or Esc to cancel)"
		} else {
			m.statusMessage = "Ready"
		}
	}
	return m, nil
}

func (m *Model) handleStartBookmarkFromTicket() (tea.Model, tea.Cmd) {
	if m.viewMode == ViewTickets && m.GetSelectedTicket() >= 0 && m.GetSelectedTicket() < len(m.ticketList) && m.jjService != nil {
		ticket := m.ticketList[m.GetSelectedTicket()]
		m.startBookmarkFromTicket(ticket)
	}
	return m, nil
}

func (m *Model) handleTransitionToInProgress() (tea.Model, tea.Cmd) {
	if m.viewMode != ViewTickets || m.ticketService == nil || !m.statusChangeMode || m.transitionInProgress {
		return m, nil
	}
	if m.GetSelectedTicket() < 0 || m.GetSelectedTicket() >= len(m.ticketList) {
		return m, nil
	}
	// Find "in progress" transition (must contain "progress" or "start" but NOT "not start")
	for _, t := range m.availableTransitions {
		lowerName := strings.ToLower(t.Name)
		isInProgress := strings.Contains(lowerName, "progress") ||
			(strings.Contains(lowerName, "start") && !strings.Contains(lowerName, "not start") && !strings.Contains(lowerName, "not_start"))
		if isInProgress {
			m.transitionInProgress = true
			m.ticketsTabModel.SetTransitionInProgress(true)
			ticket := m.ticketList[m.GetSelectedTicket()]
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
	if m.GetSelectedTicket() < 0 || m.GetSelectedTicket() >= len(m.ticketList) {
		return m, nil
	}
	// Find "done" transition
	for _, t := range m.availableTransitions {
		lowerName := strings.ToLower(t.Name)
		if strings.Contains(lowerName, "done") || strings.Contains(lowerName, "complete") || strings.Contains(lowerName, "resolve") {
			m.transitionInProgress = true
			m.ticketsTabModel.SetTransitionInProgress(true)
			ticket := m.ticketList[m.GetSelectedTicket()]
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
	if m.GetSelectedTicket() < 0 || m.GetSelectedTicket() >= len(m.ticketList) {
		return m, nil
	}
	// Find "blocked" transition
	for _, t := range m.availableTransitions {
		lowerName := strings.ToLower(t.Name)
		if strings.Contains(lowerName, "block") {
			m.transitionInProgress = true
			m.ticketsTabModel.SetTransitionInProgress(true)
			ticket := m.ticketList[m.GetSelectedTicket()]
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
	if m.GetSelectedTicket() < 0 || m.GetSelectedTicket() >= len(m.ticketList) {
		return m, nil
	}
	// Find "not started" transition
	for _, t := range m.availableTransitions {
		lowerName := strings.ToLower(t.Name)
		if strings.Contains(lowerName, "not") && strings.Contains(lowerName, "start") {
			m.transitionInProgress = true
			m.ticketsTabModel.SetTransitionInProgress(true)
			ticket := m.ticketList[m.GetSelectedTicket()]
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
		m.graphTabModel.SetEditingCommitID("")
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
	if m.viewMode == ViewCreatePR && m.isGitHubAvailable() && m.jjService != nil {
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
	changedFiles := m.graphTabModel.GetChangedFiles()
	if m.jjService == nil || len(changedFiles) == 0 {
		return m, nil
	}
	selFile := m.graphTabModel.GetSelectedFile()
	if selFile < 0 || selFile >= len(changedFiles) {
		return m, nil
	}
	if m.repository == nil || m.GetSelectedCommit() < 0 || m.GetSelectedCommit() >= len(m.repository.Graph.Commits) {
		return m, nil
	}
	commit := m.repository.Graph.Commits[m.GetSelectedCommit()]
	if commit.Immutable {
		m.statusMessage = "Cannot move file: commit is immutable"
		return m, nil
	}
	file := changedFiles[selFile]
	commitID := commit.ChangeID
	// "Move to Parent" - creates a new commit BEFORE this one (toward main/root)
	m.statusMessage = fmt.Sprintf("Moving %s to new parent commit...", file.Path)
	return m, m.moveFileToParent(commitID, file.Path)
}

func (m *Model) handleMoveFileDown() (tea.Model, tea.Cmd) {
	if m.viewMode != ViewCommitGraph || m.graphFocused {
		return m, nil
	}
	changedFiles := m.graphTabModel.GetChangedFiles()
	if m.jjService == nil || len(changedFiles) == 0 {
		return m, nil
	}
	selFile := m.graphTabModel.GetSelectedFile()
	if selFile < 0 || selFile >= len(changedFiles) {
		return m, nil
	}
	if m.repository == nil || m.GetSelectedCommit() < 0 || m.GetSelectedCommit() >= len(m.repository.Graph.Commits) {
		return m, nil
	}
	commit := m.repository.Graph.Commits[m.GetSelectedCommit()]
	if commit.Immutable {
		m.statusMessage = "Cannot move file: commit is immutable"
		return m, nil
	}
	file := changedFiles[selFile]
	commitID := commit.ChangeID
	// "Move to Child" - creates a new commit AFTER this one (toward tips/branches)
	m.statusMessage = fmt.Sprintf("Moving %s to new child commit...", file.Path)
	return m, m.moveFileToChild(commitID, file.Path)
}

func (m *Model) handleRevertFile() (tea.Model, tea.Cmd) {
	if m.viewMode != ViewCommitGraph || m.graphFocused {
		return m, nil
	}
	changedFiles := m.graphTabModel.GetChangedFiles()
	if m.jjService == nil || len(changedFiles) == 0 {
		return m, nil
	}
	selFile := m.graphTabModel.GetSelectedFile()
	if selFile < 0 || selFile >= len(changedFiles) {
		return m, nil
	}
	if m.repository == nil || m.GetSelectedCommit() < 0 || m.GetSelectedCommit() >= len(m.repository.Graph.Commits) {
		return m, nil
	}
	commit := m.repository.Graph.Commits[m.GetSelectedCommit()]
	if commit.Immutable {
		m.statusMessage = "Cannot revert file: commit is immutable"
		return m, nil
	}
	file := changedFiles[selFile]
	commitID := commit.ChangeID
	m.statusMessage = fmt.Sprintf("Reverting changes to %s...", file.Path)
	return m, m.revertFile(commitID, file.Path)
}
