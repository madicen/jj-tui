package tui

// Messages represent user actions in the TUI.
// Tests should verify these messages are produced, not coordinates.

// TabSelectedMsg is emitted when a tab is clicked
type TabSelectedMsg struct {
	Tab ViewMode
}

// CommitSelectedMsg is emitted when a commit is clicked
type CommitSelectedMsg struct {
	Index    int
	CommitID string
}

// ActionMsg is emitted when an action button is clicked
type ActionMsg struct {
	Action ActionType
}

// ActionType represents the type of action triggered
type ActionType string

const (
	ActionQuit     ActionType = "quit"
	ActionRefresh  ActionType = "refresh"
	ActionNewPR    ActionType = "new_pr"
	ActionCheckout ActionType = "checkout"
	ActionEdit     ActionType = "edit"
	ActionSquash   ActionType = "squash"
	ActionHelp     ActionType = "help"
)

// ViewMode represents different views in the TUI
type ViewMode int

const (
	ViewCommitGraph ViewMode = iota
	ViewPullRequests
	ViewJira
	ViewSettings
	ViewHelp
	ViewCreatePR
	ViewEditDescription
)

func (v ViewMode) String() string {
	switch v {
	case ViewCommitGraph:
		return "commit_graph"
	case ViewPullRequests:
		return "pull_requests"
	case ViewJira:
		return "jira"
	case ViewSettings:
		return "settings"
	case ViewHelp:
		return "help"
	case ViewCreatePR:
		return "create_pr"
	case ViewEditDescription:
		return "edit_description"
	default:
		return "unknown"
	}
}

