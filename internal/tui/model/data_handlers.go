package model

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/data"
	"github.com/madicen/jj-tui/internal/tui/state"
	graphtab "github.com/madicen/jj-tui/internal/tui/tabs/graph"
	prstab "github.com/madicen/jj-tui/internal/tui/tabs/prs"
	settingstab "github.com/madicen/jj-tui/internal/tui/tabs/settings"
)

// handleDataServicesInitializedMsg applies initialized services and repository; starts tick and PR load.
// Kept for tests or code paths that still send the full message.
//
//nolint:staticcheck // SA1019: transitional handler intentionally still processes the deprecated one-shot message.
func (m *Model) handleDataServicesInitializedMsg(msg data.ServicesInitializedMsg) (tea.Model, tea.Cmd) {
	m.silentReloadInFlight = false
	m.appState.JJService = msg.JJService
	m.appState.GitHubService = msg.GitHubService
	m.appState.TicketService = msg.TicketService
	m.appState.UpdateRepository(msg.Repository)
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
	m.appState.UpdateRepository(msg.Repository)
	m.appState.DemoMode = msg.DemoMode
	m.appState.Loading = false
	m.appState.StatusMessage = fmt.Sprintf("Loaded %d commits", len(msg.Repository.Graph.Commits))
	if m.appState.Repository != nil {
		m.appState.Repository.PRs = nil
	}
	m.propagateRepository()
	m.prsTabModel.SetGithubService(false)
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
	m.appState.DefaultBranch = msg.DefaultBranch
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
		m.applyEffects(effShowError{msg.Err})
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
			m.applyEffects(effShowError{fmt.Errorf("post-create push failed: %w\nUse Push all bookmarks to retry once you've resolved the underlying issue", msg.PushErr)})
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
	return m, m.applyEffects(effReloadRepository{})
}

// handlePushResultMsg processes the outcome of a standalone Push current / Push all action from
// the Repository remote panel. Mirrors handleRemoteOpResultMsg's success/failure handling but
// stays distinct because the panel needs different status text and because no origin URL state
// changes — only the remote bookmarks and the repo PRs view.
func (m *Model) handlePushResultMsg(msg data.PushResultMsg) (tea.Model, tea.Cmd) {
	m.appState.Loading = false
	if msg.Err != nil {
		// P5.5: push failures are usually transient (network / auth); offer Retry that re-runs
		// the same push (msg.All carries whether this was Push all vs Push current).
		return m, m.applyEffects(effShowRetryableError{
			err:   msg.Err,
			retry: data.PushBookmarksCmd(m.appState.JJService, msg.All),
		})
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
	return m, m.applyEffects(effReloadRepository{})
}

// handleDataRepositoryLoadedMsg delegates to shared applyRepositoryLoaded.
func (m *Model) handleDataRepositoryLoadedMsg(msg data.RepositoryLoadedMsg) (tea.Model, tea.Cmd) {
	return m.applyRepositoryLoaded(msg.Repository)
}

// handleActionsRepositoryLoadedMsg delegates to shared applyRepositoryLoaded.
func (m *Model) handleActionsRepositoryLoadedMsg(msg graphtab.RepositoryLoadedMsg) (tea.Model, tea.Cmd) {
	return m.applyRepositoryLoaded(msg.Repository)
}

// handleOpenPRsResolvedMsg merges targeted per-branch open-PR lookups into the repository's PR list
// (deduped by PR number) so the graph can offer "Update PR" for branches whose PR was missing from
// the bulk list. Existing entries win to avoid clobbering richer data (e.g. merged/closed state).
func (m *Model) handleOpenPRsResolvedMsg(msg prstab.OpenPRsResolvedMsg) (tea.Model, tea.Cmd) {
	if m.appState.Repository == nil || len(msg.Prs) == 0 {
		return m, nil
	}
	existingByNumber := make(map[int]bool)
	for _, pr := range m.appState.Repository.PRs {
		existingByNumber[pr.Number] = true
	}
	added := false
	for _, pr := range msg.Prs {
		if existingByNumber[pr.Number] {
			continue
		}
		m.appState.Repository.PRs = append(m.appState.Repository.PRs, pr)
		existingByNumber[pr.Number] = true
		added = true
	}
	if added {
		m.graphTabModel.OnRepositoryLoaded(m.appState.Repository)
		m.prsTabModel.OnRepositoryLoaded(m.appState.Repository)
	}
	return m, nil
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
		m.appState.UpdateRepository(msg.Repository)
		m.appState.Repository.PRs = oldPRs
		m.propagateRepository()
		m.prsTabModel.SetGithubService(m.isGitHubAvailable())
		newCount := len(msg.Repository.Graph.Commits)
		if newCount != oldCount && m.errorModal.GetError() == nil {
			m.appState.StatusMessage = fmt.Sprintf("Updated: %d commits", newCount)
		}
	}
	return m, nil
}

// handleTickMsg runs auto-refresh and ensures changed files for selected commit; forwards PR tick to PRs tab.
func (m *Model) handleTickMsg(now time.Time) (tea.Model, tea.Cmd) {
	// Don't run background refresh/updates if a modal is showing or we're in a blocking flow
	isBlockingView := m.appState.ViewMode == state.ViewEditDescription ||
		m.appState.ViewMode == state.ViewCreatePR ||
		m.appState.ViewMode == state.ViewCreateBookmark ||
		m.appState.ViewMode == state.ViewGitHubLogin ||
		m.appState.ViewMode == state.ViewFileDiff ||
		m.graphTabModel.IsInRebaseMode() ||
		m.graphTabModel.IsInMergeMode()

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
	// P5.2: opt-in silent auto-refresh. shouldSilentReload gates on ui.auto_refresh_seconds
	// (0/off by default), the configured minimum spacing, and the modal-open / in-flight guards.
	if m.shouldSilentReload(now) {
		revset := ""
		if m.appState.Config != nil {
			revset = m.appState.Config.GraphRevset
			// Mirror LoadRepository's mine() intersection so the silent background
			// refresh produces the same graph as the foreground load. Without this,
			// the periodic tick would silently widen the revset and reintroduce
			// other contributors' commits between user-initiated reloads.
			if m.appState.Config.GraphFilterToMine() {
				revset = jj.ApplyMineFilterToRevset(revset)
			}
			m.appState.JJService.BookmarkListPreferTracked = m.appState.Config.BranchesFilterToTrackedAndMine()
		}
		m.silentReloadInFlight = true
		m.lastAutoRefresh = now
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

// shouldSilentReload reports whether handleTickMsg should kick off a P5.2 silent background
// graph reload on this tick. It is the single guard for the feature and is intentionally
// conservative: auto-refresh must never clobber in-progress work, so ANY open modal (form,
// error, warning, workspaces, operations, evolog split, file diff) or in-flight/loading state
// suppresses it, as do the in-graph rebase/merge modes (which aren't modals). It also honors
// the configured minimum spacing so the faster heartbeat tick can't refresh more often than
// ui.auto_refresh_seconds. Returns false when the feature is off (interval <= 0, the default).
func (m *Model) shouldSilentReload(now time.Time) bool {
	if m.appState.JJService == nil {
		return false
	}
	interval := m.appState.Config.AutoRefreshInterval()
	if interval <= 0 {
		return false // auto-refresh disabled (default)
	}
	// Never overlap or clobber work already in flight.
	if m.silentReloadInFlight || m.appState.Loading || m.aiGenOverlayActive {
		return false
	}
	// Any open modal suppresses the background refresh (ModalStack read-model covers form modals,
	// the error/warning overlays, and the graph-overlay modals like workspaces/operations/evolog).
	stack := m.modalStack()
	if stack.Len() > 0 {
		return false
	}
	// Rebase/merge are in-graph modes rather than chromed modals, so guard them explicitly.
	if m.graphTabModel.IsInRebaseMode() || m.graphTabModel.IsInMergeMode() {
		return false
	}
	// Respect the configured minimum spacing between silent reloads.
	if !m.lastAutoRefresh.IsZero() && now.Sub(m.lastAutoRefresh) < interval {
		return false
	}
	return true
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
