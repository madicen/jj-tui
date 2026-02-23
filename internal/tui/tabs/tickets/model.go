package tickets

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tickets"
	"github.com/madicen/jj-tui/internal/tui/mouse"
)

// Model represents the state of the Tickets tab
type Model struct {
	zoneManager          *zone.Manager
	ticketList           []tickets.Ticket
	selectedTicket       int
	listYOffset          int // Scroll offset for list (details stay fixed)
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
	// scrollToSelectedTicket: when true, next render will adjust listYOffset to keep selection in view (key/click only; mouse scroll can move selection off screen)
	scrollToSelectedTicket bool
}

// NewModel creates a new Tickets tab model. zoneManager may be nil (e.g. in tests).
// Default dimensions (80x24) ensure wheel scroll works before first View()/SetDimensions, same as Graph viewports.
func NewModel(zoneManager *zone.Manager) Model {
	return Model{
		zoneManager:   zoneManager,
		selectedTicket: -1,
		loading:        false,
		width:          80,
		height:         24,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// SetDimensions sets the content area size (used for list-only scrolling)
func (m *Model) SetDimensions(width, height int) {
	m.width = width
	m.height = height
}

// Update handles messages for the Tickets tab
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Do not use full window size: we get content-area dimensions from the main model via SetDimensions()
		// so the list uses the correct height (below header, above status bar), same as the Graph tab.
		return m, nil
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case zone.MsgZoneInBounds:
		return m.handleZoneClick(msg.Zone)
	case tea.MouseMsg:
		// Wheel: IsWheel() + raw X11 fallback so we accept any terminal encoding; scroll without requiring list to be clicked first
		isWheel := tea.MouseEvent(msg).IsWheel() || msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown
		if isWheel {
			isUp := msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelLeft
			if isUp {
				m.listYOffset -= 3
				if m.listYOffset < 0 {
					m.listYOffset = 0
				}
			} else {
				m.listYOffset += 3
			}
			return m, nil
		}
	}
	return m, nil
}

// View renders the Tickets tab (pointer receiver so render can persist listYOffset clamp)
func (m *Model) View() string {
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
	case "c":
		return m, Request{ToggleStatusChangeMode: true}.Cmd()
	case "i", "D", "B", "N":
		if m.statusChangeMode && !m.transitionInProgress && m.selectedTicket >= 0 && m.selectedTicket < len(m.ticketList) {
			if id := m.transitionIDByKey(msg.String()); id != "" {
				return m, Request{TransitionID: id}.Cmd()
			}
		}
		return m, nil
	case "o":
		return m, Request{OpenInBrowser: true}.Cmd()
	case "enter", "e":
		if m.selectedTicket >= 0 && m.selectedTicket < len(m.ticketList) {
			return m, Request{StartBookmarkFromTicket: true}.Cmd()
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) transitionIDByKey(key string) string {
	for _, t := range m.availableTransitions {
		lower := strings.ToLower(t.Name)
		switch key {
		case "i":
			if strings.Contains(lower, "progress") {
				return t.ID
			}
			if strings.Contains(lower, "start") && !strings.Contains(lower, "not start") && !strings.Contains(lower, "not_start") {
				return t.ID
			}
		case "D":
			if strings.Contains(lower, "done") || strings.Contains(lower, "complete") || strings.Contains(lower, "resolve") {
				return t.ID
			}
		case "B":
			if strings.Contains(lower, "block") {
				return t.ID
			}
		case "N":
			if strings.Contains(lower, "not") && strings.Contains(lower, "start") {
				return t.ID
			}
		}
	}
	return ""
}

// handleZoneClick handles zone clicks; returns a request cmd for actions.
func (m Model) handleZoneClick(z *zone.ZoneInfo) (Model, tea.Cmd) {
	if m.zoneManager == nil || z == nil {
		return m, nil
	}
	for i := range m.ticketList {
		if m.zoneManager.Get(mouse.ZoneJiraTicket(i)) == z {
			m.selectedTicket = i
			m.scrollToSelectedTicket = true
			return m, nil
		}
	}
	if m.zoneManager.Get(mouse.ZoneJiraCreateBranch) == z {
		return m, Request{StartBookmarkFromTicket: true}.Cmd()
	}
	if m.zoneManager.Get(mouse.ZoneJiraChangeStatus) == z {
		return m, Request{ToggleStatusChangeMode: true}.Cmd()
	}
	if m.zoneManager.Get(mouse.ZoneTicketOpenBrowser) == z {
		return m, Request{OpenInBrowser: true}.Cmd()
	}
	if m.statusChangeMode && !m.transitionInProgress && m.selectedTicket >= 0 && m.selectedTicket < len(m.ticketList) {
		for i, t := range m.availableTransitions {
			zoneID := mouse.ZoneJiraTransition + fmt.Sprintf("%d", i)
			if m.zoneManager.Get(zoneID) == z {
				return m, Request{TransitionID: t.ID}.Cmd()
			}
		}
	}
	return m, nil
}

// Accessors

// GetSelectedTicket returns the index of the selected ticket
func (m *Model) GetSelectedTicket() int {
	return m.selectedTicket
}

// GetListYOffset returns the list scroll offset (for tests and accessors)
func (m *Model) GetListYOffset() int {
	return m.listYOffset
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
