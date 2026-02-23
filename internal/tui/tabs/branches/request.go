package branches

import tea "github.com/charmbracelet/bubbletea"

// Request is sent to the main model to run branch actions (main has jjService, etc.).
type Request struct {
	TrackBranch          bool
	UntrackBranch        bool
	RestoreLocalBranch   bool
	DeleteBranchBookmark bool
	PushBranch           bool
	FetchAll             bool
	ResolveBookmarkConflict bool
}

// Cmd returns a tea.Cmd that sends this request.
func (r Request) Cmd() tea.Cmd {
	return func() tea.Msg { return r }
}
