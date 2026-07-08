package tickets

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// This file characterizes the Tickets settings sub-tab's current behavior BEFORE
// the P2.3 Tab-interface refactor. The parent settings model reads the provider,
// auto-in-progress flag, and GitHub-issues excluded statuses on save, and drives
// the provider dropdown overlay. The tests lock those accessors plus the
// provider<->dropdown index sync.
//
// PLAN(P2.3): keep this green across the interface migration.

// TestNewModelDefaults locks the documented defaults (no provider, auto-in-progress on).
func TestNewModelDefaults(t *testing.T) {
	m := NewModel()
	if m.GetTicketProvider() != "" {
		t.Fatalf("default provider = %q, want empty", m.GetTicketProvider())
	}
	if !m.GetAutoInProgress() {
		t.Fatal("default GetAutoInProgress() should be true")
	}
}

// TestSetTicketProviderSyncsDropdown locks that setting the provider also moves
// the dropdown selection so the rendered panel matches the value.
func TestSetTicketProviderSyncsDropdown(t *testing.T) {
	m := NewModel()
	m.SetTicketProvider("codecks")
	if m.GetTicketProvider() != "codecks" {
		t.Fatalf("GetTicketProvider = %q, want codecks", m.GetTicketProvider())
	}
	if got := m.ProviderDropdown().SelectedIndex(); got != providerIndex("codecks") {
		t.Fatalf("dropdown index = %d, want %d", got, providerIndex("codecks"))
	}
}

// TestProviderIndexMapping locks the value<->index mapping the dropdown depends on.
func TestProviderIndexMapping(t *testing.T) {
	cases := map[string]int{"": 0, "jira": 1, "codecks": 2, "github_issues": 3, "unknown": 0}
	for value, want := range cases {
		if got := providerIndex(value); got != want {
			t.Errorf("providerIndex(%q) = %d, want %d", value, got, want)
		}
	}
}

// TestGetInputViewsOnlyForGitHubIssues locks that the excluded-statuses input is
// exposed only when the provider is github_issues (the parent hides it otherwise).
func TestGetInputViewsOnlyForGitHubIssues(t *testing.T) {
	m := NewModel()
	if m.GetInputViews() != nil {
		t.Fatal("no input views expected when provider is empty")
	}
	m.SetTicketProvider("github_issues")
	if got := len(m.GetInputViews()); got != 1 {
		t.Fatalf("github_issues should expose 1 input view, got %d", got)
	}
}

// TestUpdateOnlyEditsWhenGitHubIssues locks that typed runes only reach the
// excluded-statuses input when the provider is github_issues.
func TestUpdateOnlyEditsWhenGitHubIssues(t *testing.T) {
	m := NewModel()
	// Wrong provider: typing is ignored.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if m.GetGitHubIssuesExcludedStatuses() != "" {
		t.Fatalf("typing should be ignored for non-github_issues provider, got %q", m.GetGitHubIssuesExcludedStatuses())
	}
	m.SetTicketProvider("github_issues")
	m.SetFocusedField(0) // focus the excluded-statuses input so it accepts keys
	for _, r := range "closed" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if m.GetGitHubIssuesExcludedStatuses() != "closed" {
		t.Fatalf("github_issues typing = %q, want closed", m.GetGitHubIssuesExcludedStatuses())
	}
}

// TestSetFocusedFieldClampsNegative locks that a negative index is coerced to 0.
func TestSetFocusedFieldClampsNegative(t *testing.T) {
	m := NewModel()
	m.SetFocusedField(-3)
	if m.GetFocusedField() != 0 {
		t.Fatalf("negative focus should clamp to 0, got %d", m.GetFocusedField())
	}
}

// TestAutoInProgressToggle locks the setter used by the toggle row.
func TestAutoInProgressToggle(t *testing.T) {
	m := NewModel()
	m.SetAutoInProgress(false)
	if m.GetAutoInProgress() {
		t.Fatal("SetAutoInProgress(false) did not take effect")
	}
}
