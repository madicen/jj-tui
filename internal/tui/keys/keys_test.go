package keys

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func runeKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

// TestDefaultBindingMatchesDefaultKey confirms the compiled-in defaults match
// their documented keys (behavior-preserving baseline for P3.3).
func TestDefaultBindingMatchesDefaultKey(t *testing.T) {
	km := DefaultGraphKeyMap(nil)
	if !key.Matches(runeKey('a'), km.Abandon) {
		t.Error("default graph.abandon should match 'a'")
	}
	if key.Matches(runeKey('x'), km.Abandon) {
		t.Error("default graph.abandon should not match 'x'")
	}
}

// TestConfigRebindsKey verifies a config override changes the trigger key and
// updates the help display, and that the old key no longer matches (P5.1).
func TestConfigRebindsKey(t *testing.T) {
	overrides := map[string]string{"graph.abandon": "x"}
	km := DefaultGraphKeyMap(overrides)

	if !key.Matches(runeKey('x'), km.Abandon) {
		t.Error("rebound graph.abandon should match 'x'")
	}
	if key.Matches(runeKey('a'), km.Abandon) {
		t.Error("rebound graph.abandon should no longer match 'a'")
	}
	if got := km.Abandon.Help().Key; got != "x" {
		t.Errorf("help key display should reflect the rebinding, got %q", got)
	}
	if got := km.Abandon.Help().Desc; got != "Abandon commit" {
		t.Errorf("help desc should be preserved, got %q", got)
	}
}

// TestValidateNoCollisionsByDefault ensures the shipped defaults are unambiguous.
func TestValidateNoCollisionsByDefault(t *testing.T) {
	if c := Validate(nil); len(c) != 0 {
		t.Fatalf("default bindings must not collide, got %v", c)
	}
}

// TestValidateDetectsCollision verifies a config that rebinds one action onto
// another action's key in the same scope is reported as a collision (P5.1).
func TestValidateDetectsCollision(t *testing.T) {
	overrides := map[string]string{"graph.abandon": "s"} // collides with graph.squash ("s")
	collisions := Validate(overrides)
	if len(collisions) == 0 {
		t.Fatal("expected a collision for graph key 's'")
	}
	found := false
	for _, c := range collisions {
		if c.Scope == ScopeGraph && c.Key == "s" {
			found = true
			if len(c.IDs) < 2 {
				t.Errorf("collision should list both conflicting IDs, got %v", c.IDs)
			}
		}
	}
	if !found {
		t.Errorf("expected a graph/'s' collision, got %v", collisions)
	}
	if err := CollisionError(collisions); err == nil {
		t.Error("CollisionError should return a non-nil error for collisions")
	}
}

// TestValidateCrossScopeKeysDontCollide confirms the same key in different
// scopes is fine (collisions are per-scope).
func TestValidateCrossScopeKeysDontCollide(t *testing.T) {
	// graph.abandon 'a' and (hypothetically) prs uses different keys; the default
	// maps already reuse letters like 'M'/'c' across scopes without conflict.
	if c := Validate(nil); len(c) != 0 {
		t.Fatalf("cross-scope key reuse must not be flagged, got %v", c)
	}
}
