package help

import tea "github.com/charmbracelet/bubbletea"

// Request is sent to the main model for Help tab actions (e.g. copy to clipboard).
type Request struct {
	CopyCommand string // When set, main copies this command string to clipboard
}

// Cmd returns a tea.Cmd that sends this request.
func (r Request) Cmd() tea.Cmd {
	return func() tea.Msg { return r }
}
