package keys

import "github.com/charmbracelet/bubbles/key"

// ScopePRs is the config prefix for pull-requests tab shortcuts.
const ScopePRs = "prs"

// PRsKeyMap holds the pull-requests tab's keybindings.
type PRsKeyMap struct {
	MoveDown   key.Binding
	MoveUp     key.Binding
	ScrollUp   key.Binding
	ScrollDown key.Binding
	Home       key.Binding
	End        key.Binding
	Open       key.Binding
	Merge      key.Binding
	Close      key.Binding
}

// DefaultPRsKeyMap returns the PRs bindings, applying any config overrides.
func DefaultPRsKeyMap(overrides map[string]string) PRsKeyMap {
	return PRsKeyMap{
		MoveDown:   bind(overrides, "prs.move_down", "j/↓", "Move down", "j", "down"),
		MoveUp:     bind(overrides, "prs.move_up", "k/↑", "Move up", "k", "up"),
		ScrollUp:   bind(overrides, "prs.scroll_up", "PgUp", "Scroll page up", "pgup", "ctrl+u", "ctrl+b"),
		ScrollDown: bind(overrides, "prs.scroll_down", "PgDn", "Scroll page down", "pgdown", "ctrl+d", "ctrl+f"),
		Home:       bind(overrides, "prs.home", "Home", "Scroll to top", "home"),
		End:        bind(overrides, "prs.end", "End", "Scroll to bottom", "end"),
		Open:       bind(overrides, "prs.open", "Enter/o", "Open PR in browser", "o", "enter", "e"),
		Merge:      bind(overrides, "prs.merge", "M", "Merge pull request", "M"),
		Close:      bind(overrides, "prs.close", "X", "Close pull request", "X"),
	}
}

func (k PRsKeyMap) entries() []entry {
	return []entry{
		{"prs.move_down", k.MoveDown},
		{"prs.move_up", k.MoveUp},
		{"prs.scroll_up", k.ScrollUp},
		{"prs.scroll_down", k.ScrollDown},
		{"prs.home", k.Home},
		{"prs.end", k.End},
		{"prs.open", k.Open},
		{"prs.merge", k.Merge},
		{"prs.close", k.Close},
	}
}
