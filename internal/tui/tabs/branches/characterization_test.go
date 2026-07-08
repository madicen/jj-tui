package branches

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
)

// This file characterizes the Branches tab's key-driven request routing and
// selection navigation BEFORE the P2.3 Tab-interface refactor. model.go calls
// UpdateWithApp(msg, &appState) and relies on the tab owning its own request
// execution; the tests below lock the key->Request mapping (via the app==nil
// path, which surfaces the Request as an effect cmd) and the selection clamp.
//
// PLAN(P2.3): keep this green across the interface migration.

func loadedModel(n int) Model {
	m := NewModel(nil)
	branches := make([]internal.Branch, n)
	for i := range branches {
		branches[i] = internal.Branch{Name: "b"}
	}
	m.UpdateBranches(branches)
	return m
}

// TestSelectionNavigationClamps locks j/k selection movement with bounds.
func TestSelectionNavigationClamps(t *testing.T) {
	m := loadedModel(3)
	m.SetSelectedBranch(0)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.GetSelectedBranch() != 0 {
		t.Fatalf("k at top should stay 0, got %d", m.GetSelectedBranch())
	}
	for i := 0; i < 2; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	}
	if m.GetSelectedBranch() != 2 {
		t.Fatalf("two j should reach index 2, got %d", m.GetSelectedBranch())
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.GetSelectedBranch() != 2 {
		t.Fatalf("j past end should clamp at 2, got %d", m.GetSelectedBranch())
	}
}

// TestKeyRequestMapping locks the action-key -> Request field mapping. The
// app==nil Update path returns req.Cmd(), which emits a *Request-bearing message
// we can inspect.
func TestKeyRequestMapping(t *testing.T) {
	cases := []struct {
		key   string
		check func(Request) bool
		name  string
	}{
		{"T", func(r Request) bool { return r.TrackBranch }, "TrackBranch"},
		{"U", func(r Request) bool { return r.UntrackBranch }, "UntrackBranch"},
		{"L", func(r Request) bool { return r.RestoreLocalBranch }, "RestoreLocalBranch"},
		{"P", func(r Request) bool { return r.PushBranch }, "PushBranch"},
		{"F", func(r Request) bool { return r.FetchAll }, "FetchAll"},
		{"c", func(r Request) bool { return r.ResolveBookmarkConflict }, "ResolveBookmarkConflict"},
		{"x", func(r Request) bool { return r.DeleteBranchBookmark }, "DeleteBranchBookmark"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := loadedModel(1)
			_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tc.key)})
			if cmd == nil {
				t.Fatalf("key %q should emit a request cmd", tc.key)
			}
			msg := cmd()
			req, ok := msg.(Request)
			if !ok {
				t.Fatalf("expected Request message, got %T", msg)
			}
			if !tc.check(req) {
				t.Fatalf("key %q did not set %s: %+v", tc.key, tc.name, req)
			}
		})
	}
}

// TestOpenRemoteInputCapturesKeys locks that pressing 't' opens the inline
// track-by-name input and that Esc closes it (the input owns the keyboard while
// open, which model.go relies on to not steal keys).
func TestOpenRemoteInputCapturesKeys(t *testing.T) {
	m := loadedModel(1)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	// While the input is open, 'P' should type into the input, not emit a push
	// request.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("P")})
	if cmd != nil {
		if _, ok := cmd().(Request); ok {
			t.Fatal("keys while remote input is open should not emit a Request")
		}
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	// After Esc, 'P' should again emit a push request.
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("P")})
	if cmd == nil {
		t.Fatal("after closing remote input, P should emit a request")
	}
	if _, ok := cmd().(Request); !ok {
		t.Fatalf("expected Request after Esc, got %T", cmd())
	}
}
