package settings

import tea "github.com/charmbracelet/bubbletea"

// Callbacks are provided by the main model to run settings actions.
type Callbacks struct {
	SaveSettings      func() tea.Cmd
	SaveSettingsLocal func() tea.Cmd
}

// ExecuteResult holds the result of ExecuteRequest.
// When NeedCancel is true, the model should run handleSettingsCancel (no async cmd).
type ExecuteResult struct {
	Cmd       tea.Cmd
	StatusMsg string
	NeedCancel bool
}

// ExecuteRequest validates the request and runs the appropriate callback.
func ExecuteRequest(r Request, cb *Callbacks) ExecuteResult {
	if cb == nil {
		return ExecuteResult{}
	}
	if r.Cancel {
		return ExecuteResult{NeedCancel: true}
	}
	if r.SaveSettings && cb.SaveSettings != nil {
		return ExecuteResult{Cmd: cb.SaveSettings()}
	}
	if r.SaveSettingsLocal && cb.SaveSettingsLocal != nil {
		return ExecuteResult{Cmd: cb.SaveSettingsLocal()}
	}
	return ExecuteResult{}
}

// CleanupCallbacks are provided by the main model to run cleanup actions (delete bookmarks, abandon commits).
type CleanupCallbacks struct {
	DeleteAllBookmarks    func() tea.Cmd
	AbandonOldCommits     func() tea.Cmd
}

// ConfirmCleanup returns the command for the given confirming cleanup type, or nil.
func ConfirmCleanup(confirmingType string, cb *CleanupCallbacks) tea.Cmd {
	if cb == nil {
		return nil
	}
	switch confirmingType {
	case "delete_bookmarks":
		if cb.DeleteAllBookmarks != nil {
			return cb.DeleteAllBookmarks()
		}
	case "abandon_old_commits":
		if cb.AbandonOldCommits != nil {
			return cb.AbandonOldCommits()
		}
	}
	return nil
}

// Status messages for cleanup flows (model sets these when starting/cancelling).
const (
	StartDeleteBookmarksStatus    = "Press Y to confirm deletion of all bookmarks, or N to cancel"
	StartAbandonOldCommitsStatus  = "Press Y to confirm abandoning commits before origin/main, or N to cancel"
	CancelCleanupStatus          = "Cleanup cancelled"
)
