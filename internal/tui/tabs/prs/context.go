package prs

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/github"
	"github.com/madicen/jj-tui/internal/tui/state"
)

// ContextProvider is implemented by the main model so the PRs tab can build context without depending on model package.
type ContextProvider interface {
	GetRepository() *internal.Repository
	GetSelectedPR() int
	IsGitHubAvailable() bool
	IsDemoMode() bool
	GetGitHubService() *github.Service
	GetGitHubInfo() string
}

// BuildRequestContextFromApp builds RequestContext from app state and the PRs tab model (for UpdateWithApp flow).
func BuildRequestContextFromApp(app *state.AppState, m *Model) *RequestContext {
	if app == nil || m == nil {
		return nil
	}
	githubOK := app.GitHubService != nil
	return BuildRequestContext(&ContextInput{
		Repository:    app.Repository,
		SelectedPR:    m.GetSelectedPR(),
		GitHubOK:      githubOK,
		DemoMode:      app.DemoMode,
		GitHubService: app.GitHubService,
		GitHubInfo:    app.GithubInfo,
	})
}

// BuildRequestContextFrom builds RequestContext from a provider (e.g. main model).
func BuildRequestContextFrom(p ContextProvider) *RequestContext {
	if p == nil {
		return nil
	}
	return BuildRequestContext(&ContextInput{
		Repository:    p.GetRepository(),
		SelectedPR:    p.GetSelectedPR(),
		GitHubOK:      p.IsGitHubAvailable(),
		DemoMode:      p.IsDemoMode(),
		GitHubService: p.GetGitHubService(),
		GitHubInfo:    p.GetGitHubInfo(),
	})
}

// EnterTabProvider is implemented by main for EnterTab (status + load cmd).
type EnterTabProvider interface {
	GetRepository() *internal.Repository
	IsGitHubAvailable() bool
	GetGitHubService() *github.Service
	GetGitHubInfo() string
	IsDemoMode() bool
}

// EnterTab returns status message and optional load command when navigating to the PRs tab.
func EnterTab(p EnterTabProvider) (status string, cmd tea.Cmd) {
	status = EnterTabStatus(p.IsGitHubAvailable())
	if !p.IsGitHubAvailable() {
		return status, nil
	}
	existing := 0
	if p.GetRepository() != nil {
		existing = len(p.GetRepository().PRs)
	}
	return status, LoadPRsCmd(p.GetGitHubService(), p.GetGitHubInfo(), p.IsDemoMode(), existing)
}

// RequestContext is passed from the main model so the PRs tab can validate
// and execute requests without depending on the model package.
type RequestContext struct {
	Repository    *internal.Repository
	SelectedPR    int
	GitHubOK      bool // whether GitHub service is available
	DemoMode      bool
	GitHubService *github.Service
	GitHubInfo    string
}

// ContextInput is the data needed to build a RequestContext. Main passes this from its state.
type ContextInput struct {
	Repository    *internal.Repository
	SelectedPR    int
	GitHubOK      bool
	DemoMode      bool
	GitHubService *github.Service
	GitHubInfo    string
}

// BuildRequestContext builds RequestContext from input. The PRs tab owns what context it needs.
func BuildRequestContext(input *ContextInput) *RequestContext {
	if input == nil {
		return nil
	}
	return &RequestContext{
		Repository:    input.Repository,
		SelectedPR:    input.SelectedPR,
		GitHubOK:      input.GitHubOK,
		DemoMode:      input.DemoMode,
		GitHubService: input.GitHubService,
		GitHubInfo:    input.GitHubInfo,
	}
}

// EnterTabStatus returns the status message when navigating to the PRs tab.
func EnterTabStatus(githubOK bool) string {
	if githubOK {
		return "Loading PRs..."
	}
	return "GitHub service not initialized"
}

// SelectedPRValid returns true if SelectedPR is in range and repository has PRs.
func (c *RequestContext) SelectedPRValid() bool {
	if c.Repository == nil {
		return false
	}
	return c.SelectedPR >= 0 && c.SelectedPR < len(c.Repository.PRs)
}

// SelectedPRData returns the selected PR or nil.
func (c *RequestContext) SelectedPRData() *internal.GitHubPR {
	if !c.SelectedPRValid() {
		return nil
	}
	pr := c.Repository.PRs[c.SelectedPR]
	return &pr
}
