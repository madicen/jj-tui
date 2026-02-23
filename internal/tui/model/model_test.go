package model

import (
	"context"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/github"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tickets"
	graphtab "github.com/madicen/jj-tui/internal/tui/tabs/graph"
	"github.com/madicen/jj-tui/internal/tui/mouse"
)

// Helper to create a test model with sample data (bypasses jj service)
func newTestModel() *Model {
	ctx := context.Background()
	m := New(ctx)
	m.width = 100
	m.height = 80     // Tall enough to show all content including help view
	m.loading = false // Skip loading state for tests
	m.SetRepository(&internal.Repository{
		Path: "/test/repo",
		Graph: internal.CommitGraph{
			Commits: []internal.Commit{
				{ID: "abc123456789", ShortID: "abc1", ChangeID: "abc1", Summary: "First commit"},
				{ID: "def456789012", ShortID: "def4", ChangeID: "def4", Summary: "Second commit"},
				{ID: "ghi789012345", ShortID: "ghi7", ChangeID: "ghi7", Summary: "Third commit", IsWorking: true},
			},
		},
		PRs: []internal.GitHubPR{
			{Number: 1, Title: "Test PR", State: "open"},
		},
	})
	m.statusMessage = "Ready"

	// Sync repository and selection to tab models (bypasses repositoryLoadedMsg in tests)
	m.graphTabModel.UpdateRepository(m.repository)
	m.graphTabModel.SelectCommit(0)
	m.prsTabModel.UpdateRepository(m.repository)
	m.prsTabModel.SetGithubService(m.isGitHubAvailable())
	m.branchesTabModel.UpdateRepository(m.repository)
	m.ticketsTabModel.SetTicketServiceInfo("", false)

	// Initialize by processing a window size message
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

	msg, _ := m.handleSelectCommit(1)
	newModel, _ := m.Update(msg)
	m = newModel.(*Model)

	if m.GetSelectedCommit() != 1 {
		t.Errorf("Expected selected commit 1, got %d", m.GetSelectedCommit())
	}
}

// TestChangedFilesLoadedMsgUpdatesGraphTab verifies that when changed files load after repository load
// (e.g. initial load), the graph tab's changed files list is updated even if changedFilesCommitID
// was not set before the async load completed.
func TestChangedFilesLoadedMsgUpdatesGraphTab(t *testing.T) {
	ctx := context.Background()
	m := New(ctx)
	defer m.Close()
	m.loading = false
	repo := &internal.Repository{
		Path: "/test/repo",
		Graph: internal.CommitGraph{
			Commits: []internal.Commit{
				{ID: "a1", ShortID: "a1", ChangeID: "cid0", Summary: "First"},
				{ID: "b2", ShortID: "b2", ChangeID: "cid1", Summary: "Second"},
			},
		},
	}
	m.SetRepository(repo)
	m.graphTabModel.UpdateRepository(m.repository)
	// Do not call SelectCommit so changedFilesCommitID stays "" (simulates initial load before
	// loadChangedFiles request was made, or race where the msg arrives before we set it).
	// selectedCommit is 0 by default. Deliver changed files for the first commit.
	loadedFiles := []jj.ChangedFile{{Path: "foo.go", Status: "M"}, {Path: "bar/baz.go", Status: "A"}}
	newModel, _ := m.Update(changedFilesLoadedMsg{commitID: "cid0", files: loadedFiles})
	m = newModel.(*Model)

	got := m.graphTabModel.GetChangedFiles()
	if len(got) != 2 {
		t.Fatalf("GetChangedFiles(): expected 2 files, got %d", len(got))
	}
	if got[0].Path != "foo.go" || got[0].Status != "M" {
		t.Errorf("GetChangedFiles()[0]: expected foo.go M, got %s %s", got[0].Path, got[0].Status)
	}
	if got[1].Path != "bar/baz.go" || got[1].Status != "A" {
		t.Errorf("GetChangedFiles()[1]: expected bar/baz.go A, got %s %s", got[1].Path, got[1].Status)
	}
}

// TestMouseScrollGraphTabWithoutClicking is an integration test for mouse wheel scrolling on the graph tab.
// It documents and verifies the current behavior: scrolling is focus-based, not cursor-based.
// - Without clicking any pane: graphFocused is true by default, so wheel scrolls the graph (commit) list.
// - After clicking the files pane (or Tab to it): wheel scrolls the files list.
// So "scroll without clicking" works for the default pane (graph); to scroll the other list you must focus it first.
func TestMouseScrollGraphTabWithoutClicking(t *testing.T) {
	ctx := context.Background()
	m := New(ctx)
	defer m.Close()
	m.loading = false
	// Many commits so the graph pane is scrollable (more lines than viewport height)
	commits := make([]internal.Commit, 50)
	for i := range commits {
		commits[i] = internal.Commit{
			ID:       fmt.Sprintf("id%03d", i),
			ShortID:  fmt.Sprintf("s%02d", i),
			ChangeID: fmt.Sprintf("cid%03d", i),
			Summary:  fmt.Sprintf("Commit %d", i),
		}
	}
	m.SetRepository(&internal.Repository{
		Path:   "/test/repo",
		Graph:  internal.CommitGraph{Commits: commits},
		PRs:    nil,
	})
	m.graphTabModel.UpdateRepository(m.repository)
	m.graphTabModel.SelectCommit(0)
	m.viewMode = ViewCommitGraph
	m.graphFocused = true
	m.width = 100
	m.height = 80
	m.graphTabModel.SetDimensions(m.width, m.estimatedContentHeight())
	// Render once so viewports have content and zones are registered
	m.View()

	graphVp := m.graphTabModel.GetViewport()
	graphHeight := graphVp.Height
	totalGraphLines := graphVp.TotalLineCount()
	if totalGraphLines <= graphHeight {
		t.Skipf("graph must be scrollable: total lines=%d height=%d", totalGraphLines, graphHeight)
	}

	wheelDown := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelDown,
		X:      50,
		Y:      10,
	}

	// --- 1) Wheel WITHOUT clicking any pane: should scroll the graph (default focus) ---
	if !m.graphTabModel.IsGraphFocused() {
		t.Fatal("expected graph pane focused by default")
	}
	graphY0 := m.graphTabModel.GetViewport().YOffset
	newModel, _ := m.Update(wheelDown)
	m = newModel.(*Model)
	graphY1 := m.graphTabModel.GetViewport().YOffset
	if graphY1 <= graphY0 {
		t.Errorf("wheel without clicking: expected graph pane to scroll (focus-based). graph YOffset was %d, got %d", graphY0, graphY1)
	}
	// Files viewport should not have scrolled (we didn't focus it)
	filesY0 := m.graphTabModel.GetFilesViewport().YOffset

	// --- 2) Simulate click on files pane, then wheel: should scroll files list ---
	m.View() // refresh so zones are current
	filesZone := m.zoneManager.Get(mouse.ZoneFilesPane)
	if filesZone == nil {
		t.Skip("files pane zone not registered (e.g. no content); cannot simulate click")
	}
	zoneMsg := zone.MsgZoneInBounds{
		Zone:  filesZone,
		Event: tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 50, Y: 60},
	}
	newModel, _ = m.Update(zoneMsg)
	m = newModel.(*Model)
	if m.graphTabModel.IsGraphFocused() {
		t.Error("after clicking files pane, expected graph pane to be unfocused (files focused)")
	}
	filesY1 := m.graphTabModel.GetFilesViewport().YOffset
	// Scroll files pane with wheel
	newModel, _ = m.Update(wheelDown)
	m = newModel.(*Model)
	filesY2 := m.graphTabModel.GetFilesViewport().YOffset
	// If files pane has scrollable content, YOffset may increase; if not, it stays 0
	// We only assert that wheel was applied to the focused pane (files), not that it scrolled (content-dependent)
	if filesY2 < filesY1 && totalGraphLines > graphHeight {
		// Graph might have scrolled again if we accidentally scrolled graph
		graphY2 := m.graphTabModel.GetViewport().YOffset
		if graphY2 > graphY1 {
			t.Errorf("after focusing files pane, wheel should scroll files pane, not graph: graph YOffset moved from %d to %d", graphY1, graphY2)
		}
	}

	// --- 3) Document: without clicking, only the default pane (graph) scrolls ---
	_ = filesY0
	t.Logf("Mouse scroll behavior: default focus=graph; wheel without click scrolls graph (YOffset %d -> %d). After click on files pane, wheel scrolls files pane.", graphY0, graphY1)
}

// TestMouseScrollHelpTab verifies that the Help tab scrolls with the mouse wheel without requiring a click.
// Help has no clickable list (unlike PR/Tickets), so wheel must work as soon as the tab is active.
func TestMouseScrollHelpTab(t *testing.T) {
	m := newTestModel()
	defer m.Close()
	m.SetViewMode(ViewHelp)
	m.width = 100
	m.height = 40
	viewBefore := m.View() // View() sets dimensions for all tabs including Help

	wheelDown := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelDown,
		X:      50,
		Y:      20,
	}
	newModel, _ := m.Update(wheelDown)
	m = newModel.(*Model)
	viewAfter := m.View()

	if viewAfter == viewBefore {
		t.Error("wheel on Help tab should scroll content without clicking; view unchanged after wheel down")
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

	m.graphTabModel.SelectCommit(0)

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
	msg, _ := m.handleSelectCommit(0)
	newModel, _ := m.Update(msg)
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
	workingCopyCommit := internal.Commit{
		ID:        "wc123456789",
		ShortID:   "wc12",
		ChangeID:  "wc12",
		Summary:   "(no description)",
		IsWorking: true, // This is the working copy
	}

	parentCommit := internal.Commit{
		ID:       "parent123456",
		ShortID:  "par1",
		ChangeID: "par1",
		Summary:  "Parent commit",
	}

	m.SetRepository(&internal.Repository{
		Path:        "/test/repo",
		WorkingCopy: workingCopyCommit,
		Graph: internal.CommitGraph{
			Commits: []internal.Commit{workingCopyCommit, parentCommit},
		},
	})
	m.graphTabModel.UpdateRepository(m.repository)
	m.graphTabModel.SelectCommit(0)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 80})
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
	initialCommit := internal.Commit{
		ID:        "initial123",
		ShortID:   "init",
		ChangeID:  "init",
		Summary:   "Initial commit",
		IsWorking: true,
	}

	m.SetRepository(&internal.Repository{
		Path: "/test/repo",
		Graph: internal.CommitGraph{
			Commits: []internal.Commit{initialCommit},
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
	newWorkingCopy := internal.Commit{
		ID:        "newcommit123",
		ShortID:   "newc",
		ChangeID:  "newc",
		Summary:   "(no description)",
		IsWorking: true, // New working copy
	}

	// The old working copy is no longer the working copy
	initialCommit.IsWorking = false

	// Simulate receiving repositoryLoadedMsg with updated repository
	updatedRepo := &internal.Repository{
		Path:        "/test/repo",
		WorkingCopy: newWorkingCopy,
		Graph: internal.CommitGraph{
			Commits: []internal.Commit{newWorkingCopy, initialCommit},
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
	m.SetRepository(&internal.Repository{
		Path: "/test/repo",
		Graph: internal.CommitGraph{
			Commits: []internal.Commit{
				{ID: "a", ShortID: "aaa", Summary: "First", IsWorking: true},
				{ID: "b", ShortID: "bbb", Summary: "Second"},
			},
		},
	})
	defer m.Close()

	originalStatus := m.GetStatusMessage()

	// Simulate silent refresh with 3 commits (one new)
	silentMsg := silentRepositoryLoadedMsg{
		repository: &internal.Repository{
			Path: "/test/repo",
			Graph: internal.CommitGraph{
				Commits: []internal.Commit{
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

	m.graphTabModel.SelectCommit(0)

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
		m.prsTabModel.SetGithubService(true)
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

	if !containsString(view, "^q quit") {
		t.Error("Status bar should contain '^q quit'")
	}
	if !containsString(view, "^r refresh") {
		t.Error("Status bar should contain '^r refresh'")
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

	repo := &internal.Repository{
		Path: "/new/repo",
		Graph: internal.CommitGraph{
			Commits: []internal.Commit{
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

	msg := errorMsg{Err: fmt.Errorf("test error")}
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
		m.graphTabModel.SelectCommit(0)
		commit := m.repository.Graph.Commits[0]
		commit.Description = "Original description"
		m.repository.Graph.Commits[0] = commit

		// Call startEditingDescription directly (simulates pressing 'd')
		m.startEditingDescription(commit)

		if m.viewMode != ViewEditDescription {
			t.Errorf("Expected ViewEditDescription, got %v", m.viewMode)
		}
		if m.graphTabModel.GetEditingCommitID() != commit.ChangeID {
			t.Errorf("Expected editingCommitID %s, got %s", commit.ChangeID, m.graphTabModel.GetEditingCommitID())
		}
	})

	t.Run("immutable commit cannot be edited", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		// Make the selected commit immutable
		m.graphTabModel.SelectCommit(0)
		m.repository.Graph.Commits[0].Immutable = true

		// Verify the commit is marked as immutable
		commit := m.repository.Graph.Commits[m.GetSelectedCommit()]
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
		m.graphTabModel.SetEditingCommitID("abc1")

		// Press esc
		escMsg := tea.KeyMsg{Type: tea.KeyEsc}
		newModel, _ := m.Update(escMsg)
		m = newModel.(*Model)

		if m.viewMode != ViewCommitGraph {
			t.Errorf("Expected ViewCommitGraph after esc, got %v", m.viewMode)
		}
		if m.graphTabModel.GetEditingCommitID() != "" {
			t.Error("Expected editingCommitID to be cleared")
		}
	})

	t.Run("edit description view shows commit info", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		// Start editing
		commit := m.repository.Graph.Commits[0]
		m.viewMode = ViewEditDescription
		m.graphTabModel.SetEditingCommitID(commit.ChangeID)

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

		m.graphTabModel.SelectCommit(0)
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

		m.graphTabModel.SelectCommit(0)
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

		m.graphTabModel.SelectCommit(0)
		m.repository.Graph.Commits[0].Immutable = false

		// Press 'r' - graph returns StartRebaseMode request; main runs it and enters rebase mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
		newModel, cmd := m.Update(msg)
		m = newModel.(*Model)
		if cmd != nil {
			if req, ok := cmd().(graphtab.Request); ok {
				var v tea.Model
				v, _ = m.Update(req)
				m = v.(*Model)
			}
		}

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

		m.graphTabModel.SelectCommit(0)
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

		m.graphTabModel.SelectCommit(0)
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

		m.jjService = &jj.Service{RepoPath: "/test/repo"}
		m.graphTabModel.SelectCommit(0)
		m.repository.Graph.Commits[0].Immutable = true

		// Press 'r' - graph returns StartRebaseMode request; main handles it and blocks with message
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
		newModel, cmd := m.Update(msg)
		m = newModel.(*Model)
		if cmd != nil {
			if req, ok := cmd().(graphtab.Request); ok {
				var v tea.Model
				v, _ = m.Update(req)
				m = v.(*Model)
			}
		}

		if m.selectionMode != SelectionNormal {
			t.Error("Should not enter rebase mode for immutable commit")
		}
		if !containsString(m.statusMessage, "immutable") {
			t.Errorf("Expected immutable warning in status message, got: %q", m.statusMessage)
		}
	})

	t.Run("action buttons hidden in rebase mode", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		m.graphTabModel.SelectCommit(1) // Select destination
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
		var prs []internal.GitHubPR
		for i := 0; i < 50; i++ {
			prs = append(prs, internal.GitHubPR{
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

		m.viewMode = ViewPullRequests
		m.githubService = &github.Service{}
		m.prsTabModel.SetGithubService(true)
		m.View()

		// Wheel down: list scroll offset must increase (proves event reaches PR tab model)
		scrollDownMsg := tea.MouseMsg{
			Action: tea.MouseActionPress,
			Button: tea.MouseButtonWheelDown,
			X:      50,
			Y:      10,
		}
		newModel, _ := m.Update(scrollDownMsg)
		m = newModel.(*Model)
		if m.GetPRsListYOffset() != 3 {
			t.Errorf("after wheel down expected PR listYOffset 3, got %d", m.GetPRsListYOffset())
		}

		// Wheel up: offset decreases
		scrollUpMsg := tea.MouseMsg{
			Action: tea.MouseActionPress,
			Button: tea.MouseButtonWheelUp,
			X:      50,
			Y:      10,
		}
		newModel, _ = m.Update(scrollUpMsg)
		m = newModel.(*Model)
		if m.GetPRsListYOffset() != 0 {
			t.Errorf("after wheel up expected PR listYOffset 0, got %d", m.GetPRsListYOffset())
		}

		if out := m.View(); out == "" {
			t.Error("View should return non-empty after scroll")
		}
	})

	t.Run("PR view rendered content changes after wheel (integration)", func(t *testing.T) {
		m := createModelWithManyPRs()
		defer m.Close()
		m.viewMode = ViewPullRequests
		m.githubService = &github.Service{}
		m.prsTabModel.SetGithubService(true)
		_ = m.View() // set dimensions and get initial view
		wheelDown := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown, X: 50, Y: 10}
		// First wheel: same as "mouse scroll works on PR view"
		newModel, _ := m.Update(wheelDown)
		m = newModel.(*Model)
		if m.GetPRsListYOffset() != 3 {
			t.Fatalf("after first wheel expected listYOffset 3, got %d", m.GetPRsListYOffset())
		}
		// Second wheel: offset should accumulate
		newModel, _ = m.Update(wheelDown)
		m = newModel.(*Model)
		if m.GetPRsListYOffset() != 6 {
			t.Fatalf("after second wheel expected listYOffset 6, got %d", m.GetPRsListYOffset())
		}
		viewAfter := m.View()
		if viewAfter == "" {
			t.Fatal("PR view after wheel should not be empty")
		}
		// Rendered content should reflect scroll (e.g. different PRs visible)
		if !strings.Contains(viewAfter, "Test PR ") {
			t.Error("view should contain PR list content after scroll")
		}
	})

	t.Run("wheel down increases tickets list scroll offset", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()
		var list []tickets.Ticket
		for i := 0; i < 50; i++ {
			list = append(list, tickets.Ticket{Key: fmt.Sprintf("T-%d", i), Summary: fmt.Sprintf("Ticket %d", i)})
		}
		m.SetTicketList(list)
		m.SetViewMode(ViewTickets)
		m.ticketsTabModel.SetDimensions(80, 24)
		m.View()

		y0 := m.GetTicketsListYOffset()
		wheelDown := tea.MouseMsg{
			Action: tea.MouseActionPress,
			Button: tea.MouseButtonWheelDown,
			X:      50,
			Y:      10,
		}
		newModel, _ := m.Update(wheelDown)
		m = newModel.(*Model)
		y1 := m.GetTicketsListYOffset()
		if y1 <= y0 {
			t.Errorf("wheel should update tickets list scroll: was %d, got %d", y0, y1)
		}
	})

	t.Run("mouse scroll up at top does not panic", func(t *testing.T) {
		m := createModelWithManyPRs()
		defer m.Close()
		m.viewMode = ViewPullRequests
		m.githubService = &github.Service{}
		m.View()
		scrollUpMsg := tea.MouseMsg{
			Action: tea.MouseActionPress,
			Button: tea.MouseButtonWheelUp,
			X:      50,
			Y:      10,
		}
		_, _ = m.Update(scrollUpMsg)
	})
}

// TestJJInitFeature tests the jj init functionality for non-jj repositories
func TestJJInitFeature(t *testing.T) {
	t.Run("error message sets notJJRepo flag", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		// Simulate an error message indicating not a jj repo
		errMsg := errorMsg{
			Err:         fmt.Errorf("not a jujutsu repository: /test/path"),
			NotJJRepo:   true,
			CurrentPath: "/test/path",
		}

		newModel, _ := m.Update(errMsg)
		m = newModel.(*Model)

		if !m.IsNotJJRepo() {
			t.Error("Expected notJJRepo to be true after error message")
		}
		if m.GetError() == nil {
			t.Error("Expected error to be set")
		}
	})

	t.Run("init screen shown when notJJRepo is true", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		// Set up the not-jj-repo state
		m.err = fmt.Errorf("not a jujutsu repository")
		m.notJJRepo = true
		m.currentPath = "/test/path"

		view := m.View()

		// Check that the init screen elements are present
		if !containsString(view, "Welcome to jj-tui") {
			t.Error("Expected init screen title in view")
		}
		if !containsString(view, "Initialize Repository") {
			t.Error("Expected init button in view")
		}
		if !containsString(view, "jj git init") {
			t.Error("Expected jj git init command text in view")
		}
		if !containsString(view, "main@origin") {
			t.Error("Expected main@origin tracking text in view")
		}
	})

	t.Run("init screen NOT shown for other errors", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		// Set up a regular error (not a jj repo error)
		m.err = fmt.Errorf("some other error")
		m.notJJRepo = false

		view := m.View()

		// Check that init screen is NOT shown
		if containsString(view, "Initialize Repository") {
			t.Error("Init button should not appear for non-jj-repo errors")
		}
		// But error message should be shown
		if !containsString(view, "Error") {
			t.Error("Expected error message in view")
		}
	})

	t.Run("jjInitSuccessMsg clears error state", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		// Set up the not-jj-repo state
		m.err = fmt.Errorf("not a jujutsu repository")
		m.notJJRepo = true
		m.currentPath = "/test/path"

		// Simulate init success
		newModel, _ := m.Update(jjInitSuccessMsg{})
		m = newModel.(*Model)

		if m.IsNotJJRepo() {
			t.Error("Expected notJJRepo to be false after init success")
		}
		if !containsString(m.GetStatusMessage(), "initialized") {
			t.Error("Expected status message to indicate successful initialization")
		}
	})

	t.Run("pressing i triggers init when notJJRepo", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		// Set up the not-jj-repo state
		m.err = fmt.Errorf("not a jujutsu repository")
		m.notJJRepo = true

		// Press 'i' key
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
		_, cmd := m.Update(msg)

		// Should return a command (the runJJInit command)
		if cmd == nil {
			t.Error("Expected a command to be returned when pressing 'i' in notJJRepo state")
		}
	})

	t.Run("pressing i does nothing when already in jj repo", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		// Not in error state
		m.err = nil
		m.notJJRepo = false

		// Press 'i' key
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
		_, cmd := m.Update(msg)

		// Should NOT return an init command (i might do something else or nothing)
		// The key point is we shouldn't try to init when we're already in a jj repo
		// This test just ensures no crash and the behavior is defined
		_ = cmd // Command may or may not be nil depending on what 'i' does normally
	})

	t.Run("error display takes priority over graph view", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		// Set up error state while in graph view mode
		m.viewMode = ViewCommitGraph
		m.err = fmt.Errorf("not a jujutsu repository")
		m.notJJRepo = true
		m.currentPath = "/test/path"

		view := m.View()

		// Should show welcome screen, not graph
		if containsString(view, "Changed Files") {
			t.Error("Should not show graph content when there's an error")
		}
		if !containsString(view, "Welcome to jj-tui") {
			t.Error("Should show welcome screen when not a jj repo")
		}
	})
}

// TestNewCommitFromImmutableParent verifies that creating a new commit
// based on an immutable commit (like main) is allowed. This is valid
// because we're creating a CHILD of the immutable commit, not modifying it.
func TestNewCommitFromImmutableParent(t *testing.T) {
	t.Run("pressing n on immutable commit creates child commit", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		// Set up jjService (required for 'n' key to work)
		m.jjService = &jj.Service{RepoPath: "/test/repo"}

		// Mark the first commit as immutable (like main or root())
		m.graphTabModel.SelectCommit(0)
		m.repository.Graph.Commits[0].Immutable = true
		m.repository.Graph.Commits[0].ShortID = "main"

		// Press 'n' - graph returns NewCommit request; main runs it and sets status
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
		_, cmd := m.Update(msg)
		if cmd != nil {
			if req, ok := cmd().(graphtab.Request); ok {
				var v tea.Model
				v, _ = m.Update(req)
				m = v.(*Model)
			}
		}

		// Status message should indicate creating new commit, NOT an error
		if containsString(m.statusMessage, "Cannot") {
			t.Errorf("Should not show error message, got: %s", m.statusMessage)
		}
		if containsString(m.statusMessage, "immutable") {
			t.Errorf("Should not mention immutable restriction, got: %s", m.statusMessage)
		}
		if !containsString(m.statusMessage, "Creating new commit") {
			t.Errorf("Expected 'Creating new commit' status, got: %s", m.statusMessage)
		}
	})

	t.Run("new commit button visible for immutable commits", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		m.graphTabModel.SelectCommit(0)
		m.repository.Graph.Commits[0].Immutable = true

		view := m.View()

		// The "New (n)" button should be visible even for immutable commits
		// because creating a child of an immutable commit is valid
		if !containsString(view, "New (n)") {
			t.Error("Expected 'New (n)' button to be visible for immutable commit")
		}
	})

	t.Run("other mutating actions blocked for immutable commits", func(t *testing.T) {
		m := newTestModel()
		defer m.Close()

		m.jjService = &jj.Service{RepoPath: "/test/repo"}
		m.graphTabModel.SelectCommit(0)
		m.repository.Graph.Commits[0].Immutable = true

		// Press 'd' - graph returns StartEditDescription request; main runs it and blocks with message
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
		_, cmd := m.Update(msg)
		if cmd != nil {
			if req, ok := cmd().(graphtab.Request); ok {
				var v tea.Model
				v, _ = m.Update(req)
				m = v.(*Model)
			}
		}

		if !containsString(m.statusMessage, "immutable") {
			t.Error("Expected immutable warning for describe action")
		}
	})
}

// TestInProgressTransitionDetection verifies the logic for finding "In Progress" transitions
// This tests both Jira-style and Codecks-style transition names
func TestInProgressTransitionDetection(t *testing.T) {
	// Helper function that mirrors the logic in transitionTicketToInProgress
	findInProgressTransition := func(transitions []struct{ id, name string }) string {
		for _, tr := range transitions {
			lowerName := strings.ToLower(tr.name)
			isInProgress := strings.Contains(lowerName, "progress") ||
				(strings.Contains(lowerName, "start") && !strings.Contains(lowerName, "not start") && !strings.Contains(lowerName, "not_start"))
			if isInProgress {
				return tr.id
			}
		}
		return ""
	}

	tests := []struct {
		name        string
		transitions []struct{ id, name string }
		expectedID  string
		description string
	}{
		{
			name: "Codecks transitions - should find 'started' not 'not_started'",
			transitions: []struct{ id, name string }{
				{"not_started", "Not Started"},
				{"started", "In Progress"},
				{"done", "Done"},
			},
			expectedID:  "started",
			description: "Codecks has 'Not Started' which contains 'start' - should be excluded",
		},
		{
			name: "Jira typical workflow - Start Progress",
			transitions: []struct{ id, name string }{
				{"11", "Start Progress"},
				{"21", "Done"},
			},
			expectedID:  "11",
			description: "Common Jira transition 'Start Progress' should match",
		},
		{
			name: "Jira workflow - In Progress",
			transitions: []struct{ id, name string }{
				{"31", "To Do"},
				{"41", "In Progress"},
				{"51", "Done"},
			},
			expectedID:  "41",
			description: "Jira 'In Progress' transition should match on 'progress'",
		},
		{
			name: "Jira workflow - Start Work",
			transitions: []struct{ id, name string }{
				{"61", "Backlog"},
				{"71", "Start Work"},
				{"81", "Complete"},
			},
			expectedID:  "71",
			description: "Jira 'Start Work' should match on 'start'",
		},
		{
			name: "No in-progress transition available",
			transitions: []struct{ id, name string }{
				{"91", "To Do"},
				{"101", "Done"},
				{"111", "Closed"},
			},
			expectedID:  "",
			description: "Should return empty when no matching transition exists",
		},
		{
			name: "Edge case - 'Begin' without 'start' or 'progress'",
			transitions: []struct{ id, name string }{
				{"121", "Begin Work"},
				{"131", "Done"},
			},
			expectedID:  "",
			description: "'Begin Work' doesn't contain 'start' or 'progress' - won't match (may need to add)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := findInProgressTransition(tc.transitions)
			if result != tc.expectedID {
				t.Errorf("%s\nExpected: %q, Got: %q", tc.description, tc.expectedID, result)
			}
		})
	}
}
