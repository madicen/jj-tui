package help

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/actions"
)

// ExecuteRequest runs the help request (e.g. copy command to clipboard).
// Returns (cmd, statusMsg). When statusMsg is non-empty, the model should set it and return the cmd.
func ExecuteRequest(r Request) (tea.Cmd, string) {
	if r.CopyCommand == "" {
		return nil, ""
	}
	return actions.CopyToClipboard(r.CopyCommand), "Copied: " + r.CopyCommand
}
