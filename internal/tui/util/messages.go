package util

// ErrorMsg indicates an action failed; used by multiple tabs.
type ErrorMsg struct {
	Err error
}

// ClipboardCopiedMsg indicates clipboard operation result.
type ClipboardCopiedMsg struct {
	Success bool
	Err     error
}
