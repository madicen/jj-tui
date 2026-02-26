package descedit

import tea "github.com/charmbracelet/bubbletea"

// DescriptionLoadedMsg contains loaded description.
type DescriptionLoadedMsg struct {
	CommitID    string
	Description string
}

// DescriptionSavedMsg indicates description was saved.
type DescriptionSavedMsg struct {
	CommitID string
}

// SaveRequestedMsg is sent when the user requests save (e.g. ctrl+s); main forwards to modal which responds with PerformSaveCmd.
type SaveRequestedMsg struct{}

// CancelRequestedMsg is sent when the user requests cancel (e.g. esc); main forwards to modal which responds with PerformCancelCmd.
type CancelRequestedMsg struct{}

// SaveRequestedCmd returns a command that sends SaveRequestedMsg.
func SaveRequestedCmd() tea.Cmd {
	return func() tea.Msg { return SaveRequestedMsg{} }
}

// CancelRequestedCmd returns a command that sends CancelRequestedMsg.
func CancelRequestedCmd() tea.Cmd {
	return func() tea.Msg { return CancelRequestedMsg{} }
}

// PerformSaveMsg is sent by the modal so main runs SaveDescriptionCmd(CommitID, Description).
type PerformSaveMsg struct {
	CommitID    string
	Description string
}

// PerformCancelMsg is sent when the user cancels; main leaves edit view and sets status.
type PerformCancelMsg struct{}

// PerformSaveCmd returns a command that sends PerformSaveMsg.
func PerformSaveCmd(commitID, description string) tea.Cmd {
	return func() tea.Msg { return PerformSaveMsg{CommitID: commitID, Description: description} }
}

// PerformCancelCmd returns a command that sends PerformCancelMsg.
func PerformCancelCmd() tea.Cmd {
	return func() tea.Msg { return PerformCancelMsg{} }
}
