package branches

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/state"
)

// ContextProvider is implemented by the main model so the Branches tab can build context without depending on model package.
type ContextProvider interface {
	GetBranches() []internal.Branch
	GetSelectedBranch() int
	GetJJService() *jj.Service
}

// BuildRequestContextFromApp builds RequestContext from app state and the branches tab model (for UpdateWithApp flow).
func BuildRequestContextFromApp(app *state.AppState, m *Model) *RequestContext {
	if app == nil || m == nil {
		return nil
	}
	return BuildRequestContext(&ContextInput{
		BranchList:     m.GetBranches(),
		SelectedBranch: m.GetSelectedBranch(),
		JJService:      app.JJService,
	})
}

// BuildRequestContextFrom builds RequestContext from a provider (e.g. main model).
func BuildRequestContextFrom(p ContextProvider) *RequestContext {
	if p == nil {
		return nil
	}
	return BuildRequestContext(&ContextInput{
		BranchList:     p.GetBranches(),
		SelectedBranch: p.GetSelectedBranch(),
		JJService:      p.GetJJService(),
	})
}

// EnterTabProvider is implemented by main for EnterTab (status + load cmd).
type EnterTabProvider interface {
	GetJJService() *jj.Service
	GetBranchLimit() int
}

// EnterTab returns status message and load command when navigating to the Branches tab.
func EnterTab(p EnterTabProvider) (status string, cmd tea.Cmd) {
	status = EnterTabStatus()
	if p == nil || p.GetJJService() == nil {
		return status, nil
	}
	return status, LoadBranchesCmd(p.GetJJService(), p.GetBranchLimit())
}

// RequestContext is passed from the main model so the Branches tab can validate
// and execute requests without depending on the model package.
type RequestContext struct {
	BranchList     []internal.Branch
	SelectedBranch int
	JJService      *jj.Service
}

// ContextInput is the data needed to build a RequestContext. Main passes this from its state.
type ContextInput struct {
	BranchList     []internal.Branch
	SelectedBranch int
	JJService      *jj.Service
}

// BuildRequestContext builds RequestContext from input. The Branches tab owns what context it needs.
func BuildRequestContext(input *ContextInput) *RequestContext {
	if input == nil {
		return nil
	}
	return &RequestContext{
		BranchList:     input.BranchList,
		SelectedBranch: input.SelectedBranch,
		JJService:      input.JJService,
	}
}

// EnterTabStatus returns the status message when navigating to the Branches tab.
func EnterTabStatus() string {
	return "Loading branches..."
}

// SelectedBranchValid returns true if SelectedBranch is in range.
func (c *RequestContext) SelectedBranchValid() bool {
	return c.SelectedBranch >= 0 && c.SelectedBranch < len(c.BranchList)
}

// SelectedBranchData returns the selected branch or nil.
func (c *RequestContext) SelectedBranchData() *internal.Branch {
	if !c.SelectedBranchValid() {
		return nil
	}
	b := c.BranchList[c.SelectedBranch]
	return &b
}
