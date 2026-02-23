package branches

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// Callbacks are provided by the main model to run branch actions.
type Callbacks struct {
	TrackBranch             func(branchName, remote string) tea.Cmd
	UntrackBranch           func(branchName, remote string) tea.Cmd
	RestoreLocalBranch      func(branchName, commitID string) tea.Cmd
	DeleteBranchBookmark    func(branchName string) tea.Cmd
	PushBranch              func(branchName string) tea.Cmd
	FetchAll                func() tea.Cmd
	LoadBookmarkConflictInfo func(bookmarkName string) tea.Cmd
}

// ExecuteRequest validates the request and runs the appropriate callback.
func ExecuteRequest(r Request, ctx *RequestContext, cb *Callbacks) (tea.Cmd, string) {
	if ctx == nil || cb == nil {
		return nil, ""
	}

	if r.FetchAll {
		if cb.FetchAll != nil {
			return cb.FetchAll(), ""
		}
		return nil, ""
	}

	if !ctx.SelectedBranchValid() {
		return nil, ""
	}
	branch := ctx.SelectedBranchData()
	if branch == nil {
		return nil, ""
	}

	switch {
	case r.TrackBranch:
		if branch.IsLocal || branch.IsTracked {
			return nil, "Branch is already tracked"
		}
		if cb.TrackBranch != nil {
			return cb.TrackBranch(branch.Name, branch.Remote), ""
		}
	case r.UntrackBranch:
		if !branch.IsTracked {
			return nil, "Branch is not tracked"
		}
		if cb.UntrackBranch != nil {
			return cb.UntrackBranch(branch.Name, branch.Remote), ""
		}
	case r.RestoreLocalBranch:
		if !branch.LocalDeleted {
			return nil, "Branch local copy is not deleted"
		}
		if cb.RestoreLocalBranch != nil {
			return cb.RestoreLocalBranch(branch.Name, branch.CommitID), ""
		}
	case r.DeleteBranchBookmark:
		if !branch.IsLocal {
			return nil, "Can only delete local bookmarks"
		}
		if branch.HasConflict {
			return nil, "This bookmark has diverged. Resolve the conflict first (press 'c')."
		}
		if cb.DeleteBranchBookmark != nil {
			return cb.DeleteBranchBookmark(branch.Name), ""
		}
	case r.PushBranch:
		if !branch.IsLocal {
			return nil, "Can only push local branches"
		}
		if cb.PushBranch != nil {
			return cb.PushBranch(branch.Name), ""
		}
	case r.ResolveBookmarkConflict:
		if !branch.HasConflict {
			return nil, "This bookmark is not conflicted"
		}
		if cb.LoadBookmarkConflictInfo != nil {
			return cb.LoadBookmarkConflictInfo(branch.Name), ""
		}
	default:
		return nil, ""
	}
	return nil, ""
}

// StatusMessageForRequest returns a short status message for the given request (e.g. "Tracking branch x..."). Caller can use this when about to run the cmd.
func StatusMessageForRequest(r Request, ctx *RequestContext) string {
	if ctx == nil || !ctx.SelectedBranchValid() {
		return ""
	}
	branch := ctx.SelectedBranchData()
	if branch == nil {
		return ""
	}
	switch {
	case r.TrackBranch:
		return fmt.Sprintf("Tracking branch %s...", branch.Name)
	case r.UntrackBranch:
		return fmt.Sprintf("Untracking branch %s...", branch.Name)
	case r.RestoreLocalBranch:
		return fmt.Sprintf("Restoring local branch %s...", branch.Name)
	case r.DeleteBranchBookmark:
		return fmt.Sprintf("Deleting bookmark %s...", branch.Name)
	case r.PushBranch:
		return fmt.Sprintf("Pushing branch %s...", branch.Name)
	case r.FetchAll:
		return "Fetching from all remotes..."
	case r.ResolveBookmarkConflict:
		return "Loading conflict info..."
	}
	return ""
}
