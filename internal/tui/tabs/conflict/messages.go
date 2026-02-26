package conflict

import tea "github.com/charmbracelet/bubbletea"

// PerformCancelMsg is sent when the user cancels the conflict dialog (esc); main sets view and status.
type PerformCancelMsg struct{}

// PerformResolveMsg is sent when the user confirms resolve (enter); main runs ResolveBookmarkConflictCmd.
type PerformResolveMsg struct {
	BookmarkName string
	Resolution   string // "keep_local" or "reset_remote"
}

// PerformCancelCmd returns a command that sends PerformCancelMsg.
func PerformCancelCmd() tea.Cmd {
	return func() tea.Msg { return PerformCancelMsg{} }
}

// PerformResolveCmd returns a command that sends PerformResolveMsg with the given params.
func PerformResolveCmd(bookmarkName, resolution string) tea.Cmd {
	return func() tea.Msg { return PerformResolveMsg{BookmarkName: bookmarkName, Resolution: resolution} }
}

// BookmarkConflictResolvedMsg is sent when a bookmark conflict has been resolved (keep local or reset to remote).
type BookmarkConflictResolvedMsg struct {
	BookmarkName string
	Resolution   string // "keep_local" or "reset_remote"
	Err          error
}
