package keys

import "github.com/charmbracelet/bubbles/key"

// ScopeGraph is the config prefix for commit-graph tab shortcuts.
const ScopeGraph = "graph"

// GraphKeyMap holds the commit-graph tab's keybindings.
type GraphKeyMap struct {
	MoveDown        key.Binding
	MoveUp          key.Binding
	ToggleFocus     key.Binding
	Scroll          key.Binding
	CancelSelection key.Binding
	Rebase          key.Binding
	Merge           key.Binding
	Checkout        key.Binding
	NewCommit       key.Binding
	EditDescription key.Binding
	Squash          key.Binding
	Abandon         key.Binding
	Absorb          key.Binding
	Duplicate       key.Binding
	CreateBookmark  key.Binding
	DeleteBookmark  key.Binding
	UpdatePR        key.Binding
	CreatePR        key.Binding
	ResolveConflict key.Binding
	MoveDelta       key.Binding
	EvologSplit     key.Binding
	MoveFileUp      key.Binding
	MoveFileDown    key.Binding
	RevertFile      key.Binding
	ViewFileDiff    key.Binding
	OpenExternal    key.Binding
}

// DefaultGraphKeyMap returns the graph bindings, applying any config overrides.
func DefaultGraphKeyMap(overrides map[string]string) GraphKeyMap {
	return GraphKeyMap{
		MoveDown:        bind(overrides, "graph.move_down", "j/↓", "Move down", "j", "down"),
		MoveUp:          bind(overrides, "graph.move_up", "k/↑", "Move up", "k", "up"),
		ToggleFocus:     bind(overrides, "graph.toggle_focus", "Tab", "Switch focus: graph ↔ files", "tab"),
		Scroll:          bind(overrides, "graph.scroll", "PgUp/PgDn", "Scroll graph/files pane", "pgup", "pgdown", "ctrl+u", "ctrl+d", "home", "end", "ctrl+f", "ctrl+b"),
		CancelSelection: bind(overrides, "graph.cancel", "Esc", "Close menu / cancel selection", "esc", "q"),
		Rebase:          bind(overrides, "graph.rebase", "r", "Rebase commit (with descendants)", "r"),
		Merge:           bind(overrides, "graph.merge", "M", "Merge from: pick a source to merge into the selected commit (e.g. merge main into current bookmark)", "M"),
		Checkout:        bind(overrides, "graph.checkout", "Enter/e", "Edit selected commit (jj edit)", "enter", "e"),
		NewCommit:       bind(overrides, "graph.new_commit", "n", "Create new commit from selected", "n"),
		EditDescription: bind(overrides, "graph.edit_description", "d", "Edit description; or resolve divergent when commit is divergent", "d"),
		Squash:          bind(overrides, "graph.squash", "s", "Squash commit into parent", "s"),
		Abandon:         bind(overrides, "graph.abandon", "a", "Abandon commit", "a"),
		Absorb:          bind(overrides, "graph.absorb", "A", "Absorb working-copy changes into ancestors", "A"),
		Duplicate:       bind(overrides, "graph.duplicate", "D", "Duplicate commit", "D"),
		CreateBookmark:  bind(overrides, "graph.create_bookmark", "m", "Create/move bookmark on commit", "m"),
		DeleteBookmark:  bind(overrides, "graph.delete_bookmark", "x", "Delete bookmark from commit", "x"),
		UpdatePR:        bind(overrides, "graph.update_pr", "u", "Update existing PR with new commits", "u"),
		CreatePR:        bind(overrides, "graph.create_pr", "c", "Create new PR from commit chain", "c"),
		ResolveConflict: bind(overrides, "graph.resolve_conflict", "C", "Resolve diverged bookmark (when shown): graph pane focused; same flow as Branches (c)", "C"),
		MoveDelta:       bind(overrides, "graph.move_delta", "f", "Forgot new commit? Stack on bookmark@origin (avoid force-push)", "f"),
		EvologSplit:     bind(overrides, "graph.evolog_split", "z", "split (experimental, when shown): jj evolog parent + step file list; o patch; p plan overlay (Enter runs split from overlay); s / ✧^g AI suggest; Graph (g) vs preview after split; FAQ bases on evolog row you pick, not main unless you choose that row; if AI says no split, Enter twice (or j/k); d optional AI describe; moves change (and feature bookmark if present)", "z"),
		MoveFileUp:      bind(overrides, "graph.move_file_up", "[", "Move selected file up", "["),
		MoveFileDown:    bind(overrides, "graph.move_file_down", "]", "Move selected file down", "]"),
		RevertFile:      bind(overrides, "graph.revert_file", "v", "Revert selected file", "v"),
		ViewFileDiff:    bind(overrides, "graph.view_file_diff", "o", "View full jj diff for selected changed file (files pane)", "o"),
		OpenExternal:    bind(overrides, "graph.open_external", "O", "Open selected file in external editor (files pane; set editor in Settings → Advanced)", "O"),
	}
}

func (k GraphKeyMap) entries() []entry {
	return []entry{
		{"graph.move_down", k.MoveDown},
		{"graph.move_up", k.MoveUp},
		{"graph.toggle_focus", k.ToggleFocus},
		{"graph.scroll", k.Scroll},
		{"graph.cancel", k.CancelSelection},
		{"graph.rebase", k.Rebase},
		{"graph.merge", k.Merge},
		{"graph.checkout", k.Checkout},
		{"graph.new_commit", k.NewCommit},
		{"graph.edit_description", k.EditDescription},
		{"graph.squash", k.Squash},
		{"graph.abandon", k.Abandon},
		{"graph.absorb", k.Absorb},
		{"graph.duplicate", k.Duplicate},
		{"graph.create_bookmark", k.CreateBookmark},
		{"graph.delete_bookmark", k.DeleteBookmark},
		{"graph.update_pr", k.UpdatePR},
		{"graph.create_pr", k.CreatePR},
		{"graph.resolve_conflict", k.ResolveConflict},
		{"graph.move_delta", k.MoveDelta},
		{"graph.evolog_split", k.EvologSplit},
		{"graph.move_file_up", k.MoveFileUp},
		{"graph.move_file_down", k.MoveFileDown},
		{"graph.revert_file", k.RevertFile},
		{"graph.view_file_diff", k.ViewFileDiff},
		{"graph.open_external", k.OpenExternal},
	}
}
