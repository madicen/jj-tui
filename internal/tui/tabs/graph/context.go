package graph

import (
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/jj"
)

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
}

// IsSelectedCommitValid returns true if the selected commit index is valid.
func (c *RequestContext) IsSelectedCommitValid() bool {
	if c.Repository == nil {
		return false
	}
	return c.SelectedCommit >= 0 && c.SelectedCommit < len(c.Repository.Graph.Commits)
}
