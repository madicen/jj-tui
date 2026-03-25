package graph

import (
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/state"
)

// ContextProvider is implemented by the main model so the graph tab can build context without depending on model package.
type ContextProvider interface {
	GetJJService() *jj.Service
	GetRepository() *internal.Repository
	GetSelectedCommit() int
	GetRebaseSourceCommit() int
	GetChangedFiles() []jj.ChangedFile
	GetChangedFilesCommitID() string
	GetSelectedFile() int
	IsGraphFocused() bool
	IsGitHubAvailable() bool
	GetCreatePRBranch() string
	IsDemoMode() bool
}

// BuildRequestContextFrom builds RequestContext from a provider (e.g. main model).
func BuildRequestContextFrom(p ContextProvider) *RequestContext {
	if p == nil || p.GetRepository() == nil {
		return nil
	}
	return BuildRequestContext(&ContextInput{
		JJService:            p.GetJJService(),
		Repository:           p.GetRepository(),
		SelectedCommit:       p.GetSelectedCommit(),
		RebaseSourceCommit:   p.GetRebaseSourceCommit(),
		ChangedFiles:         p.GetChangedFiles(),
		ChangedFilesCommitID: p.GetChangedFilesCommitID(),
		SelectedFile:         p.GetSelectedFile(),
		GraphFocused:         p.IsGraphFocused(),
		GitHubAvailable:      p.IsGitHubAvailable(),
		CreatePRBranch:       p.GetCreatePRBranch(),
		DemoMode:             p.IsDemoMode(),
	})
}

// RequestContext is passed from the main model so the graph tab can execute
// requests (run jj commands) without depending on the model package.
type RequestContext struct {
	JJService            *jj.Service
	Repository           *internal.Repository
	SelectedCommit       int
	RebaseSourceCommit   int
	ChangedFiles         []jj.ChangedFile
	ChangedFilesCommitID string
	SelectedFile         int
	GraphFocused         bool
	GitHubAvailable      bool
	CreatePRBranch       string // branch that would be used for Create PR for selected commit (to block main/master)
	DemoMode             bool
}

// ContextInput is the data needed to build a RequestContext. Main passes this from its state.
type ContextInput struct {
	JJService            *jj.Service
	Repository           *internal.Repository
	SelectedCommit       int
	RebaseSourceCommit   int
	ChangedFiles         []jj.ChangedFile
	ChangedFilesCommitID string
	SelectedFile         int
	GraphFocused         bool
	GitHubAvailable      bool
	CreatePRBranch       string
	DemoMode             bool
}

// BuildRequestContext builds RequestContext from input. The graph package owns what context it needs.
func BuildRequestContext(input *ContextInput) *RequestContext {
	if input == nil || input.Repository == nil {
		return nil
	}
	return &RequestContext{
		JJService:            input.JJService,
		Repository:           input.Repository,
		SelectedCommit:       input.SelectedCommit,
		RebaseSourceCommit:   input.RebaseSourceCommit,
		ChangedFiles:         input.ChangedFiles,
		ChangedFilesCommitID: input.ChangedFilesCommitID,
		SelectedFile:         input.SelectedFile,
		GraphFocused:         input.GraphFocused,
		GitHubAvailable:      input.GitHubAvailable,
		CreatePRBranch:       input.CreatePRBranch,
		DemoMode:             input.DemoMode,
	}
}

// IsSelectedCommitValid returns true if the selected commit index is valid.
func (c *RequestContext) IsSelectedCommitValid() bool {
	if c.Repository == nil {
		return false
	}
	return c.SelectedCommit >= 0 && c.SelectedCommit < len(c.Repository.Graph.Commits)
}

// BuildRequestContextFromApp builds RequestContext from app state and graph model.
// Used when the graph tab processes requests internally (Update(msg, app)).
func BuildRequestContextFromApp(app *state.AppState, m *GraphModel) *RequestContext {
	if app == nil || app.Repository == nil {
		return nil
	}
	githubAvailable := app.GitHubService != nil || app.DemoMode
	return BuildRequestContext(&ContextInput{
		JJService:            app.JJService,
		Repository:           app.Repository,
		SelectedCommit:       m.GetSelectedCommit(),
		RebaseSourceCommit:   m.GetRebaseSourceCommit(),
		ChangedFiles:         m.GetChangedFiles(),
		ChangedFilesCommitID: m.GetChangedFilesCommitID(),
		SelectedFile:         m.GetSelectedFile(),
		GraphFocused:         m.IsGraphFocused(),
		GitHubAvailable:      githubAvailable,
		CreatePRBranch:       m.GetCreatePRBranch(),
		DemoMode:             app.DemoMode,
	})
}
