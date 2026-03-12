package model

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/tui/data"
	"github.com/madicen/jj-tui/internal/tui/state"
	graphtab "github.com/madicen/jj-tui/internal/tui/tabs/graph"
	prstab "github.com/madicen/jj-tui/internal/tui/tabs/prs"
	settingstab "github.com/madicen/jj-tui/internal/tui/tabs/settings"
)

// handleDataServicesInitializedMsg applies initialized services and repository; starts tick and PR load.
// Kept for tests or code paths that still send the full message.
func (m *Model) handleDataServicesInitializedMsg(msg data.ServicesInitializedMsg) (tea.Model, tea.Cmd) {
	m.appState.JJService = msg.JJService
	m.appState.GitHubService = msg.GitHubService
	m.appState.TicketService = msg.TicketService
	m.appState.Repository = msg.Repository
	m.appState.GithubInfo = msg.GitHubInfo
	m.appState.DemoMode = msg.DemoMode
	m.appState.Loading = false
	m.appState.StatusMessage = fmt.Sprintf("Loaded %d commits", len(msg.Repository.Graph.Commits))
	if m.appState.DemoMode {
		m.appState.StatusMessage += " (demo mode)"
	} else if m.appState.GitHubService != nil {
		m.appState.StatusMessage += " (GitHub connected)"
	} else if msg.GitHubInfo != "" {
		m.appState.StatusMessage += fmt.Sprintf(" (GitHub: %s)", msg.GitHubInfo)
	}
	if m.appState.TicketService != nil {
		m.appState.StatusMessage += fmt.Sprintf(" (%s connected)", m.appState.TicketService.GetProviderName())
	} else if msg.TicketError != nil {
		m.appState.StatusMessage += fmt.Sprintf(" (Tickets error: %v)", msg.TicketError)
	}
	var cmds []tea.Cmd
	cmds = append(cmds, m.tickCmd())
	if m.isGitHubAvailable() {
		cmds = append(cmds, prstab.LoadPRsCmd(m.appState.GitHubService, m.appState.GithubInfo, m.appState.DemoMode, 0))
		cmds = append(cmds, prstab.PrTickCmd())
	}
	if m.graphTabModel.GetSelectedCommit() < 0 && len(msg.Repository.Graph.Commits) > 0 {
		m.graphTabModel.SelectCommit(0)
		commit := msg.Repository.Graph.Commits[0]
		cmds = append(cmds, graphtab.LoadChangedFilesCmd(m.appState.JJService, commit.ChangeID))
	}
	return m, tea.Batch(cmds...)
}

// handleRepoReadyMsg shows the graph immediately and kicks off GitHub/ticket load in the background.
// Changed files are loaded on the next frame (via loadChangedFilesTriggerMsg) so the graph paints first.
func (m *Model) handleRepoReadyMsg(msg data.RepoReadyMsg) (tea.Model, tea.Cmd) {
	m.appState.JJService = msg.JJService
	m.appState.Repository = msg.Repository
	m.appState.DemoMode = msg.DemoMode
	m.appState.Loading = false
	m.appState.StatusMessage = fmt.Sprintf("Loaded %d commits", len(msg.Repository.Graph.Commits))
	if m.appState.Repository != nil {
		m.appState.Repository.PRs = nil
	}
	m.graphTabModel.UpdateRepository(m.appState.Repository)
	m.prsTabModel.UpdateRepository(m.appState.Repository)
	m.prsTabModel.SetGithubService(false)
	m.branchesTabModel.UpdateRepository(m.appState.Repository)
	m.ticketsTabModel.UpdateRepository(m.appState.Repository)
	m.settingsTabModel.UpdateRepository(m.appState.Repository)
	m.helpTabModel.UpdateRepository(m.appState.Repository)
	var cmds []tea.Cmd
	cmds = append(cmds, m.tickCmd())
	if m.graphTabModel.GetSelectedCommit() < 0 && len(msg.Repository.Graph.Commits) > 0 {
		m.graphTabModel.SelectCommit(0)
	}
	// Load changed files on next frame so the graph is painted first; then we run jj diff --summary for the selected commit.
	cmds = append(cmds, tea.Tick(0, func(time.Time) tea.Msg { return loadChangedFilesTriggerMsg{} }))
	cmds = append(cmds, data.LoadAuxServicesCmd(msg.DemoMode, msg.Owner, msg.RepoName, msg.GitHubInfoFromURL))
	return m, tea.Batch(cmds...)
}

// handleAuxServicesReadyMsg applies GitHub and ticket services after they load in the background.
func (m *Model) handleAuxServicesReadyMsg(msg data.AuxServicesReadyMsg) (tea.Model, tea.Cmd) {
	m.appState.GitHubService = msg.GitHubService
	m.appState.TicketService = msg.TicketService
	m.appState.GithubInfo = msg.GitHubInfo
	// Append GitHub/ticket info to existing "Loaded N commits" status
	if m.appState.DemoMode {
		m.appState.StatusMessage += " (demo mode)"
	} else if m.appState.GitHubService != nil {
		m.appState.StatusMessage += " (GitHub connected)"
	} else if msg.GitHubInfo != "" {
		m.appState.StatusMessage += fmt.Sprintf(" (GitHub: %s)", msg.GitHubInfo)
	}
	if m.appState.TicketService != nil {
		m.appState.StatusMessage += fmt.Sprintf(" (%s connected)", m.appState.TicketService.GetProviderName())
	} else if msg.TicketError != nil {
		m.appState.StatusMessage += fmt.Sprintf(" (Tickets error: %v)", msg.TicketError)
	}
	var cmds []tea.Cmd
	cmds = append(cmds, m.tickCmd())
	if m.isGitHubAvailable() {
		cmds = append(cmds, prstab.LoadPRsCmd(m.appState.GitHubService, m.appState.GithubInfo, m.appState.DemoMode, 0))
		cmds = append(cmds, prstab.PrTickCmd())
	}
	m.prsTabModel.SetGithubService(m.isGitHubAvailable())
	return m, tea.Batch(cmds...)
}

// handleDataRepositoryLoadedMsg delegates to shared applyRepositoryLoaded.
func (m *Model) handleDataRepositoryLoadedMsg(msg data.RepositoryLoadedMsg) (tea.Model, tea.Cmd) {
	return m.applyRepositoryLoaded(msg.Repository)
}

// handleActionsRepositoryLoadedMsg delegates to shared applyRepositoryLoaded.
func (m *Model) handleActionsRepositoryLoadedMsg(msg graphtab.RepositoryLoadedMsg) (tea.Model, tea.Cmd) {
	return m.applyRepositoryLoaded(msg.Repository)
}

// handleDataSilentRepositoryLoadedMsg applies silent repo update and propagates to all tabs.
func (m *Model) handleDataSilentRepositoryLoadedMsg(msg data.SilentRepositoryLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Repository != nil {
		oldCount := 0
		var oldPRs []internal.GitHubPR
		if m.appState.Repository != nil {
			oldCount = len(m.appState.Repository.Graph.Commits)
			oldPRs = m.appState.Repository.PRs
		}
		m.appState.Repository = msg.Repository
		m.appState.Repository.PRs = oldPRs
		m.graphTabModel.UpdateRepository(m.appState.Repository)
		m.prsTabModel.UpdateRepository(m.appState.Repository)
		m.prsTabModel.SetGithubService(m.isGitHubAvailable())
		m.branchesTabModel.UpdateRepository(m.appState.Repository)
		m.ticketsTabModel.UpdateRepository(m.appState.Repository)
		m.settingsTabModel.UpdateRepository(m.appState.Repository)
		m.helpTabModel.UpdateRepository(m.appState.Repository)
		newCount := len(msg.Repository.Graph.Commits)
		if newCount != oldCount && m.errorModal.GetError() == nil {
			m.appState.StatusMessage = fmt.Sprintf("Updated: %d commits", newCount)
		}
	}
	return m, nil
}

// handleTickMsg runs auto-refresh and ensures changed files for selected commit; forwards PR tick to PRs tab.
func (m *Model) handleTickMsg() (tea.Model, tea.Cmd) {
	// Don't run background refresh/updates if a modal is showing or we're in a blocking flow
	isBlockingView := m.appState.ViewMode == state.ViewEditDescription ||
		m.appState.ViewMode == state.ViewCreatePR ||
		m.appState.ViewMode == state.ViewCreateBookmark ||
		m.appState.ViewMode == state.ViewGitHubLogin ||
		m.graphTabModel.IsInRebaseMode()

	if m.errorModal.GetError() != nil || isBlockingView {
		return m, m.tickCmd()
	}
	var cmds []tea.Cmd
	if m.appState.ViewMode == state.ViewCommitGraph && m.appState.Repository != nil && m.appState.JJService != nil {
		commits := m.appState.Repository.Graph.Commits
		idx := m.graphTabModel.GetSelectedCommit()
		if idx >= 0 && idx < len(commits) {
			wantCommitID := commits[idx].ChangeID
			if m.graphTabModel.GetChangedFilesCommitID() != wantCommitID {
				cmds = append(cmds, graphtab.LoadChangedFilesCmd(m.appState.JJService, wantCommitID))
			}
		}
	}
	if !m.appState.Loading && m.appState.JJService != nil && m.appState.ViewMode != state.ViewEditDescription && m.appState.ViewMode != state.ViewCreatePR && m.appState.ViewMode != state.ViewCreateTicket && m.appState.ViewMode != state.ViewCreateBookmark && !m.graphTabModel.IsInRebaseMode() {
		revset := ""
		if m.appState.Config != nil {
			revset = m.appState.Config.GraphRevset
		}
		cmds = append(cmds, data.LoadRepositorySilent(m.appState.JJService, revset))
	}
	prInput := prstab.PrTickInput{
		IsPRView:      m.appState.ViewMode == state.ViewPullRequests,
		Loading:       m.appState.Loading,
		HasError:      m.errorModal.GetError() != nil,
		GitHubService: m.appState.GitHubService,
		GithubInfo:    m.appState.GithubInfo,
		DemoMode:      m.appState.DemoMode,
		ExistingCount: 0,
	}
	if m.appState.Repository != nil {
		prInput.ExistingCount = len(m.appState.Repository.PRs)
	}
	_, prCmd := m.prsTabModel.Update(prInput)
	if prCmd != nil {
		cmds = append(cmds, prCmd)
	}
	cmds = append(cmds, m.tickCmd())
	return m, tea.Batch(cmds...)
}

// handleReauthNeededEffect applies PR tab's reauth request (clear GitHub, start login).
func (m *Model) handleReauthNeededEffect(e prstab.ApplyReauthNeededEffect) (tea.Model, tea.Cmd) {
	m.appState.StatusMessage = e.Reason
	cfg, _ := config.Load()
	if cfg != nil {
		cfg.ClearGitHub()
		_ = cfg.Save()
	}
	_ = os.Unsetenv("GITHUB_TOKEN")
	m.appState.GitHubService = nil
	return m, settingstab.StartGitHubLoginCmd()
}
