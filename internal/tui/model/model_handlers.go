package model

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	aitab "github.com/madicen/jj-tui/internal/tui/ai"
	"github.com/madicen/jj-tui/internal/tui/data"
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

// model_handlers.go holds the *Model methods that drive tab-specific behavior
// (navigation, request processing, form submit/save, undo/redo, etc.). These
// call into the concrete tab packages; housing them here keeps model.go's own
// imports of those packages construction-time-only (see init.go / tab_adapters.go).

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
