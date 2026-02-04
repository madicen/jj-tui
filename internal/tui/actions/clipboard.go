package actions

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// CopyToClipboard copies text to the system clipboard
func CopyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd

		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("pbcopy")
		case "linux":
			if _, err := exec.LookPath("xclip"); err == nil {
				cmd = exec.Command("xclip", "-selection", "clipboard")
			} else if _, err := exec.LookPath("xsel"); err == nil {
				cmd = exec.Command("xsel", "--clipboard", "--input")
			} else {
				return ClipboardCopiedMsg{Success: false, Err: fmt.Errorf("no clipboard utility found")}
			}
		case "windows":
			cmd = exec.Command("clip")
		default:
			return ClipboardCopiedMsg{Success: false, Err: fmt.Errorf("clipboard not supported on %s", runtime.GOOS)}
		}

		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err != nil {
			return ClipboardCopiedMsg{Success: false, Err: err}
		}
		return ClipboardCopiedMsg{Success: true}
	}
}

