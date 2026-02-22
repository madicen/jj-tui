package tickets

import (
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tickets"
)

// Model represents the state of the Tickets tab
type Model struct {
	zoneManager          *zone.Manager
	ticketList           []tickets.Ticket
	selectedTicket       int
	availableTransitions []tickets.Transition
	transitionInProgress bool
	statusChangeMode     bool
	loadingTransitions   bool
	loading              bool
	err                  error
	statusMessage        string
	width                int
	height               int
	providerName         string // e.g. "Jira", "Codecks"
	jiraService          bool   // whether a ticket service is connected
}

// NewModel creates a new Tickets tab model. zoneManager may be nil (e.g. in tests).
func NewModel(zoneManager *zone.Manager) Model {
	return Model{
		zoneManager:   zoneManager,
		selectedTicket: -1,
		loading:        false,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the Tickets tab
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}
	return m, nil
}

// View renders the Tickets tab
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	return m.renderTickets()
}

// SetTicketServiceInfo sets provider name and whether a ticket service is connected (used by main model)
func (m *Model) SetTicketServiceInfo(providerName string, connected bool) {
	m.providerName = providerName
	m.jiraService = connected
}

// SetAvailableTransitions sets the available status transitions (called by main model when loaded)
func (m *Model) SetAvailableTransitions(t []tickets.Transition) {
	m.availableTransitions = t
}

// SetTransitionInProgress sets whether a transition is in progress (called by main model)
func (m *Model) SetTransitionInProgress(inProgress bool) {
	m.transitionInProgress = inProgress
}

// SetStatusChangeMode sets whether status change buttons are expanded (called by main model)
func (m *Model) SetStatusChangeMode(mode bool) {
	m.statusChangeMode = mode
}

// handleKeyMsg handles keyboard input specific to the Tickets tab
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.selectedTicket < len(m.ticketList)-1 {
			m.selectedTicket++
		}
		return m, nil
	case "k", "up":
		if m.selectedTicket > 0 {
			m.selectedTicket--
		}
		return m, nil
	case "esc":
		if m.statusChangeMode {
			m.statusChangeMode = false
		}
		return m, nil
	}
	return m, nil
}

// Accessors

// GetSelectedTicket returns the index of the selected ticket
func (m *Model) GetSelectedTicket() int {
	return m.selectedTicket
}

// SetSelectedTicket sets the selected ticket index
func (m *Model) SetSelectedTicket(idx int) {
	if idx >= 0 && idx < len(m.ticketList) {
		m.selectedTicket = idx
	}
}

// GetTickets returns the ticket list
func (m *Model) GetTickets() []tickets.Ticket {
	return m.ticketList
}

// UpdateTickets updates the ticket list
func (m *Model) UpdateTickets(ticketList []tickets.Ticket) {
	m.ticketList = ticketList
	if len(ticketList) > 0 && m.selectedTicket < 0 {
		m.selectedTicket = 0
	}
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
	// Repos may be updated but tickets are loaded separately
	// This is a no-op for tickets but required for interface consistency
}

// GetAvailableTransitions returns available transitions
func (m *Model) GetAvailableTransitions() []tickets.Transition {
	return m.availableTransitions
}

// IsStatusChangeMode returns whether we're in status change mode
func (m *Model) IsStatusChangeMode() bool {
	return m.statusChangeMode
}
