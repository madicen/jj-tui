package keys

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
)

// entry pairs a scope-qualified binding ID (e.g. "graph.abandon") with its
// resolved key.Binding, used for per-scope collision detection.
type entry struct {
	id      string
	binding key.Binding
}

// Collision describes one trigger key bound to more than one action in a scope.
type Collision struct {
	Scope string
	Key   string
	IDs   []string
}

func (c Collision) String() string {
	return fmt.Sprintf("%s: key %q is bound to %s", c.Scope, c.Key, strings.Join(c.IDs, " and "))
}

// detectCollisions returns every trigger key bound to more than one action
// within a single scope, sorted for stable output.
func detectCollisions(scope string, entries []entry) []Collision {
	byKey := map[string][]string{}
	for _, e := range entries {
		for _, k := range e.binding.Keys() {
			byKey[k] = append(byKey[k], e.id)
		}
	}
	var out []Collision
	for k, ids := range byKey {
		if len(ids) > 1 {
			sort.Strings(ids)
			out = append(out, Collision{Scope: scope, Key: k, IDs: ids})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// Validate resolves every scope's KeyMap from the given config overrides and
// returns any per-scope key collisions. An empty result means the effective
// bindings are unambiguous.
func Validate(overrides map[string]string) []Collision {
	km := DefaultKeyMaps(overrides)
	scopes := []struct {
		name    string
		entries []entry
	}{
		{ScopeGlobal, km.Global.entries()},
		{ScopeGraph, km.Graph.entries()},
		{ScopeBranches, km.Branches.entries()},
		{ScopePRs, km.PRs.entries()},
		{ScopeTickets, km.Tickets.entries()},
		{ScopeHelp, km.Help.entries()},
	}
	out := make([]Collision, 0, len(scopes))
	for _, s := range scopes {
		out = append(out, detectCollisions(s.name, s.entries)...)
	}
	return out
}

// CollisionError renders collisions as a single error suitable for the error
// modal, or nil when there are none.
func CollisionError(collisions []Collision) error {
	if len(collisions) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString("Keybinding conflicts in config (\"keys\"); falling back to defaults:\n")
	for _, c := range collisions {
		b.WriteString("  • ")
		b.WriteString(c.String())
		b.WriteString("\n")
	}
	//nolint:err113 // dynamic user-facing config validation message, not a sentinel error.
	return fmt.Errorf("%s", strings.TrimRight(b.String(), "\n"))
}
