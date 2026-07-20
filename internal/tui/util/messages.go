package util

import tea "github.com/charmbracelet/bubbletea"

// ErrorMsg indicates an action failed; used by multiple tabs.
type ErrorMsg struct {
	Err error
	// StatusOnly, when true, tells the main model to show Err in the status bar and clear loading
	// without opening the error modal. Use sparingly for lightweight failures where a blocking
	// modal would be disproportionate (e.g. external editor could not start).
	StatusOnly bool
	// Retry, when non-nil, arms the error modal's Retry button (and post–kill-gpg-agent
	// auto-retry) with this command. Callers typically set it to the same tea.Cmd factory
	// that produced the failure.
	Retry tea.Cmd
}

// ClipboardCopiedMsg indicates clipboard operation result.
type ClipboardCopiedMsg struct {
	Success bool
	Err     error
}
