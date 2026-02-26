package prform

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
)

// PRCreatedMsg indicates a PR was created.
type PRCreatedMsg struct {
	PR *internal.GitHubPR
}

// CancelRequestedMsg is sent when the user cancels (esc); main forwards to modal which responds with PerformCancelCmd.
type CancelRequestedMsg struct{}

// SubmitRequestedMsg is sent when the user submits (ctrl+s); main forwards to modal which responds with PerformSubmitCmd.
type SubmitRequestedMsg struct{}

// CancelRequestedCmd returns a command that sends CancelRequestedMsg.
func CancelRequestedCmd() tea.Cmd {
	return func() tea.Msg { return CancelRequestedMsg{} }
}

// SubmitRequestedCmd returns a command that sends SubmitRequestedMsg.
func SubmitRequestedCmd() tea.Cmd {
	return func() tea.Msg { return SubmitRequestedMsg{} }
}

// PerformCancelMsg is sent by the modal so main leaves PR form view and sets status.
type PerformCancelMsg struct{}

// PerformSubmitMsg is sent by the modal so main runs submitPR (create PR).
type PerformSubmitMsg struct{}

// PerformCancelCmd returns a command that sends PerformCancelMsg.
func PerformCancelCmd() tea.Cmd {
	return func() tea.Msg { return PerformCancelMsg{} }
}

// PerformSubmitCmd returns a command that sends PerformSubmitMsg.
func PerformSubmitCmd() tea.Cmd {
	return func() tea.Msg { return PerformSubmitMsg{} }
}
