package descedit

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/data"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/util"
)

// LoadDescriptionCmd fetches the complete description for a commit.
func LoadDescriptionCmd(svc *jj.Service, commitID string) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return util.ErrorMsg{Err: fmt.Errorf("jj service not available")}
		}
		desc, err := svc.GetCommitDescription(context.Background(), commitID)
		if err != nil {
			return util.ErrorMsg{Err: fmt.Errorf("failed to load description: %w", err)}
		}
		return DescriptionLoadedMsg{CommitID: commitID, Description: desc}
	}
}

// SaveDescriptionCmd saves a commit description.
func SaveDescriptionCmd(svc *jj.Service, commitID, description string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.DescribeCommit(context.Background(), commitID, description); err != nil {
			return util.ErrorMsg{Err: fmt.Errorf("failed to update description: %w", err)}
		}
		return DescriptionSavedMsg{CommitID: commitID}
	}
}

// HandleDescriptionSavedMsg mutates app (ViewMode, StatusMessage) and returns the Cmd to run.
func HandleDescriptionSavedMsg(msg DescriptionSavedMsg, app *state.AppState) tea.Cmd {
	app.ViewMode = state.ViewCommitGraph
	app.StatusMessage = fmt.Sprintf("Description updated for %s", msg.CommitID)
	return data.LoadRepository(app.JJService)
}

// DescriptionLoadedInput is the context main sends when forwarding DescriptionLoadedMsg (for building suggested description).
type DescriptionLoadedInput struct {
	CommitID       string
	Description    string
	Repository     *internal.Repository
	CommitIdx      int
	TicketKeys     map[string]string // bookmark name -> ticket short display key
	FindBookmarkFn func(*internal.Repository, int) string
}

// SuggestDescriptionForLoad returns the description to set in the modal (may prepend ticket short ID if empty).
func SuggestDescriptionForLoad(input DescriptionLoadedInput) string {
	description := input.Description
	if description == "(no description)" {
		description = ""
	}
	if description != "" {
		return description
	}
	if input.Repository == nil || input.CommitIdx < 0 || input.CommitIdx >= len(input.Repository.Graph.Commits) {
		return ""
	}
	commit := input.Repository.Graph.Commits[input.CommitIdx]
	var foundShortID string
	for _, branch := range commit.Branches {
		if shortID, ok := input.TicketKeys[branch]; ok {
			foundShortID = shortID
			break
		}
		if shortID, ok := input.TicketKeys[internal.LocalBookmarkName(branch)]; ok {
			foundShortID = shortID
			break
		}
	}
	if foundShortID == "" && input.FindBookmarkFn != nil {
		ancestorBookmark := input.FindBookmarkFn(input.Repository, input.CommitIdx)
		if ancestorBookmark != "" {
			if shortID, ok := input.TicketKeys[ancestorBookmark]; ok {
				foundShortID = shortID
			}
		}
	}
	if foundShortID != "" {
		return foundShortID + " "
	}
	return ""
}

// StartEditing prepares the description-edit modal for the given commit and returns the updated model and status message.
// Caller sets view mode and runs LoadDescriptionCmd(jjService, commit.ChangeID).
func StartEditing(modal Model, commit internal.Commit, width, height int) (Model, string) {
	m, _ := modal.PrepareForCommit(commit, width, height)
	return m, fmt.Sprintf("Loading description for %s...", commit.ShortID)
}
