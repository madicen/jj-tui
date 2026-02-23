package graph

import tea "github.com/charmbracelet/bubbletea"

// Request is sent to the main model so it can run jj/git commands (main has jjService).
// The graph tab only holds UI state; it returns a Request as tea.Msg for actions.
type Request struct {
	// LoadChangedFiles loads changed files for the given commit (e.g. after j/k).
	LoadChangedFiles *string

	// Commit actions (main uses graph tab's current selection when processing).
	Checkout            bool
	Squash              bool
	Abandon             bool
	StartEditDescription bool
	NewCommit           bool
	StartRebaseMode     bool
	PerformRebase       bool // Enter in rebase mode; DestIndex in RebaseDestIndex
	RebaseDestIndex     int  // used when PerformRebase is true

	// Divergent: main loads dialog then switches view.
	ResolveDivergent *string // ChangeID

	// Bookmark/PR (main uses selection).
	CreateBookmark bool
	DeleteBookmark bool
	CreatePR       bool
	UpdatePR       bool

	// File operations (files pane focused).
	MoveFileUp   bool
	MoveFileDown bool
	RevertFile   bool
}

// Cmd returns a tea.Cmd that sends this request to the program.
func (r Request) Cmd() tea.Cmd {
	return func() tea.Msg { return r }
}
