package branches

import (
	"context"
	"fmt"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/data"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/tabs/bookmark"
	"github.com/madicen/jj-tui/internal/tui/tabs/prs"
)

// LoadBranchesCmd returns a command that lists branches (with sorting) and sends BranchesLoadedMsg.
func LoadBranchesCmd(jjSvc *jj.Service, branchLimit int) tea.Cmd {
	if jjSvc == nil {
		return nil
	}
	svc := jjSvc
	return func() tea.Msg {
		branches, err := svc.ListBranches(context.Background(), branchLimit)
		if err != nil {
			return BranchesLoadedMsg{Err: err}
		}
		sort.Slice(branches, func(i, j int) bool {
			if branches[i].IsLocal != branches[j].IsLocal {
				return branches[i].IsLocal
			}
			iAhead := branches[i].Ahead > 0 && branches[i].Behind == 0
			jAhead := branches[j].Ahead > 0 && branches[j].Behind == 0
			if iAhead != jAhead {
				return iAhead
			}
			if iAhead {
				if branches[i].Ahead != branches[j].Ahead {
					return branches[i].Ahead > branches[j].Ahead
				}
			} else {
				if branches[i].Behind != branches[j].Behind {
					return branches[i].Behind < branches[j].Behind
				}
			}
			return branches[i].Name < branches[j].Name
		})
		return BranchesLoadedMsg{Branches: branches}
	}
}

// TrackBranchCmd returns a command that tracks a branch (returns BranchActionMsg).
func TrackBranchCmd(jjSvc *jj.Service, branchName, remote string) tea.Cmd {
	return TrackBranch(jjSvc, branchName, remote)
}

// UntrackBranchCmd returns a command that untracks a branch.
func UntrackBranchCmd(jjSvc *jj.Service, branchName, remote string) tea.Cmd {
	return UntrackBranch(jjSvc, branchName, remote)
}

// RestoreLocalBranchCmd returns a command that restores a local branch.
func RestoreLocalBranchCmd(jjSvc *jj.Service, branchName, commitID string) tea.Cmd {
	return RestoreLocalBranch(jjSvc, branchName, commitID)
}

// DeleteBranchBookmarkCmd returns a command that deletes a branch bookmark.
func DeleteBranchBookmarkCmd(jjSvc *jj.Service, branchName string) tea.Cmd {
	return DeleteBranchBookmark(jjSvc, branchName)
}

// PushBranchCmd returns a command that pushes a branch.
func PushBranchCmd(jjSvc *jj.Service, branchName string) tea.Cmd {
	return PushBranch(jjSvc, branchName)
}

// FetchAllRemotesCmd returns a command that fetches from all remotes.
func FetchAllRemotesCmd(jjSvc *jj.Service) tea.Cmd {
	return FetchAllRemotes(jjSvc)
}

// LoadBookmarkConflictInfoCmd returns a command that loads bookmark conflict info (returns BookmarkConflictInfoMsg).
func LoadBookmarkConflictInfoCmd(jjSvc *jj.Service, bookmarkName string) tea.Cmd {
	return LoadBookmarkConflictInfo(jjSvc, bookmarkName)
}

// TrackBranch starts tracking a remote branch.
func TrackBranch(svc *jj.Service, branchName, remote string) tea.Cmd {
	if svc == nil {
		return nil
	}
	return func() tea.Msg {
		err := svc.TrackBranch(context.Background(), branchName, remote)
		if err != nil {
			return BranchActionMsg{Action: "track", Branch: branchName, Err: err}
		}
		return BranchActionMsg{Action: "track", Branch: branchName}
	}
}

// UntrackBranch stops tracking a remote branch.
func UntrackBranch(svc *jj.Service, branchName, remote string) tea.Cmd {
	if svc == nil {
		return nil
	}
	return func() tea.Msg {
		err := svc.UntrackBranch(context.Background(), branchName, remote)
		if err != nil {
			return BranchActionMsg{Action: "untrack", Branch: branchName, Err: err}
		}
		return BranchActionMsg{Action: "untrack", Branch: branchName}
	}
}

// RestoreLocalBranch restores a deleted local branch from its tracked remote.
func RestoreLocalBranch(svc *jj.Service, branchName, commitID string) tea.Cmd {
	if svc == nil {
		return nil
	}
	return func() tea.Msg {
		err := svc.RestoreLocalBranch(context.Background(), branchName, commitID)
		if err != nil {
			return BranchActionMsg{Action: "restore", Branch: branchName, Err: err}
		}
		return BranchActionMsg{Action: "restore", Branch: branchName}
	}
}

// DeleteBranchBookmark deletes a local bookmark.
func DeleteBranchBookmark(svc *jj.Service, branchName string) tea.Cmd {
	if svc == nil {
		return nil
	}
	return func() tea.Msg {
		err := svc.DeleteBookmark(context.Background(), branchName)
		if err != nil {
			return BranchActionMsg{Action: "delete", Branch: branchName, Err: err}
		}
		return BranchActionMsg{Action: "delete", Branch: branchName}
	}
}

// PushBranch pushes a local branch to remote.
func PushBranch(svc *jj.Service, branchName string) tea.Cmd {
	if svc == nil {
		return nil
	}
	return func() tea.Msg {
		err := svc.PushBranch(context.Background(), branchName)
		if err != nil {
			return BranchActionMsg{Action: "push", Branch: branchName, Err: err}
		}
		return BranchActionMsg{Action: "push", Branch: branchName}
	}
}

// FetchAllRemotes fetches from all remotes.
func FetchAllRemotes(svc *jj.Service) tea.Cmd {
	if svc == nil {
		return nil
	}
	return func() tea.Msg {
		err := svc.FetchAllRemotes(context.Background())
		if err != nil {
			return BranchActionMsg{Action: "fetch", Err: err}
		}
		return BranchActionMsg{Action: "fetch"}
	}
}

// LoadBookmarkConflictInfo loads information about a conflicted bookmark.
func LoadBookmarkConflictInfo(svc *jj.Service, bookmarkName string) tea.Cmd {
	if svc == nil {
		return nil
	}
	return func() tea.Msg {
		localID, remoteID, localSummary, remoteSummary, err := svc.GetBookmarkConflictInfo(context.Background(), bookmarkName)
		return BookmarkConflictInfoMsg{
			BookmarkName:  bookmarkName,
			LocalID:       localID,
			RemoteID:      remoteID,
			LocalSummary:  localSummary,
			RemoteSummary: remoteSummary,
			Err:           err,
		}
	}
}

// ExecuteRequest validates the request and returns (statusMsg, cmd). Main sets statusMsg and returns the cmd.
func ExecuteRequest(r Request, ctx *RequestContext) (statusMsg string, cmd tea.Cmd) {
	if ctx == nil {
		return "", nil
	}

	if r.FetchAll {
		return "Fetching from all remotes...", FetchAllRemotesCmd(ctx.JJService)
	}

	if !ctx.SelectedBranchValid() {
		return "", nil
	}
	branch := ctx.SelectedBranchData()
	if branch == nil {
		return "", nil
	}

	switch {
	case r.TrackBranch:
		if branch.IsLocal || branch.IsTracked {
			return "Branch is already tracked", nil
		}
		return fmt.Sprintf("Tracking branch %s...", branch.Name), TrackBranchCmd(ctx.JJService, branch.Name, branch.Remote)
	case r.UntrackBranch:
		if !branch.IsTracked {
			return "Branch is not tracked", nil
		}
		return fmt.Sprintf("Untracking branch %s...", branch.Name), UntrackBranchCmd(ctx.JJService, branch.Name, branch.Remote)
	case r.RestoreLocalBranch:
		if !branch.LocalDeleted {
			return "Branch local copy is not deleted", nil
		}
		return fmt.Sprintf("Restoring local branch %s...", branch.Name), RestoreLocalBranchCmd(ctx.JJService, branch.Name, branch.CommitID)
	case r.DeleteBranchBookmark:
		if !branch.IsLocal {
			return "Can only delete local bookmarks", nil
		}
		if branch.HasConflict {
			return "This bookmark has diverged. Resolve the conflict first (press 'c').", nil
		}
		return fmt.Sprintf("Deleting bookmark %s...", branch.Name), DeleteBranchBookmarkCmd(ctx.JJService, branch.Name)
	case r.PushBranch:
		if !branch.IsLocal {
			return "Can only push local branches", nil
		}
		return fmt.Sprintf("Pushing branch %s...", branch.Name), PushBranchCmd(ctx.JJService, branch.Name)
	case r.ResolveBookmarkConflict:
		if !branch.HasConflict {
			return "This bookmark is not conflicted", nil
		}
		return "Loading conflict info...", LoadBookmarkConflictInfoCmd(ctx.JJService, branch.Name)
	default:
		return "", nil
	}
}

// HandleBranchPushedMsg mutates app (StatusMessage) and returns the Cmd to run.
func HandleBranchPushedMsg(msg prs.BranchPushedMsg, app *state.AppState) tea.Cmd {
	app.StatusMessage = fmt.Sprintf("Pushed %s to remote", msg.Branch)
	existing := 0
	if app.Repository != nil {
		existing = len(app.Repository.PRs)
	}
	return tea.Batch(
		data.LoadRepository(app.JJService),
		prs.LoadPRsCmd(app.GitHubService, app.GithubInfo, app.DemoMode, existing),
	)
}

// HandleBookmarkDeletedMsg mutates app (ViewMode, StatusMessage) and returns the Cmd to run.
func HandleBookmarkDeletedMsg(msg bookmark.BookmarkDeletedMsg, app *state.AppState) tea.Cmd {
	app.ViewMode = state.ViewCommitGraph
	app.StatusMessage = fmt.Sprintf("Bookmark '%s' deleted", msg.BookmarkName)
	existing := 0
	if app.Repository != nil {
		existing = len(app.Repository.PRs)
	}
	return tea.Batch(
		data.LoadRepository(app.JJService),
		prs.LoadPRsCmd(app.GitHubService, app.GithubInfo, app.DemoMode, existing),
	)
}
