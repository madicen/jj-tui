package state

// ViewMode represents different views in the TUI.
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
	ViewGitHubLogin      // GitHub Device Flow login
	ViewBookmarkConflict // Bookmark conflict resolution dialog
	ViewDivergentCommit  // Divergent commit resolution dialog
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
	case ViewBookmarkConflict:
		return "bookmark_conflict"
	case ViewDivergentCommit:
		return "divergent_commit"
	default:
		return "unknown"
	}
}
