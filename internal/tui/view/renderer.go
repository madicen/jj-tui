// Package view provides rendering functions for the TUI views.
package view

import (
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen-utilities/jj-tui/v2/internal/models"
)

// Renderer provides rendering capabilities with zone support
type Renderer struct {
	Zone *zone.Manager
}

// New creates a new Renderer
func New(z *zone.Manager) *Renderer {
	return &Renderer{Zone: z}
}

// ChangedFile represents a file changed in a commit
type ChangedFile struct {
	Path   string
	Status string // M=modified, A=added, D=deleted
}

// GraphData contains data needed for commit graph rendering
type GraphData struct {
	Repository         *models.Repository
	SelectedCommit     int
	InRebaseMode       bool            // True when selecting rebase destination
	RebaseSourceCommit int             // Index of commit being rebased
	OpenPRBranches     map[string]bool // Map of branch names that have open PRs
	CommitPRBranch     map[int]string  // Maps commit index to PR branch it can push to (including descendants)
	CommitBookmark     map[int]string  // Maps commit index to bookmark it can create a PR with (including descendants)
	ChangedFiles       []ChangedFile   // Changed files for the selected commit
}

// PRData contains data needed for PR rendering
type PRData struct {
	Repository    *models.Repository
	SelectedPR    int
	GithubService bool // whether GitHub is connected
}

// PRResult contains the split rendering for PRs
type PRResult struct {
	FixedHeader    string // Details section that stays fixed
	ScrollableList string // List that scrolls
	FullContent    string // Full content for non-split views
}

// JiraData contains data needed for Jira rendering
type JiraData struct {
	Tickets        []JiraTicket
	SelectedTicket int
	JiraService    bool // whether Jira is connected
}

// JiraResult contains the split rendering for Jira
type JiraResult struct {
	FixedHeader    string // Details section that stays fixed
	ScrollableList string // List that scrolls
	FullContent    string // Full content for non-split views
}

// JiraTicket represents a Jira ticket for rendering
type JiraTicket struct {
	Key         string
	Summary     string
	Status      string
	Type        string
	Priority    string
	Description string
}

// SettingsData contains data needed for settings rendering
type SettingsData struct {
	Inputs        []InputView
	FocusedField  int
	GithubService bool
	JiraService   bool
}

// InputView represents a text input for rendering
type InputView struct {
	View string
}

// DescriptionData contains data needed for description editing
type DescriptionData struct {
	Repository      *models.Repository
	EditingCommitID string
	InputView       string
}

// CreatePRData contains data needed for create PR view
type CreatePRData struct {
	Repository     *models.Repository
	SelectedCommit int
	GithubService  bool
	TitleInput     string
	BodyInput      string
	HeadBranch     string
	BaseBranch     string
	FocusedField   int // 0=title, 1=body
}

// BookmarkData contains data needed for bookmark creation view
type BookmarkData struct {
	Repository        *models.Repository
	CommitIndex       int // -1 means creating new branch from main
	NameInput         string
	ExistingBookmarks []string // List of existing bookmarks that can be moved
	SelectedBookmark  int      // Index of selected existing bookmark (-1 for new)
	FromJira          bool     // True if creating bookmark from Jira ticket
	JiraTicketKey     string   // Jira ticket key when FromJira is true
}
