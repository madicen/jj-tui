// Package keys centralizes jj-tui's keybindings as bubbles/key Bindings.
//
// PLAN(P3.3): keys were previously hardcoded string switches in
// internal/tui/model/keys.go and each tab's handler. This package introduces a
// KeyMap struct per scope (global + one per tab) whose fields are key.Binding
// values carrying both the trigger keys and the help text. Handlers match with
// key.Matches instead of comparing msg.String() to literals; the help tab is fed
// from the same bindings. The default keys reproduce the previous behavior
// exactly.
//
// PLAN(P5.1): each KeyMap constructor accepts a config override map
// (scope.action -> key) so users can rebind keys; collisions are validated per
// scope (see Validate) and surfaced in the error modal.
package keys

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
)

// bind builds a key.Binding from an override-aware key list. When the config
// map has an entry for id, that single key replaces the defaults and the help
// key column is updated to match; otherwise the compiled-in defaults are used.
func bind(overrides map[string]string, id, disp, desc string, defaults ...string) key.Binding {
	keys := defaults
	if overrides != nil {
		if v, ok := overrides[id]; ok {
			if v = strings.TrimSpace(v); v != "" {
				keys = []string{v}
				disp = v
			}
		}
	}
	return key.NewBinding(key.WithKeys(keys...), key.WithHelp(disp, desc))
}
