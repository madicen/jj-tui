package model

import (
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/jj"
	"github.com/madicen/jj-tui/internal/models"
)

// Auto-refresh interval
const autoRefreshInterval = 2 * time.Second

// isGitHubAvailable returns true if GitHub functionality is available
// (either through a real service connection or demo mode)
func (m *Model) isGitHubAvailable() bool {
	return m.githubService != nil || m.demoMode
}

// openURL opens a URL in the default browser
func openURL(url string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "linux":
			cmd = exec.Command("xdg-open", url)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		default:
			return nil
		}
		_ = cmd.Start()
		return nil
	}
}

// isSelectedCommitValid returns true if selectedCommit points to a valid commit
func (m *Model) isSelectedCommitValid() bool {
	return m.repository != nil &&
		m.selectedCommit >= 0 &&
		m.selectedCommit < len(m.repository.Graph.Commits)
}

// refreshRepository starts a refresh of the repository data.
// Also refreshes PRs if GitHub is connected and tickets if ticket service is available.
func (m *Model) refreshRepository() tea.Cmd {
	m.statusMessage = "Refreshing..."
	m.loading = true
	var cmds []tea.Cmd
	cmds = append(cmds, m.loadRepository())
	if m.isGitHubAvailable() {
		cmds = append(cmds, m.loadPRs())
	}
	if m.ticketService != nil {
		cmds = append(cmds, m.loadTickets())
	}
	return tea.Batch(cmds...)
}

func If[T any](condition bool, trueCmd, falseCmd T) T {
	if condition {
		return trueCmd
	}
	return falseCmd
}

func (m *Model) createIsZoneClickedFunc(clickedZone *zone.ZoneInfo) func(string) bool {
	return func(clickedZoneID string) bool {
		return m.zoneManager.Get(clickedZoneID) == clickedZone
	}
}

// findCommitsWithEmptyDescriptions finds all commits from the selected commit
// back to main that have empty descriptions (excluding immutable/root commits)
func (m *Model) findCommitsWithEmptyDescriptions() []models.Commit {
	if m.repository == nil || !m.isSelectedCommitValid() {
		return nil
	}

	commits := m.repository.Graph.Commits
	var emptyDescCommits []models.Commit

	// Walk from selected commit back through parents until we hit an immutable commit
	visited := make(map[string]bool)
	queue := []int{m.selectedCommit}

	// Build index for parent lookup
	idToIndex := make(map[string]int)
	for i, c := range commits {
		idToIndex[c.ID] = i
		idToIndex[c.ChangeID] = i
	}

	for len(queue) > 0 {
		idx := queue[0]
		queue = queue[1:]

		if idx < 0 || idx >= len(commits) {
			continue
		}

		commit := commits[idx]
		if visited[commit.ID] {
			continue
		}
		visited[commit.ID] = true

		// Stop at immutable commits (like main)
		if commit.Immutable {
			continue
		}

		// Check if description is empty (just whitespace counts as empty)
		desc := strings.TrimSpace(commit.Description)
		if desc == "" || desc == "(no description)" {
			emptyDescCommits = append(emptyDescCommits, commit)
		}

		// Add parents to queue
		for _, parentID := range commit.Parents {
			if parentIdx, ok := idToIndex[parentID]; ok && !visited[commits[parentIdx].ID] {
				queue = append(queue, parentIdx)
			}
		}
	}

	return emptyDescCommits
}

// checkBookmarkNameExists checks if a bookmark name already exists in the repository
func (m *Model) checkBookmarkNameExists(name string) bool {
	if name == "" {
		return false
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}

	// Check against the full branch list (includes all local and remote branches)
	// This is the most comprehensive source
	for _, branch := range m.branchList {
		if branch.Name == name {
			return true
		}
	}

	// Also check against branches on commits in the graph
	// This serves as a backup if branchList isn't loaded yet
	if m.repository != nil {
		for _, commit := range m.repository.Graph.Commits {
			for _, branchName := range commit.Branches {
				if branchName == name {
					return true
				}
			}
		}
	}

	// Also check the existingBookmarks list (bookmarks that can be moved)
	for _, bookmark := range m.existingBookmarks {
		if bookmark == name {
			return true
		}
	}

	return false
}

// updateBookmarkNameExists updates the bookmarkNameExists flag based on current input
func (m *Model) updateBookmarkNameExists() {
	name := m.bookmarkNameInput.Value()

	// If sanitization is enabled, check the sanitized name
	if m.settingsSanitizeBookmarks {
		name = jj.SanitizeBookmarkName(name)
	}

	m.bookmarkNameExists = m.checkBookmarkNameExists(name)
}
