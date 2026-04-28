package planner

import "fmt"

// EvologPlanner maps a raw diff into an ordered list of atomic commit proposals and validates them.
// Implementations may use LLM, heuristics, or AST-backed strategies (see ChunkStrategy).
type EvologPlanner interface {
	Analyze(diff string) ([]CommitProposal, error)
	Validate(proposals []CommitProposal) error
}

// ChunkStrategy names a diff-chunking backend (strategy pattern). Analyze implementations
// may delegate to one or more strategies.
type ChunkStrategy interface {
	Name() string
}

// StubPlanner returns no proposals until LLM/heuristic strategies are wired. Use for tests
// and incremental UI integration.
type StubPlanner struct{}

func (StubPlanner) Analyze(diff string) ([]CommitProposal, error) {
	if diff == "" {
		return nil, nil
	}
	return nil, fmt.Errorf("planner: Analyze not implemented (diff length %d)", len(diff))
}

func (StubPlanner) Validate(proposals []CommitProposal) error {
	return ValidateProposals(proposals)
}
