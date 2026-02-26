package branches

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
)

// BranchActionMsg is sent when a branch action completes (track, untrack, restore, delete, push, fetch).
type BranchActionMsg struct {
	Action string // "track", "untrack", "restore", "delete", "push", "fetch"
	Branch string
	Err    error
}

// BookmarkConflictInfoMsg contains info about a conflicted bookmark.
type BookmarkConflictInfoMsg struct {
	BookmarkName  string
	LocalID       string
	RemoteID      string
	LocalSummary  string
	RemoteSummary string
	Err           error
}

// BranchesLoadedMsg is sent when branches have been loaded (or load failed with Err).
type BranchesLoadedMsg struct {
	Branches []internal.Branch
	Err      error
}

// Request is sent to the main model to run branch actions (main has jjService, etc.).
type Request struct {
	TrackBranch             bool
	UntrackBranch           bool
	RestoreLocalBranch      bool
	DeleteBranchBookmark    bool
	PushBranch              bool
	FetchAll                bool
	ResolveBookmarkConflict bool
}

// Cmd returns a tea.Cmd that sends this request.
func (r Request) Cmd() tea.Cmd {
	return func() tea.Msg { return r }
}

// Effect types: the tab sends these to the main model so it can update app state and status.

// ApplyBranchesLoadedEffect tells main to set status and optionally update bookmark conflict sources.
type ApplyBranchesLoadedEffect struct {
	Err                  error
	StatusMessage        string
	InCreateBookmarkView bool
}

// ApplyBranchActionEffect tells main to set status/error after a branch action and run reload cmds.
type ApplyBranchActionEffect struct {
	Err           error
	StatusMessage string
}

func (e ApplyBranchesLoadedEffect) Cmd() tea.Cmd {
	return func() tea.Msg { return e }
}

func (e ApplyBranchActionEffect) Cmd() tea.Cmd {
	return func() tea.Msg { return e }
}

// BranchesLoadedInput is the context main sends when forwarding BranchesLoadedMsg.
type BranchesLoadedInput struct {
	BranchesLoadedMsg
	InCreateBookmarkView bool
	HasError             bool
}
