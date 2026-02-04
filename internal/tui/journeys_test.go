package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/github"
	"github.com/madicen/jj-tui/internal/models"
	"github.com/madicen/jj-tui/internal/testutil"
)

// TestJourney_BrowseCommitsAndSwitchViews tests the basic navigation flow
func TestJourney_BrowseCommitsAndSwitchViews(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	// Step 1: Start in commit graph view
	if m.viewMode != ViewCommitGraph {
		t.Fatalf("Expected to start in ViewCommitGraph, got %v", m.viewMode)
	}

	// Step 2: Switch to PRs view with 'p'
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m = newModel.(*Model)
	if m.viewMode != ViewPullRequests {
		t.Errorf("Expected ViewPullRequests after pressing p, got %v", m.viewMode)
	}

	// Step 3: Switch to Tickets view with 't'
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	m = newModel.(*Model)
	if m.viewMode != ViewJira {
		t.Errorf("Expected ViewJira after pressing t, got %v", m.viewMode)
	}

	// Step 4: Switch to Help view with 'h'
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	m = newModel.(*Model)
	if m.viewMode != ViewHelp {
		t.Errorf("Expected ViewHelp after pressing h, got %v", m.viewMode)
	}

	// Step 5: Return to Graph with 'g'
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	m = newModel.(*Model)
	if m.viewMode != ViewCommitGraph {
		t.Errorf("Expected ViewCommitGraph after pressing g, got %v", m.viewMode)
	}
}

// TestJourney_PRStateColors tests that PR states are correctly colored
func TestJourney_PRStateColors(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	// Set up repository with PRs of different states
	m.repository.PRs = []models.GitHubPR{
		{Number: 1, Title: "Open PR", State: "open"},
		{Number: 2, Title: "Merged PR", State: "merged"},
		{Number: 3, Title: "Closed PR", State: "closed"},
	}

	// Switch to PR view
	m.viewMode = ViewPullRequests
	// Need a github service (empty struct works for view testing)
	m.githubService = &github.Service{}

	view := m.View()

	// Verify all PRs are displayed
	if !containsString(view, "Open PR") {
		t.Error("View should contain 'Open PR'")
	}
	if !containsString(view, "Merged PR") {
		t.Error("View should contain 'Merged PR'")
	}
	if !containsString(view, "Closed PR") {
		t.Error("View should contain 'Closed PR'")
	}

	// The view should contain the colored indicators (●)
	if !containsString(view, "●") {
		t.Error("View should contain status indicators")
	}
}

// TestJourney_TicketsViewWithProvider tests the tickets view with different providers
func TestJourney_TicketsViewWithProvider(t *testing.T) {
	tests := []struct {
		name         string
		mockService  *testutil.MockTicketService
		expectTitle  string
	}{
		{
			name:         "JiraProvider",
			mockService:  testutil.NewMockJiraService(),
			expectTitle:  "Jira",
		},
		{
			name:         "CodecksProvider",
			mockService:  testutil.NewMockCodecksService(),
			expectTitle:  "Codecks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel()
			defer m.Close()

			// Set up the mock ticket service
			m.ticketService = tt.mockService
			m.ticketList = tt.mockService.Tickets
			m.viewMode = ViewJira

			view := m.View()

			// Verify the provider name is shown
			if !containsString(view, tt.expectTitle) {
				t.Errorf("View should contain provider name '%s'", tt.expectTitle)
			}

			// Verify tickets are displayed
			for _, ticket := range tt.mockService.Tickets {
				if !containsString(view, ticket.Summary) {
					t.Errorf("View should contain ticket summary '%s'", ticket.Summary)
				}
			}
		})
	}
}

// TestJourney_TicketNavigation tests navigating through tickets
func TestJourney_TicketNavigation(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	mockService := testutil.NewMockJiraService()
	m.ticketService = mockService
	m.ticketList = mockService.Tickets
	m.viewMode = ViewJira
	m.selectedTicket = 0

	// Navigate down through tickets
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.selectedTicket != 1 {
		t.Errorf("Expected selectedTicket=1, got %d", m.selectedTicket)
	}

	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.selectedTicket != 2 {
		t.Errorf("Expected selectedTicket=2, got %d", m.selectedTicket)
	}

	// Boundary check - should not go past last ticket
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.selectedTicket != 2 {
		t.Errorf("Expected selectedTicket=2 (at boundary), got %d", m.selectedTicket)
	}

	// Navigate back up
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.selectedTicket != 1 {
		t.Errorf("Expected selectedTicket=1, got %d", m.selectedTicket)
	}
}

// TestJourney_PRNavigation tests navigating through PRs
func TestJourney_PRNavigation(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	m.repository.PRs = []models.GitHubPR{
		{Number: 1, Title: "PR 1", State: "open"},
		{Number: 2, Title: "PR 2", State: "open"},
		{Number: 3, Title: "PR 3", State: "merged"},
	}
	m.githubService = &github.Service{}
	m.viewMode = ViewPullRequests
	m.selectedPR = 0

	// Navigate down
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.selectedPR != 1 {
		t.Errorf("Expected selectedPR=1, got %d", m.selectedPR)
	}

	// Navigate to end
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.selectedPR != 2 {
		t.Errorf("Expected selectedPR=2, got %d", m.selectedPR)
	}

	// Boundary check
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.selectedPR != 2 {
		t.Errorf("Expected selectedPR=2 (at boundary), got %d", m.selectedPR)
	}

	// Navigate back to start
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.selectedPR != 0 {
		t.Errorf("Expected selectedPR=0, got %d", m.selectedPR)
	}

	// Boundary check at top
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.selectedPR != 0 {
		t.Errorf("Expected selectedPR=0 (at boundary), got %d", m.selectedPR)
	}
}

// TestJourney_ErrorHandling tests error state handling
func TestJourney_ErrorHandling(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	// Simulate an error
	m = updateModel(m, errorMsg{err: fmt.Errorf("test error")})

	if m.err == nil {
		t.Error("Expected error to be set")
	}

	view := m.View()
	if !containsString(view, "Error") {
		t.Error("View should show error message")
	}

	// Press Esc to dismiss
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyEsc})

	if m.err != nil {
		t.Error("Error should be cleared after Esc")
	}
}

// TestJourney_NotJJRepoError tests the not-a-jj-repo error state
func TestJourney_NotJJRepoError(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	// Simulate "not a jj repo" error
	m = updateModel(m, errorMsg{
		err:         fmt.Errorf("not a jujutsu repository"),
		notJJRepo:   true,
		currentPath: "/test/path",
	})

	if !m.notJJRepo {
		t.Error("notJJRepo flag should be set")
	}

	view := m.View()
	if !containsString(view, "Not a Jujutsu Repository") {
		t.Error("View should show 'Not a Jujutsu Repository' message")
	}
	if !containsString(view, "Initialize") {
		t.Error("View should show Initialize button")
	}
}

// TestJourney_SettingsView tests the settings view
func TestJourney_SettingsView(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	// Switch to settings
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(",")})
	if m.viewMode != ViewSettings {
		t.Fatalf("Expected ViewSettings, got %v", m.viewMode)
	}

	view := m.View()

	// Check for key UI elements
	if !containsString(view, "Settings") {
		t.Error("Settings view should show title")
	}
	if !containsString(view, "GitHub") {
		t.Error("Settings view should show GitHub section")
	}
	if !containsString(view, "Jira") {
		t.Error("Settings view should show Jira section")
	}
	if !containsString(view, "Codecks") {
		t.Error("Settings view should show Codecks section")
	}
	if !containsString(view, "Save") {
		t.Error("Settings view should show Save button")
	}

	// Test field navigation with Tab
	initialField := m.settingsFocusedField
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyTab})
	if m.settingsFocusedField == initialField {
		t.Error("Tab should move to next field")
	}

	// Test cancel with Esc
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.viewMode != ViewCommitGraph {
		t.Error("Esc should return to commit graph")
	}
}

// TestJourney_HelpView tests the help view
func TestJourney_HelpView(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	// Switch to help
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	if m.viewMode != ViewHelp {
		t.Fatalf("Expected ViewHelp, got %v", m.viewMode)
	}

	view := m.View()

	// Check for key help content
	if !containsString(view, "Shortcuts") {
		t.Error("Help view should show shortcuts")
	}
	if !containsString(view, "Graph") {
		t.Error("Help view should mention Graph shortcuts")
	}
	if !containsString(view, "Pull Request") {
		t.Error("Help view should mention PR shortcuts")
	}
	if !containsString(view, "Tickets") {
		t.Error("Help view should mention Tickets shortcuts")
	}
}

// TestJourney_ImmutableCommitProtection tests that immutable commits can't be modified
func TestJourney_ImmutableCommitProtection(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	// Select an immutable commit
	m.repository.Graph.Commits[0].Immutable = true
	m.selectedCommit = 0

	view := m.View()

	// Should show immutable message instead of action buttons
	if containsString(view, "Edit (e)") {
		t.Error("Edit button should not appear for immutable commit")
	}
	if containsString(view, "Describe (d)") {
		t.Error("Describe button should not appear for immutable commit")
	}
	if !containsString(view, "immutable") {
		t.Error("Should show immutable message")
	}
}

// updateModel is a helper that casts the update result back to *Model
func updateModel(m *Model, msg tea.Msg) *Model {
	newModel, _ := m.Update(msg)
	return newModel.(*Model)
}

