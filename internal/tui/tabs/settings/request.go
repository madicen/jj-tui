package settings

import tea "github.com/charmbracelet/bubbletea"

// Request is sent to the main model for Settings tab actions.
type Request struct {
	Cancel            bool // Leave settings without saving
	SaveSettings      bool // Save settings (e.g. ctrl+s / enter on last field)
	SaveSettingsLocal bool // Save to local .jj-tui.json (ctrl+l)
}

// Cmd returns a tea.Cmd that sends this request.
func (r Request) Cmd() tea.Cmd {
	return func() tea.Msg { return r }
}
