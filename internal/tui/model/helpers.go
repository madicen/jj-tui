package model

import (
	"os/exec"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Auto-refresh interval
const autoRefreshInterval = 2 * time.Second

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
// Also refreshes PRs if GitHub is connected.
func (m *Model) refreshRepository() tea.Cmd {
	m.statusMessage = "Refreshing..."
	m.loading = true
	if m.githubService != nil {
		return tea.Batch(m.loadRepository(), m.loadPRs())
	}
	return m.loadRepository()
}

