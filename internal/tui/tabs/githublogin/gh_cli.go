package githublogin

import (
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

// GhCLIAuthFinishedMsg is sent after `gh auth login` exits (success or failure).
type GhCLIAuthFinishedMsg struct {
	Err error
}

// GhAuthLoginCmd runs interactive `gh auth login` with the real terminal (suspends the TUI).
func GhAuthLoginCmd() tea.Cmd {
	if _, err := exec.LookPath("gh"); err != nil {
		return func() tea.Msg {
			return GhCLIAuthFinishedMsg{Err: err}
		}
	}
	c := exec.Command("gh", "auth", "login")
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return GhCLIAuthFinishedMsg{Err: err}
	})
}
