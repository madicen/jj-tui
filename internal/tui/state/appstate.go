package state

import (
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/github"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tickets"
)

// AppState holds shared application state and services. The main model owns it;
// submodels receive *AppState and can read/mutate it. Config is the current app
// config (reloaded after settings save).
type AppState struct {
	// Data and services
	Repository    *internal.Repository
	JJService     *jj.Service
	GitHubService *github.Service
	TicketService tickets.Service
	Config        *config.Config

	// UI/routing state (submodels can read and set these)
	ViewMode      ViewMode
	StatusMessage string
	Loading       bool // Busy overlay (spinner): first PR/tickets fetch, create PR, fetch-all, etc.
	DemoMode      bool
	GithubInfo    string

	// DefaultBranch is the resolved default branch of the GitHub repository (e.g. "main",
	// "master", "trunk"). Populated by LoadAuxServicesCmd after the GitHub service is
	// constructed. May be empty when no GitHub service is available, the repo isn't on
	// GitHub, or the lookup hasn't completed yet — callers should fall back to "main" in
	// that case to preserve the legacy hardcoded behavior. Used by the Create PR form to
	// pick a base branch that actually exists on the remote.
	DefaultBranch string

	// PRsLoadedOnce is set after the first GitHub PR list load completes (success or error).
	PRsLoadedOnce bool
	// TicketsLoadedOnce is set after the first ticket list load completes (success or error).
	TicketsLoadedOnce bool
	// BranchRemoteFetchPending: branches tab started "fetch all remotes"; main batches spinner with the cmd.
	BranchRemoteFetchPending bool
}

// HasRepository returns true if repository data is loaded.
func (a *AppState) HasRepository() bool { return a.Repository != nil }

// HasJJ returns true if the jj service is available.
func (a *AppState) HasJJ() bool { return a.JJService != nil }
