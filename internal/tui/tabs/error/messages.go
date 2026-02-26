package error

import tea "github.com/charmbracelet/bubbletea"

// Request messages sent by the error modal so main can clear, set view/status, and run the right cmd.
// (Init-repo screen uses initrepo package messages.)

// RequestDismissMsg is sent on esc; main clears error, sets view, sets status, runs tick.
type RequestDismissMsg struct{}

// RequestRefreshMsg is sent on ctrl+r; main clears error, sets view, runs refresh.
type RequestRefreshMsg struct{}

// RequestCopyMsg is sent on c; main runs copy (reads error from modal) and sets copied.
type RequestCopyMsg struct{}

// RequestDismissCmd returns a command that sends RequestDismissMsg.
func RequestDismissCmd() tea.Cmd {
	return func() tea.Msg { return RequestDismissMsg{} }
}

// RequestRefreshCmd returns a command that sends RequestRefreshMsg.
func RequestRefreshCmd() tea.Cmd {
	return func() tea.Msg { return RequestRefreshMsg{} }
}

// RequestCopyCmd returns a command that sends RequestCopyMsg.
func RequestCopyCmd() tea.Cmd {
	return func() tea.Msg { return RequestCopyMsg{} }
}
