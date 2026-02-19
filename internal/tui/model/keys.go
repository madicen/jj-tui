package model

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/state"
)

// handleKeyMsg handles keyboard input. Overlay models (init repo, error, warning) get keys first
// and return request cmds; main's Update handles those messages. Then view-specific modals
// get keys, then global shortcuts.
func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Overlay: init-repo screen (not a jj repo). Returns request cmds for main to handle.
	if m.initRepoModel.Path() != "" {
		updated, cmd := m.initRepoModel.Update(msg)
		m.initRepoModel = updated
		return m, cmd
	}

	// Overlay: error modal. Returns request cmds (RequestDismissMsg, RequestRefreshMsg, RequestCopyMsg) for main to handle.
	if m.errorModal.GetError() != nil {
		updated, cmd := m.errorModal.Update(msg)
		m.errorModal = updated
		return m, cmd
	}

	// Overlay: warning modal. Returns PerformCancelCmd, EditCommitRequestedCmd, or tea.Quit for main to handle.
	if m.warningModal.IsShown() {
		updated, cmd := m.warningModal.Update(msg)
		m.warningModal = updated
		return m, cmd
	}

	// View-specific modals: forward to the active view's submodel.
	switch m.appState.ViewMode {
	case state.ViewEditDescription:
		updated, cmd := m.desceditModal.Update(msg)
		m.desceditModal = updated
		return m, cmd
	case state.ViewSettings:
		updated, cmd := m.settingsTabModel.Update(msg)
		m.settingsTabModel = updated
		return m, cmd
	case state.ViewCreatePR:
		updated, cmd := m.prFormModal.Update(msg)
		m.prFormModal = updated
		return m, cmd
	case state.ViewCreateBookmark:
		updated, cmd := m.bookmarkModal.Update(msg)
		m.bookmarkModal = updated
		m.bookmarkModal.UpdateNameExistsFromInput(m.appState.Config != nil && m.appState.Config.ShouldSanitizeBookmarkNames())
		return m, cmd
	case state.ViewBookmarkConflict:
		updated, cmd := m.conflictModal.Update(msg)
		m.conflictModal = updated
		return m, cmd
	case state.ViewDivergentCommit:
		updated, cmd := m.divergentModal.Update(msg)
		m.divergentModal = updated
		return m, cmd
	case state.ViewGitHubLogin:
		updated, cmd := m.githubLoginModel.Update(msg)
		m.githubLoginModel = updated
		return m, cmd
	}

	// Global shortcuts (and Esc/Tab when not in a modal).
	switch msg.String() {
	case "ctrl+q", "ctrl+c":
		return m, tea.Quit
	case "g":
		return m.handleNavigateToGraphTab()
	case "p":
		return m.handleNavigateToPRTab()
	case "t":
		return m.handleNavigateToTicketsTab()
	case "b":
		return m.handleNavigateToBranchesTab()
	case ",":
		return m.handleNavigateToSettingsTab()
	case "h", "?":
		return m.handleNavigateToHelpTab()
	case "ctrl+r":
		return m, m.refreshRepository()
	case "ctrl+z":
		return m.handleUndo()
	case "ctrl+y":
		return m.handleRedo()
	case "esc":
		if m.appState.ViewMode == state.ViewTickets && m.ticketsTabModel.IsStatusChangeMode() {
			m.ticketsTabModel.SetStatusChangeMode(false)
			m.appState.StatusMessage = "Ready"
			return m, nil
		}
		if m.appState.ViewMode != state.ViewCommitGraph {
			m.appState.ViewMode = state.ViewCommitGraph
		}
	case "tab":
		if m.appState.ViewMode != state.ViewCommitGraph {
			m.appState.ViewMode = state.ViewCommitGraph
		}
	}
	return m, nil
}
