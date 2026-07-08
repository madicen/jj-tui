package model

// Model-level dispatch, modal-flow, and effect-dispatcher tests. These lock the
// root-model orchestration the tab-interface migration must preserve:
//   - tab-switch key routing,
//   - modal open/close gating per modal kind,
//   - NavigateMsg handling for the common kinds,
//   - the effect dispatcher outcomes (error routing, resolve-open-PRs,
//     reload-repository, load-branches, set-bookmark-conflict-sources),
//   - additional chromedSlot z-order combinations.
//
// PLAN(P2.3): keep these green across the tab-interface migration.

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/integrations/github"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/state"
)

func keyMsg(s string) tea.KeyMsg {
	if s == "esc" {
		return tea.KeyMsg{Type: tea.KeyEsc}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// TestTabSwitchKeyRouting locks the top-level tab-switch keybindings.
func TestTabSwitchKeyRouting(t *testing.T) {
	cases := []struct {
		start state.ViewMode
		key   string
		want  state.ViewMode
	}{
		{state.ViewCommitGraph, "p", state.ViewPullRequests},
		{state.ViewCommitGraph, "t", state.ViewTickets},
		{state.ViewCommitGraph, "b", state.ViewBranches},
		{state.ViewCommitGraph, ",", state.ViewSettings},
		{state.ViewCommitGraph, "h", state.ViewHelp},
		{state.ViewHelp, "g", state.ViewCommitGraph},
		{state.ViewPullRequests, "esc", state.ViewCommitGraph},
		{state.ViewHelp, "esc", state.ViewCommitGraph},
	}
	for _, tc := range cases {
		name := fmt.Sprintf("%v_%s", tc.start, tc.key)
		t.Run(name, func(t *testing.T) {
			m := newGoldenModel(t)
			defer m.Close()
			m.SetViewMode(tc.start)
			newModel, _ := m.Update(keyMsg(tc.key))
			m = newModel.(*Model)
			if m.GetViewMode() != tc.want {
				t.Fatalf("key %q from %v: got %v, want %v", tc.key, tc.start, m.GetViewMode(), tc.want)
			}
		})
	}
}

// TestModalOpenCloseGating verifies each modal kind opens (ViewMode + chromedSlot
// key) and that returning to a tab clears the chrome.
func TestModalOpenCloseGating(t *testing.T) {
	cases := []struct {
		vm  state.ViewMode
		key string
	}{
		{state.ViewEditDescription, "descedit"},
		{state.ViewCreatePR, "pr"},
		{state.ViewCreateTicket, "ticket"},
		{state.ViewCreateBookmark, "bookmark"},
		{state.ViewGitHubLogin, "githublogin"},
		{state.ViewBookmarkConflict, "conflict"},
		{state.ViewDivergentCommit, "divergent"},
		{state.ViewEvologSplit, "evolog"},
		{state.ViewFileDiff, "filediff"},
		{state.ViewWorkspaces, "workspaces"},
	}
	for _, tc := range cases {
		t.Run(tc.key+"_open", func(t *testing.T) {
			m := newGoldenModel(t)
			defer m.Close()
			m.appState.ViewMode = tc.vm
			key, _, _, _ := m.chromedSlot()
			if key != tc.key {
				t.Fatalf("ViewMode %v: chromedSlot key = %q, want %q", tc.vm, key, tc.key)
			}
			// Closing back to a plain tab clears the chrome.
			m.appState.ViewMode = state.ViewCommitGraph
			if key, _, _, _ := m.chromedSlot(); key != "" {
				t.Fatalf("after returning to graph, chromedSlot key = %q, want empty", key)
			}
		})
	}
}

// TestNavigateBackToGraphClosesModals: the common "return to graph" navigate
// kinds reset ViewMode to the commit graph.
func TestNavigateBackToGraph(t *testing.T) {
	for _, kind := range []state.NavigateKind{
		state.NavigateBackToGraph,
		state.NavigateDismissError,
	} {
		t.Run(fmt.Sprintf("%v", kind), func(t *testing.T) {
			m := newGoldenModel(t)
			defer m.Close()
			m.appState.ViewMode = state.ViewPullRequests
			newModel, _ := m.Update(state.NavigateTarget{Kind: kind}.Cmd()())
			m = newModel.(*Model)
			if m.GetViewMode() != state.ViewCommitGraph {
				t.Fatalf("%v should return to graph, got %v", kind, m.GetViewMode())
			}
		})
	}
}

// TestCreateBookmarkModalOpenCloseFlow exercises the full open→cancel flow for
// the create-bookmark modal, proving the modal opens and Esc restores the graph.
func TestCreateBookmarkModalOpenCloseFlow(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()
	m.graphTabModel.SelectCommit(1)
	m.startCreateBookmark()
	if m.appState.ViewMode != state.ViewCreateBookmark {
		t.Fatalf("startCreateBookmark should open the bookmark modal, got %v", m.appState.ViewMode)
	}
	if key, _, _, _ := m.chromedSlot(); key != "bookmark" {
		t.Fatalf("bookmark modal should be chromed, got %q", key)
	}
	// Esc cancels; drive the returned cmds to closure.
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(*Model)
	for cmd != nil {
		newModel, cmd = m.Update(cmd())
		m = newModel.(*Model)
	}
	if m.appState.ViewMode != state.ViewCommitGraph {
		t.Fatalf("Esc should close bookmark modal back to graph, got %v", m.appState.ViewMode)
	}
}

// TestEffectShowAndClearError locks the error-routing effects.
func TestEffectShowAndClearError(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()

	boom := fmt.Errorf("boom")
	if cmd := m.applyEffect(effShowError{err: boom}); cmd != nil {
		t.Error("effShowError should not return a command")
	}
	if m.errorModal.GetError() == nil {
		t.Fatal("effShowError should populate the error modal")
	}
	if cmd := m.applyEffect(effClearError{}); cmd != nil {
		t.Error("effClearError should not return a command")
	}
	if m.errorModal.GetError() != nil {
		t.Fatal("effClearError should clear the error modal")
	}
}

// TestEffectReloadCommandsReturnCmds locks that the async reload effects produce
// a runnable command (the migration must keep these wired through applyEffect).
func TestEffectReloadCommandsReturnCmds(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()
	m.appState.JJService = &jj.Service{RepoPath: "/test/repo"}
	m.appState.GitHubService = &github.Service{}

	if cmd := m.applyEffect(effReloadRepository{}); cmd == nil {
		t.Error("effReloadRepository should return a load command")
	}
	if cmd := m.applyEffect(effLoadBranches{}); cmd == nil {
		t.Error("effLoadBranches should return a load command")
	}
	// effResolveOpenPRs returns a command only when some local bookmark still
	// lacks a matched open PR; here we just assert it routes without panicking
	// (nil is valid when every bookmark is already resolved).
	_ = m.applyEffect(effResolveOpenPRs{})
}

// TestEffectSetBookmarkConflictSources locks that the effect mutates the bookmark
// modal without returning a command (and without panicking on nil config paths).
func TestEffectSetBookmarkConflictSources(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()
	m.branchesTabModel.UpdateBranches(goldenBranches())
	if cmd := m.applyEffect(effSetBookmarkConflictSources{}); cmd != nil {
		t.Error("effSetBookmarkConflictSources should not return a command")
	}
}

// TestApplyEffectsBatches verifies applyEffects batches multiple command-producing
// effects into a single runnable command.
func TestApplyEffectsBatches(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()
	m.appState.JJService = &jj.Service{RepoPath: "/test/repo"}

	if cmd := m.applyEffects(); cmd != nil {
		t.Error("applyEffects() with no effects should return nil")
	}
	cmd := m.applyEffects(effLoadBranches{}, effReloadRepository{})
	if cmd == nil {
		t.Fatal("applyEffects with two load effects should return a batched command")
	}
}

// TestChromedSlotZOrderAdditionalCombos extends the committed z-order snapshot
// with combinations it does not cover: error/warning standing alone (no ViewMode
// overlay), the workspaces overlay, and error winning over the workspaces overlay.
func TestChromedSlotZOrderAdditionalCombos(t *testing.T) {
	cases := []struct {
		name    string
		apply   func(m *Model)
		wantKey string
	}{
		{
			name:    "error_alone_on_graph",
			apply:   func(m *Model) { m.appState.ViewMode = state.ViewCommitGraph; m.errorModal.SetError(fmt.Errorf("x"), false, "") },
			wantKey: "error",
		},
		{
			name:    "warning_alone_on_graph",
			apply:   func(m *Model) { m.appState.ViewMode = state.ViewCommitGraph; m.warningModal.Show("t", "m", nil) },
			wantKey: "warning",
		},
		{
			name:    "workspaces_viewmode",
			apply:   func(m *Model) { m.appState.ViewMode = state.ViewWorkspaces },
			wantKey: "workspaces",
		},
		{
			name: "error_over_workspaces",
			apply: func(m *Model) {
				m.appState.ViewMode = state.ViewWorkspaces
				m.errorModal.SetError(fmt.Errorf("x"), false, "")
			},
			wantKey: "error",
		},
		{
			name: "warning_over_workspaces",
			apply: func(m *Model) {
				m.appState.ViewMode = state.ViewWorkspaces
				m.warningModal.Show("t", "m", nil)
			},
			wantKey: "warning",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newGoldenModel(t)
			defer m.Close()
			tc.apply(m)
			key, _, _, _ := m.chromedSlot()
			if key != tc.wantKey {
				t.Fatalf("chromedSlot key = %q, want %q", key, tc.wantKey)
			}
		})
	}
}
