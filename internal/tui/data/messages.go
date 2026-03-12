package data

import (
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/github"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tickets"
)

// InitErrorMsg is sent when initialization fails (e.g. not a jj repo). Main converts to model error.
type InitErrorMsg struct {
	Err         error
	NotJJRepo   bool
	CurrentPath string
}

// ServicesInitializedMsg is sent when jj, GitHub, and ticket services are initialized.
// Deprecated: initialization is now two-phase (RepoReadyMsg then AuxServicesReadyMsg).
type ServicesInitializedMsg struct {
	JJService     *jj.Service
	GitHubService *github.Service
	TicketService tickets.Service
	TicketError   error
	Repository    *internal.Repository
	GitHubInfo    string
	DemoMode      bool
}

// RepoReadyMsg is sent as soon as jj service and repository are ready so the UI can show the graph.
// A follow-up LoadAuxServicesCmd continues loading GitHub and ticket services in the background.
type RepoReadyMsg struct {
	JJService         *jj.Service
	Repository        *internal.Repository
	DemoMode          bool
	Owner             string // for GitHub/ticket; may be empty
	RepoName          string
	GitHubInfoFromURL string // e.g. "repo=owner/name (no token)" or "no remote configured"
}

// AuxServicesReadyMsg is sent after GitHub and ticket services are ready (after RepoReadyMsg).
type AuxServicesReadyMsg struct {
	GitHubService *github.Service
	TicketService tickets.Service
	TicketError   error
	GitHubInfo    string
}

// RepositoryLoadedMsg is sent when repository data is loaded (refresh).
type RepositoryLoadedMsg struct {
	Repository *internal.Repository
}

// SilentRepositoryLoadedMsg is for background refresh (no status update).
type SilentRepositoryLoadedMsg struct {
	Repository *internal.Repository
}

// JJInitSuccessMsg is sent when jj git init succeeds.
type JJInitSuccessMsg struct{}
