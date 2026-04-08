package filediff

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/integrations/jj"
)

// FileDiffLoadedMsg carries jj diff output for the graph file-diff modal.
type FileDiffLoadedMsg struct {
	Seq  int
	Text string
	Err  error
}

// LoadFileDiffCmd runs jj diff for one file at a revision and sends FileDiffLoadedMsg.
func LoadFileDiffCmd(svc *jj.Service, seq int, changeID, path string) tea.Cmd {
	if svc == nil || seq <= 0 {
		return nil
	}
	ch := strings.TrimSpace(changeID)
	p := strings.TrimSpace(path)
	if ch == "" || p == "" {
		return nil
	}
	return func() tea.Msg {
		text, err := svc.DiffRevisionFile(context.Background(), ch, p)
		if err != nil {
			return FileDiffLoadedMsg{Seq: seq, Err: err}
		}
		t := strings.TrimSpace(text)
		if t == "" {
			text = "(no changes in diff output for this path vs parents)"
		}
		return FileDiffLoadedMsg{Seq: seq, Text: text}
	}
}
