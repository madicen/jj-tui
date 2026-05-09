package model

import (
	"fmt"
	"os"
	"strings"
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
	m.silentReloadInFlight = false
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
		cmds = append(cmds, m.wrapFirstPRLoadCmd(prstab.LoadPRsCmd(m.appState.GitHubService, m.appState.GithubInfo, m.appState.DemoMode, 0)))
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
	m.silentReloadInFlight = false
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
		cmds = append(cmds, m.wrapFirstPRLoadCmd(prstab.LoadPRsCmd(m.appState.GitHubService, m.appState.GithubInfo, m.appState.DemoMode, 0)))
		cmds = append(cmds, prstab.PrTickCmd())
	}
	m.prsTabModel.SetGithubService(m.isGitHubAvailable())
	return m, tea.Batch(cmds...)
}

// handleRemoteOpResultMsg processes the outcome of an Apply / CreateGh / Remove origin command
// dispatched from Settings → GitHub → Repository remote. On success: refresh the cached origin
// shown in the panel, set a status message, and reload the repo so PR / branch flows pick up
// the new remote bookmarks. On failure: surface the error in the modal so the user can read it
// (Retry isn't useful here because the command is idempotent and the user can simply re-press
// Apply with corrections).
//
// Special-case for Op == RemoteOpCreateGh: the command attempts an inline `jj git push` after
// creating the GitHub repo. PushErr being non-nil while Err is nil is the soft-failure case
// (repo created, push failed) — we surface the push error in the modal but keep the new origin
// in place so the user can retry via the Push all bookmarks button without re-creating.
func (m *Model) handleRemoteOpResultMsg(msg data.RemoteOpResultMsg) (tea.Model, tea.Cmd) {
	m.appState.Loading = false
	if msg.Err != nil {
		m.errorModal.SetError(msg.Err, false, "")
		// Refresh anyway so the panel shows whatever state we ended up in (e.g. the user
		// changed origin but the fetch failed; current origin should still update).
		m.refreshSettingsOriginURL()
		return m, nil
	}
	switch msg.Op {
	case data.RemoteOpApply:
		if msg.PreviousURL == "" {
			m.appState.StatusMessage = fmt.Sprintf("Added origin %s", msg.NewURL)
		} else if msg.PreviousURL != msg.NewURL {
			m.appState.StatusMessage = fmt.Sprintf("Updated origin → %s", msg.NewURL)
		} else {
			m.appState.StatusMessage = "Origin already set to that URL; refreshed"
		}
	case data.RemoteOpCreateGh:
		base := "Created GitHub repo"
		if msg.NewURL != "" {
			base = fmt.Sprintf("Created GitHub repo (%s)", msg.NewURL)
		}
		switch {
		case msg.PushErr != nil:
			// Soft-failure: create succeeded, push didn't. Status reads the success-side, the
			// modal carries the failure detail so the user knows to retry the push.
			m.appState.StatusMessage = base + "; push failed (see error)"
			m.errorModal.SetError(fmt.Errorf("post-create push failed: %w\nUse Push all bookmarks to retry once you've resolved the underlying issue", msg.PushErr), false, "")
		case msg.PushedCount > 0:
			m.appState.StatusMessage = fmt.Sprintf("%s and pushed %d bookmark(s): %s", base, msg.PushedCount, strings.Join(msg.PushedNames, ", "))
		default:
			m.appState.StatusMessage = base + " (no bookmarks to push yet)"
		}
	case data.RemoteOpRemove:
		m.appState.StatusMessage = fmt.Sprintf("Removed origin (was %s)", msg.PreviousURL)
		// Clear the input so the user doesn't re-Apply the same URL by accident on the next
		// keystroke. They can retype if they want to re-add it.
		m.settingsTabModel.GetGitHubModel().SetOriginURL("")
	}
	m.refreshSettingsOriginURL()
	// Reload the repo (and branches) so any newly fetched remote bookmarks appear immediately.
	cmds := []tea.Cmd{
		data.LoadRepository(m.appState.JJService),
	}
	return m, tea.Batch(cmds...)
}

// handlePushResultMsg processes the outcome of a standalone Push current / Push all action from
// the Repository remote panel. Mirrors handleRemoteOpResultMsg's success/failure handling but
// stays distinct because the panel needs different status text and because no origin URL state
// changes — only the remote bookmarks and the repo PRs view.
func (m *Model) handlePushResultMsg(msg data.PushResultMsg) (tea.Model, tea.Cmd) {
	m.appState.Loading = false
	if msg.Err != nil {
		m.errorModal.SetError(msg.Err, false, "")
		return m, nil
	}
	switch {
	case msg.PushedCount == 0:
		m.appState.StatusMessage = "Nothing to push (no local bookmarks yet)"
		return m, nil
	case msg.All:
		if len(msg.PushedNames) > 0 {
			m.appState.StatusMessage = fmt.Sprintf("Pushed %d bookmark(s) to origin: %s", msg.PushedCount, strings.Join(msg.PushedNames, ", "))
		} else {
			m.appState.StatusMessage = fmt.Sprintf("Pushed %d bookmark(s) to origin", msg.PushedCount)
		}
	default:
		if len(msg.PushedNames) > 0 {
			m.appState.StatusMessage = fmt.Sprintf("Pushed bookmark %s to origin", msg.PushedNames[0])
		} else {
			m.appState.StatusMessage = "Pushed current bookmark to origin"
		}
	}
	// Reload the repo so the graph picks up new remote-tracking bookmarks (e.g. main@origin).
	return m, data.LoadRepository(m.appState.JJService)
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
	m.silentReloadInFlight = false
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
		m.appState.ViewMode == state.ViewFileDiff ||
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
	if !m.silentReloadInFlight && !m.appState.Loading && !m.aiGenOverlayActive && m.appState.JJService != nil && m.appState.ViewMode != state.ViewEditDescription && m.appState.ViewMode != state.ViewCreatePR && m.appState.ViewMode != state.ViewCreateTicket && m.appState.ViewMode != state.ViewCreateBookmark && m.appState.ViewMode != state.ViewFileDiff && (m.appState.ViewMode != state.ViewEvologSplit || !m.evologSplitModal.SuggestLoading()) && !m.graphTabModel.IsInRebaseMode() {
		revset := ""
		if m.appState.Config != nil {
			revset = m.appState.Config.GraphRevset
		}
		m.silentReloadInFlight = true
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
	m.appState.Loading = false
	m.appState.StatusMessage = e.Reason
	cfg, _ := config.Load()
	if cfg != nil {
		cfg.ClearGitHub()
		_ = cfg.Save()
	}
	_ = os.Unsetenv("GITHUB_TOKEN")
	m.appState.GitHubService = nil
	src := config.GitHubTokenSourceSaved
	if cfg != nil {
		src = cfg.GitHubTokenSourceOrDefault()
	}
	if src == config.GitHubTokenSourceGhCLI {
		return m, settingstab.StartGitHubCLILoginShowCmd()
	}
	return m, settingstab.StartGitHubLoginCmd()
}
