package util

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// KillGPGAgentResultMsg is the result of KillGPGAgentCmd.
type KillGPGAgentResultMsg struct {
	Err error
}

// KillGPGAgentCmd runs `gpgconf --kill gpg-agent`.
func KillGPGAgentCmd() tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("gpgconf", "--kill", "gpg-agent").CombinedOutput()
		if err != nil {
			msg := strings.TrimSpace(string(out))
			if msg != "" {
				return KillGPGAgentResultMsg{Err: fmt.Errorf("kill gpg-agent: %w: %s", err, msg)}
			}
			return KillGPGAgentResultMsg{Err: fmt.Errorf("kill gpg-agent: %w", err)}
		}
		return KillGPGAgentResultMsg{}
	}
}
