package prs

import "github.com/madicen/jj-tui/internal"

// RequestContext is passed from the main model so the PRs tab can validate
// and execute requests without depending on the model package.
type RequestContext struct {
	Repository   *internal.Repository
	SelectedPR   int
	GitHubOK     bool // whether GitHub service is available
	DemoMode     bool
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
