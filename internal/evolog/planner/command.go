package planner

// SplitCommand is one reversible step toward applying a CommitProposal (command pattern).
// Concrete implementations will wrap jj/git primitives; DryRun may build commands without executing.
type SplitCommand interface {
	// Describe returns a short human-readable label for previews and logs.
	Describe() string
	// DryRun when true must not mutate the repository.
	Execute(dryRun bool) error
	// Undo restores pre-step state when the engine supports rollback (optional per command).
	Undo() error
}
