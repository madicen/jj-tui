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
	ViewCreateTicket // Create Ticket modal (from Tickets tab)
	ViewEditDescription
	ViewCreateBookmark
	ViewGitHubLogin      // GitHub login (device flow or GitHub CLI)
	ViewBookmarkConflict // Bookmark conflict resolution dialog
	ViewDivergentCommit  // Divergent commit resolution dialog
	ViewEvologSplit      // Experimental evolog-driven stack split (FAQ-style)
	ViewFileDiff         // Full-file diff for selected changed file (graph overlay)
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
	case ViewCreateTicket:
		return "create_ticket"
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
	case ViewEvologSplit:
		return "evolog_split"
	case ViewFileDiff:
		return "file_diff"
	default:
		return "unknown"
	}
}
