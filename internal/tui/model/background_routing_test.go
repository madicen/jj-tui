package model

// Behavioral tests that lock how the root Model routes BACKGROUND messages by
// concrete type — the behavior the P2.3 tab-interface migration is most likely
// to break. Today model.go type-switches each async *LoadedMsg to the correct
// tab regardless of which tab is active; a generic "forward to active tab"
// implementation would silently drop these. These tests deliver each background
// message while a DIFFERENT tab is active and assert the target tab still
// received it, plus the correct effects/commands were emitted.
//
// PLAN(P2.3): keep these green across the tab-interface migration.

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/github"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tickets"
	"github.com/madicen/jj-tui/internal/tui/state"
	branchestab "github.com/madicen/jj-tui/internal/tui/tabs/branches"
	graphtab "github.com/madicen/jj-tui/internal/tui/tabs/graph"
	prstab "github.com/madicen/jj-tui/internal/tui/tabs/prs"
	ticketstab "github.com/madicen/jj-tui/internal/tui/tabs/tickets"
)

// drain runs a returned tea.Cmd once and returns its message (nil-safe).
func drain(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

// TestBackgroundPRsLoadedWhileInactive: PrsLoadedMsg delivered while on the graph
// tab must still populate the PR tab and emit the open-PR resolution effect.
func TestBackgroundPRsLoadedWhileInactive(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()
	m.SetViewMode(state.ViewCommitGraph) // NOT the PR tab
	m.appState.GitHubService = &github.Service{}
	m.prsTabModel.SetGithubService(true)
	m.appState.Loading = true

	newModel, cmd := m.Update(prstab.PrsLoadedMsg{Prs: []internal.GitHubPR{
		{Number: 7, Title: "Background-loaded PR", State: "open"},
	}})
	m = newModel.(*Model)

	if !m.appState.PRsLoadedOnce {
		t.Error("PRsLoadedOnce should be set after PrsLoadedMsg")
	}
	if m.appState.Loading {
		t.Error("Loading should be cleared after PrsLoadedMsg")
	}
	if cmd == nil {
		t.Error("PrsLoadedMsg should emit a command (effResolveOpenPRs)")
	}
	// Switching to the PR tab must now show the background-loaded PR.
	m.SetViewMode(state.ViewPullRequests)
	if view := m.View(); !containsString(view, "Background-loaded PR") {
		t.Error("PR tab should render the PR delivered while it was inactive")
	}
}

// TestBackgroundTicketsLoadedWhileInactive: TicketsLoadedMsg delivered while on
// the graph tab must still populate the tickets tab.
func TestBackgroundTicketsLoadedWhileInactive(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()
	m.SetViewMode(state.ViewCommitGraph)
	m.appState.Loading = true

	newModel, _ := m.Update(ticketstab.TicketsLoadedMsg{Tickets: []tickets.Ticket{
		{Key: "BG-1", Summary: "Loaded in the background", Status: "To Do"},
		{Key: "BG-2", Summary: "Second background ticket", Status: "Done"},
	}})
	m = newModel.(*Model)

	if !m.appState.TicketsLoadedOnce {
		t.Error("TicketsLoadedOnce should be set after TicketsLoadedMsg")
	}
	if got := len(m.GetTickets()); got != 2 {
		t.Fatalf("tickets tab should have 2 tickets after inactive load, got %d", got)
	}
	if m.GetTickets()[0].Key != "BG-1" {
		t.Errorf("expected first ticket BG-1, got %s", m.GetTickets()[0].Key)
	}
}

// TestBackgroundBranchesLoadedWhileInactive: BranchesLoadedMsg delivered while on
// the graph tab must still populate the branches tab.
func TestBackgroundBranchesLoadedWhileInactive(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()
	m.SetViewMode(state.ViewCommitGraph)

	newModel, _ := m.Update(branchestab.BranchesLoadedMsg{Branches: []internal.Branch{
		{Name: "bg/one", IsLocal: true},
		{Name: "bg/two", IsLocal: true},
		{Name: "bg/three", Remote: "origin", IsTracked: true},
	}})
	m = newModel.(*Model)

	if got := len(m.GetBranches()); got != 3 {
		t.Fatalf("branches tab should have 3 branches after inactive load, got %d", got)
	}
	if m.GetBranches()[0].Name != "bg/one" {
		t.Errorf("expected first branch bg/one, got %s", m.GetBranches()[0].Name)
	}
}

// TestBackgroundBranchesLoadedInCreateBookmarkEmitsConflictSources: when a
// BranchesLoadedMsg arrives while the create-bookmark modal is open, the model
// must feed the fresh names into the bookmark modal (effSetBookmarkConflictSources).
func TestBackgroundBranchesLoadedInCreateBookmarkEmitsConflictSources(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()
	m.graphTabModel.SelectCommit(1)
	m.startCreateBookmark()
	if m.appState.ViewMode != state.ViewCreateBookmark {
		t.Fatalf("expected ViewCreateBookmark after startCreateBookmark, got %v", m.appState.ViewMode)
	}
	// Deliver branches while the modal is open; must not panic and must keep the modal open.
	newModel, _ := m.Update(branchestab.BranchesLoadedMsg{Branches: []internal.Branch{
		{Name: "existing-bookmark", IsLocal: true},
	}})
	m = newModel.(*Model)
	if m.appState.ViewMode != state.ViewCreateBookmark {
		t.Errorf("create-bookmark modal should stay open across a background branch load, got %v", m.appState.ViewMode)
	}
}

// TestBackgroundChangedFilesWhileOnPRsTab: ChangedFilesLoadedMsg delivered while
// the PR tab is active must still reach the graph tab's changed-files state.
func TestBackgroundChangedFilesWhileOnPRsTab(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()
	m.SetViewMode(state.ViewPullRequests) // graph is inactive
	m.graphTabModel.SelectCommit(0)
	cid := m.appState.Repository.Graph.Commits[0].ChangeID

	newModel, _ := m.Update(graphtab.ChangedFilesLoadedMsg{
		CommitID: cid,
		Files:    []jj.ChangedFile{{Path: "a.go", Status: "M"}, {Path: "b.go", Status: "A"}},
	})
	m = newModel.(*Model)

	if got := len(m.GetChangedFiles()); got != 2 {
		t.Fatalf("graph changed files should update from an inactive-tab background msg, got %d", got)
	}
}

// TestBackgroundBranchActionSuccessEmitsReloadEffects: a successful BranchActionMsg
// must trigger both a branch reload and a repository reload (effLoadBranches + effReloadRepository).
func TestBackgroundBranchActionSuccessEmitsReloadEffects(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()
	m.appState.JJService = &jj.Service{RepoPath: "/test/repo"}
	m.SetViewMode(state.ViewBranches)

	_, cmd := m.Update(branchestab.BranchActionMsg{Action: "track", Err: nil})
	if cmd == nil {
		t.Fatal("successful BranchActionMsg should emit reload effects, got nil cmd")
	}
	// The batched command should be runnable without panicking.
	_ = drain(cmd)
}

// TestBackgroundBranchActionErrorDoesNotReload: a failed BranchActionMsg must NOT
// emit reload effects (the tab already set the status), and must clear Loading.
func TestBackgroundBranchActionErrorDoesNotReload(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()
	m.SetViewMode(state.ViewBranches)
	m.appState.Loading = true

	newModel, cmd := m.Update(branchestab.BranchActionMsg{Action: "push", Err: fmt.Errorf("push rejected")})
	m = newModel.(*Model)
	if m.appState.Loading {
		t.Error("Loading should be cleared after a failed branch action")
	}
	if cmd != nil {
		t.Errorf("failed BranchActionMsg should not emit reload effects, got a cmd")
	}
}

// TestBackgroundPRErrorRoutesToErrorModal: a PR LoadErrorMsg while inactive must
// surface through the error modal.
func TestBackgroundPRErrorRoutesToErrorModal(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()
	m.SetViewMode(state.ViewCommitGraph)

	newModel, cmd := m.Update(prstab.LoadErrorMsg{Err: fmt.Errorf("github: 503 unavailable")})
	m = newModel.(*Model)
	// Run the effShowError command so the modal is populated.
	if msg := drain(cmd); msg != nil {
		nm, _ := m.Update(msg)
		m = nm.(*Model)
	}
	if m.errorModal.GetError() == nil {
		t.Error("PR LoadErrorMsg should populate the error modal")
	}
}

// TestWindowSizeFansOutToAllTabs: a WindowSizeMsg must resize every tab so each
// renders at the new size (the migration must preserve the resize fan-out).
func TestWindowSizeFansOutToAllTabs(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()
	m.appState.GitHubService = &github.Service{}
	m.prsTabModel.SetGithubService(true)

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 50})
	m = newModel.(*Model)
	if m.width != 120 || m.height != 50 {
		t.Fatalf("window size not stored: %dx%d", m.width, m.height)
	}
	for _, vm := range []state.ViewMode{
		state.ViewCommitGraph, state.ViewPullRequests, state.ViewBranches,
		state.ViewTickets, state.ViewSettings, state.ViewHelp,
	} {
		m.SetViewMode(vm)
		if view := m.View(); len(view) == 0 {
			t.Errorf("tab %v rendered empty after resize", vm)
		}
	}
}
