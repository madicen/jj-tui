package codecks

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// This file characterizes the Codecks settings sub-tab's current behavior BEFORE
// the P2.3 Tab-interface refactor. The tab maps config fields onto a shared
// form.Model; the tests lock the accessor<->field-index mapping and focus/typing
// routing the parent settings model relies on.
//
// PLAN(P2.3): keep this green across the interface migration.

// TestAccessorsAreIndependent locks that each getter/setter pair maps to its own
// field with no cross-contamination.
func TestAccessorsAreIndependent(t *testing.T) {
	m := NewModel()
	m.SetSubdomain("sub")
	m.SetToken("tok")
	m.SetProject("proj")
	m.SetExcludedStatuses("done")

	if m.GetSubdomain() != "sub" {
		t.Errorf("GetSubdomain = %q, want sub", m.GetSubdomain())
	}
	if m.GetToken() != "tok" {
		t.Errorf("GetToken = %q, want tok", m.GetToken())
	}
	if m.GetProject() != "proj" {
		t.Errorf("GetProject = %q, want proj", m.GetProject())
	}
	if m.GetExcludedStatuses() != "done" {
		t.Errorf("GetExcludedStatuses = %q, want done", m.GetExcludedStatuses())
	}
}

// TestAPIKeyAliasesToken locks that GetAPIKey/SetAPIKey are aliases of the token
// field (kept for compatibility with older callers).
func TestAPIKeyAliasesToken(t *testing.T) {
	m := NewModel()
	m.SetToken("via-token")
	if m.GetAPIKey() != "via-token" {
		t.Fatalf("GetAPIKey = %q, want via-token (should alias token)", m.GetAPIKey())
	}
	m.SetAPIKey("via-apikey")
	if m.GetToken() != "via-apikey" {
		t.Fatalf("SetAPIKey should write the token field, GetToken = %q", m.GetToken())
	}
}

// TestGetInputViewsCountStable locks the field count at 4.
func TestGetInputViewsCountStable(t *testing.T) {
	m := NewModel()
	if got := len(m.GetInputViews()); got != 4 {
		t.Fatalf("GetInputViews() length = %d, want 4", got)
	}
}

// TestUpdateNavCyclesFocus locks j/k focus movement with clamping.
func TestUpdateNavCyclesFocus(t *testing.T) {
	m := NewModel()
	for i := 0; i < 3; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		if m.GetFocusedField() != i+1 {
			t.Fatalf("j from %d landed on %d", i, m.GetFocusedField())
		}
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.GetFocusedField() != 3 {
		t.Fatalf("j past last should clamp at 3, got %d", m.GetFocusedField())
	}
}

// TestUpdateTypingRoutesToFocusedField locks typed runes land in the focused
// field only.
func TestUpdateTypingRoutesToFocusedField(t *testing.T) {
	m := NewModel()
	m.SetFocusedField(fieldSubdomain)
	for _, r := range "team" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if m.GetSubdomain() != "team" {
		t.Fatalf("typing into subdomain = %q, want team", m.GetSubdomain())
	}
	if m.GetToken() != "" {
		t.Fatalf("typing leaked into token: %q", m.GetToken())
	}
}
