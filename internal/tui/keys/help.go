package keys

import "github.com/charmbracelet/bubbles/key"

// ScopeHelp is the config prefix for help tab shortcuts.
const ScopeHelp = "help"

// HelpKeyMap holds the help tab's sub-tab navigation keybindings.
type HelpKeyMap struct {
	PrevTab   key.Binding
	NextTab   key.Binding
	SwitchTab key.Binding
}

// DefaultHelpKeyMap returns the help bindings, applying any config overrides.
func DefaultHelpKeyMap(overrides map[string]string) HelpKeyMap {
	return HelpKeyMap{
		PrevTab:   bind(overrides, "help.prev_tab", "^j", "Previous sub-tab (Shortcuts ↔ History)", "ctrl+j"),
		NextTab:   bind(overrides, "help.next_tab", "^k", "Next sub-tab", "ctrl+k"),
		SwitchTab: bind(overrides, "help.switch_tab", "Tab", "Next sub-tab", "tab"),
	}
}

func (k HelpKeyMap) entries() []entry {
	return []entry{
		{"help.prev_tab", k.PrevTab},
		{"help.next_tab", k.NextTab},
		{"help.switch_tab", k.SwitchTab},
	}
}
