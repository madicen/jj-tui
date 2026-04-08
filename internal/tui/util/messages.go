package util

// ErrorMsg indicates an action failed; used by multiple tabs.
type ErrorMsg struct {
	Err error
	// StatusOnly, when true, tells the main model to show Err in the status bar and clear loading
	// without opening the error modal (used for expected failures such as git push rejected).
	StatusOnly bool
}

// ClipboardCopiedMsg indicates clipboard operation result.
type ClipboardCopiedMsg struct {
	Success bool
	Err     error
}
