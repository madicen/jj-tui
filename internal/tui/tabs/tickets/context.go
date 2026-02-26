package tickets

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tickets"
	"github.com/madicen/jj-tui/internal/tui/state"
)

// ContextProvider is implemented by the main model so the Tickets tab can build context without depending on model package.
type ContextProvider interface {
	GetTickets() []tickets.Ticket
	GetSelectedTicket() int
	GetAvailableTransitions() []tickets.Transition
	GetTransitionInProgress() bool
	GetTicketService() tickets.Service
	GetIsStatusChangeMode() bool
}

// BuildRequestContextFromApp builds RequestContext from app state and the tickets tab model (for UpdateWithApp flow).
func BuildRequestContextFromApp(app *state.AppState, m *Model) *RequestContext {
	if app == nil || m == nil {
		return nil
	}
	return BuildRequestContext(&ContextInput{
		TicketList:           m.GetTickets(),
		SelectedTicket:       m.GetSelectedTicket(),
		AvailableTransitions: m.GetAvailableTransitions(),
		TransitionInProgress: m.GetTransitionInProgress(),
		TicketService:        app.TicketService,
		IsStatusChangeMode:   m.IsStatusChangeMode(),
	})
}

// BuildRequestContextFrom builds RequestContext from a provider (e.g. main model).
func BuildRequestContextFrom(p ContextProvider) *RequestContext {
	if p == nil {
		return nil
	}
	return BuildRequestContext(&ContextInput{
		TicketList:           p.GetTickets(),
		SelectedTicket:       p.GetSelectedTicket(),
		AvailableTransitions: p.GetAvailableTransitions(),
		TransitionInProgress: p.GetTransitionInProgress(),
		TicketService:        p.GetTicketService(),
		IsStatusChangeMode:   p.GetIsStatusChangeMode(),
	})
}

// EnterTabProvider is implemented by main for EnterTab (status + load cmd).
type EnterTabProvider interface {
	GetTicketService() tickets.Service
	IsDemoMode() bool
}

// EnterTab returns status message and optional load command when navigating to the Tickets tab.
func EnterTab(p EnterTabProvider) (status string, cmd tea.Cmd) {
	hasService := p != nil && p.GetTicketService() != nil
	status = EnterTabStatus(hasService)
	if !hasService {
		return status, nil
	}
	return status, LoadTicketsCmd(p.GetTicketService(), p.IsDemoMode())
}

// RequestContext is passed from the main model so the Tickets tab can validate
// and execute requests.
type RequestContext struct {
	TicketList           []tickets.Ticket
	SelectedTicket       int
	AvailableTransitions []tickets.Transition
	TransitionInProgress bool
	TicketService        tickets.Service // for GetTicketURL; can be nil
	IsStatusChangeMode   bool            // current status-change expansion; used to set ToggleModeStatus when NeedToggleMode
}

// ContextInput is the data needed to build a RequestContext. Main passes this from its state.
type ContextInput struct {
	TicketList           []tickets.Ticket
	SelectedTicket       int
	AvailableTransitions []tickets.Transition
	TransitionInProgress bool
	TicketService        tickets.Service
	IsStatusChangeMode   bool
}

// BuildRequestContext builds RequestContext from input. The Tickets tab owns what context it needs.
func BuildRequestContext(input *ContextInput) *RequestContext {
	if input == nil {
		return nil
	}
	return &RequestContext{
		TicketList:           input.TicketList,
		SelectedTicket:       input.SelectedTicket,
		AvailableTransitions: input.AvailableTransitions,
		TransitionInProgress: input.TransitionInProgress,
		TicketService:        input.TicketService,
		IsStatusChangeMode:   input.IsStatusChangeMode,
	}
}

// EnterTabStatus returns the status message when navigating to the Tickets tab.
func EnterTabStatus(hasService bool) string {
	if hasService {
		return "Loading tickets..."
	}
	return ""
}

// SelectedTicketValid returns true if SelectedTicket is in range.
func (c *RequestContext) SelectedTicketValid() bool {
	return c.SelectedTicket >= 0 && c.SelectedTicket < len(c.TicketList)
}

// SelectedTicketData returns the selected ticket or nil.
func (c *RequestContext) SelectedTicketData() *tickets.Ticket {
	if !c.SelectedTicketValid() {
		return nil
	}
	t := c.TicketList[c.SelectedTicket]
	return &t
}

// TransitionName returns the name for the given transition ID.
func (c *RequestContext) TransitionName(transitionID string) string {
	for _, t := range c.AvailableTransitions {
		if t.ID == transitionID {
			return t.Name
		}
	}
	return ""
}
