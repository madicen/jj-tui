package model

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/state"
	conflicttab "github.com/madicen/jj-tui/internal/tui/tabs/conflict"
	divergenttab "github.com/madicen/jj-tui/internal/tui/tabs/divergent"
	evologsplittab "github.com/madicen/jj-tui/internal/tui/tabs/evologsplit"
	filedifftab "github.com/madicen/jj-tui/internal/tui/tabs/filediff"
	graphtab "github.com/madicen/jj-tui/internal/tui/tabs/graph"
	workspacestab "github.com/madicen/jj-tui/internal/tui/tabs/workspaces"
)

// navigate_overlays.go holds the per-domain NavigateKind handlers for the
// graph-anchored overlay modals: evolog split, file diff, bookmark
// conflict / divergent resolution, and workspaces.

// handleNavigateEvolog covers opening and performing an evolog-driven split.
func (m *Model) handleNavigateEvolog(t state.NavigateTarget) (tea.Model, tea.Cmd, bool) {
	switch t.Kind {
	case state.NavigateOpenEvologSplit:
		m.evologPostSplitDescribe = false
		m.evologStepwiseRemainderAfterSplit = nil
		m.evologStepwiseBookmarkName = ""
		m.evologDescribePreviewActive = false
		m.evologDescribePreviewFromPlan = false
		m.evologDescribeSkipParent = false
		m.evologDescribeParent = ""
		m.evologDescribeChild = ""
		m.evologPrecomputedDescribeParent = ""
		m.evologPrecomputedDescribeChild = ""
		bn := graphtab.FeatureBookmarkForSplit(t.Commit.Branches)
		m.evologSplitModal = m.evologSplitModal.SetDimensions(m.width, m.height).WithSuggestConfig(m.appState.Config)
		descDef := m.appState.Config != nil && m.appState.Config.DefaultEvologPostSplitDescribe()
		m.evologSplitModal.Show(t.Commit, bn, descDef)
		m.appState.ViewMode = state.ViewEvologSplit
		m.appState.StatusMessage = "Loading jj evolog…"
		return m, evologsplittab.LoadEvologCmd(m.appState.JJService, bn, t.Commit), true
	case state.NavigatePerformEvologSplit:
		m.evologSplitModal.ResetOutcomePreviewForPerformSplit()
		m.evologPostSplitDescribe = t.EvologDescribeAfterSplit
		m.evologPrecomputedDescribeParent = strings.TrimSpace(t.EvologPrecomputedDescribeParent)
		m.evologPrecomputedDescribeChild = strings.TrimSpace(t.EvologPrecomputedDescribeChild)
		m.evologStepwiseRemainderAfterSplit = append([]string(nil), t.EvologStepwiseRemainder...)
		m.evologStepwiseBookmarkName = t.EvologBookmarkName
		m.appState.StatusMessage = "Splitting change…"
		m.appState.Loading = true
		return m, tea.Batch(
			evologsplittab.PerformEvologSplitCmd(
				m.appState.JJService,
				t.EvologBookmarkName,
				t.EvologTipChangeID,
				t.EvologTipCommitHint,
				t.EvologBaseCommitID,
				t.EvologMultiBaseCommitIDs,
				t.EvologFilesetsFirst,
				t.EvologHunkPeelRounds,
			),
			m.startBusySpinnerCmd(),
		), true
	default:
		return m, nil, false
	}
}

// handleNavigateFileDiff covers opening and closing the full-file diff overlay.
func (m *Model) handleNavigateFileDiff(t state.NavigateTarget) (tea.Model, tea.Cmd, bool) {
	switch t.Kind {
	case state.NavigateCloseFileDiff:
		m.fileDiffModal.Hide()
		if isStaleFileDiffGlobalStatus(m.appState.StatusMessage) {
			m.appState.StatusMessage = ""
		}
		if m.evologSplitModal.IsShown() {
			m.appState.ViewMode = state.ViewEvologSplit
		} else {
			m.restoreModalUnderlayOrGraph()
		}
		return m, nil, true
	case state.NavigateOpenFileDiff:
		if raw := strings.TrimSpace(t.FileDiffRawGit); raw != "" {
			m.fileDiffModal = m.fileDiffModal.SetDimensions(m.width, m.height)
			m.fileDiffModal = m.fileDiffModal.ShowPreloadedStyledDiff(
				strings.TrimSpace(t.FileDiffOverlayTitle),
				strings.TrimSpace(t.FileDiffOverlaySubtitle),
				raw,
			)
			m.appState.ViewMode = state.ViewFileDiff
			m.appState.StatusMessage = ""
			return m, nil, true
		}
		path := strings.TrimSpace(t.FileDiffPath)
		if path == "" || m.appState.JJService == nil {
			m.appState.StatusMessage = "Cannot open file diff"
			return m, nil, true
		}
		m.fileDiffModal = m.fileDiffModal.SetDimensions(m.width, m.height)
		seq := m.fileDiffModal.BeginLoad(t.Commit, path)
		m.appState.ViewMode = state.ViewFileDiff
		m.appState.StatusMessage = "Loading file diff…"
		return m, filedifftab.LoadFileDiffCmd(m.appState.JJService, seq, t.Commit.ChangeID, path), true
	default:
		return m, nil, false
	}
}

// handleNavigateConflictDivergent covers bookmark-conflict close/resolve and
// divergent-commit resolve.
func (m *Model) handleNavigateConflictDivergent(t state.NavigateTarget) (tea.Model, tea.Cmd, bool) {
	switch t.Kind {
	case state.NavigateCloseBookmarkConflict:
		m.conflictModal.Hide()
		if m.bookmarkConflictReturnValid {
			m.appState.ViewMode = m.bookmarkConflictReturnView
		} else {
			m.appState.ViewMode = state.ViewBranches
		}
		m.bookmarkConflictReturnValid = false
		if t.StatusMessage != "" {
			m.appState.StatusMessage = t.StatusMessage
		}
		return m, nil, true
	case state.NavigateResolveConflict:
		m.appState.StatusMessage = "Resolving bookmark conflict..."
		return m, conflicttab.ResolveBookmarkConflictCmd(m.appState.JJService, t.ConflictBookmarkName, t.ConflictResolution), true
	case state.NavigateResolveDivergent:
		m.appState.StatusMessage = "Resolving divergent commit..."
		return m, divergenttab.ResolveDivergentCommitCmd(m.appState.JJService, t.DivergentChangeID, t.DivergentKeepCommitID), true
	default:
		return m, nil, false
	}
}

// handleNavigateWorkspaces covers the workspaces overlay add/forget/close.
func (m *Model) handleNavigateWorkspaces(t state.NavigateTarget) (tea.Model, tea.Cmd, bool) {
	switch t.Kind {
	case state.NavigateAddWorkspace:
		m.appState.Loading = true
		m.appState.StatusMessage = "Adding workspace…"
		return m, workspacestab.AddWorkspaceCmd(m.appState.JJService, t.WorkspacePath), true
	case state.NavigateForgetWorkspace:
		m.appState.Loading = true
		m.appState.StatusMessage = "Forgetting workspace…"
		return m, workspacestab.ForgetWorkspaceCmd(m.appState.JJService, t.WorkspaceName), true
	case state.NavigateCloseWorkspaces:
		m.workspacesModal.Hide()
		m.appState.ViewMode = state.ViewCommitGraph
		if t.StatusMessage != "" {
			m.appState.StatusMessage = t.StatusMessage
		}
		return m, nil, true
	default:
		return m, nil, false
	}
}
