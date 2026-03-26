package conflict

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/data"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/tabs/branches"
)

// ResolveBookmarkConflictCmd runs jj resolve for the bookmark and sends BookmarkConflictResolvedMsg.
func ResolveBookmarkConflictCmd(jjSvc *jj.Service, bookmarkName, resolution string) tea.Cmd {
	if jjSvc == nil {
		return nil
	}
	svc := jjSvc
	return func() tea.Msg {
		var err error
		if resolution == "keep_local" {
			err = svc.ResolveBookmarkConflictKeepLocal(context.Background(), bookmarkName)
		} else {
			err = svc.ResolveBookmarkConflictResetToRemote(context.Background(), bookmarkName)
		}
		return BookmarkConflictResolvedMsg{
			BookmarkName: bookmarkName,
			Resolution:   resolution,
			Err:          err,
		}
	}
}

// ShowConflictInfo is returned when the handler wants main to show the conflict modal.
type ShowConflictInfo struct {
	BookmarkName  string
	LocalID       string
	RemoteID      string
	LocalSummary  string
	RemoteSummary string
}

// HandleBookmarkConflictInfoMsg mutates app when err; otherwise returns info for main to show the modal.
func HandleBookmarkConflictInfoMsg(msg branches.BookmarkConflictInfoMsg, app *state.AppState) (tea.Cmd, *ShowConflictInfo) {
	if msg.Err != nil {
		app.StatusMessage = fmt.Sprintf("Error loading conflict info: %v", msg.Err)
		app.ViewMode = state.ViewBranches
		return nil, nil
	}
	return nil, &ShowConflictInfo{
		BookmarkName:  msg.BookmarkName,
		LocalID:       msg.LocalID,
		RemoteID:      msg.RemoteID,
		LocalSummary:  msg.LocalSummary,
		RemoteSummary: msg.RemoteSummary,
	}
}

// HandleBookmarkConflictResolvedMsg mutates app StatusMessage and returns the Cmd to run.
// Main sets ViewMode to the tab the user was on when opening the dialog.
// branchLimit is used for LoadBranchesCmd (e.g. from settings).
func HandleBookmarkConflictResolvedMsg(msg BookmarkConflictResolvedMsg, app *state.AppState, branchLimit int) tea.Cmd {
	if msg.Err != nil {
		app.StatusMessage = fmt.Sprintf("Error resolving conflict: %v", msg.Err)
		return nil
	}
	resolutionDesc := "kept local version"
	if msg.Resolution == "reset_remote" {
		resolutionDesc = "reset to remote"
	}
	app.StatusMessage = fmt.Sprintf("Bookmark '%s' conflict resolved (%s)", msg.BookmarkName, resolutionDesc)
	return tea.Batch(
		data.LoadRepository(app.JJService),
		branches.LoadBranchesCmd(app.JJService, branchLimit),
	)
}
