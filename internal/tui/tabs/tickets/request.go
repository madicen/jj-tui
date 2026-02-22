package tickets

import tea "github.com/charmbracelet/bubbletea"

// Request is sent to the main model to run ticket actions (main has ticketService, jjService, etc.).
type Request struct {
	OpenInBrowser           bool
	ToggleStatusChangeMode  bool
	StartBookmarkFromTicket bool
	TransitionID            string // When set, main runs transitionTicket(TransitionID)
}

// Cmd returns a tea.Cmd that sends this request.
func (r Request) Cmd() tea.Cmd {
	return func() tea.Msg { return r }
}
