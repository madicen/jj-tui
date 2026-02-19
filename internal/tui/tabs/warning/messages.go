package warning

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
)

// PerformCancelMsg is sent when the user cancels the warning (esc); main sets status to "Cancelled".
type PerformCancelMsg struct{}

// PerformCancelCmd returns a command that sends PerformCancelMsg.
func PerformCancelCmd() tea.Cmd {
	return func() tea.Msg { return PerformCancelMsg{} }
}

// EditCommitRequestedMsg is sent when the user presses Enter. Main forwards to modal; modal responds with PerformEditCommitCmd.
type EditCommitRequestedMsg struct {
	Commit internal.Commit
}

// PerformEditCommitMsg is sent by the modal so main navigates to the commit and starts edit (modal already hid itself).
type PerformEditCommitMsg struct {
	Commit internal.Commit
}

// EditCommitRequestedCmd returns a command that sends EditCommitRequestedMsg for the given commit.
func EditCommitRequestedCmd(commit internal.Commit) tea.Cmd {
	return func() tea.Msg { return EditCommitRequestedMsg{Commit: commit} }
}

// PerformEditCommitCmd returns a command that sends PerformEditCommitMsg.
func PerformEditCommitCmd(commit internal.Commit) tea.Cmd {
	return func() tea.Msg { return PerformEditCommitMsg{Commit: commit} }
}
