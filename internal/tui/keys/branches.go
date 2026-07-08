package keys

import "github.com/charmbracelet/bubbles/key"

// ScopeBranches is the config prefix for branches tab shortcuts.
const ScopeBranches = "branches"

// BranchesKeyMap holds the branches tab's keybindings.
type BranchesKeyMap struct {
	MoveDown        key.Binding
	MoveUp          key.Binding
	TrackByName     key.Binding
	Track           key.Binding
	Untrack         key.Binding
	Restore         key.Binding
	Delete          key.Binding
	Push            key.Binding
	Fetch           key.Binding
	ResolveConflict key.Binding
}

// DefaultBranchesKeyMap returns the branches bindings, applying any config overrides.
func DefaultBranchesKeyMap(overrides map[string]string) BranchesKeyMap {
	return BranchesKeyMap{
		MoveDown:        bind(overrides, "branches.move_down", "j/↓", "Move down", "j", "down"),
		MoveUp:          bind(overrides, "branches.move_up", "k/↑", "Move up", "k", "up"),
		TrackByName:     bind(overrides, "branches.track_by_name", "t", "Pull & track remote branch by name", "t"),
		Track:           bind(overrides, "branches.track", "T", "Track remote branch", "T"),
		Untrack:         bind(overrides, "branches.untrack", "U", "Untrack remote branch", "U"),
		Restore:         bind(overrides, "branches.restore", "L", "Restore deleted local branch", "L"),
		Delete:          bind(overrides, "branches.delete", "x", "Delete local bookmark", "x"),
		Push:            bind(overrides, "branches.push", "P", "Push local branch to remote", "P"),
		Fetch:           bind(overrides, "branches.fetch", "F", "Fetch from all remotes", "F"),
		ResolveConflict: bind(overrides, "branches.resolve_conflict", "c", "Resolve conflicted bookmark", "c"),
	}
}
