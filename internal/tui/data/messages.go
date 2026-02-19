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
type ServicesInitializedMsg struct {
	JJService     *jj.Service
	GitHubService *github.Service
	TicketService tickets.Service
	TicketError   error
	Repository    *internal.Repository
	GitHubInfo    string
	DemoMode      bool
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
