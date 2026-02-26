package data

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// RunJJInit runs `jj git init` in the current directory and optionally tracks main@origin.
// Returns a cmd that sends JJInitSuccessMsg or InitErrorMsg.
func RunJJInit() tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("jj", "git", "init")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return InitErrorMsg{
				Err:       fmt.Errorf("failed to initialize repository: %s", strings.TrimSpace(string(output))),
				NotJJRepo: true,
			}
		}
		trackCmd := exec.Command("jj", "bookmark", "track", "main@origin")
		_, _ = trackCmd.CombinedOutput()
		return JJInitSuccessMsg{}
	}
}
