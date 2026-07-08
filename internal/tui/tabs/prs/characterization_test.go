package prs

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
)

// This file characterizes the PRs tab's key-driven request routing and
// selection navigation BEFORE the P2.3 Tab-interface refactor. model.go calls
// UpdateWithApp(msg, &appState); the tests lock the key->Request mapping (via
// the app==nil path, which surfaces the Request as an effect cmd) and the
// selection clamp, both of which the interface migration must preserve.
//
// PLAN(P2.3): keep this green across the interface migration.

func loadedModel(n int) Model {
	m := NewModel(nil)
	prs := make([]internal.GitHubPR, n)
	repo := &internal.Repository{PRs: prs}
	m.OnRepositoryLoaded(repo)
	return m
}

// TestSelectionNavigationClamps locks j/k selection movement with bounds.
func TestSelectionNavigationClamps(t *testing.T) {
	m := loadedModel(3)
	m.SetSelectedPR(0)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.GetSelectedPR() != 0 {
		t.Fatalf("k at top should stay 0, got %d", m.GetSelectedPR())
	}
	for i := 0; i < 2; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	}
	if m.GetSelectedPR() != 2 {
		t.Fatalf("two j should reach 2, got %d", m.GetSelectedPR())
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.GetSelectedPR() != 2 {
		t.Fatalf("j past end should clamp at 2, got %d", m.GetSelectedPR())
	}
}

// TestKeyRequestMapping locks the action-key -> Request field mapping.
func TestKeyRequestMapping(t *testing.T) {
	cases := []struct {
		key   string
		check func(Request) bool
		name  string
	}{
		{"o", func(r Request) bool { return r.OpenInBrowser }, "OpenInBrowser"},
		{"M", func(r Request) bool { return r.MergePR }, "MergePR"},
		{"X", func(r Request) bool { return r.ClosePR }, "ClosePR"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := loadedModel(2)
			m.SetSelectedPR(0)
			_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tc.key)})
			if cmd == nil {
				t.Fatalf("key %q should emit a request cmd", tc.key)
			}
			req, ok := cmd().(Request)
			if !ok {
				t.Fatalf("expected Request message, got %T", cmd())
			}
			if !tc.check(req) {
				t.Fatalf("key %q did not set %s: %+v", tc.key, tc.name, req)
			}
		})
	}
}

// TestActionKeysNoOpWithoutSelection locks that M/X/o do nothing when no PR is
// selected or the list is empty (guards against acting on an invalid index).
func TestActionKeysNoOpWithoutSelection(t *testing.T) {
	m := loadedModel(0) // empty list -> selectedPR = -1
	for _, key := range []string{"o", "M", "X"} {
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		if cmd != nil {
			t.Fatalf("key %q with no selection should be a no-op, got cmd", key)
		}
	}
}

// TestNoRepositoryIsSafe locks that navigation before any repository load does
// not panic and stays at the sentinel selection.
func TestNoRepositoryIsSafe(t *testing.T) {
	m := NewModel(nil)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.GetSelectedPR() != -1 {
		t.Fatalf("selection should stay -1 before repository load, got %d", m.GetSelectedPR())
	}
}
