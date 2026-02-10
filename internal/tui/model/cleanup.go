package model

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// startDeleteBookmarks initiates the bookmark deletion confirmation
func (m *Model) startDeleteBookmarks() {
	m.confirmingCleanup = "delete_bookmarks"
	m.statusMessage = "Press Y to confirm deletion of all bookmarks, or N to cancel"
}

// startAbandonOldCommits initiates the abandon old commits confirmation
func (m *Model) startAbandonOldCommits() {
	m.confirmingCleanup = "abandon_old_commits"
	m.statusMessage = "Press Y to confirm abandoning commits before origin/main, or N to cancel"
}

// deleteAllBookmarks executes the deletion of all bookmarks
func (m *Model) deleteAllBookmarks() tea.Cmd {
	if m.jjService == nil || m.repository == nil {
		return func() tea.Msg {
			return cleanupCompletedMsg{
				success: false,
				err:     fmt.Errorf("jj service or repository not initialized"),
			}
		}
	}

	jjSvc := m.jjService
	repo := m.repository

	return func() tea.Msg {
		ctx := context.Background()

		// Collect all bookmark names to delete from commits
		bookmarkMap := make(map[string]bool)
		for _, commit := range repo.Graph.Commits {
			for _, branch := range commit.Branches {
				bookmarkMap[branch] = true
			}
		}

		if len(bookmarkMap) == 0 {
			return cleanupCompletedMsg{
				success: true,
				message: "No bookmarks to delete",
			}
		}

		// Delete each bookmark
		var deletedCount int
		for bookmarkName := range bookmarkMap {
			err := jjSvc.DeleteBookmark(ctx, bookmarkName)
			if err == nil {
				deletedCount++
			}
		}

		return cleanupCompletedMsg{
			success: true,
			message: fmt.Sprintf("Deleted %d bookmarks", deletedCount),
		}
	}
}

// abandonCommitsBeforeOriginMain abandons all commits before origin/main
func (m *Model) abandonCommitsBeforeOriginMain() tea.Cmd {
	if m.jjService == nil || m.repository == nil {
		return func() tea.Msg {
			return cleanupCompletedMsg{
				success: false,
				err:     fmt.Errorf("jj service or repository not initialized"),
			}
		}
	}

	jjSvc := m.jjService
	repo := m.repository

	return func() tea.Msg {
		ctx := context.Background()

		// Use jj directly to find main@origin's change_id
		mainCommitID, err := jjSvc.GetRevisionChangeID(ctx, "main@origin")
		if err != nil || mainCommitID == "" {
			return cleanupCompletedMsg{
				success: false,
				err:     fmt.Errorf("could not find main@origin - make sure to track it first"),
			}
		}

		// Abandon all mutable commits (they are not ancestors of main@origin)
		var abandonedCount int
		for _, commit := range repo.Graph.Commits {
			// Don't abandon the working copy, immutable commits, or main itself
			if commit.IsWorking || commit.Immutable || commit.ChangeID == mainCommitID {
				continue
			}

			err := jjSvc.AbandonCommit(ctx, commit.ChangeID)
			if err == nil {
				abandonedCount++
			}
		}

		return cleanupCompletedMsg{
			success: true,
			message: fmt.Sprintf("Abandoned %d commits", abandonedCount),
		}
	}
}

// cancelCleanup cancels the current cleanup operation
func (m *Model) cancelCleanup() {
	m.confirmingCleanup = ""
	m.statusMessage = "Cleanup cancelled"
}

// confirmCleanup executes the confirmed cleanup operation
func (m *Model) confirmCleanup() tea.Cmd {
	switch m.confirmingCleanup {
	case "delete_bookmarks":
		m.confirmingCleanup = ""
		return m.deleteAllBookmarks()
	case "abandon_old_commits":
		m.confirmingCleanup = ""
		return m.abandonCommitsBeforeOriginMain()
	default:
		return nil
	}
}
