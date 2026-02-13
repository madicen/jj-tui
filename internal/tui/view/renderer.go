// Package view provides rendering functions for the TUI views.
package view

import (
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/models"
)

// Renderer provides rendering capabilities with zone support
type Renderer struct {
	Zone *zone.Manager
}

// New creates a new Renderer
func New(z *zone.Manager) *Renderer {
	// Ensure zone manager is never nil
	if z == nil {
		z = zone.New()
	}
	return &Renderer{Zone: z}
}

// Mark safely wraps the zone manager's Mark function
// If Zone is nil (shouldn't happen but defensive), returns content unchanged
func (r *Renderer) Mark(id, content string) string {
	if r == nil || r.Zone == nil {
		return content
	}
	return r.Zone.Mark(id, content)
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
	GraphFocused       bool            // True if graph pane has focus
	SelectedFile       int             // Index of selected file in changed files list
}

// GraphResult contains the split rendering for commit graph view
type GraphResult struct {
	GraphContent string // Commit graph (scrollable)
	ActionsBar   string // Actions buttons (fixed in middle)
	FilesContent string // Changed files (scrollable)
	FullContent  string // Full content for non-split views
}

// PRData contains data needed for PR rendering
type PRData struct {
	Repository    *models.Repository
	SelectedPR    int
	GithubService bool // whether GitHub is connected
	Width         int  // viewport width for separator lines
}

// PRResult contains the split rendering for PRs
type PRResult struct {
	FixedHeader    string // Details section that stays fixed
	ScrollableList string // List that scrolls
	FullContent    string // Full content for non-split views
}

// JiraData contains data needed for Jira/Tickets rendering
// TicketTransition represents a possible status transition
type TicketTransition struct {
	ID   string
	Name string
}

type JiraData struct {
	Tickets              []JiraTicket
	SelectedTicket       int
	JiraService          bool               // whether a ticket service is connected
	ProviderName         string             // name of the ticket provider (e.g., "Jira", "Codecks")
	AvailableTransitions []TicketTransition // available status transitions for selected ticket
	TransitionInProgress bool               // whether a transition is currently being executed
	StatusChangeMode     bool               // whether status change buttons are expanded
	Width                int                // viewport width for separator lines
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
	DisplayKey  string // Short key for display (e.g., "#51" for Codecks, "PROJ-123" for Jira)
	Summary     string
	Status      string
	Type        string
	Priority    string
	Description string
}

// SettingsTab represents the active tab within settings
type SettingsTab int

const (
	SettingsTabGitHub SettingsTab = iota
	SettingsTabJira
	SettingsTabCodecks
	SettingsTabTickets // New tab for ticket provider selection
	SettingsTabBranches
	SettingsTabAdvanced
)

// SettingsData contains data needed for settings rendering
type SettingsData struct {
	Inputs         []InputView
	FocusedField   int
	GithubService  bool
	JiraService    bool
	HasLocalConfig bool        // True if .jj-tui.json exists in current directory
	ConfigSource   string      // Path to the currently loaded config
	ActiveTab      SettingsTab // Which settings tab is active

	// GitHub filter toggles
	ShowMergedPRs     bool
	ShowClosedPRs     bool
	OnlyMyPRs         bool
	PRLimit           int
	PRRefreshInterval int // in seconds, 0 = disabled

	// Ticket provider settings
	TicketProvider         string // Current provider: "jira", "codecks", "github_issues", or ""
	TicketProviderName     string // Display name of current provider
	AutoInProgressOnBranch bool   // Auto-transition ticket to "In Progress" when creating branch

	// Provider availability (based on what's configured)
	JiraConfigured         bool
	CodecksConfigured      bool
	GitHubIssuesConfigured bool // true if GitHub is connected (can use GitHub Issues)

	// Branch settings
	BranchLimit       int  // Max branches to calculate stats for
	SanitizeBookmarks bool // Auto-fix invalid bookmark names

	// Advanced tab state
	ConfirmingCleanup string // "" = not confirming, "delete_bookmarks", "abandon_old_commits"
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
	NameExists        bool     // True if the entered name matches an existing bookmark
}

// BookmarkConflictData contains data needed for bookmark conflict resolution view
type BookmarkConflictData struct {
	BookmarkName   string // Name of the conflicted bookmark
	LocalCommitID  string // Local commit ID
	RemoteCommitID string // Remote commit ID
	LocalSummary   string // Local commit summary
	RemoteSummary  string // Remote commit summary
	SelectedOption int    // 0=Keep Local, 1=Reset to Remote
}

// DivergentCommitData contains data needed for divergent commit resolution view
type DivergentCommitData struct {
	ChangeID    string   // The change ID that's divergent
	CommitIDs   []string // All commit hashes sharing this change ID
	Summaries   []string // Summary of each divergent commit
	SelectedIdx int      // Which commit to keep (0-indexed)
}
