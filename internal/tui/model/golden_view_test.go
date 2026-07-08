package model

// Golden View() coverage for the primary content tabs, rendered through the root
// Model.View() (the real composite: chrome + tab body + status bar). These lock
// the FULL rendered string per tab/state so the P2.3 tab-interface migration
// fails loudly if it changes any user-visible output.
//
// Determinism: goldenTheme() pins the global lipgloss color profile and theme so
// ANSI output is identical across machines/CI. Fixtures use fixed dates/IDs and
// stable ordering. Settings is golden-locked separately in its own package
// (internal/tui/tabs/settings) because its root-model render depends on
// exec.LookPath("gh"); see that package's golden test.
//
// Update goldens intentionally with:  go test ./internal/tui/model -run TestGoldenView -update
//
// PLAN(P2.3): keep these green across the tab-interface migration.

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/github"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/testutil"
	"github.com/madicen/jj-tui/internal/tickets"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

// fixedDate is a stable timestamp for commit fixtures so any date rendering is deterministic.
var fixedDate = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

// goldenTheme pins global rendering state so View() output is byte-identical
// across machines and TTY/no-TTY environments. Call at the top of every golden test.
func goldenTheme() {
	testutil.ForceDeterministicRendering()
	// Pin the default theme explicitly so a machine-local config or a prior test
	// that mutated the global styles can't shift colors under us.
	styles.SetTheme("#7E00AF", "#50FA7B", "#6272A4")
}

// goldenRepo returns a deterministic, reasonably rich repository fixture:
// a working copy, a bookmarked tip, an immutable trunk, and a conflicted commit.
func goldenRepo() *internal.Repository {
	commits := []internal.Commit{
		{
			ID: "aaaa111122223333", ShortID: "aaaa", ChangeID: "zkztqwxr",
			Author: "Ada Lovelace", Email: "ada@example.com", Date: fixedDate,
			Summary: "Add golden test safety net", Description: "Add golden test safety net",
			IsWorking: true, Branches: []string{"feature/goldens"},
		},
		{
			ID: "bbbb222233334444", ShortID: "bbbb", ChangeID: "nmqrstuv",
			Author: "Grace Hopper", Email: "grace@example.com", Date: fixedDate,
			Summary: "Refactor tab dispatch", Description: "Refactor tab dispatch",
			Parents: []string{"aaaa111122223333"},
		},
		{
			ID: "cccc333344445555", ShortID: "cccc", ChangeID: "wpqxlmno",
			Author: "Alan Turing", Email: "alan@example.com", Date: fixedDate,
			Summary: "Resolve merge conflict", Description: "Resolve merge conflict",
			Parents: []string{"bbbb222233334444"}, Conflicts: true,
		},
		{
			ID: "dddd444455556666", ShortID: "dddd", ChangeID: "hijklabc",
			Author: "Edsger Dijkstra", Email: "edsger@example.com", Date: fixedDate,
			Summary: "Initial commit", Description: "Initial commit",
			Parents: []string{"cccc333344445555"}, Immutable: true, Branches: []string{"main"},
		},
	}
	return &internal.Repository{
		Path:        "/test/repo",
		WorkingCopy: commits[0],
		Graph:       internal.CommitGraph{Commits: commits},
		PRs: []internal.GitHubPR{
			{Number: 42, Title: "Add golden test safety net", State: "open", HeadBranch: "feature/goldens", BaseBranch: "main", CheckStatus: internal.CheckStatusSuccess, ReviewStatus: internal.ReviewStatusApproved},
			{Number: 41, Title: "Refactor tab dispatch", State: "open", HeadBranch: "refactor/tabs", BaseBranch: "main", CheckStatus: internal.CheckStatusPending, ReviewStatus: internal.ReviewStatusPending, IsDraft: true},
			{Number: 40, Title: "Old merged work", State: "merged", HeadBranch: "chore/cleanup", BaseBranch: "main", CheckStatus: internal.CheckStatusFailure, ReviewStatus: internal.ReviewStatusChangesRequested},
		},
	}
}

func goldenBranches() []internal.Branch {
	return []internal.Branch{
		{Name: "main", IsLocal: true, IsCurrent: true, ShortID: "dddd", CommitID: "dddd444455556666"},
		{Name: "feature/goldens", IsLocal: true, ShortID: "aaaa", CommitID: "aaaa111122223333", Ahead: 2},
		{Name: "refactor/tabs", Remote: "origin", IsTracked: true, ShortID: "bbbb", CommitID: "bbbb222233334444"},
		{Name: "hotfix/urgent", IsLocal: true, HasConflict: true, ShortID: "cccc", CommitID: "cccc333344445555"},
	}
}

func goldenTickets() []tickets.Ticket {
	return []tickets.Ticket{
		{Key: "PROJ-101", Summary: "Wire up the tab registry", Status: "In Progress", Type: "Task"},
		{Key: "PROJ-102", Summary: "Document the migration", Status: "To Do", Type: "Story"},
		{Key: "PROJ-103", Summary: "Fix the flaky golden test", Status: "Done", Type: "Bug"},
	}
}

// newGoldenModel builds a deterministic root Model with a loaded repository,
// fixed dimensions, and pinned theme, ready for View() golden capture.
func newGoldenModel(t *testing.T) *Model {
	t.Helper()
	goldenTheme()
	m := New(context.Background())
	m.width = 100
	m.height = 40
	m.appState.Loading = false
	m.SetRepository(goldenRepo())
	m.appState.StatusMessage = "Ready"
	m.graphTabModel.OnRepositoryLoaded(m.appState.Repository)
	m.graphTabModel.SelectCommit(0)
	m.prsTabModel.OnRepositoryLoaded(m.appState.Repository)
	m.branchesTabModel.OnRepositoryLoaded(m.appState.Repository)
	m.branchesTabModel.UpdateBranches(goldenBranches())
	m.ticketsTabModel.SetTicketServiceInfo("Jira", true)
	m.SetTicketList(goldenTickets())
	m.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
	return m
}

func TestGoldenViewGraph(t *testing.T) {
	t.Run("loaded_with_selection", func(t *testing.T) {
		m := newGoldenModel(t)
		defer m.Close()
		m.SetViewMode(state.ViewCommitGraph)
		m.graphTabModel.SelectCommit(1)
		selCID := m.appState.Repository.Graph.Commits[1].ChangeID
		m.graphTabModel.SetChangedFiles([]jj.ChangedFile{
			{Path: "internal/tui/model/model.go", Status: "M"},
			{Path: "internal/tui/tab/registry.go", Status: "A"},
			{Path: "old/removed.go", Status: "D"},
		}, selCID)
		testutil.AssertGolden(t, "graph/loaded_with_selection", m.View())
	})

	t.Run("loaded_working_copy_selected", func(t *testing.T) {
		m := newGoldenModel(t)
		defer m.Close()
		m.SetViewMode(state.ViewCommitGraph)
		m.graphTabModel.SelectCommit(0)
		testutil.AssertGolden(t, "graph/loaded_working_copy_selected", m.View())
	})

	t.Run("immutable_selected", func(t *testing.T) {
		m := newGoldenModel(t)
		defer m.Close()
		m.SetViewMode(state.ViewCommitGraph)
		m.graphTabModel.SelectCommit(3) // immutable trunk
		testutil.AssertGolden(t, "graph/immutable_selected", m.View())
	})

	t.Run("empty_loading", func(t *testing.T) {
		goldenTheme()
		m := New(context.Background())
		m.width = 100
		m.height = 40
		m.appState.Loading = true
		m.appState.StatusMessage = "Loading repository…"
		defer m.Close()
		m.SetViewMode(state.ViewCommitGraph)
		m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
		testutil.AssertGolden(t, "graph/empty_loading", m.View())
	})
}

func TestGoldenViewPRs(t *testing.T) {
	t.Run("loaded_with_github", func(t *testing.T) {
		m := newGoldenModel(t)
		defer m.Close()
		m.appState.GitHubService = &github.Service{}
		m.prsTabModel.SetGithubService(true)
		m.SetViewMode(state.ViewPullRequests)
		m.SetSelectedPR(0)
		testutil.AssertGolden(t, "prs/loaded_with_github", m.View())
	})

	t.Run("no_github", func(t *testing.T) {
		m := newGoldenModel(t)
		defer m.Close()
		m.prsTabModel.SetGithubService(false)
		m.SetViewMode(state.ViewPullRequests)
		testutil.AssertGolden(t, "prs/no_github", m.View())
	})
}

func TestGoldenViewBranches(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()
	m.SetViewMode(state.ViewBranches)
	m.branchesTabModel.SetSelectedBranch(1)
	testutil.AssertGolden(t, "branches/loaded_with_selection", m.View())
}

func TestGoldenViewTickets(t *testing.T) {
	t.Run("loaded_with_service", func(t *testing.T) {
		m := newGoldenModel(t)
		defer m.Close()
		m.SetViewMode(state.ViewTickets)
		m.SetSelectedTicket(0)
		testutil.AssertGolden(t, "tickets/loaded_with_service", m.View())
	})

	t.Run("no_service", func(t *testing.T) {
		m := newGoldenModel(t)
		defer m.Close()
		m.ticketsTabModel.SetTicketServiceInfo("", false)
		m.SetTicketList(nil)
		m.SetViewMode(state.ViewTickets)
		testutil.AssertGolden(t, "tickets/no_service", m.View())
	})
}

func TestGoldenViewHelp(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()
	m.SetViewMode(state.ViewHelp)
	m.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
	testutil.AssertGolden(t, "help/shortcuts", m.View())
}
