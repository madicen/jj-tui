package graph

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/actions"
)

// ExecuteRequest runs the requested graph action using the given context.
// Returns a command to run and an optional status message (e.g. validation error).
// If statusMsg is non-empty, the caller should set status and return without running cmd.
func ExecuteRequest(r Request, ctx *RequestContext) (cmd tea.Cmd, statusMsg string) {
	if ctx == nil || ctx.JJService == nil {
		return nil, ""
	}

	if r.Checkout {
		return executeCheckout(ctx)
	}
	if r.Squash {
		return executeSquash(ctx)
	}
	if r.Abandon {
		return executeAbandon(ctx)
	}
	if r.PerformRebase {
		return executePerformRebase(r.RebaseDestIndex, ctx)
	}
	if r.DeleteBookmark {
		return executeDeleteBookmark(ctx)
	}
	if r.MoveFileUp {
		return executeMoveFileUp(ctx)
	}
	if r.MoveFileDown {
		return executeMoveFileDown(ctx)
	}
	if r.RevertFile {
		return executeRevertFile(ctx)
	}
	if r.NewCommit {
		return executeNewCommit(ctx)
	}
	return nil, ""
}

func executeCheckout(ctx *RequestContext) (tea.Cmd, string) {
	if !ctx.IsSelectedCommitValid() {
		return nil, ""
	}
	commit := ctx.Repository.Graph.Commits[ctx.SelectedCommit]
	if commit.Immutable {
		return nil, "Cannot edit: commit is immutable"
	}
	return actions.Checkout(ctx.JJService, commit.ChangeID), ""
}

func executeSquash(ctx *RequestContext) (tea.Cmd, string) {
	if !ctx.IsSelectedCommitValid() {
		return nil, ""
	}
	commit := ctx.Repository.Graph.Commits[ctx.SelectedCommit]
	if commit.Immutable {
		return nil, "Cannot squash: commit is immutable"
	}
	return actions.Squash(ctx.JJService, commit.ChangeID), ""
}

func executeAbandon(ctx *RequestContext) (tea.Cmd, string) {
	if !ctx.IsSelectedCommitValid() {
		return nil, ""
	}
	commit := ctx.Repository.Graph.Commits[ctx.SelectedCommit]
	if commit.Immutable {
		return nil, "Cannot abandon: commit is immutable"
	}
	if commit.Divergent {
		// Caller should show divergent dialog instead
		return nil, "__divergent__"
	}
	return actions.Abandon(ctx.JJService, commit.ChangeID), ""
}

func executePerformRebase(destIndex int, ctx *RequestContext) (tea.Cmd, string) {
	if !ctx.IsSelectedCommitValid() || ctx.RebaseSourceCommit < 0 ||
		ctx.RebaseSourceCommit >= len(ctx.Repository.Graph.Commits) ||
		destIndex < 0 || destIndex >= len(ctx.Repository.Graph.Commits) {
		return nil, ""
	}
	if ctx.RebaseSourceCommit == destIndex {
		return nil, "Cannot rebase commit onto itself"
	}
	sourceCommit := ctx.Repository.Graph.Commits[ctx.RebaseSourceCommit]
	destCommit := ctx.Repository.Graph.Commits[destIndex]
	return actions.Rebase(ctx.JJService, sourceCommit.ChangeID, destCommit.ChangeID), ""
}

func executeDeleteBookmark(ctx *RequestContext) (tea.Cmd, string) {
	if !ctx.IsSelectedCommitValid() {
		return nil, ""
	}
	commit := ctx.Repository.Graph.Commits[ctx.SelectedCommit]
	if len(commit.Branches) == 0 {
		return nil, "No bookmark on this commit to delete"
	}
	// DeleteBookmark needs bookmark name; we use the first branch on the commit
	bookmarkName := commit.Branches[0]
	return actions.DeleteBookmark(ctx.JJService, bookmarkName), ""
}

func executeMoveFileUp(ctx *RequestContext) (tea.Cmd, string) {
	if ctx.GraphFocused || len(ctx.ChangedFiles) == 0 {
		return nil, ""
	}
	if ctx.SelectedFile < 0 || ctx.SelectedFile >= len(ctx.ChangedFiles) {
		return nil, ""
	}
	if !ctx.IsSelectedCommitValid() {
		return nil, ""
	}
	commit := ctx.Repository.Graph.Commits[ctx.SelectedCommit]
	if commit.Immutable {
		return nil, "Cannot move file: commit is immutable"
	}
	file := ctx.ChangedFiles[ctx.SelectedFile]
	return actions.SplitFileToParent(ctx.JJService, commit.ChangeID, file.Path), ""
}

func executeMoveFileDown(ctx *RequestContext) (tea.Cmd, string) {
	if ctx.GraphFocused || len(ctx.ChangedFiles) == 0 {
		return nil, ""
	}
	if ctx.SelectedFile < 0 || ctx.SelectedFile >= len(ctx.ChangedFiles) {
		return nil, ""
	}
	if !ctx.IsSelectedCommitValid() {
		return nil, ""
	}
	commit := ctx.Repository.Graph.Commits[ctx.SelectedCommit]
	if commit.Immutable {
		return nil, "Cannot move file: commit is immutable"
	}
	file := ctx.ChangedFiles[ctx.SelectedFile]
	return actions.MoveFileToChild(ctx.JJService, commit.ChangeID, file.Path), ""
}

func executeRevertFile(ctx *RequestContext) (tea.Cmd, string) {
	if ctx.GraphFocused || len(ctx.ChangedFiles) == 0 {
		return nil, ""
	}
	if ctx.SelectedFile < 0 || ctx.SelectedFile >= len(ctx.ChangedFiles) {
		return nil, ""
	}
	if !ctx.IsSelectedCommitValid() {
		return nil, ""
	}
	commit := ctx.Repository.Graph.Commits[ctx.SelectedCommit]
	if commit.Immutable {
		return nil, "Cannot revert file: commit is immutable"
	}
	file := ctx.ChangedFiles[ctx.SelectedFile]
	return actions.RevertFile(ctx.JJService, commit.ChangeID, file.Path), ""
}

func executeNewCommit(ctx *RequestContext) (tea.Cmd, string) {
	parentCommitID := ""
	if ctx.IsSelectedCommitValid() {
		commit := ctx.Repository.Graph.Commits[ctx.SelectedCommit]
		parentCommitID = commit.ChangeID
	}
	return actions.NewCommit(ctx.JJService, parentCommitID), ""
}

// StatusNeedsDivergentDialog is returned by ExecuteRequest when the user
// requested abandon but the commit is divergent; the main model should
// show the divergent commit resolution dialog instead.
const StatusNeedsDivergentDialog = "__divergent__"
