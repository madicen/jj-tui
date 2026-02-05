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

// startTrackOriginMain initiates the track origin/main process
func (m *Model) startTrackOriginMain() {
	m.confirmingCleanup = "track_origin_main"
	m.statusMessage = "Fetching origin/main..."
	m.loading = true
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

		// Find the commit ID for origin/main
		var mainCommitID string
		for _, commit := range repo.Graph.Commits {
			for _, branch := range commit.Branches {
				if branch == "origin/main" {
					mainCommitID = commit.ChangeID
					break
				}
			}
			if mainCommitID != "" {
				break
			}
		}

		if mainCommitID == "" {
			return cleanupCompletedMsg{
				success: false,
				err:     fmt.Errorf("could not find origin/main in repository"),
			}
		}

		// Collect commits before origin/main and abandon them
		var abandonedCount int
		for _, commit := range repo.Graph.Commits {
			// Don't abandon the working copy or main itself
			if commit.IsWorking || commit.ChangeID == mainCommitID {
				continue
			}

			// Check if this commit has origin/main
			hasOriginMain := false
			for _, branch := range commit.Branches {
				if branch == "origin/main" {
					hasOriginMain = true
					break
				}
			}

			// Abandon if it's not main and doesn't have origin/main
			if !hasOriginMain {
				err := jjSvc.AbandonCommit(ctx, commit.ChangeID)
				if err == nil {
					abandonedCount++
				}
			}
		}

		return cleanupCompletedMsg{
			success: true,
			message: fmt.Sprintf("Abandoned %d commits", abandonedCount),
		}
	}
}

// trackOriginMain fetches from origin to update tracking
func (m *Model) trackOriginMain() tea.Cmd {
	if m.jjService == nil {
		return func() tea.Msg {
			return cleanupCompletedMsg{
				success: false,
				err:     fmt.Errorf("jj service not initialized"),
			}
		}
	}

	return func() tea.Msg {
		ctx := context.Background()
		fetchOut, err := m.jjService.FetchFromGit(ctx)
		if err != nil {
			return cleanupCompletedMsg{
				success: false,
				err:     fmt.Errorf("failed to fetch from origin: %w", err),
			}
		}

		// After fetching, we'll return success - the repository will be reloaded
		// which will show the updated remote bookmarks
		message := "Fetched remote updates. Remote bookmarks have been updated."
		if fetchOut != "" {
			message += "\n" + fetchOut
		}

		return cleanupCompletedMsg{
			success: true,
			message: message,
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
	case "track_origin_main":
		m.confirmingCleanup = ""
		return m.trackOriginMain()
	default:
		return nil
	}
}

// Helper function to check if a slice contains a string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
