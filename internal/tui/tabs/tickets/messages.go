package tickets

import (
	tea "github.com/charmbracelet/bubbletea"
	ticketdomain "github.com/madicen/jj-tui/internal/tickets"
)

// TicketsLoadedMsg is sent when tickets are loaded from the ticket service.
type TicketsLoadedMsg struct {
	Tickets []ticketdomain.Ticket
}

// TransitionsLoadedMsg is sent when available transitions are loaded for a ticket.
type TransitionsLoadedMsg struct {
	Transitions []ticketdomain.Transition
}

// TransitionCompletedMsg is sent when a ticket status transition completes.
type TransitionCompletedMsg struct {
	TicketKey string
	NewStatus string
	Err       error
}

// LoadErrorMsg is sent when loading tickets fails (main shows error modal).
type LoadErrorMsg struct {
	Err error
}

// Request is sent to the main model to run ticket actions (main has ticketService, jjService, etc.).
type Request struct {
	OpenInBrowser          bool
	ToggleStatusChangeMode bool
	StartBookmarkFromTicket bool
	TransitionID           string // When set, main runs transitionTicket(TransitionID)
}

// Cmd returns a tea.Cmd that sends this request.
func (r Request) Cmd() tea.Cmd {
	return func() tea.Msg { return r }
}

// Effect types: the tab sends these to the main model so it can update app state and status.

// ApplyTicketsLoadedEffect tells main to set status and start loading transitions.
type ApplyTicketsLoadedEffect struct {
	StatusMessage string
}

// ApplyTransitionsLoadedEffect is a no-op effect; main just applied the tab update. Used for consistency.
type ApplyTransitionsLoadedEffect struct{}

// ApplyTransitionCompletedEffect tells main to set status/error and optionally reload tickets.
type ApplyTransitionCompletedEffect struct {
	Err           error
	StatusMessage string
	ReloadTickets bool
}

// ApplyTicketsLoadErrorEffect tells main to show the load error in the error modal.
type ApplyTicketsLoadErrorEffect struct {
	Err error
}

func (e ApplyTicketsLoadedEffect) Cmd() tea.Cmd {
	return func() tea.Msg { return e }
}

func (e ApplyTransitionsLoadedEffect) Cmd() tea.Cmd {
	return func() tea.Msg { return e }
}

func (e ApplyTransitionCompletedEffect) Cmd() tea.Cmd {
	return func() tea.Msg { return e }
}

func (e ApplyTicketsLoadErrorEffect) Cmd() tea.Cmd {
	return func() tea.Msg { return e }
}

// TicketsLoadedInput is the context main sends when forwarding TicketsLoadedMsg.
type TicketsLoadedInput struct {
	Tickets      []ticketdomain.Ticket
	ProviderName string
	HasService   bool
}

// OpenURLEffect tells main to open a URL in the browser.
type OpenURLEffect struct {
	URL string
}

// ToggleModeEffect tells main to toggle status change mode and set status.
type ToggleModeEffect struct {
	Status string
}

// OpenURLEffectCmd returns a cmd that sends OpenURLEffect to main.
func OpenURLEffectCmd(url string) tea.Cmd {
	return func() tea.Msg { return OpenURLEffect{URL: url} }
}

// ToggleModeEffectCmd returns a cmd that sends ToggleModeEffect to main.
func ToggleModeEffectCmd(status string) tea.Cmd {
	return func() tea.Msg { return ToggleModeEffect{Status: status} }
}

// OpenCreateBookmarkFromTicketEffect tells main to open the bookmark modal to create a branch from main using the ticket key.
type OpenCreateBookmarkFromTicketEffect struct {
	TicketKey   string
	Title       string
	DisplayKey  string
}

// Cmd returns a tea.Cmd that sends this effect to main.
func (e OpenCreateBookmarkFromTicketEffect) Cmd() tea.Cmd {
	return func() tea.Msg { return e }
}
