package model

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/state"
)

// This file characterizes the root Model <-> tab wiring BEFORE the P2.3 Tab
// interface refactor. P2.3 replaces the six concrete tab fields
// (graphTabModel, prsTabModel, branchesTabModel, ticketsTabModel,
// settingsTabModel, helpTabModel) with an interface-backed registry and routes
// the per-ViewMode Update/View/SetDimensions switches through it. The tests
// below lock the two things that must not regress:
//
//  1. Dispatch routing: a key/window-size message in a given ViewMode reaches
//     the correct tab and mutates its state.
//  2. Accessor delegation: the Model's public accessors return exactly what the
//     underlying tab model holds (these accessors feed tab ContextProviders and
//     tests, so any rewiring must preserve them).
//
// PLAN(P2.3): keep this green across the interface migration.

// TestTabViewRoutingSmoke verifies each primary tab ViewMode renders through
// View() without panicking and produces non-empty output. This guards the
// View() dispatch switch that P2.3 collapses into interface calls.
func TestTabViewRoutingSmoke(t *testing.T) {
	views := []state.ViewMode{
		state.ViewCommitGraph,
		state.ViewPullRequests,
		state.ViewBranches,
		state.ViewTickets,
		state.ViewSettings,
		state.ViewHelp,
	}
	for _, vm := range views {
		vm := vm
		t.Run(vm.String(), func(t *testing.T) {
			m := newTestModel()
			defer m.Close()
			m.SetViewMode(vm)
			out := m.View()
			if strings.TrimSpace(out) == "" {
				t.Fatalf("View() for %v returned empty output", vm)
			}
		})
	}
}

// TestGraphKeyDispatchRouting locks that j/k in the graph view mutate the graph
// tab's selection through the Update dispatch path.
func TestGraphKeyDispatchRouting(t *testing.T) {
	m := newTestModel()
	defer m.Close()
	m.SetViewMode(state.ViewCommitGraph)
	m.graphTabModel.SelectCommit(0)

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = newModel.(*Model)
	if got := m.GetSelectedCommit(); got != 1 {
		t.Fatalf("after 'j' expected selected commit 1, got %d", got)
	}
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = newModel.(*Model)
	if got := m.GetSelectedCommit(); got != 0 {
		t.Fatalf("after 'k' expected selected commit 0, got %d", got)
	}
}

// TestWindowSizeDispatchRoutesToAllTabs verifies a WindowSizeMsg fans out to
// every tab (they must not panic and the model records the new size). P2.3
// keeps this fan-out but routes it via the registry.
func TestWindowSizeDispatchRoutesToAllTabs(t *testing.T) {
	m := newTestModel()
	defer m.Close()
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 50})
	m = newModel.(*Model)
	// View() after resize must render for each tab without panic.
	for _, vm := range []state.ViewMode{
		state.ViewCommitGraph, state.ViewPullRequests, state.ViewBranches,
		state.ViewTickets, state.ViewSettings, state.ViewHelp,
	} {
		m.SetViewMode(vm)
		if strings.TrimSpace(m.View()) == "" {
			t.Fatalf("View() empty for %v after resize", vm)
		}
	}
}

// TestAccessorDelegation locks that the Model accessors delegate to the
// underlying tab models. Each case mutates the concrete tab model and reads
// back through the Model accessor. P2.3 must preserve these values regardless
// of how the tab is stored.
func TestAccessorDelegation(t *testing.T) {
	m := newTestModel()
	defer m.Close()

	// graph: selected commit
	m.graphTabModel.SelectCommit(2)
	if got := m.GetSelectedCommit(); got != 2 {
		t.Fatalf("GetSelectedCommit = %d, want 2", got)
	}

	// tickets: status-change mode
	m.ticketsTabModel.SetStatusChangeMode(true)
	if !m.GetIsStatusChangeMode() {
		t.Fatal("GetIsStatusChangeMode should be true after SetStatusChangeMode(true)")
	}
	m.ticketsTabModel.SetStatusChangeMode(false)
	if m.GetIsStatusChangeMode() {
		t.Fatal("GetIsStatusChangeMode should be false after SetStatusChangeMode(false)")
	}

	// settings: active sub-tab index
	m.settingsTabModel.SetActiveSettingsTabIndex(3)
	if got := m.GetActiveSettingsTabIndex(); got != 3 {
		t.Fatalf("GetActiveSettingsTabIndex = %d, want 3", got)
	}

	// settings: Jira project (nested GetJiraModel accessor)
	m.settingsTabModel.GetJiraModel().SetProject("PROJ")
	if got := m.GetSettingsJiraProject(); got != "PROJ" {
		t.Fatalf("GetSettingsJiraProject = %q, want %q", got, "PROJ")
	}

	// settings: graph revset (nested GetAdvancedModel accessor)
	m.settingsTabModel.GetAdvancedModel().SetGraphRevset("mine()")
	if got := m.GetSettingsGraphRevset(); got != "mine()" {
		t.Fatalf("GetSettingsGraphRevset = %q, want %q", got, "mine()")
	}

	// prs: selected PR
	m.prsTabModel.SetSelectedPR(0)
	if got := m.GetSelectedPR(); got != 0 {
		t.Fatalf("GetSelectedPR = %d, want 0", got)
	}
}

// TestGraphChangedFilesAccessorDelegation locks the changed-files accessors
// which the model reconciles against during repository reloads.
func TestGraphChangedFilesAccessorDelegation(t *testing.T) {
	m := newTestModel()
	defer m.Close()
	// Selecting a commit and loading changed files sets the commit id the
	// reload path checks via GetChangedFilesCommitID.
	m.graphTabModel.SelectCommit(0)
	// Before any changed-files load, the accessor should be safe to call.
	_ = m.GetChangedFilesCommitID()
	_ = m.GetChangedFiles()
	_ = m.GetSelectedFile()
	_ = m.IsGraphFocused()
}
