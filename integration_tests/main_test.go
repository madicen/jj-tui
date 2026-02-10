package integration_tests

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/github"
	"github.com/madicen/jj-tui/internal/jj"
	"github.com/madicen/jj-tui/internal/models"
	"github.com/madicen/jj-tui/internal/testutil"
	"github.com/madicen/jj-tui/internal/tui"
)

// =============================================================================
// Test Repository Helpers
// =============================================================================

// TestRepository manages a test jj repository
type TestRepository struct {
	Path string
	t    *testing.T
}

// NewTestRepository creates a new test repository
func NewTestRepository(t *testing.T) *TestRepository {
	tempDir, err := os.MkdirTemp("", "jj-tui-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	repo := &TestRepository{
		Path: tempDir,
		t:    t,
	}

	// Initialize jj repository
	if err := repo.runCommand("jj", "git", "init"); err != nil {
		t.Fatalf("Failed to initialize jj repository: %v", err)
	}

	// Set up basic git config
	_ = repo.runCommand("jj", "config", "set", "--repo", "user.name", "Test User")
	_ = repo.runCommand("jj", "config", "set", "--repo", "user.email", "test@example.com")

	return repo
}

// Cleanup removes the test repository
func (r *TestRepository) Cleanup() {
	if r.Path != "" {
		os.RemoveAll(r.Path)
	}
}

// runCommand executes a command in the repository directory
func (r *TestRepository) runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = r.Path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// writeFile writes content to a file in the repository
func (r *TestRepository) writeFile(filename, content string) error {
	fullPath := filepath.Join(r.Path, filename)
	return os.WriteFile(fullPath, []byte(content), 0644)
}

// commitFile creates a file and commits it
func (r *TestRepository) commitFile(filename, content, message string) error {
	if err := r.writeFile(filename, content); err != nil {
		return err
	}
	return r.runCommand("jj", "commit", "-m", message)
}

// =============================================================================
// TUI Test Helpers
// =============================================================================

// newTestModel creates a new TUI model for testing
func newTestModel() *tui.Model {
	ctx := context.Background()
	m := tui.New(ctx)
	m.SetDimensions(100, 80)
	m.SetLoading(false)
	m.SetRepository(&models.Repository{
		Path: "/test/repo",
		Graph: models.CommitGraph{
			Commits: []models.Commit{
				{ID: "abc123456789", ShortID: "abc1", ChangeID: "abc1", Summary: "First commit"},
				{ID: "def456789012", ShortID: "def4", ChangeID: "def4", Summary: "Second commit"},
				{ID: "ghi789012345", ShortID: "ghi7", ChangeID: "ghi7", Summary: "Third commit", IsWorking: true},
			},
		},
		PRs: []models.GitHubPR{
			{Number: 1, Title: "Test PR", State: "open"},
		},
	})

	// Initialize viewport by processing a window size message
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 80})

	return m
}

// updateModel is a helper that casts the update result back to *Model
func updateModel(m *tui.Model, msg tea.Msg) *tui.Model {
	newModel, _ := m.Update(msg)
	return newModel.(*tui.Model)
}

// containsString checks if s contains substr
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

// =============================================================================
// JJ Service Integration Tests
// =============================================================================

// TestJJServiceBasicOperations tests basic jj service operations
func TestJJServiceBasicOperations(t *testing.T) {
	// Skip if jj is not available
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj command not available")
	}

	repo := NewTestRepository(t)
	defer repo.Cleanup()

	ctx := context.Background()
	service, err := jj.NewService(repo.Path)
	if err != nil {
		t.Fatalf("Failed to create jj service: %v", err)
	}

	t.Run("GetRepository", func(t *testing.T) {
		repository, err := service.GetRepository(ctx)
		if err != nil {
			t.Fatalf("Failed to get repository: %v", err)
		}

		if repository.Path != repo.Path {
			t.Errorf("Expected path %s, got %s", repo.Path, repository.Path)
		}

		if len(repository.Graph.Commits) == 0 {
			t.Error("Expected at least one commit (working copy)")
		}
	})

	t.Run("CreateCommit", func(t *testing.T) {
		// Create a file and commit it
		if err := repo.writeFile("test.txt", "Hello, World!"); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		if err := service.CreateNewCommit(ctx, "Add test file"); err != nil {
			t.Fatalf("Failed to create commit: %v", err)
		}

		// Verify the commit was created
		repository, err := service.GetRepository(ctx)
		if err != nil {
			t.Fatalf("Failed to get repository after commit: %v", err)
		}

		// Should have at least 2 commits now (initial + our commit)
		if len(repository.Graph.Commits) < 2 {
			t.Errorf("Expected at least 2 commits, got %d", len(repository.Graph.Commits))
		}

		// Find our commit
		found := false
		for _, commit := range repository.Graph.Commits {
			if commit.Summary == "Add test file" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Could not find the created commit")
		}
	})
}

// TestCommitGraphVisualization tests commit graph visualization
func TestCommitGraphVisualization(t *testing.T) {
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj command not available")
	}

	repo := NewTestRepository(t)
	defer repo.Cleanup()

	ctx := context.Background()
	service, err := jj.NewService(repo.Path)
	if err != nil {
		t.Fatalf("Failed to create jj service: %v", err)
	}

	// Create a series of commits
	commits := []struct {
		filename string
		content  string
		message  string
	}{
		{"file1.txt", "Content 1", "First commit"},
		{"file2.txt", "Content 2", "Second commit"},
		{"file3.txt", "Content 3", "Third commit"},
	}

	for _, commit := range commits {
		if err := repo.commitFile(commit.filename, commit.content, commit.message); err != nil {
			t.Fatalf("Failed to create commit %s: %v", commit.message, err)
		}
	}

	// Get repository state
	repository, err := service.GetRepository(ctx)
	if err != nil {
		t.Fatalf("Failed to get repository: %v", err)
	}

	// Verify we have the expected commits
	if len(repository.Graph.Commits) < len(commits) {
		t.Errorf("Expected at least %d commits, got %d", len(commits), len(repository.Graph.Commits))
	}

	// Verify commit messages are present
	commitMessages := make(map[string]bool)
	for _, commit := range repository.Graph.Commits {
		commitMessages[commit.Summary] = true
	}

	for _, expected := range commits {
		if !commitMessages[expected.message] {
			t.Errorf("Expected commit message '%s' not found", expected.message)
		}
	}
}

// =============================================================================
// PR Workflow Tests
// =============================================================================

// TestPRWorkflow tests the PR creation workflow
func TestPRWorkflow(t *testing.T) {
	// This test focuses on the data structures and workflow
	// without actually creating GitHub PRs

	t.Run("CreatePRRequest", func(t *testing.T) {
		req := &models.CreatePRRequest{
			Title:      "Test PR",
			Body:       "This is a test PR",
			BaseBranch: "main",
			HeadBranch: "feature-branch",
			CommitIDs:  []string{"commit1", "commit2"},
			Draft:      false,
		}

		if req.Title != "Test PR" {
			t.Errorf("Expected title 'Test PR', got '%s'", req.Title)
		}

		if len(req.CommitIDs) != 2 {
			t.Errorf("Expected 2 commit IDs, got %d", len(req.CommitIDs))
		}
	})

	t.Run("UpdatePRRequest", func(t *testing.T) {
		req := &models.UpdatePRRequest{
			Title:     "Updated PR Title",
			CommitIDs: []string{"commit1", "commit2", "commit3"},
		}

		if req.Title != "Updated PR Title" {
			t.Errorf("Expected title 'Updated PR Title', got '%s'", req.Title)
		}

		if len(req.CommitIDs) != 3 {
			t.Errorf("Expected 3 commit IDs, got %d", len(req.CommitIDs))
		}
	})
}

// TestRepositoryState tests repository state management
func TestRepositoryState(t *testing.T) {
	repo := &models.Repository{
		Path: "/test/path",
		WorkingCopy: models.Commit{
			ID:        "working",
			ShortID:   "work",
			Summary:   "Working copy",
			IsWorking: true,
		},
		Graph: models.CommitGraph{
			Commits: []models.Commit{
				{
					ID:      "commit1",
					ShortID: "com1",
					Summary: "First commit",
					Parents: []string{},
				},
				{
					ID:      "commit2",
					ShortID: "com2",
					Summary: "Second commit",
					Parents: []string{"commit1"},
				},
			},
			Connections: map[string][]string{
				"commit1": {"commit2"},
			},
		},
		PRs: []models.GitHubPR{},
	}

	if repo.Path != "/test/path" {
		t.Errorf("Expected path '/test/path', got '%s'", repo.Path)
	}

	if !repo.WorkingCopy.IsWorking {
		t.Error("Expected working copy to be marked as working")
	}

	if len(repo.Graph.Commits) != 2 {
		t.Errorf("Expected 2 commits, got %d", len(repo.Graph.Commits))
	}

	if len(repo.Graph.Connections["commit1"]) != 1 {
		t.Errorf("Expected commit1 to have 1 connection, got %d", len(repo.Graph.Connections["commit1"]))
	}
}

// =============================================================================
// TUI Journey Tests
// =============================================================================

// TestJourney_BrowseCommitsAndSwitchViews tests the basic navigation flow
func TestJourney_BrowseCommitsAndSwitchViews(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	// Step 1: Start in commit graph view
	if m.GetViewMode() != tui.ViewCommitGraph {
		t.Fatalf("Expected to start in ViewCommitGraph, got %v", m.GetViewMode())
	}

	// Step 2: Switch to PRs view with 'p'
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	if m.GetViewMode() != tui.ViewPullRequests {
		t.Errorf("Expected ViewPullRequests after pressing p, got %v", m.GetViewMode())
	}

	// Step 3: Switch to Tickets view with 't'
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	if m.GetViewMode() != tui.ViewJira {
		t.Errorf("Expected ViewJira after pressing t, got %v", m.GetViewMode())
	}

	// Step 4: Switch to Help view with 'h'
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	if m.GetViewMode() != tui.ViewHelp {
		t.Errorf("Expected ViewHelp after pressing h, got %v", m.GetViewMode())
	}

	// Step 5: Return to Graph with 'g'
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	if m.GetViewMode() != tui.ViewCommitGraph {
		t.Errorf("Expected ViewCommitGraph after pressing g, got %v", m.GetViewMode())
	}
}

// TestJourney_PRStateColors tests that PR states are correctly rendered
func TestJourney_PRStateColors(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	// Set up repository with PRs of different states
	m.GetRepository().PRs = []models.GitHubPR{
		{Number: 1, Title: "Open PR", State: "open"},
		{Number: 2, Title: "Merged PR", State: "merged"},
		{Number: 3, Title: "Closed PR", State: "closed"},
	}

	// Switch to PR view
	m.SetViewMode(tui.ViewPullRequests)
	m.SetGitHubService(&github.Service{})

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

// TestJourney_TicketNavigation tests navigating through tickets
func TestJourney_TicketNavigation(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	mockService := testutil.NewMockJiraService()
	m.SetTicketService(mockService)
	m.SetTicketList(mockService.Tickets)
	m.SetViewMode(tui.ViewJira)
	m.SetSelectedTicket(0)

	// Navigate down through tickets
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.GetSelectedTicket() != 1 {
		t.Errorf("Expected selectedTicket=1, got %d", m.GetSelectedTicket())
	}

	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.GetSelectedTicket() != 2 {
		t.Errorf("Expected selectedTicket=2, got %d", m.GetSelectedTicket())
	}

	// Boundary check - should not go past last ticket
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.GetSelectedTicket() != 2 {
		t.Errorf("Expected selectedTicket=2 (at boundary), got %d", m.GetSelectedTicket())
	}

	// Navigate back up
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.GetSelectedTicket() != 1 {
		t.Errorf("Expected selectedTicket=1, got %d", m.GetSelectedTicket())
	}
}

// TestJourney_PRNavigation tests navigating through PRs
func TestJourney_PRNavigation(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	m.GetRepository().PRs = []models.GitHubPR{
		{Number: 1, Title: "PR 1", State: "open"},
		{Number: 2, Title: "PR 2", State: "open"},
		{Number: 3, Title: "PR 3", State: "merged"},
	}
	m.SetGitHubService(&github.Service{})
	m.SetViewMode(tui.ViewPullRequests)
	m.SetSelectedPR(0)

	// Navigate down
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.GetSelectedPR() != 1 {
		t.Errorf("Expected selectedPR=1, got %d", m.GetSelectedPR())
	}

	// Navigate to end
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.GetSelectedPR() != 2 {
		t.Errorf("Expected selectedPR=2, got %d", m.GetSelectedPR())
	}

	// Boundary check
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.GetSelectedPR() != 2 {
		t.Errorf("Expected selectedPR=2 (at boundary), got %d", m.GetSelectedPR())
	}

	// Navigate back to start
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.GetSelectedPR() != 0 {
		t.Errorf("Expected selectedPR=0, got %d", m.GetSelectedPR())
	}

	// Boundary check at top
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.GetSelectedPR() != 0 {
		t.Errorf("Expected selectedPR=0 (at boundary), got %d", m.GetSelectedPR())
	}
}

// TestJourney_ErrorHandling tests error state handling
func TestJourney_ErrorHandling(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	// Simulate an error by updating with an error message
	m = updateModel(m, tui.ErrorMsg(fmt.Errorf("test error")))

	if m.GetError() == nil {
		t.Error("Expected error to be set")
	}

	view := m.View()
	if !containsString(view, "Error") {
		t.Error("View should show error message")
	}

	// Press Esc to dismiss
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyEsc})

	if m.GetError() != nil {
		t.Error("Error should be cleared after Esc")
	}
}

// TestJourney_SettingsView tests the settings view
func TestJourney_SettingsView(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	// Switch to settings
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(",")})
	if m.GetViewMode() != tui.ViewSettings {
		t.Fatalf("Expected ViewSettings, got %v", m.GetViewMode())
	}

	view := m.View()

	// Check for key UI elements
	if !containsString(view, "Settings") {
		t.Error("Settings view should show title")
	}
	if !containsString(view, "GitHub") {
		t.Error("Settings view should show GitHub tab")
	}
	if !containsString(view, "Jira") {
		t.Error("Settings view should show Jira tab")
	}
	if !containsString(view, "Codecks") {
		t.Error("Settings view should show Codecks tab")
	}
	if !containsString(view, "Save") {
		t.Error("Settings view should show Save button")
	}

	// Test field navigation with Tab
	initialField := m.GetSettingsFocusedField()
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyTab})
	if m.GetSettingsFocusedField() == initialField {
		t.Error("Tab should move to next field")
	}

	// Test cancel with Esc
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.GetViewMode() != tui.ViewCommitGraph {
		t.Error("Esc should return to commit graph")
	}
}

// TestJourney_HelpView tests the help view
func TestJourney_HelpView(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	// Switch to help
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	if m.GetViewMode() != tui.ViewHelp {
		t.Fatalf("Expected ViewHelp, got %v", m.GetViewMode())
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

	// Mark the first commit as immutable
	repo := m.GetRepository()
	repo.Graph.Commits[0].Immutable = true

	// Select the immutable commit
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // Select first commit
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}) // Back to top

	view := m.View()

	// Should show immutable message instead of action buttons
	if containsString(view, "Edit (e)") && repo.Graph.Commits[m.GetSelectedCommit()].Immutable {
		t.Error("Edit button should not appear for immutable commit")
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

// BenchmarkRepositoryLoad benchmarks repository loading
func BenchmarkRepositoryLoad(b *testing.B) {
	if _, err := exec.LookPath("jj"); err != nil {
		b.Skip("jj command not available")
	}

	repo := NewTestRepository(&testing.T{})
	defer repo.Cleanup()

	// Create some commits for a more realistic benchmark
	for i := 0; i < 10; i++ {
		filename := fmt.Sprintf("file%d.txt", i)
		content := fmt.Sprintf("Content %d", i)
		message := fmt.Sprintf("Commit %d", i)
		_ = repo.commitFile(filename, content, message)
	}

	ctx := context.Background()
	service, err := jj.NewService(repo.Path)
	if err != nil {
		b.Fatalf("Failed to create jj service: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := service.GetRepository(ctx)
		if err != nil {
			b.Fatalf("Failed to get repository: %v", err)
		}
	}
}
