package model

import (
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/github"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tickets"
	"github.com/madicen/jj-tui/internal/tui/state"
	helptab "github.com/madicen/jj-tui/internal/tui/tabs/help"
	"github.com/madicen/jj-tui/internal/tui/util"
)

// Model accessor methods for testing and external access
// These methods provide controlled access to internal model state

// GetViewMode returns the current view mode
func (m *Model) GetViewMode() state.ViewMode {
	return m.appState.ViewMode
}

// GetSelectedCommit returns the selected commit index (from graph tab)
func (m *Model) GetSelectedCommit() int {
	return m.graphTabModel.GetSelectedCommit()
}

// GetStatusMessage returns the status message
func (m *Model) GetStatusMessage() string {
	return m.appState.StatusMessage
}

// GetRepository returns the repository
func (m *Model) GetRepository() *internal.Repository {
	return m.appState.Repository
}

// GetSelectedPR returns the selected PR index (from PR tab)
func (m *Model) GetSelectedPR() int {
	return m.prsTabModel.GetSelectedPR()
}

// GetSelectedTicket returns the selected ticket index (from tickets tab)
func (m *Model) GetSelectedTicket() int {
	return m.ticketsTabModel.GetSelectedTicket()
}

// GetSelectedBranch returns the selected branch index (from branches tab)
func (m *Model) GetSelectedBranch() int {
	return m.branchesTabModel.GetSelectedBranch()
}

// GetBranches returns the branch list (for tabs that need context).
func (m *Model) GetBranches() []internal.Branch {
	return m.branchesTabModel.GetBranches()
}

// GetJJService returns the jj service (for tabs that need context).
func (m *Model) GetJJService() *jj.Service {
	return m.appState.JJService
}

// GetRebaseSourceCommit returns the graph tab's rebase source commit index.
func (m *Model) GetRebaseSourceCommit() int {
	return m.graphTabModel.GetRebaseSourceCommit()
}

// GetChangedFiles returns the graph tab's changed files list.
func (m *Model) GetChangedFiles() []jj.ChangedFile {
	return m.graphTabModel.GetChangedFiles()
}

// GetChangedFilesCommitID returns the graph tab's changed-files commit ID.
func (m *Model) GetChangedFilesCommitID() string {
	return m.graphTabModel.GetChangedFilesCommitID()
}

// GetSelectedFile returns the graph tab's selected file index.
func (m *Model) GetSelectedFile() int {
	return m.graphTabModel.GetSelectedFile()
}

// IsGraphFocused returns whether the graph (not files) pane is focused.
func (m *Model) IsGraphFocused() bool {
	return m.graphTabModel.IsGraphFocused()
}

// IsGitHubAvailable returns whether GitHub functionality is available (for tab context providers).
func (m *Model) IsGitHubAvailable() bool {
	return m.isGitHubAvailable()
}

// GetCreatePRBranch returns the branch that would be used for Create PR for the selected commit (for graph ContextProvider).
func (m *Model) GetCreatePRBranch() string {
	return m.graphTabModel.GetCreatePRBranch()
}

// IsDemoMode returns whether the app is in demo mode (for tab context providers).
func (m *Model) IsDemoMode() bool {
	return m.appState.DemoMode
}

// GetGitHubService returns the GitHub service (for tab context providers).
func (m *Model) GetGitHubService() *github.Service {
	return m.appState.GitHubService
}

// GetGitHubInfo returns the GitHub diagnostic info (for tab context providers).
func (m *Model) GetGitHubInfo() string {
	return m.appState.GithubInfo
}

// GetBranchLimit returns the configured branch limit (for Branches tab EnterTab).
func (m *Model) GetBranchLimit() int {
	return m.settingsTabModel.GetSettingsBranchLimit()
}

// GetTickets returns the tickets list (for tab context providers).
func (m *Model) GetTickets() []tickets.Ticket {
	return m.ticketsTabModel.GetTickets()
}

// GetAvailableTransitions returns the tickets tab's available transitions.
func (m *Model) GetAvailableTransitions() []tickets.Transition {
	return m.ticketsTabModel.GetAvailableTransitions()
}

// GetTransitionInProgress returns whether a ticket transition is in progress.
func (m *Model) GetTransitionInProgress() bool {
	return m.ticketsTabModel.GetTransitionInProgress()
}

// GetTicketService returns the ticket service (for tab context providers).
func (m *Model) GetTicketService() tickets.Service {
	if m.appState.TicketService == nil {
		return nil
	}
	if util.IsNilInterface(m.appState.TicketService) {
		return nil
	}
	return m.appState.TicketService
}

// GetIsStatusChangeMode returns whether the tickets tab is in status-change mode.
func (m *Model) GetIsStatusChangeMode() bool {
	return m.ticketsTabModel.IsStatusChangeMode()
}

// GetPRsListYOffset returns the PR list scroll offset (for tests)
func (m *Model) GetPRsListYOffset() int {
	return m.prsTabModel.GetListYOffset()
}

// GetTicketsListYOffset returns the tickets list scroll offset (for tests)
func (m *Model) GetTicketsListYOffset() int {
	return m.ticketsTabModel.GetListYOffset()
}

// GetBranchesListYOffset returns the branches list scroll offset (for tests)
func (m *Model) GetBranchesListYOffset() int {
	return m.branchesTabModel.GetListYOffset()
}

// GetSettingsFocusedField returns the focused settings field index
func (m *Model) GetSettingsFocusedField() int {
	return m.settingsTabModel.GetFocusedField()
}

// GetSettingsTab returns the active settings sub-tab index (0–5: GitHub, Jira, Codecks, Tickets, Branches, Advanced).
func (m *Model) GetSettingsTab() int {
	return m.settingsTabModel.GetSettingsTab()
}

// SetSettingsTab sets the active settings sub-tab (for testing).
func (m *Model) SetSettingsTab(tab int) {
	m.settingsTabModel.SetSettingsTab(tab)
}

// GetSettingsGraphRevset returns the graph revset string from the Advanced settings panel.
func (m *Model) GetSettingsGraphRevset() string {
	return m.settingsTabModel.GetAdvancedModel().GetGraphRevset()
}

// SetSettingsGraphRevset sets the graph revset string in the Advanced settings panel (for testing).
func (m *Model) SetSettingsGraphRevset(revset string) {
	m.settingsTabModel.GetAdvancedModel().SetGraphRevset(revset)
}

// GetSettingsJiraProject returns the Jira "project for new issues" from the settings panel (for testing).
func (m *Model) GetSettingsJiraProject() string {
	return m.settingsTabModel.GetJiraModel().GetProject()
}

// GetSettingsJiraProjectFilter returns the Jira "project filter" from the settings panel (for testing).
func (m *Model) GetSettingsJiraProjectFilter() string {
	return m.settingsTabModel.GetJiraModel().GetProjectFilter()
}

// GetError returns the current error
func (m *Model) GetError() error {
	return m.errorModal.GetError()
}

// IsNotJJRepo returns whether we're in a non-jj repo state
func (m *Model) IsNotJJRepo() bool {
	return m.initRepoModel.Path() != ""
}

// SetTicketService sets the ticket service for testing
func (m *Model) SetTicketService(svc tickets.Service) {
	m.appState.TicketService = svc
}

// SetTicketList sets the ticket list for testing (updates tickets tab)
func (m *Model) SetTicketList(list []tickets.Ticket) {
	m.ticketsTabModel.UpdateTickets(list)
}

// SetGitHubService sets the GitHub service for testing and syncs to the PRs tab so it shows the PR list instead of "GitHub not connected".
func (m *Model) SetGitHubService(svc *github.Service) {
	m.appState.GitHubService = svc
	m.prsTabModel.SetGithubService(m.isGitHubAvailable())
}

// SetViewMode sets the view mode for testing
func (m *Model) SetViewMode(mode state.ViewMode) {
	m.appState.ViewMode = mode
	if mode == state.ViewHelp {
		m.helpTabModel.SetCommandHistoryEntries(helptab.BuildCommandHistoryEntries(m.appState.JJService))
	}
}

// SetSelectedPR sets the selected PR index for testing
func (m *Model) SetSelectedPR(idx int) {
	m.prsTabModel.SetSelectedPR(idx)
}

// SetSelectedTicket sets the selected ticket index for testing
func (m *Model) SetSelectedTicket(idx int) {
	m.ticketsTabModel.SetSelectedTicket(idx)
}

// SetLoading sets the loading state for testing
func (m *Model) SetLoading(loading bool) {
	m.appState.Loading = loading
}

// SetDimensions sets width and height for testing
func (m *Model) SetDimensions(width, height int) {
	m.width = width
	m.height = height
}

// commitIdxForChangeID returns the index of the commit with the given change ID, or -1.
func commitIdxForChangeID(repo *internal.Repository, changeID string) int {
	if repo == nil || changeID == "" {
		return -1
	}
	for i, c := range repo.Graph.Commits {
		if c.ChangeID == changeID || c.ID == changeID {
			return i
		}
	}
	return -1
}

// Close releases resources
func (m *Model) Close() {
	if m.zoneManager != nil {
		m.zoneManager.Close()
	}
}
