package initrepo

import tea "github.com/charmbracelet/bubbletea"

// Request messages sent to main when user acts on the init-repo screen.

// RequestDismissMsg is sent on Esc; main clears init-repo screen and returns to graph.
type RequestDismissMsg struct{}

// RequestInitMsg is sent on "i" or init button click; main runs jj init.
type RequestInitMsg struct{}

// RequestDismissCmd returns a command that sends RequestDismissMsg.
func RequestDismissCmd() tea.Cmd {
	return func() tea.Msg { return RequestDismissMsg{} }
}

// RequestInitCmd returns a command that sends RequestInitMsg.
func RequestInitCmd() tea.Cmd {
	return func() tea.Msg { return RequestInitMsg{} }
}
