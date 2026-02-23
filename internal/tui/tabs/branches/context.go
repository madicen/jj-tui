package branches

import "github.com/madicen/jj-tui/internal"

// RequestContext is passed from the main model so the Branches tab can validate
// and execute requests without depending on the model package.
type RequestContext struct {
	BranchList     []internal.Branch
	SelectedBranch int
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
