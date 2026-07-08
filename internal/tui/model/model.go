package model

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	bubbledropdown "github.com/madicen/bubble-dropdown"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	aitab "github.com/madicen/jj-tui/internal/tui/ai"
	"github.com/madicen/jj-tui/internal/tui/data"
	"github.com/madicen/jj-tui/internal/tui/genmenu"
	"github.com/madicen/jj-tui/internal/tui/state"
	bookmarktab "github.com/madicen/jj-tui/internal/tui/tabs/bookmark"
	branchestab "github.com/madicen/jj-tui/internal/tui/tabs/branches"
	conflicttab "github.com/madicen/jj-tui/internal/tui/tabs/conflict"
	descedittab "github.com/madicen/jj-tui/internal/tui/tabs/descedit"
	divergenttab "github.com/madicen/jj-tui/internal/tui/tabs/divergent"
	evologsplittab "github.com/madicen/jj-tui/internal/tui/tabs/evologsplit"
	filedifftab "github.com/madicen/jj-tui/internal/tui/tabs/filediff"
	graphtab "github.com/madicen/jj-tui/internal/tui/tabs/graph"
	"github.com/madicen/jj-tui/internal/tui/tabs/help/commandhistory"
	prformtab "github.com/madicen/jj-tui/internal/tui/tabs/prform"
	prstab "github.com/madicen/jj-tui/internal/tui/tabs/prs"
	settingstab "github.com/madicen/jj-tui/internal/tui/tabs/settings"
	ticketformtab "github.com/madicen/jj-tui/internal/tui/tabs/ticketform"
	ticketstab "github.com/madicen/jj-tui/internal/tui/tabs/tickets"
	workspacestab "github.com/madicen/jj-tui/internal/tui/tabs/workspaces"
	"github.com/madicen/jj-tui/internal/tui/util"
)

// estimatedContentHeight returns height available for tab content (excluding header/status).
// Used in Update() when delegating to tabs so viewport/list dimensions are correct for scroll handling.
func (m *Model) estimatedContentHeight() int {
	return max(m.height-4, 1)
}

func (m *Model) beginModalUnderlay() {
	m.modalUnderlayView = m.appState.ViewMode
	m.modalUnderlayValid = true
}

func (m *Model) clearModalUnderlay() {
	m.modalUnderlayValid = false
}

// restoreModalUnderlayOrGraph restores the tab from beginModalUnderlay, or the graph if none.
func (m *Model) restoreModalUnderlayOrGraph() {
	if m.modalUnderlayValid {
		m.appState.ViewMode = m.modalUnderlayView
		m.modalUnderlayValid = false
		return
	}
	m.appState.ViewMode = state.ViewCommitGraph
}

func (m *Model) clearAIGenOverlay() {
	m.aiGenOverlayActive = false
}

// clearPendingAIRetry forgets any saved AI replay target. Called on successful generation, on
// error dismissal, and whenever we transition back to the main views without a form modal open.
func (m *Model) clearPendingAIRetry() {
	m.pendingAIRetryActive = false
	m.pendingAIRetryKind = 0
	m.pendingAIRetryOverrideProfile = ""
}

// resolveAIOverride looks up the optional one-shot profile override on t against the
// active config. Returns nil when the override is empty or the profile is not found.
// A missing profile name is treated as "use active" (the menu can only be populated
// from the live profile list, so this is defensive only).
func (m *Model) resolveAIOverride(t state.NavigateTarget) *config.AIProfile {
	name := strings.TrimSpace(t.AIOverrideProfile)
	if name == "" || m.appState.Config == nil {
		return nil
	}
	if p, ok := m.appState.Config.FindAIProfile(name); ok {
		return &p
	}
	return nil
}

// aiGenStatusMessage decorates a default generating-message with the profile name
// when an override is in use so the user can see which model is actually running.
func aiGenStatusMessage(base string, override *config.AIProfile) string {
	if override == nil {
		return base
	}
	return fmt.Sprintf("%s (using %s)", base, override.Name)
}

// pushAIProfilesToFormModals propagates the current config's AI profile list and
// active profile name to every form modal that has a long-press generate menu.
// Called whenever a generate-bearing modal opens and after settings changes
// alter the saved profile list. When cfg is nil we still push an empty list so
// the modals don't carry stale state from a previous repo.
func (m *Model) pushAIProfilesToFormModals() {
	var profiles []config.AIProfile
	active := ""
	if m.appState.Config != nil {
		profiles = m.appState.Config.AIProfileList()
		active = m.appState.Config.ActiveAIProfile().Name
	}
	m.desceditModal.SetAIProfiles(profiles, active)
	m.prFormModal.SetAIProfiles(profiles, active)
	m.bookmarkModal.SetAIProfiles(profiles, active)
	m.ticketFormModal.SetAIProfiles(profiles, active)
}

// activeFormModalGenMenu returns the genmenu.State for the currently shown
// form modal that has a generate-button long-press popover, or nil when no
// such modal is active. Used by the mouse-routing path to forward press /
// motion / release events and by the view layer to overlay the popover.
func (m *Model) activeFormModalGenMenu() *genmenu.State {
	switch m.appState.ViewMode {
	case state.ViewEditDescription:
		return m.desceditModal.MenuState()
	case state.ViewCreatePR:
		return m.prFormModal.MenuState()
	case state.ViewCreateBookmark:
		return m.bookmarkModal.MenuState()
	case state.ViewCreateTicket:
		return m.ticketFormModal.MenuState()
	}
	return nil
}

// forwardMouseToActiveFormModal forwards a tea.MouseMsg to whichever form modal
// owns the active view so the modal's long-press genmenu can advance its state.
// Returns the cmd from the modal Update (typically the tick cmd on a fresh press
// or a NavigateGenerate* cmd on release over a menu row).
func (m *Model) forwardMouseToActiveFormModal(msg tea.MouseMsg) tea.Cmd {
	switch m.appState.ViewMode {
	case state.ViewEditDescription:
		updated, cmd := m.desceditModal.Update(msg)
		m.desceditModal = updated
		return cmd
	case state.ViewCreatePR:
		updated, cmd := m.prFormModal.Update(msg)
		m.prFormModal = updated
		return cmd
	case state.ViewCreateBookmark:
		updated, cmd := m.bookmarkModal.Update(msg)
		m.bookmarkModal = updated
		return cmd
	case state.ViewCreateTicket:
		updated, cmd := m.ticketFormModal.Update(msg)
		m.ticketFormModal = updated
		return cmd
	}
	return nil
}

// forwardGenMenuTickToActiveFormModal routes a genmenu.TickMsg to the active
// form modal so its long-press tick can pop the menu when still pressed.
func (m *Model) forwardGenMenuTickToActiveFormModal(msg genmenu.TickMsg) tea.Cmd {
	switch m.appState.ViewMode {
	case state.ViewEditDescription:
		updated, cmd := m.desceditModal.Update(msg)
		m.desceditModal = updated
		return cmd
	case state.ViewCreatePR:
		updated, cmd := m.prFormModal.Update(msg)
		m.prFormModal = updated
		return cmd
	case state.ViewCreateBookmark:
		updated, cmd := m.bookmarkModal.Update(msg)
		m.bookmarkModal = updated
		return cmd
	case state.ViewCreateTicket:
		updated, cmd := m.ticketFormModal.Update(msg)
		m.ticketFormModal = updated
		return cmd
	}
	return nil
}

// activeFormModalGenMenuOverlay returns (view, x, y) for the currently visible
// long-press popover, or ("", 0, 0) when none is shown. Used in view_helpers.go.
func (m *Model) activeFormModalGenMenuOverlay() (string, int, int) {
	switch m.appState.ViewMode {
	case state.ViewEditDescription:
		return m.desceditModal.MenuOverlay()
	case state.ViewCreatePR:
		return m.prFormModal.MenuOverlay()
	case state.ViewCreateBookmark:
		return m.bookmarkModal.MenuOverlay()
	case state.ViewCreateTicket:
		return m.ticketFormModal.MenuOverlay()
	}
	return "", 0, 0
}

// buildSettingsViewOpts builds ViewOpts for the settings tab (used when entering settings or on resize).
func (m *Model) buildSettingsViewOpts() settingstab.ViewOpts {
	ticketName := ""
	if m.appState.TicketService != nil {
		ticketName = m.appState.TicketService.GetProviderName()
	}
	_, ghLookErr := exec.LookPath("gh")
	return settingstab.ViewOpts{
		GitHubAvailable:   m.isGitHubAvailable(),
		TicketServiceName: ticketName,
		Config:            m.appState.Config,
		ContentHeight:     m.estimatedContentHeight(),
		GhAvailable:       ghLookErr == nil,
	}
}

// Auto-refresh interval for the repository view.
// Kept at 5s to limit CPU and allocation churn from repeated jj log + parse.
const autoRefreshInterval = 5 * time.Second

// tickCmd returns a command that sends a tick after the refresh interval.
func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(autoRefreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// isGitHubAvailable returns true if GitHub functionality is available (real service or demo mode).
func (m *Model) isGitHubAvailable() bool {
	return m.appState.GitHubService != nil || m.appState.DemoMode
}

// isSelectedCommitValid returns true if selected commit index points to a valid commit.
func (m *Model) isSelectedCommitValid() bool {
	return m.appState.Repository != nil &&
		m.GetSelectedCommit() >= 0 &&
		m.GetSelectedCommit() < len(m.appState.Repository.Graph.Commits)
}

// applyRepositoryLoaded applies a loaded repository from data or actions package (shared logic).
func (m *Model) applyRepositoryLoaded(repo *internal.Repository) (*Model, tea.Cmd) {
	m.silentReloadInFlight = false
	var oldPRs []internal.GitHubPR
	if m.appState.Repository != nil {
		oldPRs = m.appState.Repository.PRs
	}
	m.appState.Repository = repo
	m.appState.Repository.PRs = oldPRs
	m.appState.Loading = false
	if m.appState.JJService == nil {
		jjSvc, _ := jj.NewService("")
		m.appState.JJService = jjSvc
	}
	m.appState.StatusMessage = fmt.Sprintf("Loaded %d commits", len(repo.Graph.Commits))
	m.graphTabModel.UpdateRepository(m.appState.Repository)
	m.prsTabModel.UpdateRepository(m.appState.Repository)
	m.prsTabModel.SetGithubService(m.isGitHubAvailable())
	m.branchesTabModel.UpdateRepository(m.appState.Repository)
	m.ticketsTabModel.UpdateRepository(m.appState.Repository)
	m.settingsTabModel.UpdateRepository(m.appState.Repository)
	m.helpTabModel.UpdateRepository(m.appState.Repository)
	var cmds []tea.Cmd
	cmds = append(cmds, m.tickCmd())
	if m.appState.GitHubService != nil {
		existing := 0
		if m.appState.Repository != nil {
			existing = len(m.appState.Repository.PRs)
		}
		cmds = append(cmds, m.wrapFirstPRLoadCmd(prstab.LoadPRsCmd(m.appState.GitHubService, m.appState.GithubInfo, m.appState.DemoMode, existing)))
	}
	commits := repo.Graph.Commits
	if len(commits) > 0 {
		idx := m.graphTabModel.GetSelectedCommit()
		if idx < 0 {
			idx = 0
		}
		m.graphTabModel.SelectCommit(idx)
		cmds = append(cmds, graphtab.LoadChangedFilesCmd(m.appState.JJService, commits[idx].ChangeID))
	}
	return m, tea.Batch(cmds...)
}

// bookmarksNeedingPRLookup collects local bookmark names in the graph that should be resolved to an
// open PR via a targeted query. It skips the default branch and bookmarks already matched to an open
// PR in the current list, and caps the count so a graph with many bookmarks can't fan out unboundedly.
func (m *Model) bookmarksNeedingPRLookup() []string {
	if m.appState.Repository == nil {
		return nil
	}
	openPRBranches := make(map[string]bool)
	for _, pr := range m.appState.Repository.PRs {
		if pr.State == "open" {
			openPRBranches[pr.HeadBranch] = true
		}
	}
	const maxLookups = 25
	seen := make(map[string]bool)
	var names []string
	for _, commit := range m.appState.Repository.Graph.Commits {
		for _, branch := range commit.Branches {
			local := util.LocalBookmarkName(branch)
			if local == "" || seen[local] {
				continue
			}
			if m.appState.DefaultBranch != "" && local == m.appState.DefaultBranch {
				continue
			}
			if local == "main" || local == "master" || local == "trunk" {
				continue
			}
			if openPRBranches[local] || openPRBranches[branch] {
				continue
			}
			seen[local] = true
			names = append(names, local)
			if len(names) >= maxLookups {
				return names
			}
		}
	}
	return names
}

// refreshRepository starts a refresh of the repository data.
func (m *Model) refreshRepository() tea.Cmd {
	m.appState.StatusMessage = "Refreshing..."
	var cmds []tea.Cmd
	if m.appState.JJService == nil {
		cmds = append(cmds, data.InitializeServices(m.appState.DemoMode))
	} else {
		cmds = append(cmds, m.applyEffects(effReloadRepository{}))
		// Branches tab keeps its own list (trunk graph, HasConflict); ^r must reload it too or diverged
		// bookmarks look stale after resolve until the user switches tabs or something else loads branches.
		cmds = append(cmds, m.applyEffects(effLoadBranches{}))
	}
	if m.isGitHubAvailable() {
		existing := 0
		if m.appState.Repository != nil {
			existing = len(m.appState.Repository.PRs)
		}
		cmds = append(cmds, m.wrapFirstPRLoadCmd(prstab.LoadPRsCmd(m.appState.GitHubService, m.appState.GithubInfo, m.appState.DemoMode, existing)))
	}
	svc := m.appState.TicketService
	if svc != nil && !util.IsNilInterface(svc) {
		cmds = append(cmds, ticketstab.LoadTicketsCmd(svc, m.appState.DemoMode))
	}
	return tea.Batch(cmds...)
}

// createIsZoneClickedFuncWithEvent returns a function that checks if the given zone ID contains the mouse event.
func (m *Model) createIsZoneClickedFuncWithEvent(event tea.MouseMsg) func(string) bool {
	return func(zoneID string) bool {
		z := m.zoneManager.Get(zoneID)
		return z != nil && z.InBounds(event)
	}
}

// --- Handlers: main routes to tabs; tabs own context (BuildRequestContextFrom) and execution (ExecuteRequest / EnterTab). ---

// processGraphRequest runs a graph request via the graph tab; ApplyResult mutates app and returns cmd.
func (m *Model) processGraphRequest(r graphtab.Request) (tea.Model, tea.Cmd) {
	if r.Checkout || r.Squash || r.Abandon || r.NewCommit || r.PerformRebase || r.DragRebase || r.ResolveDivergent != nil || r.CreateBookmark || r.DeleteBookmark || r.CreatePR || r.UpdatePR || r.MoveFileUp || r.MoveFileDown || r.RevertFile || r.MoveDeltaOntoOrigin || r.StartEvologSplit || r.ResolveBookmarkConflict || r.Duplicate || r.Backout {
		m.redoOperationID = ""
	}
	ctx := graphtab.BuildRequestContextFrom(m)
	res := graphtab.HandleRequest(r, ctx)
	cmd := graphtab.ApplyResult(res, &m.graphTabModel, ctx, &m.appState)
	return m, m.wrapGraphTabCmd(cmd)
}

func (m *Model) handleHelpRequest(r commandhistory.Request) (tea.Model, tea.Cmd) {
	statusMsg, cmd := commandhistory.ExecuteRequest(r)
	if statusMsg != "" {
		m.appState.StatusMessage = statusMsg
	}
	return m, cmd
}

func (m *Model) handleSettingsRequest(r settingstab.Request) (tea.Model, tea.Cmd) {
	statusMsg, cmd := settingstab.ExecuteRequest(r)
	if statusMsg != "" {
		m.appState.StatusMessage = statusMsg
	}
	return m, cmd
}

func (m *Model) handleNavigateToGraphTab() (tea.Model, tea.Cmd) {
	m.appState.ViewMode = state.ViewCommitGraph
	m.appState.StatusMessage = "Loading commit graph"
	return m, m.refreshRepository()
}

func (m *Model) handleNavigateToPRTab() (tea.Model, tea.Cmd) {
	m.appState.ViewMode = state.ViewPullRequests
	status, cmd := prstab.EnterTab(m)
	m.appState.StatusMessage = status
	if cmd != nil {
		cmd = m.wrapFirstPRLoadCmd(cmd)
	}
	return m, cmd
}

func (m *Model) handleNavigateToTicketsTab() (tea.Model, tea.Cmd) {
	m.appState.ViewMode = state.ViewTickets
	status, cmd := ticketstab.EnterTab(m)
	m.appState.StatusMessage = status
	if cmd != nil && !m.appState.TicketsLoadedOnce {
		m.appState.Loading = true
		m.appState.StatusMessage = "Loading tickets…"
		return m, tea.Batch(cmd, m.startBusySpinnerCmd())
	}
	return m, cmd
}

func (m *Model) handleNavigateToSettingsTab() (tea.Model, tea.Cmd) {
	m.appState.ViewMode = state.ViewSettings
	m.settingsTabModel.SetViewOpts(m.buildSettingsViewOpts())
	m.refreshSettingsOriginURL()
	return m, m.settingsTabModel.EnterTab()
}

// refreshSettingsOriginURL queries the jj service for the current `origin` URL and caches it on
// the GitHub settings sub-model so renderRepositoryRemote can display "Current origin: …" /
// "(none configured)". Called on Settings open and after every successful Apply / Create /
// Remove operation so the cached value reflects reality without a refresh shortcut.
func (m *Model) refreshSettingsOriginURL() {
	gh := m.settingsTabModel.GetGitHubModel()
	if m.appState.JJService == nil {
		gh.SetCurrentOrigin("")
		return
	}
	url, err := m.appState.JJService.GetGitRemoteURL(context.Background())
	if err != nil || strings.TrimSpace(url) == "" {
		gh.SetCurrentOrigin("")
		return
	}
	gh.SetCurrentOrigin(strings.TrimSpace(url))
	// Pre-fill the input with the existing URL so users see what's there and can edit/replace it
	// rather than retyping from scratch. Empty input is left empty.
	if gh.GetOriginURL() == "" {
		gh.SetOriginURL(strings.TrimSpace(url))
	}
}

func (m *Model) handleNavigateToHelpTab() (tea.Model, tea.Cmd) {
	m.appState.ViewMode = state.ViewHelp
	m.refreshHelpCommandHistory()
	m.helpTabModel.SetSelectedCommand(0)
	m.appState.StatusMessage = "Loaded Help"
	return m, nil
}

func (m *Model) handleNavigateToWorkspaces() (tea.Model, tea.Cmd) {
	if m.appState.JJService == nil {
		return m, nil
	}
	m.appState.Loading = true
	m.appState.StatusMessage = "Loading workspaces…"
	return m, workspacestab.LoadWorkspacesCmd(m.appState.JJService)
}

func (m *Model) handleNavigateToBranchesTab() (tea.Model, tea.Cmd) {
	m.appState.ViewMode = state.ViewBranches
	status, cmd := branchestab.EnterTab(m)
	m.appState.StatusMessage = status
	return m, cmd
}

// handleNavigate performs view changes that only main can do (it owns modals and cross-tab state).
func (m *Model) handleNavigate(t state.NavigateTarget) (tea.Model, tea.Cmd) {
	if t.Kind == state.NavigateSaveDescription || t.Kind == state.NavigateSubmitBookmark || t.Kind == state.NavigateSubmitPR || t.Kind == state.NavigateSubmitTicket || t.Kind == state.NavigateResolveConflict || t.Kind == state.NavigateResolveDivergent || t.Kind == state.NavigateRunInit || t.Kind == state.NavigatePerformEvologSplit {
		m.redoOperationID = ""
	}
	switch t.Kind {
	case state.NavigateEditDescription:
		// If we're entering edit-description from the empty-description warning, ensure the warning is closed.
		m.warningModal.Hide()
		if m.appState.Repository != nil {
			for i, c := range m.appState.Repository.Graph.Commits {
				if c.ChangeID == t.Commit.ChangeID {
					m.graphTabModel.SelectCommit(i)
					break
				}
			}
		}
		return m.startEditingDescription(t.Commit)
	case state.NavigateCreateBookmark:
		m.startCreateBookmark()
		return m, m.applyEffects(effLoadBranches{})
	case state.NavigateCreateBookmarkFromTicket:
		m.beginModalUnderlay()
		m.appState.ViewMode = state.ViewCreateBookmark
		m.appState.StatusMessage = bookmarktab.OpenCreateBookmarkFromTicket(&m.bookmarkModal, m.appState.Repository, t.TicketKey, t.TicketTitle, t.TicketDisplayKey, m.branchesTabModel.BuildBookmarkNameConflictSources(), m.appState.Config != nil && m.appState.Config.ShouldSanitizeBookmarkNames(), ModalInnerWidth(m.width))
		m.pushAIProfilesToFormModals()
		return m, nil
	case state.NavigateWarning:
		m.warningModal.Show(t.WarningTitle, t.WarningMessage, t.WarningCommits)
		return m, nil
	case state.NavigateCreatePR:
		m.startCreatePR()
		return m, nil
	case state.NavigateBackToGraph:
		m.clearAIGenOverlay()
		m.clearPendingAIRetry()
		m.evologSplitModal.Hide()
		m.evologStepwiseRemainderAfterSplit = nil
		m.evologStepwiseBookmarkName = ""
		m.fileDiffModal.Hide()
		m.restoreModalUnderlayOrGraph()
		m.appState.Loading = false
		if t.StatusMessage != "" {
			m.appState.StatusMessage = t.StatusMessage
		}
		return m, nil
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
		return m, evologsplittab.LoadEvologCmd(m.appState.JJService, bn, t.Commit)
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
		return m, nil
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
			return m, nil
		}
		path := strings.TrimSpace(t.FileDiffPath)
		if path == "" || m.appState.JJService == nil {
			m.appState.StatusMessage = "Cannot open file diff"
			return m, nil
		}
		m.fileDiffModal = m.fileDiffModal.SetDimensions(m.width, m.height)
		seq := m.fileDiffModal.BeginLoad(t.Commit, path)
		m.appState.ViewMode = state.ViewFileDiff
		m.appState.StatusMessage = "Loading file diff…"
		return m, filedifftab.LoadFileDiffCmd(m.appState.JJService, seq, t.Commit.ChangeID, path)
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
		)
	case state.NavigateBackToBranches:
		m.appState.ViewMode = state.ViewBranches
		if t.StatusMessage != "" {
			m.appState.StatusMessage = t.StatusMessage
		}
		return m, nil
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
		return m, nil
	case state.NavigateBackToSettings:
		m.appState.ViewMode = state.ViewSettings
		if t.StatusMessage != "" {
			m.appState.StatusMessage = t.StatusMessage
		}
		return m, nil
	case state.NavigateDismissError:
		m.errorModal.ClearError()
		m.clearPendingAIRetry()
		// If a form modal (Edit Description, PR/Ticket/Bookmark, GitHub login) is open, keep it
		// open after dismissing the error. Previously we forced ViewMode back to the graph,
		// which silently discarded whatever the user had typed. Errors triggered from these
		// views are typically AI/network failures the user wants to recover from inline.
		if !m.isFormModalView() {
			m.appState.ViewMode = state.ViewCommitGraph
		}
		if t.StatusMessage != "" {
			m.appState.StatusMessage = t.StatusMessage
		}
		if t.RefreshAfterDismiss {
			return m, m.refreshRepository()
		}
		return m, m.tickCmd()
	case state.NavigateDismissInit:
		m.initRepoModel.SetPath("")
		m.appState.ViewMode = state.ViewCommitGraph
		if t.StatusMessage != "" {
			m.appState.StatusMessage = t.StatusMessage
		}
		return m, m.tickCmd()
	case state.NavigateGitHubLoginCancel:
		m.githubLoginModel.ClearFlow()
		m.clearModalUnderlay()
		m.appState.ViewMode = state.ViewSettings
		if t.StatusMessage != "" {
			m.appState.StatusMessage = t.StatusMessage
		}
		return m, nil
	case state.NavigateSaveDescription:
		// A second save while the first describe is still running causes parallel jj operations on the
		// same revision → divergent commits (same message, sibling children of one parent).
		if m.appState.Loading || m.aiGenOverlayActive {
			return m, nil
		}
		if t.SaveCommitID != "" && m.appState.JJService != nil {
			m.appState.Loading = true
			m.appState.StatusMessage = "Saving description…"
			cmd := graphtab.SaveDescriptionCmd(m.appState.JJService, t.SaveCommitID, t.SaveDescription)
			return m, tea.Batch(cmd, m.startBusySpinnerCmd())
		}
		return m, nil
	case state.NavigateSubmitBookmark:
		if m.appState.JJService != nil {
			cmd, status := bookmarktab.SubmitBookmark(&m.bookmarkModal, m.appState.Repository, m.appState.Config, m.appState.JJService)
			m.appState.StatusMessage = status
			if cmd == nil {
				return m, nil
			}
			m.appState.Loading = true
			// Batch the spinner tick so the busy overlay animates while jj creates the bookmark
			// and the repo reloads (cleared by applyRepositoryLoaded). Test harnesses that drain
			// cmds one message per step must expand the resulting tea.BatchMsg.
			return m, tea.Batch(cmd, m.startBusySpinnerCmd())
		}
		return m, nil
	case state.NavigateSubmitPR:
		if m.isGitHubAvailable() && m.appState.JJService != nil {
			return m, m.submitPR()
		}
		return m, nil
	case state.NavigateResolveConflict:
		m.appState.StatusMessage = "Resolving bookmark conflict..."
		return m, conflicttab.ResolveBookmarkConflictCmd(m.appState.JJService, t.ConflictBookmarkName, t.ConflictResolution)
	case state.NavigateResolveDivergent:
		m.appState.StatusMessage = "Resolving divergent commit..."
		return m, divergenttab.ResolveDivergentCommitCmd(m.appState.JJService, t.DivergentChangeID, t.DivergentKeepCommitID)
	case state.NavigateAddWorkspace:
		m.appState.Loading = true
		m.appState.StatusMessage = "Adding workspace…"
		return m, workspacestab.AddWorkspaceCmd(m.appState.JJService, t.WorkspacePath)
	case state.NavigateForgetWorkspace:
		m.appState.Loading = true
		m.appState.StatusMessage = "Forgetting workspace…"
		return m, workspacestab.ForgetWorkspaceCmd(m.appState.JJService, t.WorkspaceName)
	case state.NavigateCloseWorkspaces:
		m.workspacesModal.Hide()
		m.appState.ViewMode = state.ViewCommitGraph
		if t.StatusMessage != "" {
			m.appState.StatusMessage = t.StatusMessage
		}
		return m, nil
	case state.NavigateWarningCancel:
		if t.StatusMessage != "" {
			m.appState.StatusMessage = t.StatusMessage
		}
		return m, nil
	case state.NavigateRunInit:
		m.appState.Loading = true
		switch {
		case t.InitGhCreateRepo:
			m.appState.StatusMessage = "Initializing repository and creating GitHub repo…"
		case strings.TrimSpace(t.InitRemoteURL) != "":
			m.appState.StatusMessage = "Initializing repository and adding remote…"
		default:
			m.appState.StatusMessage = "Initializing repository…"
		}
		opts := data.InitOptions{
			Colocate:      t.InitColocate,
			RemoteURL:     t.InitRemoteURL,
			GhCreateRepo:  t.InitGhCreateRepo,
			GhRepoName:    t.InitGhRepoName,
			GhRepoPrivate: t.InitGhRepoPrivate,
		}
		return m, tea.Batch(data.RunJJInit(opts), m.startBusySpinnerCmd())
	case state.NavigateRemoteApply:
		url := strings.TrimSpace(t.RemoteURL)
		if url == "" {
			// Empty URL on a tab where origin is already configured is a "I cleared the field
			// to remove origin" intent; route to remove instead so the user doesn't have to
			// remember the Ctrl+x shortcut.
			gh := m.settingsTabModel.GetGitHubModel()
			if gh.GetCurrentOrigin() != "" {
				return m, data.RemoveOriginCmd(m.appState.JJService)
			}
			m.appState.StatusMessage = "Enter a remote URL first"
			return m, nil
		}
		m.appState.Loading = true
		m.appState.StatusMessage = "Configuring origin remote…"
		return m, tea.Batch(data.ApplyOriginCmd(m.appState.JJService, url), m.startBusySpinnerCmd())
	case state.NavigateRemoteCreateGh:
		m.appState.Loading = true
		m.appState.StatusMessage = "Creating GitHub repository…"
		// Repo name is implicitly the current working directory; the data layer derives it from
		// filepath.Base when name is empty so we don't need to plumb it through here.
		return m, tea.Batch(data.CreateGhRepoCmd(m.appState.JJService, "", t.RemoteRepoPrivate), m.startBusySpinnerCmd())
	case state.NavigateRemoteRemove:
		m.appState.Loading = true
		m.appState.StatusMessage = "Removing origin remote…"
		return m, tea.Batch(data.RemoveOriginCmd(m.appState.JJService), m.startBusySpinnerCmd())
	case state.NavigatePushBookmarks:
		m.appState.Loading = true
		if t.PushAll {
			m.appState.StatusMessage = "Pushing all bookmarks to origin…"
		} else {
			m.appState.StatusMessage = "Pushing current bookmark to origin…"
		}
		return m, tea.Batch(data.PushBookmarksCmd(m.appState.JJService, t.PushAll), m.startBusySpinnerCmd())
	case state.NavigateRetryError:
		// If we have a saved AI replay target, clear the modal and re-dispatch the same
		// NavigateGenerate* request via handleNavigate. The form modal underneath stays open
		// so the user keeps any text they typed, and the spinner overlay flips back on.
		if m.pendingAIRetryActive {
			m.errorModal.ClearError()
			retryKind := m.pendingAIRetryKind
			retryOverride := m.pendingAIRetryOverrideProfile
			// pendingAIRetryActive will be set again by the NavigateGenerate* handler.
			m.pendingAIRetryActive = false
			return m.handleNavigate(state.NavigateTarget{Kind: retryKind, AIOverrideProfile: retryOverride})
		}
		// No replayable action: fall back to the legacy behavior of dismissing and refreshing
		// the repository. Today the Retry button is hidden in this case (errortab.HasRetry is
		// false), so this branch is only reached if the user binds ctrl+r elsewhere.
		m.errorModal.ClearError()
		if !m.isFormModalView() {
			m.appState.ViewMode = state.ViewCommitGraph
		}
		return m, m.refreshRepository()
	case state.NavigateBackFromPRForm:
		m.clearAIGenOverlay()
		m.clearPendingAIRetry()
		m.prFormModal.Hide()
		m.restoreModalUnderlayOrGraph()
		if t.StatusMessage != "" {
			m.appState.StatusMessage = t.StatusMessage
		}
		return m, nil
	case state.NavigateCreateTicket:
		m.startCreateTicket()
		return m, nil
	case state.NavigateBackFromTicketForm:
		m.clearAIGenOverlay()
		m.clearPendingAIRetry()
		m.ticketFormModal.Hide()
		if m.modalUnderlayValid {
			m.appState.ViewMode = m.modalUnderlayView
			m.modalUnderlayValid = false
		} else {
			m.appState.ViewMode = state.ViewTickets
		}
		if t.StatusMessage != "" {
			m.appState.StatusMessage = t.StatusMessage
		}
		return m, nil
	case state.NavigateSubmitTicket:
		return m, m.submitTicket()
	case state.NavigateGenerateCommitDescription:
		if m.appState.Config == nil || !m.appState.Config.AIConfiguredForGeneration() {
			m.appState.StatusMessage = fmt.Sprintf("Enable AI in Settings → AI and set an API key (or %s)", config.EnvAIAPIKey)
			return m, nil
		}
		changeID := m.desceditModal.GetEditingCommitID()
		if changeID == "" {
			return m, nil
		}
		override := m.resolveAIOverride(t)
		m.aiGenReqID++
		rid := m.aiGenReqID
		m.appState.StatusMessage = aiGenStatusMessage("Generating description…", override)
		m.aiGenOverlayActive = true
		m.pendingAIRetryKind = state.NavigateGenerateCommitDescription
		m.pendingAIRetryActive = true
		m.pendingAIRetryOverrideProfile = t.AIOverrideProfile
		return m, tea.Batch(
			aitab.GenerateCommitDescriptionCmd(rid, m.appState.JJService, m.appState.Config, changeID, m.desceditModal.GetCommitShortID(), m.desceditModal.GetDescriptionValue(), override),
			m.startBusySpinnerCmd(),
		)
	case state.NavigateGeneratePRForm:
		if m.appState.Config == nil || !m.appState.Config.AIConfiguredForGeneration() {
			m.appState.StatusMessage = fmt.Sprintf("Enable AI in Settings → AI and set an API key (or %s)", config.EnvAIAPIKey)
			return m, nil
		}
		repo := m.appState.Repository
		idx := m.prFormModal.GetCommitIndex()
		if repo == nil || idx < 0 || idx >= len(repo.Graph.Commits) {
			return m, nil
		}
		changeID := repo.Graph.Commits[idx].ChangeID
		override := m.resolveAIOverride(t)
		m.aiGenReqID++
		rid := m.aiGenReqID
		m.appState.StatusMessage = aiGenStatusMessage("Generating PR title and body…", override)
		m.aiGenOverlayActive = true
		m.pendingAIRetryKind = state.NavigateGeneratePRForm
		m.pendingAIRetryActive = true
		m.pendingAIRetryOverrideProfile = t.AIOverrideProfile
		return m, tea.Batch(
			aitab.GeneratePRFormCmd(rid, m.appState.JJService, m.appState.Config, changeID, m.prFormModal.GetBaseBranch(), m.prFormModal.GetHeadBranch(), m.prFormModal.GetTitle(), override),
			m.startBusySpinnerCmd(),
		)
	case state.NavigateGenerateBookmarkName:
		if m.appState.Config == nil || !m.appState.Config.AIConfiguredForGeneration() {
			m.appState.StatusMessage = fmt.Sprintf("Enable AI in Settings → AI and set an API key (or %s)", config.EnvAIAPIKey)
			return m, nil
		}
		repo := m.appState.Repository
		idx := m.bookmarkModal.GetCommitIdx()
		rev := "@"
		if repo != nil && idx >= 0 && idx < len(repo.Graph.Commits) {
			rev = repo.Graph.Commits[idx].ChangeID
		}
		hint := ""
		if m.bookmarkModal.IsFromJira() {
			hint = strings.TrimSpace(m.bookmarkModal.GetJiraKey() + " " + m.bookmarkModal.GetJiraTicketTitle())
		}
		override := m.resolveAIOverride(t)
		m.aiGenReqID++
		rid := m.aiGenReqID
		m.appState.StatusMessage = aiGenStatusMessage("Generating bookmark name…", override)
		m.aiGenOverlayActive = true
		m.pendingAIRetryKind = state.NavigateGenerateBookmarkName
		m.pendingAIRetryActive = true
		m.pendingAIRetryOverrideProfile = t.AIOverrideProfile
		return m, tea.Batch(
			aitab.GenerateBookmarkNameCmd(rid, m.appState.JJService, m.appState.Config, rev, hint, override),
			m.startBusySpinnerCmd(),
		)
	case state.NavigateGenerateTicketForm:
		if m.appState.Config == nil || !m.appState.Config.AIConfiguredForGeneration() {
			m.appState.StatusMessage = fmt.Sprintf("Enable AI in Settings → AI and set an API key (or %s)", config.EnvAIAPIKey)
			return m, nil
		}
		repo := m.appState.Repository
		idx := m.GetSelectedCommit()
		changeID := "@"
		changeShort := "@"
		if repo != nil && idx >= 0 && idx < len(repo.Graph.Commits) {
			c := repo.Graph.Commits[idx]
			changeID = c.ChangeID
			if strings.TrimSpace(c.ShortID) != "" {
				changeShort = strings.TrimSpace(c.ShortID)
			} else {
				changeShort = changeID
			}
		}
		override := m.resolveAIOverride(t)
		m.aiGenReqID++
		rid := m.aiGenReqID
		m.appState.StatusMessage = aiGenStatusMessage("Generating ticket title and description…", override)
		m.aiGenOverlayActive = true
		m.pendingAIRetryKind = state.NavigateGenerateTicketForm
		m.pendingAIRetryActive = true
		m.pendingAIRetryOverrideProfile = t.AIOverrideProfile
		return m, tea.Batch(
			aitab.GenerateTicketFormCmd(rid, m.appState.JJService, m.appState.Config, changeID, changeShort, m.ticketFormModal.GetSummary(), m.ticketFormModal.GetDescription(), override),
			m.startBusySpinnerCmd(),
		)
	default:
		return m, nil
	}
}

func (m *Model) handleUndo() (tea.Model, tea.Cmd) {
	if m.appState.JJService != nil {
		m.appState.Loading = true
		m.appState.StatusMessage = "Undoing..."
		return m, tea.Batch(graphtab.UndoCmd(m.appState.JJService), m.startBusySpinnerCmd())
	}
	return m, nil
}

func (m *Model) handleRedo() (tea.Model, tea.Cmd) {
	if m.appState.JJService != nil && m.redoOperationID != "" {
		m.appState.Loading = true
		m.appState.StatusMessage = "Redoing..."
		return m, tea.Batch(graphtab.RedoCmd(m.appState.JJService, m.redoOperationID), m.startBusySpinnerCmd())
	}
	return m, nil
}

func (m *Model) handleSelectCommit(index int) (tea.Model, tea.Cmd) {
	return m.processGraphRequest(graphtab.Request{SelectCommit: &index})
}

// startEditingDescription switches to description edit view and starts loading the description.
func (m *Model) startEditingDescription(commit internal.Commit) (tea.Model, tea.Cmd) {
	m.beginModalUnderlay()
	m.appState.ViewMode = state.ViewEditDescription
	m.desceditModal, m.appState.StatusMessage = descedittab.StartEditing(m.desceditModal, commit, ModalInnerWidth(m.width), max(m.height-24, 3))
	m.pushAIProfilesToFormModals()
	return m, descedittab.LoadDescriptionCmd(m.appState.JJService, commit.ChangeID)
}

// startCreateBookmark opens the bookmark creation dialog for the selected commit.
func (m *Model) startCreateBookmark() {
	if !m.isSelectedCommitValid() {
		m.appState.StatusMessage = "No commit selected"
		return
	}
	m.beginModalUnderlay()
	idx := m.GetSelectedCommit()
	m.appState.ViewMode = state.ViewCreateBookmark
	m.appState.StatusMessage = bookmarktab.OpenCreateBookmark(&m.bookmarkModal, m.appState.Repository, idx, m.branchesTabModel.BuildBookmarkNameConflictSources(), m.appState.Config != nil && m.appState.Config.ShouldSanitizeBookmarkNames(), ModalInnerWidth(m.width))
	m.pushAIProfilesToFormModals()
}

// startCreatePR opens the PR creation dialog for the selected commit's bookmark.
func (m *Model) startCreatePR() {
	if !m.isSelectedCommitValid() {
		m.appState.StatusMessage = "No commit selected"
		return
	}
	idx := m.GetSelectedCommit()
	contentHeight := m.estimatedContentHeight()
	res := prformtab.OpenCreatePR(&m.prFormModal, m.appState.Repository, idx, m.bookmarkModal.GetJiraBookmarkTitles(), m.appState.DefaultBranch, ModalInnerWidth(m.width), contentHeight)
	if !res.Ok {
		m.appState.StatusMessage = res.StatusMessage
		return
	}
	m.beginModalUnderlay()
	m.appState.ViewMode = state.ViewCreatePR
	m.appState.StatusMessage = res.StatusMessage
	m.pushAIProfilesToFormModals()
}

// submitPR runs the PR creation command.
func (m *Model) submitPR() tea.Cmd {
	// Avoid duplicate CreatePRCmd (e.g. double mouse release or overlapping zone deliveries)
	// while a create is already in flight.
	if m.appState.ViewMode == state.ViewCreatePR && m.appState.Loading {
		return nil
	}
	res := prformtab.SubmitPR(&m.prFormModal, m.appState.Repository, m.appState.JJService, m.appState.GitHubService, m.appState.DemoMode)
	m.appState.StatusMessage = res.StatusMessage
	if res.Cmd == nil {
		return nil
	}
	m.appState.Loading = true
	return tea.Batch(res.Cmd, m.startBusySpinnerCmd())
}

// startCreateTicket opens the Create Ticket dialog when the provider supports it.
func (m *Model) startCreateTicket() {
	contentHeight := m.estimatedContentHeight()
	res := ticketformtab.OpenCreateTicket(&m.ticketFormModal, m.appState.TicketService, ModalInnerWidth(m.width), contentHeight)
	if !res.Ok {
		m.appState.StatusMessage = res.StatusMessage
		return
	}
	m.beginModalUnderlay()
	m.appState.ViewMode = state.ViewCreateTicket
	m.appState.StatusMessage = res.StatusMessage
	m.pushAIProfilesToFormModals()
}

// submitTicket runs the create-ticket command and closes the modal on success.
func (m *Model) submitTicket() tea.Cmd {
	res := ticketformtab.SubmitTicket(&m.ticketFormModal, m.appState.TicketService, m.appState.DemoMode)
	m.appState.StatusMessage = res.StatusMessage
	if res.Cmd == nil {
		return nil
	}
	m.appState.Loading = true
	return tea.Batch(res.Cmd, m.startBusySpinnerCmd())
}

// saveSettings builds params from settings tab and runs global save.
func (m *Model) saveSettings() tea.Cmd {
	ghOwner, ghRepo := "", ""
	if m.appState.GitHubService != nil {
		ghOwner = m.appState.GitHubService.GetOwner()
		ghRepo = m.appState.GitHubService.GetRepo()
	}
	return settingstab.SaveSettings(&m.settingsTabModel, ghOwner, ghRepo)
}

// saveSettingsLocal builds params and runs local save.
func (m *Model) saveSettingsLocal() tea.Cmd {
	ghOwner, ghRepo := "", ""
	if m.appState.GitHubService != nil {
		ghOwner = m.appState.GitHubService.GetOwner()
		ghRepo = m.appState.GitHubService.GetRepo()
	}
	return settingstab.SaveSettingsLocal(&m.settingsTabModel, ghOwner, ghRepo)
}

// confirmCleanup runs the cleanup command for the current confirming type.
func (m *Model) confirmCleanup() tea.Cmd {
	return settingstab.ConfirmCleanup(&m.settingsTabModel, m.appState.JJService, m.appState.Repository)
}

// handleClipboardCopiedMsg sets status (or error modal copied flag) from copy result; kept in main (generic).
func (m *Model) handleClipboardCopiedMsg(msg util.ClipboardCopiedMsg) (tea.Model, tea.Cmd) {
	if msg.Success {
		if m.appState.ViewMode == state.ViewGitHubLogin {
			m.appState.StatusMessage = "Code copied to clipboard! Paste it in your browser."
		} else if m.errorModal.GetError() != nil {
			m.errorModal.SetCopied(true)
			m.appState.StatusMessage = "Error copied to clipboard!"
		} else {
			m.appState.StatusMessage = "Copied to clipboard!"
		}
	} else {
		m.appState.StatusMessage = fmt.Sprintf("Failed to copy: %v", msg.Err)
	}
	return m, nil
}

// SetRepository sets the repository data and syncs to tab models (e.g. for tests)
func (m *Model) SetRepository(repo *internal.Repository) {
	m.appState.Repository = repo
	m.graphTabModel.UpdateRepository(repo)
	m.prsTabModel.UpdateRepository(repo)
	m.prsTabModel.SetGithubService(m.isGitHubAvailable())
	m.branchesTabModel.UpdateRepository(repo)
	m.ticketsTabModel.UpdateRepository(repo)
	m.settingsTabModel.UpdateRepository(repo)
	m.helpTabModel.UpdateRepository(repo)
}

// Init implements tea.Model
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		data.InitializeServices(m.appState.DemoMode),
		m.tickCmd(),
	)
}

// Update implements tea.Model.
// Message responsibility: see internal/tui/model/RESPONSIBILITY.md.
// Flow: globals (SetStatus, WindowSize) → state.NavigateMsg (from submodels) →
// modal request messages (descedit/bookmark/prform/warning forward to modals) →
// async result messages (data.*, graphtab.*, prstab.*, etc.) → zone/key routing.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Lock the spinner label across the rest of this Update invocation. Submodels
	// throughout the tree set m.appState.Loading directly and write progress text to
	// StatusMessage, but StatusMessage is *also* the footer; any later footer update
	// (background PR polls, "Loaded N tickets", "Ready") would otherwise overwrite the
	// spinner's caption mid-operation. By snapshotting at the false→true edge here we
	// keep the spinner text stable for the entire duration of the loading operation
	// without having to plumb a separate setter through every call site.
	defer m.snapshotSpinnerMessage()

	switch msg := msg.(type) {
	case SetStatusMsg:
		m.appState.StatusMessage = msg.Status
		return m, nil

	case spinner.TickMsg:
		if !m.appState.Loading && !m.aiGenOverlayActive {
			return m, nil
		}
		var spinCmd tea.Cmd
		m.busySpinner, spinCmd = m.busySpinner.Update(msg)
		return m, spinCmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.errorModal.SetWidth(m.width)
		m.errorModal.SetHeight(m.height)

		inputWidth := ModalInnerWidth(m.width)

		m.desceditModal.SetDimensions(inputWidth, max(m.height-24, 3))
		m.prFormModal.GetBodyInput().SetWidth(inputWidth)
		m.prFormModal.GetTitleInput().Width = inputWidth
		m.ticketFormModal.GetBodyInput().SetWidth(inputWidth)
		m.ticketFormModal.GetTitleInput().Width = inputWidth
		// PR form body uses full content height when in create-PR view
		contentHeight := m.estimatedContentHeight()
		if m.appState.ViewMode == state.ViewCreatePR {
			const fixedFormLines = 11
			bodyH := contentHeight - fixedFormLines
			if bodyH < 3 {
				bodyH = 3
			}
			m.prFormModal.GetBodyInput().SetHeight(bodyH)
		}
		if m.appState.ViewMode == state.ViewCreateTicket {
			const fixedFormLines = 12
			bodyH := contentHeight - fixedFormLines
			if bodyH < 3 {
				bodyH = 3
			}
			m.ticketFormModal.GetBodyInput().SetHeight(bodyH)
		}
		m.bookmarkModal.GetNameInput().Width = inputWidth
		m.bookmarkModal.SetContentWidth(inputWidth)

		m.settingsTabModel.SetInputWidths(min(max(m.width-24, 36), 76))

		if m.appState.ViewMode == state.ViewSettings {
			m.settingsTabModel.SetViewOpts(m.buildSettingsViewOpts())
		}

		// Propagate dimensions to tab models so they can render
		cmds := util.PropagateUpdate(msg, &m.graphTabModel, &m.prsTabModel, &m.branchesTabModel, &m.ticketsTabModel, &m.settingsTabModel, &m.helpTabModel)
		// Set content-area height on tabs so graph/files split fills the content area (not full window)
		for _, vm := range m.tabOrder {
			m.tabRegistry[vm].SetDimensions(m.width, contentHeight)
		}
		m.evologSplitModal = m.evologSplitModal.SetDimensions(m.width, m.height).WithSuggestConfig(m.appState.Config)
		m.fileDiffModal = m.fileDiffModal.SetDimensions(m.width, m.height)
		m.divergentModal = m.divergentModal.SetDimensions(m.width, m.height)
		m.conflictModal = m.conflictModal.SetDimensions(m.width, m.height)
		m.workspacesModal = m.workspacesModal.SetDimensions(m.width, m.height)
		if len(cmds) > 0 {
			return m, tea.Batch(cmds...)
		}
		return m, nil

	case tea.KeyMsg:
		// Window chrome keyboard nudge (Alt+arrow to move, Alt+Shift+arrow
		// to resize) is consumed before any modal/tab handling so the
		// keystroke can never collide with a textinput's own bindings —
		// HandleChromeKey only matches the alt/alt-shift arrow combos and
		// returns Consumed=false for everything else.
		key, content, title, closeCmd := m.chromedSlot()
		if consumed, cmd := m.chrome.Update(msg, content, title, key, m.width, m.height, closeCmd); consumed {
			return m, cmd
		}
		if m.evologDescribePreviewActive {
			switch msg.String() {
			case "y", "Y":
				pd, cd := m.evologDescribeParent, m.evologDescribeChild
				m.evologDescribePreviewActive = false
				m.evologDescribePreviewFromPlan = false
				skipP := m.evologDescribeSkipParent
				m.evologDescribeSkipParent = false
				m.evologDescribeParent, m.evologDescribeChild = "", ""
				m.appState.Loading = true
				m.appState.StatusMessage = "Applying descriptions…"
				return m, tea.Batch(
					aitab.ApplyEvologSplitDescriptionsCmd(0, m.appState.JJService, m.appState.Config, pd, cd, skipP),
					m.startBusySpinnerCmd(),
				)
			case "n", "N", "esc":
				m.evologDescribePreviewActive = false
				m.evologDescribePreviewFromPlan = false
				m.evologDescribeSkipParent = false
				m.evologDescribeParent, m.evologDescribeChild = "", ""
				m.appState.StatusMessage = "Descriptions preview dismissed"
				return m, nil
			default:
				return m, nil
			}
		}
		if m.absorbPreviewActive {
			switch msg.String() {
			case "y", "Y":
				m.absorbPreviewActive = false
				m.absorbPreviewSummary = ""
				m.redoOperationID = ""
				m.appState.Loading = true
				m.appState.StatusMessage = "Absorbing…"
				return m, tea.Batch(
					graphtab.AbsorbApplyCmd(m.appState.JJService),
					m.startBusySpinnerCmd(),
				)
			case "n", "N", "esc":
				m.absorbPreviewActive = false
				m.absorbPreviewSummary = ""
				m.appState.StatusMessage = "Absorb cancelled"
				return m, nil
			default:
				return m, nil
			}
		}
		// When an overlay or blocking modal is showing, route keys to handleKeyMsg (init, error, warning) or view modals.
		if m.initRepoModel.Path() != "" || m.errorModal.GetError() != nil || m.warningModal.IsShown() {
			return m.handleKeyMsg(msg)
		}
		// View-specific modals (divergent, bookmark conflict): route keys to handleKeyMsg so the modal gets them.
		if m.appState.ViewMode == state.ViewDivergentCommit || m.appState.ViewMode == state.ViewBookmarkConflict || m.appState.ViewMode == state.ViewEvologSplit || m.appState.ViewMode == state.ViewFileDiff || m.appState.ViewMode == state.ViewWorkspaces {
			return m.handleKeyMsg(msg)
		}
		// Esc in Settings: close in-tab overlays (theme picker, cleanup confirm) first; otherwise leave settings.
		if m.appState.ViewMode == state.ViewSettings && msg.String() == "esc" {
			if !m.escHandledInsideSettings() {
				return m.handleNavigate(state.NavigateTarget{Kind: state.NavigateBackToGraph, StatusMessage: "Settings cancelled"})
			}
		}
		// Delegate to the active tab behind the registry (tabs own selection
		// state). The adapters apply the per-tab command wrapper the inline
		// paths used (graph → wrapGraphTabCmd; prs/branches → wrapSpinnerStart)
		// so the returned cmd is already wrapped. Per-view control flow (Settings
		// always consumes; Help swallows tab/shift+tab; Tickets swallows the
		// esc that closed status-change mode) is preserved below; every other
		// view falls through to handleKeyMsg when the tab didn't consume the key.
		if t, ok := m.tabRegistry[m.appState.ViewMode]; ok {
			switch m.appState.ViewMode {
			case state.ViewTickets:
				wasStatusChange := m.isTicketsStatusChangeMode()
				if _, cmd := t.Update(msg, &m.appState); cmd != nil {
					return m, cmd
				}
				if msg.String() == "esc" && wasStatusChange && !m.isTicketsStatusChangeMode() {
					return m, nil
				}
			case state.ViewSettings:
				_, cmd := t.Update(msg, &m.appState)
				if cmd != nil {
					return m, cmd
				}
				return m, nil
			case state.ViewHelp:
				if _, cmd := t.Update(msg, &m.appState); cmd != nil {
					return m, cmd
				}
				// Tab/shift+tab switch help sub-tab; don't fall through to handleKeyMsg (which would switch to graph)
				if msg.String() == "tab" || msg.String() == "shift+tab" {
					return m, nil
				}
			default: // graph, prs, branches
				if _, cmd := t.Update(msg, &m.appState); cmd != nil {
					return m, cmd
				}
			}
		}
		return m.handleKeyMsg(msg)

	case tea.MouseMsg:
		// Window chrome (title-bar drag, [x] close, edge resize) gets first
		// look so a drag started on the tab keeps consuming subsequent
		// motion / release events even if they cross over an underlying
		// zone. We only swallow the event here when the chrome actually
		// engaged — otherwise (Consumed=false, Pop=false) we fall through
		// so the rest of the modal still receives normal clicks. Pop fires
		// when the user clicked [x]; chromedSlot supplies the modal-specific
		// Cancel / Dismiss navigation so close-via-tab and close-via-Esc go
		// through the same teardown path.
		key, content, title, closeCmd := m.chromedSlot()
		if consumed, cmd := m.chrome.Update(msg, content, title, key, m.width, m.height, closeCmd); consumed {
			// Track press/release pairing so a [x] close (which only consumes the
			// press) doesn't leak its release into underlay zones — see the
			// chromeConsumedPress field comment.
			switch msg.Action {
			case tea.MouseActionPress:
				m.chromeConsumedPress = true
			case tea.MouseActionRelease:
				m.chromeConsumedPress = false
			}
			return m, cmd
		}
		// Chrome did NOT consume this event. If it's the release half of a click
		// whose press chrome already claimed (typically the [x] close button),
		// drop it here so bubblezone doesn't dispatch it to whatever underlay
		// zone (e.g. the graph "split" button) happens to sit beneath the modal.
		if msg.Action == tea.MouseActionRelease && m.chromeConsumedPress {
			m.chromeConsumedPress = false
			return m, nil
		}
		// Minimized-chrome pass-through: chromed modal is collapsed to its
		// tab strip and the user clicked outside that strip — they're asking
		// to interact with the underlying view, not the dormant modal. Send
		// the raw MouseMsg to the underlay tab so wheel scrolls and hover
		// updates land, then on release also fan out to bubblezone so
		// underlay zones (commits / PR rows / etc) fire. The form-modal /
		// genmenu interceptions below are skipped because the modal body
		// isn't painted right now — its zones would either be stale or
		// missing from the latest Scan.
		if m.mouseTransparent(msg.X, msg.Y) {
			cmd, _ := m.routeMouseToUnderlay(msg)
			if msg.Action == tea.MouseActionRelease {
				_, zoneCmd := m.zoneManager.AnyInBoundsAndUpdate(m, msg)
				return m, tea.Batch(cmd, zoneCmd)
			}
			return m, cmd
		}
		// Generate-bearing form modals: forward the raw MouseMsg first so the
		// long-press AI profile picker can arm/cancel/hover/resolve. The modal
		// returns a non-nil cmd only when it actually captured the event (the
		// BeginPress tick, a menu-row release that fires NavigateGenerate*, etc.).
		if genState := m.activeFormModalGenMenu(); genState != nil {
			wasShown := genState.IsShown()
			if cmd := m.forwardMouseToActiveFormModal(msg); cmd != nil {
				return m, cmd
			}
			// When the popover was visible at the start of this event and is now
			// closed (a release that landed off the menu), suppress the normal
			// release-zone dispatch so the underlying generate-chip click does
			// not also fire after the user dismissed the popover.
			if wasShown && !genState.IsShown() && msg.Action == tea.MouseActionRelease {
				return m, nil
			}
			// When the popover is currently shown, swallow non-release mouse
			// events that didn't produce a cmd so they don't leak to the tab
			// layer (hover updates already happened inside the modal Update).
			if genState.IsShown() && msg.Action != tea.MouseActionRelease {
				return m, nil
			}
		}
		// Blocking overlays and modal views: run zone check on release first so clicks reach the modal, not the tab.
		if msg.Action == tea.MouseActionRelease &&
			(m.initRepoModel.Path() != "" || m.errorModal.GetError() != nil || m.warningModal.IsShown() ||
				m.appState.ViewMode == state.ViewCreatePR || m.appState.ViewMode == state.ViewCreateTicket || m.appState.ViewMode == state.ViewEditDescription || m.appState.ViewMode == state.ViewCreateBookmark || m.appState.ViewMode == state.ViewDivergentCommit || m.appState.ViewMode == state.ViewBookmarkConflict || m.appState.ViewMode == state.ViewEvologSplit || m.appState.ViewMode == state.ViewFileDiff) {
			return m.zoneManager.AnyInBoundsAndUpdate(m, msg)
		}
		// Handle wheel: IsWheel() covers standard encodings; also accept raw X11 4/5
		isWheel := tea.MouseEvent(msg).IsWheel() || msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown
		if isWheel {
			if m.appState.ViewMode == state.ViewEvologSplit {
				updated, cmd := m.evologSplitModal.Update(msg)
				m.evologSplitModal = updated
				return m, cmd
			}
			if m.appState.ViewMode == state.ViewFileDiff {
				updated, cmd := m.fileDiffModal.Update(msg)
				m.fileDiffModal = updated
				return m, cmd
			}
			// Wheel over a primary tab: every view sized its tab before
			// delegating, so do that generically through the registry (the
			// adapters apply the same command wrappers the inline paths used).
			contentHeight := m.estimatedContentHeight()
			if t, ok := m.tabRegistry[m.appState.ViewMode]; ok {
				t.SetDimensions(m.width, contentHeight)
				if _, cmd := t.Update(msg, &m.appState); cmd != nil {
					return m, cmd
				}
			}
			return m, nil
		}
		// Delegate other mouse to active tab (same as KeyMsg) for any other scroll/click handling.
		// The list tabs (prs/branches/tickets) size themselves first so scroll works even when the
		// wheel encoding wasn't recognized above; graph/settings/help delegate without a resize,
		// matching the pre-registry behavior exactly.
		contentHeight := m.estimatedContentHeight()
		if t, ok := m.tabRegistry[m.appState.ViewMode]; ok {
			switch m.appState.ViewMode {
			case state.ViewPullRequests, state.ViewBranches, state.ViewTickets:
				t.SetDimensions(m.width, contentHeight)
			}
			if _, cmd := t.Update(msg, &m.appState); cmd != nil {
				return m, cmd
			}
		}
		if msg.Action == tea.MouseActionRelease {
			return m.zoneManager.AnyInBoundsAndUpdate(m, msg)
		}
		return m, nil

	case genmenu.TickMsg:
		// Long-press tick for the AI profile picker on the active form modal.
		// The modal gates this internally (Owner + PressID checks) so we can
		// route it unconditionally; stale ticks become no-ops.
		if cmd := m.forwardGenMenuTickToActiveFormModal(msg); cmd != nil {
			return m, cmd
		}
		return m, nil

	case zone.MsgZoneInBounds:
		// Minimized-chrome pass-through: route the zone directly to the
		// underlay tab so commits / PR rows / branches / tickets clicks
		// reach the view the user can actually see behind the collapsed
		// modal. Without this branch the existing routing keys off
		// m.appState.ViewMode and forwards every zone to the dormant form
		// modal, which silently drops it.
		if m.mouseTransparent(msg.Event.X, msg.Event.Y) {
			if cmd, handled := m.routeMouseToUnderlay(msg); handled {
				return m, cmd
			}
			return m, nil
		}
		// Blocking overlays (init, error, warning) get zone clicks first so tabs don't consume them
		if m.initRepoModel.Path() != "" || m.errorModal.GetError() != nil || m.warningModal.IsShown() {
			return m.handleZoneClick(msg)
		}
		// View modals (divergent, conflict) get zone clicks so they're not consumed by the tab
		if m.appState.ViewMode == state.ViewDivergentCommit {
			updated, cmd := m.divergentModal.Update(msg)
			m.divergentModal = updated
			return m, cmd
		}
		if m.appState.ViewMode == state.ViewBookmarkConflict {
			updated, cmd := m.conflictModal.Update(msg)
			m.conflictModal = updated
			return m, cmd
		}
		if m.appState.ViewMode == state.ViewEvologSplit {
			updated, cmd := m.evologSplitModal.Update(msg)
			m.evologSplitModal = updated
			return m, cmd
		}
		if m.appState.ViewMode == state.ViewFileDiff {
			updated, cmd := m.fileDiffModal.Update(msg)
			m.fileDiffModal = updated
			return m, cmd
		}
		// Delegate to the active content tab (graph/prs/branches/tickets) so it
		// can return requests; settings/help are handled inside handleZoneClick.
		// Graph, PRs, Branches, and Tickets already receive zone.MsgZoneInBounds
		// here before handleZoneClick runs, so handleZoneClick must not re-Update
		// them (double-processing the release).
		switch m.appState.ViewMode {
		case state.ViewCommitGraph, state.ViewPullRequests, state.ViewBranches, state.ViewTickets:
			if t, ok := m.tabRegistry[m.appState.ViewMode]; ok {
				if _, cmd := t.Update(msg, &m.appState); cmd != nil {
					return m, cmd
				}
			}
		}
		return m.handleZoneClick(msg)

	case bubbledropdown.ItemChosenMsg, bubbledropdown.ItemCanceledMsg:
		// A settings dropdown emitted its selection/cancel via a deferred cmd.
		// Route it to the settings tab so the active sub-model applies and closes.
		if m.appState.ViewMode == state.ViewSettings {
			updated, cmd := m.settingsTabModel.Update(msg)
			m.settingsTabModel = updated
			return m, cmd
		}
		return m, nil

	case state.NavigateMsg:
		return m.handleNavigate(msg.Target)

	default:
		return m.dispatchAsyncMsg(msg)
	}
}

// isStaleFileDiffGlobalStatus reports status strings tied to the file-diff overlay that should not
// linger on the status line (or a concurrent loading overlay) after the modal closes.
func isStaleFileDiffGlobalStatus(msg string) bool {
	s := strings.TrimSpace(msg)
	if s == "File diff failed" {
		return true
	}
	return strings.HasPrefix(s, "File diff —")
}
