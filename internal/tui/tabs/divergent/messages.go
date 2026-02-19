package divergent

import tea "github.com/charmbracelet/bubbletea"

// PerformCancelMsg is sent when the user cancels the divergent dialog (esc); main sets view and status.
type PerformCancelMsg struct{}

// PerformResolveMsg is sent when the user confirms resolve (enter); main runs ResolveDivergentCommitCmd.
type PerformResolveMsg struct {
	ChangeID     string
	KeepCommitID string
}

// PerformCancelCmd returns a command that sends PerformCancelMsg.
func PerformCancelCmd() tea.Cmd {
	return func() tea.Msg { return PerformCancelMsg{} }
}

// PerformResolveCmd returns a command that sends PerformResolveMsg with the given params.
func PerformResolveCmd(changeID, keepCommitID string) tea.Cmd {
	return func() tea.Msg { return PerformResolveMsg{ChangeID: changeID, KeepCommitID: keepCommitID} }
}

// DivergentCommitResolvedMsg is sent when a divergent commit has been resolved (kept one version).
type DivergentCommitResolvedMsg struct {
	ChangeID     string
	KeptCommitID string
	Err          error
}
