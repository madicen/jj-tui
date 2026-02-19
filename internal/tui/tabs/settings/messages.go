package settings

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tickets"
)

// CleanupCompletedMsg is sent when a settings cleanup operation completes (delete bookmarks, abandon old commits).
type CleanupCompletedMsg struct {
	Success bool
	Message string
	Err     error
}

// SettingsSavedMsg indicates settings were saved (used by main and settings tab).
type SettingsSavedMsg struct {
	GitHubConnected bool
	TicketService   tickets.Service
	TicketProvider  string
	SavedLocal      bool
	Err             error
}

// Request is sent to the main model for Settings tab actions.
type Request struct {
	Cancel            bool // Leave settings without saving
	SaveSettings      bool // Save settings (e.g. ctrl+s / enter on last field)
	SaveSettingsLocal bool // Save to local .jj-tui.json (ctrl+l)
}

// Cmd returns a tea.Cmd that sends this request.
func (r Request) Cmd() tea.Cmd {
	return func() tea.Msg { return r }
}

// PerformCancelMsg is sent so main leaves settings view and sets status (tab decided to cancel on esc).
type PerformCancelMsg struct{}

// PerformCancelCmd returns a command that sends PerformCancelMsg.
func PerformCancelCmd() tea.Cmd {
	return func() tea.Msg { return PerformCancelMsg{} }
}

// RequestConfirmCleanupMsg is sent when user confirms cleanup (y); main runs confirmCleanup().
type RequestConfirmCleanupMsg struct{}

// RequestCancelCleanupMsg is sent when user cancels cleanup (n/esc); main sets status to CancelCleanupStatus.
type RequestCancelCleanupMsg struct{}

// RequestConfirmCleanupCmd returns a command that sends RequestConfirmCleanupMsg.
func RequestConfirmCleanupCmd() tea.Cmd {
	return func() tea.Msg { return RequestConfirmCleanupMsg{} }
}

// RequestCancelCleanupCmd returns a command that sends RequestCancelCleanupMsg.
func RequestCancelCleanupCmd() tea.Cmd {
	return func() tea.Msg { return RequestCancelCleanupMsg{} }
}

// RequestSetStatusMsg is sent when the tab wants main to set the status line (e.g. after starting a cleanup confirmation).
type RequestSetStatusMsg struct {
	Status string
}

// RequestSetStatusCmd returns a command that sends RequestSetStatusMsg.
func RequestSetStatusCmd(status string) tea.Cmd {
	return func() tea.Msg { return RequestSetStatusMsg{Status: status} }
}

// GitHubDeviceFlowStartedMsg is sent when device flow authentication starts.
type GitHubDeviceFlowStartedMsg struct {
	DeviceCode      string
	UserCode        string
	VerificationURL string
	Interval        int
}

// GitHubLoginPollMsg is sent to continue polling for the GitHub token.
type GitHubLoginPollMsg struct {
	Interval int
}

// GitHubLoginSuccessMsg is sent when GitHub login succeeds.
type GitHubLoginSuccessMsg struct {
	Token string
}

// GitHubLoginErrorMsg is sent when starting device flow or polling fails.
type GitHubLoginErrorMsg struct {
	Err error
}
