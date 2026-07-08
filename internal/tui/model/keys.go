package model

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/util"
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
	case state.ViewCreateTicket:
		updated, cmd := m.ticketFormModal.Update(msg)
		m.ticketFormModal = updated
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
	case state.ViewWorkspaces:
		updated, cmd := m.workspacesModal.Update(msg)
		m.workspacesModal = updated
		return m, cmd
	case state.ViewOperations:
		updated, cmd := m.operationsModal.Update(msg)
		m.operationsModal = updated
		return m, cmd
	case state.ViewEvologSplit:
		updated, cmd := m.evologSplitModal.Update(msg)
		m.evologSplitModal = updated
		return m, cmd
	case state.ViewFileDiff:
		updated, cmd := m.fileDiffModal.Update(msg)
		m.fileDiffModal = updated
		return m, cmd
	case state.ViewGitHubLogin:
		updated, cmd := m.githubLoginModel.Update(msg)
		m.githubLoginModel = updated
		return m, cmd
	}

	// Global shortcuts (and Esc/Tab when not in a modal).
	switch {
	case key.Matches(msg, m.keys.Quit):
		util.FlushMouse() // sync: stop SGR mouse before quit cmd runs (avoids shell seeing "35;…M")
		return m, tea.Quit
	case key.Matches(msg, m.keys.NavGraph):
		return m.handleNavigateToGraphTab()
	case key.Matches(msg, m.keys.NavPRs):
		return m.handleNavigateToPRTab()
	case key.Matches(msg, m.keys.NavTickets):
		return m.handleNavigateToTicketsTab()
	case key.Matches(msg, m.keys.NavBranches):
		return m.handleNavigateToBranchesTab()
	case key.Matches(msg, m.keys.NavSettings):
		return m.handleNavigateToSettingsTab()
	case key.Matches(msg, m.keys.NavHelp):
		return m.handleNavigateToHelpTab()
	case key.Matches(msg, m.keys.NavWorkspaces):
		return m.handleNavigateToWorkspaces()
	case key.Matches(msg, m.keys.NavOperations):
		return m.handleNavigateToOperations()
	case key.Matches(msg, m.keys.Refresh):
		return m, m.refreshRepository()
	case key.Matches(msg, m.keys.Undo):
		return m.handleUndo()
	case key.Matches(msg, m.keys.Redo):
		return m.handleRedo()
	case key.Matches(msg, m.keys.Back):
		if m.appState.ViewMode == state.ViewTickets && m.ticketsTabModel.IsStatusChangeMode() {
			m.ticketsTabModel.SetStatusChangeMode(false)
			m.appState.StatusMessage = "Ready"
			return m, nil
		}
		if m.appState.ViewMode != state.ViewCommitGraph {
			m.appState.ViewMode = state.ViewCommitGraph
		}
	case msg.String() == "tab":
		if m.appState.ViewMode != state.ViewCommitGraph {
			m.appState.ViewMode = state.ViewCommitGraph
		}
	}
	return m, nil
}
