package githublogin

import tea "github.com/charmbracelet/bubbletea"

// PerformCancelMsg is sent when the user cancels the GitHub login (Esc).
// Main clears the login state and returns to settings view.
type PerformCancelMsg struct{}

// PerformCancelCmd returns a command that sends PerformCancelMsg.
func PerformCancelCmd() tea.Cmd {
	return func() tea.Msg { return PerformCancelMsg{} }
}
