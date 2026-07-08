package keys

import "github.com/charmbracelet/bubbles/key"

// ScopeGlobal is the config prefix for global (always-available) shortcuts.
const ScopeGlobal = "global"

// GlobalKeyMap holds the always-available navigation and top-level shortcuts
// handled by the root model after tabs and modals decline a key.
type GlobalKeyMap struct {
	NavGraph      key.Binding
	NavPRs        key.Binding
	NavTickets    key.Binding
	NavBranches   key.Binding
	NavSettings   key.Binding
	NavHelp       key.Binding
	NavWorkspaces key.Binding
	Refresh       key.Binding
	Undo          key.Binding
	Redo          key.Binding
	Back          key.Binding
	Quit          key.Binding
}

// DefaultGlobalKeyMap returns the global bindings, applying any config overrides.
func DefaultGlobalKeyMap(overrides map[string]string) GlobalKeyMap {
	return GlobalKeyMap{
		NavGraph:      bind(overrides, "global.graph", "g", "Go to commit graph", "g"),
		NavPRs:        bind(overrides, "global.prs", "p", "Go to pull requests", "p"),
		NavTickets:    bind(overrides, "global.tickets", "t", "Go to Tickets", "t"),
		NavBranches:   bind(overrides, "global.branches", "b", "Go to Branches", "b"),
		NavSettings:   bind(overrides, "global.settings", ",", "Open settings", ","),
		NavHelp:       bind(overrides, "global.help", "h/?", "Show this help", "h", "?"),
		NavWorkspaces: bind(overrides, "global.workspaces", "w", "Manage workspaces", "w"),
		Refresh:       bind(overrides, "global.refresh", "^r", "Refresh", "ctrl+r"),
		Undo:          bind(overrides, "global.undo", "^z", "Undo last jj operation", "ctrl+z"),
		Redo:          bind(overrides, "global.redo", "^y", "Redo jj operation", "ctrl+y"),
		Back:          bind(overrides, "global.back", "Esc", "Back to graph", "esc"),
		Quit:          bind(overrides, "global.quit", "^q", "Quit", "ctrl+q", "ctrl+c"),
	}
}

func (k GlobalKeyMap) entries() []entry {
	return []entry{
		{"global.graph", k.NavGraph},
		{"global.prs", k.NavPRs},
		{"global.tickets", k.NavTickets},
		{"global.branches", k.NavBranches},
		{"global.settings", k.NavSettings},
		{"global.help", k.NavHelp},
		{"global.workspaces", k.NavWorkspaces},
		{"global.refresh", k.Refresh},
		{"global.undo", k.Undo},
		{"global.redo", k.Redo},
		{"global.back", k.Back},
		{"global.quit", k.Quit},
	}
}
