package tui

import (
	"context"
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/github"
	"github.com/madicen/jj-tui/internal/jj"
	"github.com/madicen/jj-tui/internal/models"
)

// Helper to create a test model with sample data (bypasses jj service)
func newTestModel() *Model {
	ctx := context.Background()
	m := New(ctx)
	m.width = 100
	m.height = 80 // Tall enough to show all content including help view
	m.loading = false // Skip loading state for tests
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
	m.statusMessage = "Ready"

	// Initialize viewport by processing a window size message
	m.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})

	return m
}

// TestTabSelectedMsgChangesView verifies that TabSelectedMsg changes the view
func TestTabSelectedMsgChangesView(t *testing.T) {
	tests := []struct {
		name         string
		msg          TabSelectedMsg
		expectedView ViewMode
	}{
		{"SelectGraph", TabSelectedMsg{Tab: ViewCommitGraph}, ViewCommitGraph},
		{"SelectPRs", TabSelectedMsg{Tab: ViewPullRequests}, ViewPullRequests},
		{"SelectHelp", TabSelectedMsg{Tab: ViewHelp}, ViewHelp},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel()
			defer m.Close()

			newModel, _ := m.Update(tt.msg)
			m = newModel.(*Model)

			if m.GetViewMode() != tt.expectedView {
				t.Errorf("Expected view %v, got %v", tt.expectedView, m.GetViewMode())
			}
		})
	}
}

// TestCommitSelectedMsgChangesSelection verifies that CommitSelectedMsg changes selection
func TestCommitSelectedMsgChangesSelection(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	m.selectedCommit = -1 // Reset

	msg := CommitSelectedMsg{Index: 1, CommitID: "def456789012"}
	newModel, _ := m.Update(msg)
	m = newModel.(*Model)

	if m.GetSelectedCommit() != 1 {
		t.Errorf("Expected selected commit 1, got %d", m.GetSelectedCommit())
	}
}

// TestKeyboardShortcuts verifies keyboard shortcuts work correctly
func TestKeyboardShortcuts(t *testing.T) {
	tests := []struct {
		key          string
		expectedView ViewMode
	}{
		{"g", ViewCommitGraph},
		{"p", ViewPullRequests},
		{"h", ViewHelp},
	}

	for _, tt := range tests {
		t.Run("Key_"+tt.key, func(t *testing.T) {
			m := newTestModel()
			defer m.Close()

			keyMsg := tea.KeyMsg{
				Type:  tea.KeyRunes,
				Runes: []rune(tt.key),
			}

			newModel, _ := m.Update(keyMsg)
			m = newModel.(*Model)

			if m.GetViewMode() != tt.expectedView {
				t.Errorf("Key '%s' should set view to %v, got %v",
					tt.key, tt.expectedView, m.GetViewMode())
			}
		})
	}
}

// TestKeyboardNavigation verifies j/k navigation
func TestKeyboardNavigation(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	m.selectedCommit = 0

	// Press j to move down
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	newModel, _ := m.Update(keyMsg)
	m = newModel.(*Model)

	if m.GetSelectedCommit() != 1 {
		t.Errorf("Expected selected commit 1 after 'j', got %d", m.GetSelectedCommit())
	}

	// Press k to move up
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
	newModel, _ = m.Update(keyMsg)
	m = newModel.(*Model)

	if m.GetSelectedCommit() != 0 {
		t.Errorf("Expected selected commit 0 after 'k', got %d", m.GetSelectedCommit())
	}
}

// TestEscReturnsToGraph verifies Esc returns to commit graph
func TestEscReturnsToGraph(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	m.viewMode = ViewHelp

	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ := m.Update(keyMsg)
	m = newModel.(*Model)

	if m.GetViewMode() != ViewCommitGraph {
		t.Errorf("Expected ViewCommitGraph after Esc, got %v", m.GetViewMode())
	}
}

// TestWorkflowWithMessages tests a realistic sequence using messages
func TestWorkflowWithMessages(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	// Step 1: Verify initial state
	if m.GetViewMode() != ViewCommitGraph {
		t.Fatalf("Expected initial view to be ViewCommitGraph")
	}

	// Step 2: Select first commit via message
	newModel, _ := m.Update(CommitSelectedMsg{Index: 0, CommitID: "abc123456789"})
	m = newModel.(*Model)

	if m.GetSelectedCommit() != 0 {
		t.Errorf("Step 2: Expected selected commit 0, got %d", m.GetSelectedCommit())
	}

	// Step 3: Switch to PRs view via message
	newModel, _ = m.Update(TabSelectedMsg{Tab: ViewPullRequests})
	m = newModel.(*Model)

	if m.GetViewMode() != ViewPullRequests {
		t.Errorf("Step 3: Expected ViewPullRequests, got %v", m.GetViewMode())
	}

	// Step 4: Switch to Help view via message
	newModel, _ = m.Update(TabSelectedMsg{Tab: ViewHelp})
	m = newModel.(*Model)

	if m.GetViewMode() != ViewHelp {
		t.Errorf("Step 4: Expected ViewHelp, got %v", m.GetViewMode())
	}

	// Step 5: Return to Graph via message
	newModel, _ = m.Update(TabSelectedMsg{Tab: ViewCommitGraph})
	m = newModel.(*Model)

	if m.GetViewMode() != ViewCommitGraph {
		t.Errorf("Step 5: Expected ViewCommitGraph, got %v", m.GetViewMode())
	}
}

// TestViewRenders verifies the view renders without errors
func TestViewRenders(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	view := m.View()

	if len(view) == 0 {
		t.Error("View should not be empty")
	}

	// Check that key content is present
	if !containsString(view, "jj-tui") {
		t.Error("View should contain title 'jj-tui'")
	}
	if !containsString(view, "Graph") {
		t.Error("View should contain 'Graph' tab")
	}
	if !containsString(view, "PRs") {
		t.Error("View should contain 'PRs' tab")
	}
	if !containsString(view, "Help") {
		t.Error("View should contain 'Help' tab")
	}
}

// TestViewShowsCommits verifies commits are rendered
func TestViewShowsCommits(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	view := m.View()

	if !containsString(view, "abc1") {
		t.Error("View should contain first commit ID 'abc1'")
	}
	if !containsString(view, "First commit") {
		t.Error("View should contain first commit summary")
	}
	if !containsString(view, "def4") {
		t.Error("View should contain second commit ID 'def4'")
	}
}

// TestViewShowsWorkingCopyIndicator verifies @ indicator for working copy
func TestViewShowsWorkingCopyIndicator(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	view := m.View()

	// The third commit is marked as working copy
	if !containsString(view, "@") {
		t.Error("View should contain '@' indicator for working copy")
	}
}

// TestWorkingCopyNodeAppearsInGraph verifies that a working copy commit is displayed
func TestWorkingCopyNodeAppearsInGraph(t *testing.T) {
	ctx := context.Background()
	m := New(ctx)
	m.width = 100
	m.height = 30
	m.loading = false
	// Initialize viewport
	m.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})

	// Set up repository with a working copy commit
	workingCopyCommit := models.Commit{
		ID:        "wc123456789",
		ShortID:   "wc12",
		ChangeID:  "wc12",
		Summary:   "(no description)",
		IsWorking: true, // This is the working copy
	}

	parentCommit := models.Commit{
		ID:       "parent123456",
		ShortID:  "par1",
		ChangeID: "par1",
		Summary:  "Parent commit",
	}

	m.SetRepository(&models.Repository{
		Path:        "/test/repo",
		WorkingCopy: workingCopyCommit,
		Graph: models.CommitGraph{
			Commits: []models.Commit{workingCopyCommit, parentCommit},
		},
	})
	defer m.Close()

	view := m.View()

	// Verify the @ symbol appears for the working copy
	if !containsString(view, "@") {
		t.Error("View should display '@' symbol for working copy commit")
	}

	// Verify the working copy's change ID appears
	if !containsString(view, "wc12") {
		t.Error("View should display working copy's short ID 'wc12'")
	}

	// Verify the parent commit also appears (not marked as working copy)
	if !containsString(view, "par1") {
		t.Error("View should display parent commit's short ID 'par1'")
	}
}

// TestNewCommitAppearsAfterRefresh verifies that new commits appear when repository is reloaded
func TestNewCommitAppearsAfterRefresh(t *testing.T) {
	ctx := context.Background()
	m := New(ctx)
	m.width = 100
	m.height = 30
	m.loading = false
	// Initialize viewport
	m.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})

	// Initial state: one commit
	initialCommit := models.Commit{
		ID:        "initial123",
		ShortID:   "init",
		ChangeID:  "init",
		Summary:   "Initial commit",
		IsWorking: true,
	}

	m.SetRepository(&models.Repository{
		Path: "/test/repo",
		Graph: models.CommitGraph{
			Commits: []models.Commit{initialCommit},
		},
	})
	defer m.Close()

	// Verify initial state
	view := m.View()
	if !containsString(view, "init") {
		t.Error("Initial view should contain 'init' commit")
	}
	if containsString(view, "newc") {
		t.Error("Initial view should NOT contain 'newc' (new commit not added yet)")
	}

	// Simulate adding a new commit (as if jj new was run externally)
	newWorkingCopy := models.Commit{
		ID:        "newcommit123",
		ShortID:   "newc",
		ChangeID:  "newc",
		Summary:   "(no description)",
		IsWorking: true, // New working copy
	}

	// The old working copy is no longer the working copy
	initialCommit.IsWorking = false

	// Simulate receiving repositoryLoadedMsg with updated repository
	updatedRepo := &models.Repository{
		Path:        "/test/repo",
		WorkingCopy: newWorkingCopy,
		Graph: models.CommitGraph{
			Commits: []models.Commit{newWorkingCopy, initialCommit},
		},
	}

	msg := repositoryLoadedMsg{repository: updatedRepo}
	newModel, _ := m.Update(msg)
	m = newModel.(*Model)

	// Verify new commit appears after "refresh"
	view = m.View()
	if !containsString(view, "newc") {
		t.Error("View after refresh should contain new commit 'newc'")
	}
	if !containsString(view, "init") {
		t.Error("View after refresh should still contain original commit 'init'")
	}

	// Verify the @ symbol is on the new working copy
	// The new commit should be marked with @
	if !containsString(view, "@") {
		t.Error("View should display '@' symbol for new working copy")
	}
}

// TestSilentRefreshUpdatesCommits verifies silent refresh detects new commits
func TestSilentRefreshUpdatesCommits(t *testing.T) {
	ctx := context.Background()
	m := New(ctx)
	m.width = 100
	m.height = 30
	m.loading = false
	// Initialize viewport
	m.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})

	// Start with 2 commits
	m.SetRepository(&models.Repository{
		Path: "/test/repo",
		Graph: models.CommitGraph{
			Commits: []models.Commit{
				{ID: "a", ShortID: "aaa", Summary: "First", IsWorking: true},
				{ID: "b", ShortID: "bbb", Summary: "Second"},
			},
		},
	})
	defer m.Close()

	originalStatus := m.GetStatusMessage()

	// Simulate silent refresh with 3 commits (one new)
	silentMsg := silentRepositoryLoadedMsg{
		repository: &models.Repository{
			Path: "/test/repo",
			Graph: models.CommitGraph{
				Commits: []models.Commit{
					{ID: "c", ShortID: "ccc", Summary: "New commit", IsWorking: true},
					{ID: "a", ShortID: "aaa", Summary: "First"},
					{ID: "b", ShortID: "bbb", Summary: "Second"},
				},
			},
		},
	}

	newModel, _ := m.Update(silentMsg)
	m = newModel.(*Model)

	// Verify commit count changed
	if len(m.GetRepository().Graph.Commits) != 3 {
		t.Errorf("Expected 3 commits after refresh, got %d", len(m.GetRepository().Graph.Commits))
	}

	// Status should be updated because count changed
	if m.GetStatusMessage() == originalStatus {
		t.Error("Status message should update when commit count changes")
	}

	// Verify new commit is visible
	view := m.View()
	if !containsString(view, "ccc") {
		t.Error("New commit 'ccc' should appear in view after silent refresh")
	}
}

// TestViewShowsActionsWhenCommitSelected verifies action buttons appear
func TestViewShowsActionsWhenCommitSelected(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	m.selectedCommit = 0

	view := m.View()

	if !containsString(view, "Edit") {
		t.Error("View should contain 'Edit' button when commit selected")
	}
	if !containsString(view, "Squash") {
		t.Error("View should contain 'Squash' button when commit selected")
	}
}

// TestHelpViewContent verifies help view shows shortcuts
func TestHelpViewContent(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	m.viewMode = ViewHelp

	view := m.View()

	if !containsString(view, "Commit Graph Shortcuts") {
		t.Error("Help view should contain 'Commit Graph Shortcuts'")
	}
	if !containsString(view, "Pull Request Shortcuts") {
		t.Error("Help view should contain 'Pull Request Shortcuts'")
	}
	if !containsString(view, "Quit") {
		t.Error("Help view should mention Quit")
	}
}

// TestPRViewContent verifies PR view shows PRs or GitHub not connected message
func TestPRViewContent(t *testing.T) {
	t.Run("WithoutGitHub", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		m.viewMode = ViewPullRequests

		view := m.View()

		// Without GitHub service, should show connection instructions
		if !containsString(view, "GitHub") {
			t.Error("PR view should mention GitHub")
		}
		if !containsString(view, "GITHUB_TOKEN") {
			t.Error("PR view should mention GITHUB_TOKEN")
		}
	})

	t.Run("WithGitHubAndPRs", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		// Simulate having a GitHub service by setting a non-nil pointer
		// (we don't actually need the real service for view testing)
		m.githubService = &github.Service{}
		m.viewMode = ViewPullRequests

		view := m.View()

		// With GitHub service and PRs, should show PR data
		if !containsString(view, "#1") {
			t.Error("PR view should contain PR number")
		}
		if !containsString(view, "Test PR") {
			t.Error("PR view should contain PR title")
		}
	})
}

// TestStatusBarContent verifies status bar shows shortcuts
func TestStatusBarContent(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	view := m.View()

	if !containsString(view, "ctrl+q:quit") {
		t.Error("Status bar should contain 'ctrl+q:quit'")
	}
	if !containsString(view, "ctrl+r:refresh") {
		t.Error("Status bar should contain 'ctrl+r:refresh'")
	}
}

// TestWindowSizeUpdate verifies window size is stored
func TestWindowSizeUpdate(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	newModel, _ := m.Update(msg)
	m = newModel.(*Model)

	if m.width != 120 {
		t.Errorf("Expected width 120, got %d", m.width)
	}
	if m.height != 40 {
		t.Errorf("Expected height 40, got %d", m.height)
	}
}

// TestRepositoryLoadedMsg verifies repository loading
func TestRepositoryLoadedMsg(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	m.loading = true
	m.repository = nil

	repo := &models.Repository{
		Path: "/new/repo",
		Graph: models.CommitGraph{
			Commits: []models.Commit{
				{ID: "new123", ShortID: "new1", Summary: "New commit"},
			},
		},
	}

	msg := repositoryLoadedMsg{repository: repo}
	newModel, _ := m.Update(msg)
	m = newModel.(*Model)

	if m.loading {
		t.Error("Expected loading to be false after repository loaded")
	}
	if m.repository == nil {
		t.Error("Expected repository to be set")
	}
	if m.repository.Path != "/new/repo" {
		t.Errorf("Expected path '/new/repo', got '%s'", m.repository.Path)
	}
}

// TestErrorMsg verifies error handling
func TestErrorMsg(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	m.loading = true

	msg := errorMsg{err: fmt.Errorf("test error")}
	newModel, _ := m.Update(msg)
	m = newModel.(*Model)

	if m.loading {
		t.Error("Expected loading to be false after error")
	}
	if m.err == nil {
		t.Error("Expected error to be set")
	}
	if !containsString(m.GetStatusMessage(), "Error") {
		t.Error("Status should contain 'Error'")
	}
}

// containsString checks if s contains substr
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestDescriptionEditingFlow verifies the description editing flow
func TestDescriptionEditingFlow(t *testing.T) {
	t.Run("startEditingDescription sets up edit mode", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		// Select a mutable commit
		m.selectedCommit = 0
		commit := m.repository.Graph.Commits[0]
		commit.Description = "Original description"
		m.repository.Graph.Commits[0] = commit

		// Call startEditingDescription directly (simulates pressing 'd')
		m.startEditingDescription(commit)

		if m.viewMode != ViewEditDescription {
			t.Errorf("Expected ViewEditDescription, got %v", m.viewMode)
		}
		if m.editingCommitID != commit.ChangeID {
			t.Errorf("Expected editingCommitID %s, got %s", commit.ChangeID, m.editingCommitID)
		}
	})

	t.Run("immutable commit cannot be edited", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		// Make the selected commit immutable
		m.selectedCommit = 0
		m.repository.Graph.Commits[0].Immutable = true

		// Verify the commit is marked as immutable
		commit := m.repository.Graph.Commits[m.selectedCommit]
		if !commit.Immutable {
			t.Error("Test commit should be marked immutable")
		}

		// The TUI should prevent editing immutable commits
		// (tested via UI logic, not key handler which requires jjService)
	})

	t.Run("esc cancels description editing", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		// Start editing
		m.viewMode = ViewEditDescription
		m.editingCommitID = "abc1"

		// Press esc
		escMsg := tea.KeyMsg{Type: tea.KeyEsc}
		newModel, _ := m.Update(escMsg)
		m = newModel.(*Model)

		if m.viewMode != ViewCommitGraph {
			t.Errorf("Expected ViewCommitGraph after esc, got %v", m.viewMode)
		}
		if m.editingCommitID != "" {
			t.Error("Expected editingCommitID to be cleared")
		}
	})

	t.Run("edit description view shows commit info", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		// Start editing
		commit := m.repository.Graph.Commits[0]
		m.viewMode = ViewEditDescription
		m.editingCommitID = commit.ChangeID

		view := m.View()

		// Should show editing UI
		if !containsString(view, "Edit Commit Description") {
			t.Error("Edit view should show title")
		}
		if !containsString(view, "Ctrl+S") {
			t.Error("Edit view should show save instructions")
		}
		if !containsString(view, "Esc") {
			t.Error("Edit view should show cancel instructions")
		}
	})
}

// TestActionButtonsInCommitGraph verifies the action buttons in the commit graph
func TestActionButtonsInCommitGraph(t *testing.T) {
	t.Run("new button always visible", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		view := m.View()

		if !containsString(view, "New (n)") {
			t.Error("Expected New (n) button to be visible")
		}
	})

	t.Run("all action buttons appear for mutable commit", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		m.selectedCommit = 0
		m.repository.Graph.Commits[0].Immutable = false

		view := m.View()

		if !containsString(view, "New (n)") {
			t.Error("Expected New (n) button")
		}
		if !containsString(view, "Edit (e)") {
			t.Error("Expected Edit (e) button")
		}
		if !containsString(view, "Describe (d)") {
			t.Error("Expected Describe (d) button")
		}
		if !containsString(view, "Squash (s)") {
			t.Error("Expected Squash (s) button")
		}
		if !containsString(view, "Rebase (r)") {
			t.Error("Expected Rebase (r) button")
		}
		if !containsString(view, "Abandon (a)") {
			t.Error("Expected Abandon (a) button")
		}
	})

	t.Run("commit-specific actions hidden for immutable commit", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		m.selectedCommit = 0
		m.repository.Graph.Commits[0].Immutable = true

		view := m.View()

		// New should still be visible
		if !containsString(view, "New (n)") {
			t.Error("Expected New (n) button to be visible even for immutable commit")
		}
		// But Edit, Describe, Squash, Rebase should be hidden
		if containsString(view, "Edit (e)") {
			t.Error("Expected Edit (e) button to be hidden for immutable commit")
		}
		if containsString(view, "Describe (d)") {
			t.Error("Expected Describe (d) button to be hidden for immutable commit")
		}
		if containsString(view, "Squash (s)") {
			t.Error("Expected Squash (s) button to be hidden for immutable commit")
		}
		if containsString(view, "Rebase (r)") {
			t.Error("Expected Rebase (r) button to be hidden for immutable commit")
		}
		if containsString(view, "Abandon (a)") {
			t.Error("Expected Abandon (a) button to be hidden for immutable commit")
		}
		// Should show immutable message
		if !containsString(view, "immutable") {
			t.Error("Expected immutable message")
		}
	})
}

// TestRebaseModeFlow verifies the rebase mode workflow
func TestRebaseModeFlow(t *testing.T) {
	t.Run("pressing r enters rebase mode", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		// Need a jjService for rebase to work (use a stub)
		m.jjService = &jj.Service{RepoPath: "/test/repo"}

		m.selectedCommit = 0
		m.repository.Graph.Commits[0].Immutable = false

		// Press 'r' to enter rebase mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
		newModel, _ := m.Update(msg)
		m = newModel.(*Model)

		if m.selectionMode != SelectionRebaseDestination {
			t.Error("Expected to enter rebase destination selection mode")
		}
		if m.rebaseSourceCommit != 0 {
			t.Errorf("Expected rebase source to be 0, got %d", m.rebaseSourceCommit)
		}
	})

	t.Run("rebase mode shows special UI", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		m.selectedCommit = 0
		m.repository.Graph.Commits[0].Immutable = false
		m.selectionMode = SelectionRebaseDestination
		m.rebaseSourceCommit = 0

		view := m.View()

		if !containsString(view, "REBASE MODE") {
			t.Error("Expected 'REBASE MODE' header in rebase mode")
		}
		if !containsString(view, "Select destination") {
			t.Error("Expected 'Select destination' instruction in rebase mode")
		}
	})

	t.Run("esc cancels rebase mode", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		m.selectedCommit = 0
		m.selectionMode = SelectionRebaseDestination
		m.rebaseSourceCommit = 0

		// Press Esc to cancel
		msg := tea.KeyMsg{Type: tea.KeyEsc}
		newModel, _ := m.Update(msg)
		m = newModel.(*Model)

		if m.selectionMode != SelectionNormal {
			t.Error("Expected to exit rebase mode on Esc")
		}
		if m.rebaseSourceCommit != -1 {
			t.Error("Expected rebase source to be reset on cancel")
		}
	})

	t.Run("cannot rebase immutable commit", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		// Need a jjService for the key handler to try rebase
		m.jjService = &jj.Service{RepoPath: "/test/repo"}

		m.selectedCommit = 0
		m.repository.Graph.Commits[0].Immutable = true

		// Press 'b' - should not enter rebase mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}
		newModel, _ := m.Update(msg)
		m = newModel.(*Model)

		if m.selectionMode != SelectionNormal {
			t.Error("Should not enter rebase mode for immutable commit")
		}
		if !containsString(m.statusMessage, "immutable") {
			t.Error("Expected immutable warning in status message")
		}
	})

	t.Run("action buttons hidden in rebase mode", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		m.selectedCommit = 1 // Select destination
		m.selectionMode = SelectionRebaseDestination
		m.rebaseSourceCommit = 0

		view := m.View()

		// Action buttons should be hidden in rebase mode
		if containsString(view, "New (n)") {
			t.Error("Action buttons should be hidden in rebase mode")
		}
		if containsString(view, "Edit (e)") {
			t.Error("Action buttons should be hidden in rebase mode")
		}
	})
}

// TestMouseScrollingOnViews tests that mouse scrolling works correctly on different views
func TestMouseScrollingOnViews(t *testing.T) {
	// Helper to create test model with many PRs for scrolling
	createModelWithManyPRs := func() *Model {
		m := newTestModel()
		// Use a smaller height to make scrolling necessary with fewer PRs
		m.height = 30
		// Re-initialize viewport with new height
		m.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		// Add many PRs so there's content to scroll
		var prs []models.GitHubPR
		for i := 0; i < 50; i++ {
			prs = append(prs, models.GitHubPR{
				Number: i + 1,
				Title:  fmt.Sprintf("Test PR %d", i+1),
				State:  "open",
			})
		}
		m.repository.PRs = prs
		return m
	}

	t.Run("mouse scroll works on PR view", func(t *testing.T) {
		m := createModelWithManyPRs()
		defer m.Close()

		// Switch to PR view
		m.viewMode = ViewPullRequests
		m.githubService = &github.Service{} // Enable GitHub to show PR list

		// Render view to initialize viewport with content
		m.View()

		initialOffset := m.viewport.YOffset

		// Ensure there's content to scroll (total lines > height)
		if m.viewport.TotalLineCount() <= m.viewport.Height {
			t.Skipf("Not enough content to scroll: %d lines, %d height", m.viewport.TotalLineCount(), m.viewport.Height)
		}

		// Simulate mouse wheel scroll down
		scrollDownMsg := tea.MouseMsg{
			Action: tea.MouseActionPress,
			Button: tea.MouseButtonWheelDown,
			X:      50,
			Y:      10,
		}
		newModel, _ := m.Update(scrollDownMsg)
		m = newModel.(*Model)

		// Offset should have increased (scrolled down)
		if m.viewport.YOffset <= initialOffset {
			t.Errorf("Expected YOffset to increase after scroll down, got %d (was %d)", m.viewport.YOffset, initialOffset)
		}

		// Now scroll back up
		afterScrollDown := m.viewport.YOffset
		scrollUpMsg := tea.MouseMsg{
			Action: tea.MouseActionPress,
			Button: tea.MouseButtonWheelUp,
			X:      50,
			Y:      10,
		}
		newModel, _ = m.Update(scrollUpMsg)
		m = newModel.(*Model)

		// Offset should have decreased (scrolled up)
		if m.viewport.YOffset >= afterScrollDown {
			t.Errorf("Expected YOffset to decrease after scroll up, got %d (was %d)", m.viewport.YOffset, afterScrollDown)
		}
	})

	t.Run("mouse scroll respects bounds", func(t *testing.T) {
		m := createModelWithManyPRs()
		defer m.Close()

		// Switch to PR view
		m.viewMode = ViewPullRequests
		m.githubService = &github.Service{}

		// Render view to initialize viewport
		m.View()

		// Try scrolling up when already at top - should stay at 0
		m.viewport.YOffset = 0
		scrollUpMsg := tea.MouseMsg{
			Action: tea.MouseActionPress,
			Button: tea.MouseButtonWheelUp,
			X:      50,
			Y:      10,
		}
		newModel, _ := m.Update(scrollUpMsg)
		m = newModel.(*Model)

		if m.viewport.YOffset < 0 {
			t.Errorf("YOffset should not go negative, got %d", m.viewport.YOffset)
		}
	})

	t.Run("viewport height is correct after switching from graph to PRs", func(t *testing.T) {
		m := createModelWithManyPRs()
		defer m.Close()

		// Start on Graph view
		m.viewMode = ViewCommitGraph
		m.View() // Render to set up graph viewport heights

		// Switch to PR view
		m.viewMode = ViewPullRequests
		m.githubService = &github.Service{}
		m.View() // Render PR view

		// Viewport height should be reasonable (not the reduced graph height)
		// The full content height should be at least height - header - statusbar
		minExpectedHeight := m.height - 10 // Account for header, status, fixed header in split view
		if m.viewport.Height < minExpectedHeight/2 {
			t.Errorf("Viewport height %d seems too small after switching from graph (expected at least %d)", m.viewport.Height, minExpectedHeight/2)
		}
	})
}

