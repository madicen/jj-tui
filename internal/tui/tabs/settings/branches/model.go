package branches

import (
	"github.com/madicen/jj-tui/internal/config"
)

// Model represents the Branches settings sub-tab (branch limit).
type Model struct {
	branchLimit int
}

// NewModel creates a new Branches settings model with default state.
func NewModel() Model {
	return Model{branchLimit: 100}
}

// NewModelFromConfig creates a model initialized from config.
func NewModelFromConfig(cfg *config.Config) Model {
	m := NewModel()
	if cfg != nil {
		m.branchLimit = cfg.BranchLimit()
	}
	return m
}

// GetBranchLimit returns the branch limit (0 = all).
func (m *Model) GetBranchLimit() int {
	return m.branchLimit
}

// SetBranchLimit sets the branch limit.
func (m *Model) SetBranchLimit(n int) {
	if n < 0 {
		n = 0
	}
	if n > 500 {
		n = 500
	}
	m.branchLimit = n
}
