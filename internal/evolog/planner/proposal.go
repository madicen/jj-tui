package planner

// CommitProposal is one atomic commit in a planned split sequence (dry-run manifest item).
// Hunks are opaque identifiers for now (e.g. unified-diff excerpts, path keys, or future AST spans).
type CommitProposal struct {
	Hunks      []string `json:"hunks"`
	Message    string   `json:"message"`
	SequenceID int      `json:"sequence_id"`
}
