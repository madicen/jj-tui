package model

// Messages represent user actions in the TUI.
// Tests should verify these messages are produced, not coordinates.

// TabSelectedMsg is emitted when a tab is clicked
type TabSelectedMsg struct {
	Tab ViewMode
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
	ActionRebase   ActionType = "rebase"
	ActionHelp     ActionType = "help"
)

// SelectionMode indicates what the user is selecting commits for
type SelectionMode int

const (
	SelectionNormal            SelectionMode = iota // Normal selection
	SelectionRebaseDestination                      // Selecting destination for rebase
)

// ViewMode represents different views in the TUI
type ViewMode int

const (
	ViewCommitGraph ViewMode = iota
	ViewPullRequests
	ViewTickets
	ViewBranches
	ViewSettings
	ViewHelp
	ViewCreatePR
	ViewEditDescription
	ViewCreateBookmark
	ViewGitHubLogin // GitHub Device Flow login
)

func (v ViewMode) String() string {
	switch v {
	case ViewCommitGraph:
		return "commit_graph"
	case ViewPullRequests:
		return "pull_requests"
	case ViewTickets:
		return "jira"
	case ViewBranches:
		return "branches"
	case ViewSettings:
		return "settings"
	case ViewHelp:
		return "help"
	case ViewCreatePR:
		return "create_pr"
	case ViewEditDescription:
		return "edit_description"
	case ViewCreateBookmark:
		return "create_bookmark"
	case ViewGitHubLogin:
		return "github_login"
	default:
		return "unknown"
	}
}
