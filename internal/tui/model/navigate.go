package model

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/state"
)

// navigate.go holds the top-level NavigateMsg dispatcher (P2.7). The historical
// ~440-line handleNavigate switch is split into per-domain handler methods
// (navigate_forms.go, navigate_overlays.go, navigate_repo.go, navigate_ai.go),
// each a switch over its own NavigateKinds that returns handled=false for kinds
// it doesn't own. handleNavigate walks the handlers in turn and returns the
// first that claims the kind. No modal-opening kind bypasses the ModalStack:
// every case still routes through the same ViewMode / sub-model presence state
// that modalStack() (P2.4) derives from — this split only relocates the case
// bodies, it does not introduce a parallel modal-presence path.

// navHandler is a per-domain NavigateKind handler. The bool reports whether the
// handler owns (and has processed) the given kind.
type navHandler func(state.NavigateTarget) (tea.Model, tea.Cmd, bool)

// isRedoResettingNavigate reports the mutating navigations that invalidate a
// pending redo (a new operation supersedes the last-undone one).
func isRedoResettingNavigate(k state.NavigateKind) bool {
	switch k {
	case state.NavigateSaveDescription,
		state.NavigateSubmitBookmark,
		state.NavigateSubmitPR,
		state.NavigateSubmitTicket,
		state.NavigateResolveConflict,
		state.NavigateResolveDivergent,
		state.NavigateRunInit,
		state.NavigatePerformEvologSplit,
		state.NavigateRestoreOperation:
		return true
	default:
		return false
	}
}

// isMutatingJJNavigate reports navigations that create a new jj operation on the
// local repo (and therefore should surface the P5.4 undo hint). PR/ticket API
// calls, repo init, and pushes are deliberately excluded: they either aren't jj
// operations or can't be meaningfully undone with Ctrl+z.
func isMutatingJJNavigate(k state.NavigateKind) bool {
	switch k {
	case state.NavigateSaveDescription,
		state.NavigateSubmitBookmark,
		state.NavigateDeleteBookmark,
		state.NavigateResolveConflict,
		state.NavigateResolveDivergent,
		state.NavigatePerformEvologSplit,
		state.NavigateRestoreOperation:
		return true
	default:
		return false
	}
}

// handleNavigate performs view changes that only main can do (it owns modals and
// cross-tab state). It dispatches to the per-domain handlers below.
func (m *Model) handleNavigate(t state.NavigateTarget) (tea.Model, tea.Cmd) {
	if isRedoResettingNavigate(t.Kind) {
		m.redoOperationID = ""
	}
	if isMutatingJJNavigate(t.Kind) {
		m.pendingUndoHint = true
	}
	for _, h := range []navHandler{
		m.handleNavigateDescription,
		m.handleNavigateBookmark,
		m.handleNavigatePR,
		m.handleNavigateTicket,
		m.handleNavigateWarning,
		m.handleNavigateEvolog,
		m.handleNavigateFileDiff,
		m.handleNavigateConflictDivergent,
		m.handleNavigateWorkspaces,
		m.handleNavigateOperations,
		m.handleNavigateInit,
		m.handleNavigateRemote,
		m.handleNavigateError,
		m.handleNavigateTabSwitch,
		m.handleNavigateAI,
	} {
		if model, cmd, ok := h(t); ok {
			return model, cmd
		}
	}
	return m, nil
}

// handleNavigateTabSwitch covers the plain tab-return navigations.
func (m *Model) handleNavigateTabSwitch(t state.NavigateTarget) (tea.Model, tea.Cmd, bool) {
	switch t.Kind {
	case state.NavigateBackToBranches:
		m.appState.ViewMode = state.ViewBranches
		if t.StatusMessage != "" {
			m.appState.StatusMessage = t.StatusMessage
		}
		return m, nil, true
	case state.NavigateBackToSettings:
		m.appState.ViewMode = state.ViewSettings
		if t.StatusMessage != "" {
			m.appState.StatusMessage = t.StatusMessage
		}
		return m, nil, true
	default:
		return m, nil, false
	}
}

// handleNavigateWarning covers the empty-description (and similar) warning modal.
func (m *Model) handleNavigateWarning(t state.NavigateTarget) (tea.Model, tea.Cmd, bool) {
	switch t.Kind {
	case state.NavigateWarning:
		m.warningModal.Show(t.WarningTitle, t.WarningMessage, t.WarningCommits)
		return m, nil, true
	case state.NavigateWarningCancel:
		if t.StatusMessage != "" {
			m.appState.StatusMessage = t.StatusMessage
		}
		return m, nil, true
	default:
		return m, nil, false
	}
}
