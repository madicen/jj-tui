package tui

import (
	"github.com/madicen/jj-tui/internal/github"
	"github.com/madicen/jj-tui/internal/models"
	"github.com/madicen/jj-tui/internal/tickets"
)

// Model accessor methods for testing and external access
// These methods provide controlled access to internal model state

// GetViewMode returns the current view mode
func (m *Model) GetViewMode() ViewMode {
	return m.viewMode
}

// GetSelectedCommit returns the selected commit index
func (m *Model) GetSelectedCommit() int {
	return m.selectedCommit
}

// GetStatusMessage returns the status message
func (m *Model) GetStatusMessage() string {
	return m.statusMessage
}

// GetRepository returns the repository
func (m *Model) GetRepository() *models.Repository {
	return m.repository
}

// GetSelectedPR returns the selected PR index
func (m *Model) GetSelectedPR() int {
	return m.selectedPR
}

// GetSelectedTicket returns the selected ticket index
func (m *Model) GetSelectedTicket() int {
	return m.selectedTicket
}

// GetSettingsFocusedField returns the focused settings field index
func (m *Model) GetSettingsFocusedField() int {
	return m.settingsFocusedField
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

// SetTicketList sets the ticket list for testing
func (m *Model) SetTicketList(list []tickets.Ticket) {
	m.ticketList = list
}

// SetGitHubService sets the GitHub service for testing
func (m *Model) SetGitHubService(svc *github.Service) {
	m.githubService = svc
}

// SetViewMode sets the view mode for testing
func (m *Model) SetViewMode(mode ViewMode) {
	m.viewMode = mode
}

// SetSelectedPR sets the selected PR index for testing
func (m *Model) SetSelectedPR(idx int) {
	m.selectedPR = idx
}

// SetSelectedTicket sets the selected ticket index for testing
func (m *Model) SetSelectedTicket(idx int) {
	m.selectedTicket = idx
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
	if m.zone != nil {
		m.zone.Close()
	}
}

