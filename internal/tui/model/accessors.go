package model

import (
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/github"
	"github.com/madicen/jj-tui/internal/tickets"
)

// Model accessor methods for testing and external access
// These methods provide controlled access to internal model state

// GetViewMode returns the current view mode
func (m *Model) GetViewMode() ViewMode {
	return m.viewMode
}

// GetSelectedCommit returns the selected commit index (from graph tab)
func (m *Model) GetSelectedCommit() int {
	return m.graphTabModel.GetSelectedCommit()
}

// GetStatusMessage returns the status message
func (m *Model) GetStatusMessage() string {
	return m.statusMessage
}

// GetRepository returns the repository
func (m *Model) GetRepository() *internal.Repository {
	return m.repository
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

// GetError returns the current error
func (m *Model) GetError() error {
	return m.err
}

// IsNotJJRepo returns whether we're in a non-jj repo state
func (m *Model) IsNotJJRepo() bool {
	return m.notJJRepo
}

// SetTicketService sets the ticket service for testing
func (m *Model) SetTicketService(svc tickets.Service) {
	m.ticketService = svc
}

// SetTicketList sets the ticket list for testing (updates tickets tab)
func (m *Model) SetTicketList(list []tickets.Ticket) {
	m.ticketsTabModel.UpdateTickets(list)
}

// SetGitHubService sets the GitHub service for testing and syncs to the PRs tab so it shows the PR list instead of "GitHub not connected".
func (m *Model) SetGitHubService(svc *github.Service) {
	m.githubService = svc
	m.prsTabModel.SetGithubService(m.isGitHubAvailable())
}

// SetViewMode sets the view mode for testing
func (m *Model) SetViewMode(mode ViewMode) {
	m.viewMode = mode
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
	m.loading = loading
}

// SetDimensions sets width and height for testing
func (m *Model) SetDimensions(width, height int) {
	m.width = width
	m.height = height
}

// Close releases resources
func (m *Model) Close() {
	if m.zoneManager != nil {
		m.zoneManager.Close()
	}
}
