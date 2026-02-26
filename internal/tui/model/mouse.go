package model

import (
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/state"
	graphtab "github.com/madicen/jj-tui/internal/tui/tabs/graph"
)

// handleZoneClick handles clicks detected by bubblezone. Main forwards zone events to submodels or handles global zones.
func (m *Model) handleZoneClick(msg zone.MsgZoneInBounds) (tea.Model, tea.Cmd) {
	if msg.Zone == nil {
		return m, nil
	}
	userClicked := m.createIsZoneClickedFuncWithEvent(msg.Event)

	// ——— Blocking modals: forward raw zone msg to modal (modal resolves and handles) ———
	if m.initRepoModel.Path() != "" {
		updated, cmd := m.initRepoModel.Update(msg)
		m.initRepoModel = updated
		return m, cmd
	}
	if m.errorModal.GetError() != nil {
		updated, cmd := m.errorModal.Update(msg)
		m.errorModal = updated
		return m, cmd
	}
	if m.warningModal.IsShown() {
		updated, cmd := m.warningModal.Update(msg)
		m.warningModal = updated
		return m, cmd
	}

	// ——— Global zones (tab nav, status bar actions) ———
	if userClicked(mouse.ZoneTabGraph) {
		return m.handleNavigateToGraphTab()
	}
	if userClicked(mouse.ZoneTabPRs) {
		return m.handleNavigateToPRTab()
	}
	if userClicked(mouse.ZoneTabJira) {
		return m.handleNavigateToTicketsTab()
	}
	if userClicked(mouse.ZoneTabBranches) {
		return m.handleNavigateToBranchesTab()
	}
	if userClicked(mouse.ZoneTabSettings) {
		return m.handleNavigateToSettingsTab()
	}
	if userClicked(mouse.ZoneTabHelp) {
		return m.handleNavigateToHelpTab()
	}
	if userClicked(mouse.ZoneActionQuit) {
		return m, tea.Quit
	}
	if userClicked(mouse.ZoneActionRefresh) {
		return m, m.refreshRepository()
	}
	if userClicked(mouse.ZoneActionNewCommit) {
		return m.processGraphRequest(graphtab.Request{NewCommit: true})
	}
	if userClicked(mouse.ZoneActionUndo) {
		return m.handleUndo()
	}
	if userClicked(mouse.ZoneActionRedo) {
		return m.handleRedo()
	}

	// ——— Forward zone to active view's submodel (by viewMode) ———
	switch m.appState.ViewMode {
	case state.ViewCommitGraph:
		updated, cmd := m.graphTabModel.Update(msg)
		if p, ok := updated.(*graphtab.GraphModel); ok {
			m.graphTabModel = *p
		}
		return m, cmd
	case state.ViewEditDescription:
		updated, cmd := m.desceditModal.Update(msg)
		m.desceditModal = updated
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
	case state.ViewCreatePR:
		updated, cmd := m.prFormModal.Update(msg)
		m.prFormModal = updated
		return m, cmd
	case state.ViewPullRequests:
		updated, cmd := m.prsTabModel.Update(msg)
		m.prsTabModel = updated
		return m, cmd
	case state.ViewTickets:
		updated, cmd := m.ticketsTabModel.Update(msg)
		m.ticketsTabModel = updated
		return m, cmd
	case state.ViewBranches:
		updated, cmd := m.branchesTabModel.Update(msg)
		m.branchesTabModel = updated
		return m, cmd
	case state.ViewHelp:
		updated, cmd := m.helpTabModel.Update(msg)
		m.helpTabModel = updated
		return m, cmd
	case state.ViewSettings:
		updated, cmd := m.settingsTabModel.Update(msg)
		m.settingsTabModel = updated
		return m, cmd
	case state.ViewGitHubLogin:
		updated, cmd := m.githubLoginModel.Update(msg)
		m.githubLoginModel = updated
		return m, cmd
	}

	return m, nil
}

// handleAction handles action messages (e.g. from external triggers). Tab navigation and graph actions are handled in keys/mouse.
func (m *Model) handleAction(action ActionType) (tea.Model, tea.Cmd) {
	switch action {
	case ActionQuit:
		return m, tea.Quit
	case ActionRefresh:
		return m, m.refreshRepository()
	case ActionNewPR:
		m.appState.ViewMode = state.ViewCreatePR
		return m, nil
	}
	return m, nil
}
