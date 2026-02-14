package actions

import (
	"context"
	"fmt"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/models"
)

// ValidateBookmarkName returns error message if invalid, empty if valid
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

// GetExistingBookmarks returns sorted bookmarks excluding those on commitIdx
func GetExistingBookmarks(repo *models.Repository, commitIdx int) []string {
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

// CreateBookmark creates a new bookmark on a commit
func CreateBookmark(svc *jj.Service, bookmarkName, commitID string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.CreateBookmarkOnCommit(context.Background(), bookmarkName, commitID); err != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to create bookmark: %w", err)}
		}
		return BookmarkCreatedMsg{BookmarkName: bookmarkName, CommitID: commitID, WasMoved: false}
	}
}

// MoveBookmark moves an existing bookmark to a commit
func MoveBookmark(svc *jj.Service, bookmarkName, commitID string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.MoveBookmark(context.Background(), bookmarkName, commitID); err != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to move bookmark: %w", err)}
		}
		return BookmarkCreatedMsg{BookmarkName: bookmarkName, CommitID: commitID, WasMoved: true}
	}
}

// CreateBranchFromMain creates a new branch from main
// ticketKey is optional - if provided, it enables auto-transition to "In Progress"
func CreateBranchFromMain(svc *jj.Service, bookmarkName, ticketKey string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.CreateBranchFromMain(context.Background(), bookmarkName); err != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to create branch from main: %w", err)}
		}
		return BookmarkCreatedMsg{BookmarkName: bookmarkName, CommitID: "main", WasMoved: false, TicketKey: ticketKey}
	}
}

// DeleteBookmark deletes a bookmark
func DeleteBookmark(svc *jj.Service, bookmarkName string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.DeleteBookmark(context.Background(), bookmarkName); err != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to delete bookmark: %w", err)}
		}
		return BookmarkDeletedMsg{BookmarkName: bookmarkName}
	}
}

// FindBookmarkForCommit finds a bookmark from ancestors using BFS
func FindBookmarkForCommit(repo *models.Repository, commitIdx int) string {
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
