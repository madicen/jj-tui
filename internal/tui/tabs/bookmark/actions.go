package bookmark

import (
	"context"
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/data"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/tabs/tickets"
	"github.com/madicen/jj-tui/internal/tui/util"
)

// PrepareShowResult is the result of PrepareShow for opening the bookmark creation dialog.
type PrepareShowResult struct {
	ExistingBookmarks []string
	CommitShortID     string
}

// PrepareShow returns data needed to show the bookmark creation dialog (existing bookmarks, commit short ID).
func PrepareShow(repo *internal.Repository, commitIdx int) PrepareShowResult {
	if repo == nil || commitIdx < 0 || commitIdx >= len(repo.Graph.Commits) {
		return PrepareShowResult{}
	}
	commit := repo.Graph.Commits[commitIdx]
	return PrepareShowResult{
		ExistingBookmarks: GetExistingBookmarks(repo, commitIdx),
		CommitShortID:     commit.ShortID,
	}
}

// SubmitInput contains everything needed to run the bookmark submit (create or create-from-Jira).
type SubmitInput struct {
	BookmarkName              string
	CommitIdx                 int
	CommitID                  string
	FromJira                  bool
	JiraKey                   string
	JiraTitle                 string
	DisplayKey                string
	JiraBookmarkTitles        map[string]string
	TicketBookmarkDisplayKeys map[string]string
	JJService                 *jj.Service
	SanitizeBookmarks         bool
}

// SubmitCmd validates and runs the appropriate command (CreateBookmarkCmd or CreateBranchFromMain).
// Returns (cmd, ""); if validation fails returns (nil, validationError). Caller sets status from validationError.
func SubmitCmd(input SubmitInput) (tea.Cmd, string) {
	bookmarkName := strings.TrimSpace(input.BookmarkName)
	if input.SanitizeBookmarks {
		bookmarkName = jj.SanitizeBookmarkName(bookmarkName)
	}
	if err := ValidateBookmarkName(bookmarkName); err != "" {
		return nil, err
	}
	if input.FromJira {
		return CreateBranchFromMainCmd(input.JJService, bookmarkName, input.JiraKey), ""
	}
	return CreateBookmarkCmd(input.JJService, bookmarkName, input.CommitID), ""
}

// OpenCreateBookmark prepares and shows the bookmark creation dialog for the given commit.
// Caller sets view mode and status message from the returned value.
func OpenCreateBookmark(modal *Model, repo *internal.Repository, commitIdx int, conflictSources []string, sanitize bool, width int) string {
	data := PrepareShow(repo, commitIdx)
	modal.Show(commitIdx, data.ExistingBookmarks)
	modal.UpdateRepository(repo)
	modal.SetNameConflictSources(conflictSources)
	modal.UpdateNameExistsFromInput(sanitize)
	ni := modal.GetNameInput()
	ni.SetValue("")
	ni.Focus()
	ni.Width = width
	return fmt.Sprintf("Create or move bookmark on %s", data.CommitShortID)
}

// OpenCreateBookmarkFromTicket prepares and shows the bookmark creation dialog to create a branch from main for the given ticket.
// Caller sets view mode and status message from the returned value.
func OpenCreateBookmarkFromTicket(modal *Model, repo *internal.Repository, ticketKey, title, displayKey string, conflictSources []string, sanitize bool, width int) string {
	modal.Show(-1, nil)
	modal.SetFromJira(ticketKey, title, displayKey)
	modal.SetBookmarkName(ticketKey)
	modal.UpdateRepository(repo)
	modal.SetNameConflictSources(conflictSources)
	modal.UpdateNameExistsFromInput(sanitize)
	ni := modal.GetNameInput()
	ni.Focus()
	ni.Width = width
	return fmt.Sprintf("Create branch from main for ticket %s", ticketKey)
}

// SubmitBookmark builds submit input from modal state and repo/config, handles Jira side effects, and runs the submit command.
// Returns (cmd, statusOrError). Caller sets status message and returns the cmd.
func SubmitBookmark(modal *Model, repo *internal.Repository, cfg *config.Config, jjService *jj.Service) (tea.Cmd, string) {
	commitIdx := modal.GetCommitIdx()
	var commitID string
	if repo != nil && commitIdx >= 0 && commitIdx < len(repo.Graph.Commits) {
		commitID = repo.Graph.Commits[commitIdx].ChangeID
	}
	if modal.GetSelectedBookmarkIdx() >= 0 {
		return nil, "Moving existing bookmark not yet supported"
	}
	sanitize := true
	if cfg != nil {
		sanitize = cfg.ShouldSanitizeBookmarkNames()
	}
	input := SubmitInput{
		BookmarkName:              modal.GetBookmarkName(),
		CommitIdx:                 commitIdx,
		CommitID:                  commitID,
		FromJira:                  modal.IsFromJira(),
		JiraKey:                   modal.GetJiraKey(),
		JiraTitle:                 modal.GetJiraTicketTitle(),
		DisplayKey:                modal.GetTicketDisplayKey(),
		JiraBookmarkTitles:        modal.GetJiraBookmarkTitles(),
		TicketBookmarkDisplayKeys: modal.GetTicketBookmarkDisplayKeys(),
		JJService:                 jjService,
		SanitizeBookmarks:         sanitize,
	}
	if input.FromJira {
		bookmarkName := strings.TrimSpace(input.BookmarkName)
		if sanitize {
			bookmarkName = jj.SanitizeBookmarkName(bookmarkName)
		}
		if input.JiraTitle != "" && input.JiraKey != "" {
			keyForTitle := input.JiraKey
			if input.DisplayKey != "" {
				keyForTitle = input.DisplayKey
			}
			titles := modal.GetJiraBookmarkTitles()
			if titles == nil {
				titles = make(map[string]string)
			}
			titles[bookmarkName] = keyForTitle + " - " + input.JiraTitle
			modal.SetJiraBookmarkTitles(titles)
		}
		if input.DisplayKey != "" {
			keys := modal.GetTicketBookmarkDisplayKeys()
			if keys == nil {
				keys = make(map[string]string)
			}
			keys[bookmarkName] = input.DisplayKey
			modal.SetTicketBookmarkDisplayKeys(keys)
		}
		modal.ClearJiraContext()
	}
	cmd, errStr := SubmitCmd(input)
	if errStr != "" {
		return nil, errStr
	}
	if input.FromJira {
		return cmd, fmt.Sprintf("Creating branch '%s' from main...", strings.TrimSpace(input.BookmarkName))
	}
	return cmd, fmt.Sprintf("Creating bookmark '%s'...", strings.TrimSpace(input.BookmarkName))
}

// ValidateBookmarkName returns error message if invalid, empty if valid.
func ValidateBookmarkName(name string) string {
	if name == "" {
		return "Bookmark name is required"
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '/') {
			return "Invalid bookmark name. Use letters, numbers, -, _, or /"
		}
	}
	return ""
}

// GetExistingBookmarks returns sorted bookmarks excluding those on commitIdx.
func GetExistingBookmarks(repo *internal.Repository, commitIdx int) []string {
	if repo == nil || commitIdx < 0 || commitIdx >= len(repo.Graph.Commits) {
		return nil
	}
	commit := repo.Graph.Commits[commitIdx]
	existingOnCommit := make(map[string]bool)
	for _, b := range commit.Branches {
		existingOnCommit[b] = true
	}
	bookmarkSet := make(map[string]bool)
	for _, c := range repo.Graph.Commits {
		for _, b := range c.Branches {
			if !existingOnCommit[b] {
				bookmarkSet[b] = true
			}
		}
	}
	bookmarks := make([]string, 0, len(bookmarkSet))
	for b := range bookmarkSet {
		bookmarks = append(bookmarks, b)
	}
	sort.Strings(bookmarks)
	return bookmarks
}

// CreateBookmarkCmd returns a command that creates a new bookmark on a commit.
func CreateBookmarkCmd(svc *jj.Service, bookmarkName, commitID string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.CreateBookmarkOnCommit(context.Background(), bookmarkName, commitID); err != nil {
			return util.ErrorMsg{Err: fmt.Errorf("failed to create bookmark: %w", err)}
		}
		return BookmarkCreatedMsg{BookmarkName: bookmarkName, CommitID: commitID, WasMoved: false}
	}
}

// MoveBookmarkCmd returns a command that moves an existing bookmark to a commit.
func MoveBookmarkCmd(svc *jj.Service, bookmarkName, commitID string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.MoveBookmark(context.Background(), bookmarkName, commitID); err != nil {
			return util.ErrorMsg{Err: fmt.Errorf("failed to move bookmark: %w", err)}
		}
		return BookmarkCreatedMsg{BookmarkName: bookmarkName, CommitID: commitID, WasMoved: true}
	}
}

// CreateBranchFromMainCmd returns a command that creates a new branch from main.
// ticketKey is optional - if provided, it enables auto-transition to "In Progress".
func CreateBranchFromMainCmd(svc *jj.Service, bookmarkName, ticketKey string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.CreateBranchFromMain(context.Background(), bookmarkName); err != nil {
			return util.ErrorMsg{Err: fmt.Errorf("failed to create branch from main: %w", err)}
		}
		return BookmarkCreatedMsg{BookmarkName: bookmarkName, CommitID: "main", WasMoved: false, TicketKey: ticketKey}
	}
}

// DeleteBookmarkCmd returns a command that deletes a bookmark.
func DeleteBookmarkCmd(svc *jj.Service, bookmarkName string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.DeleteBookmark(context.Background(), bookmarkName); err != nil {
			return util.ErrorMsg{Err: fmt.Errorf("failed to delete bookmark: %w", err)}
		}
		return BookmarkDeletedMsg{BookmarkName: bookmarkName}
	}
}

// FindBookmarkForCommit finds a bookmark from ancestors using BFS.
func FindBookmarkForCommit(repo *internal.Repository, commitIdx int) string {
	if repo == nil || commitIdx < 0 || commitIdx >= len(repo.Graph.Commits) {
		return ""
	}
	commitIDToIndex := make(map[string]int)
	for i, commit := range repo.Graph.Commits {
		commitIDToIndex[commit.ID] = i
		commitIDToIndex[commit.ChangeID] = i
	}
	visited := make(map[int]bool)
	queue := []int{commitIdx}
	for len(queue) > 0 {
		idx := queue[0]
		queue = queue[1:]
		if visited[idx] {
			continue
		}
		visited[idx] = true
		commit := repo.Graph.Commits[idx]
		if len(commit.Branches) > 0 {
			return commit.Branches[0]
		}
		for _, parentID := range commit.Parents {
			if parentIdx, ok := commitIDToIndex[parentID]; ok {
				queue = append(queue, parentIdx)
			}
		}
	}
	return ""
}

// HandleBookmarkCreatedMsg mutates app (ViewMode, StatusMessage) and returns the Cmd to run.
func HandleBookmarkCreatedMsg(msg BookmarkCreatedMsg, app *state.AppState) tea.Cmd {
	app.ViewMode = state.ViewCommitGraph
	statusMsg := fmt.Sprintf("Bookmark '%s' created", msg.BookmarkName)
	if msg.WasMoved {
		statusMsg = fmt.Sprintf("Bookmark '%s' moved", msg.BookmarkName)
	}
	app.StatusMessage = statusMsg
	if msg.TicketKey != "" && app.TicketService != nil && app.Config != nil && app.Config.AutoInProgressOnBranch() {
		return tea.Batch(
			data.LoadRepository(app.JJService),
			tickets.TransitionTicketToInProgressCmd(app.TicketService, msg.TicketKey),
		)
	}
	return data.LoadRepository(app.JJService)
}
