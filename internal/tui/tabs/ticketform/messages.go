package ticketform

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tickets"
)

// TicketCreatedMsg is sent when a ticket was successfully created.
type TicketCreatedMsg struct {
	Ticket *tickets.Ticket
}

// CancelRequestedMsg is sent when the user cancels (Esc).
type CancelRequestedMsg struct{}

// SubmitRequestedMsg is sent when the user submits (Ctrl+S).
type SubmitRequestedMsg struct{}

// CancelRequestedCmd returns a command that sends CancelRequestedMsg.
func CancelRequestedCmd() tea.Cmd {
	return func() tea.Msg { return CancelRequestedMsg{} }
}

// SubmitRequestedCmd returns a command that sends SubmitRequestedMsg.
func SubmitRequestedCmd() tea.Cmd {
	return func() tea.Msg { return SubmitRequestedMsg{} }
}
