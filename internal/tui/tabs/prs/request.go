package prs

import tea "github.com/charmbracelet/bubbletea"

// Request is sent to the main model to run PR actions (main has githubService, openURL, etc.).
type Request struct {
	OpenInBrowser bool
	MergePR       bool
	ClosePR       bool
}

// Cmd returns a tea.Cmd that sends this request.
func (r Request) Cmd() tea.Cmd {
	return func() tea.Msg { return r }
}
