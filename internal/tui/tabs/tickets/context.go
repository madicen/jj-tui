package tickets

import "github.com/madicen/jj-tui/internal/tickets"

// RequestContext is passed from the main model so the Tickets tab can validate
// and execute requests.
type RequestContext struct {
	TicketList           []tickets.Ticket
	SelectedTicket       int
	AvailableTransitions []tickets.Transition
	TransitionInProgress bool
	TicketService        tickets.Service // for GetTicketURL; can be nil
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
