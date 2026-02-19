package bookmark

import tea "github.com/charmbracelet/bubbletea"

// BookmarkCreatedMsg indicates bookmark was created/moved.
type BookmarkCreatedMsg struct {
	BookmarkName string
	CommitID     string
	WasMoved     bool
	TicketKey    string // set when creating from a ticket (for auto-transition)
}

// BookmarkDeletedMsg indicates bookmark was deleted.
type BookmarkDeletedMsg struct {
	BookmarkName string
}

// CancelRequestedMsg is sent when the user cancels (esc); main forwards to modal which responds with PerformCancelCmd.
type CancelRequestedMsg struct{}

// SubmitRequestedMsg is sent when the user submits (enter/ctrl+s); main forwards to modal which responds with PerformSubmitCmd.
type SubmitRequestedMsg struct{}

// CancelRequestedCmd returns a command that sends CancelRequestedMsg.
func CancelRequestedCmd() tea.Cmd {
	return func() tea.Msg { return CancelRequestedMsg{} }
}

// SubmitRequestedCmd returns a command that sends SubmitRequestedMsg.
func SubmitRequestedCmd() tea.Cmd {
	return func() tea.Msg { return SubmitRequestedMsg{} }
}

// PerformCancelMsg is sent by the modal so main leaves bookmark view and sets status.
type PerformCancelMsg struct{}

// PerformSubmitMsg is sent by the modal so main runs submitBookmark (create/move bookmark).
type PerformSubmitMsg struct{}

// PerformCancelCmd returns a command that sends PerformCancelMsg.
func PerformCancelCmd() tea.Cmd {
	return func() tea.Msg { return PerformCancelMsg{} }
}

// PerformSubmitCmd returns a command that sends PerformSubmitMsg.
func PerformSubmitCmd() tea.Cmd {
	return func() tea.Msg { return PerformSubmitMsg{} }
}
