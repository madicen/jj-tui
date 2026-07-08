package jira

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// This file characterizes the Jira settings sub-tab's current behavior BEFORE
// the P2.3 Tab-interface refactor. The tab is a thin mapping of config fields
// onto a shared form.Model; the parent settings model concatenates GetInputViews
// into a flat global index and reads each getter on save. The tests lock the
// accessor<->field-index mapping (so a reorder can't silently swap values) and
// the focus/typing routing.
//
// PLAN(P2.3): keep this green across the interface migration.

// TestAccessorsAreIndependent locks that each getter/setter pair maps to its own
// field with no cross-contamination — this is the invariant a field reorder
// would break.
func TestAccessorsAreIndependent(t *testing.T) {
	m := NewModel()
	m.SetURL("url")
	m.SetUser("user")
	m.SetToken("token")
	m.SetProject("project")
	m.SetProjectFilter("filter")
	m.SetIssueType("issuetype")
	m.SetJQL("jql")
	m.SetExcludedStatuses("excluded")

	checks := []struct {
		name string
		got  string
		want string
	}{
		{"URL", m.GetURL(), "url"},
		{"User", m.GetUser(), "user"},
		{"Token", m.GetToken(), "token"},
		{"Project", m.GetProject(), "project"},
		{"ProjectFilter", m.GetProjectFilter(), "filter"},
		{"IssueType", m.GetIssueType(), "issuetype"},
		{"JQL", m.GetJQL(), "jql"},
		{"ExcludedStatuses", m.GetExcludedStatuses(), "excluded"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q (field mapping drifted)", c.name, c.got, c.want)
		}
	}
}

// TestGetInputViewsCountStable locks the field count at 8; the parent's flat
// global-index math breaks if this changes.
func TestGetInputViewsCountStable(t *testing.T) {
	m := NewModel()
	if got := len(m.GetInputViews()); got != 8 {
		t.Fatalf("GetInputViews() length = %d, want 8", got)
	}
}

// TestUpdateNavCyclesFocus locks that j/k move focus through the 8 fields with
// clamping at both ends.
func TestUpdateNavCyclesFocus(t *testing.T) {
	m := NewModel()
	if m.GetFocusedField() != 0 {
		t.Fatalf("initial focus = %d, want 0", m.GetFocusedField())
	}
	for i := 0; i < 7; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		if m.GetFocusedField() != i+1 {
			t.Fatalf("j from %d landed on %d", i, m.GetFocusedField())
		}
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.GetFocusedField() != 7 {
		t.Fatalf("j past last field should clamp at 7, got %d", m.GetFocusedField())
	}
	for i := 7; i > 0; i-- {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
		if m.GetFocusedField() != i-1 {
			t.Fatalf("k from %d landed on %d", i, m.GetFocusedField())
		}
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.GetFocusedField() != 0 {
		t.Fatalf("k below 0 should clamp at 0, got %d", m.GetFocusedField())
	}
}

// TestUpdateTypingRoutesToFocusedField locks that typed runes accumulate in the
// focused field only (the URL field here), leaving others untouched.
func TestUpdateTypingRoutesToFocusedField(t *testing.T) {
	m := NewModel()
	m.SetFocusedField(fieldURL)
	for _, r := range "abc" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if m.GetURL() != "abc" {
		t.Fatalf("typing into URL field = %q, want abc", m.GetURL())
	}
	if m.GetToken() != "" {
		t.Fatalf("typing into URL field leaked into token: %q", m.GetToken())
	}
}
