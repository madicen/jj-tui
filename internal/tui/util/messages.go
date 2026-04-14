package util

// ErrorMsg indicates an action failed; used by multiple tabs.
type ErrorMsg struct {
	Err error
	// StatusOnly, when true, tells the main model to show Err in the status bar and clear loading
	// without opening the error modal. Use sparingly for lightweight failures where a blocking
	// modal would be disproportionate (e.g. external editor could not start).
	StatusOnly bool
}

// ClipboardCopiedMsg indicates clipboard operation result.
type ClipboardCopiedMsg struct {
	Success bool
	Err     error
}
