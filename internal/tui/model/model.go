package model

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/bubble-color-picker"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/data"
	"github.com/madicen/jj-tui/internal/tui/state"
	bookmarktab "github.com/madicen/jj-tui/internal/tui/tabs/bookmark"
	branchestab "github.com/madicen/jj-tui/internal/tui/tabs/branches"
	conflicttab "github.com/madicen/jj-tui/internal/tui/tabs/conflict"
	descedittab "github.com/madicen/jj-tui/internal/tui/tabs/descedit"
	divergenttab "github.com/madicen/jj-tui/internal/tui/tabs/divergent"
	errortab "github.com/madicen/jj-tui/internal/tui/tabs/error"
	githublogintab "github.com/madicen/jj-tui/internal/tui/tabs/githublogin"
	graphtab "github.com/madicen/jj-tui/internal/tui/tabs/graph"
	helptab "github.com/madicen/jj-tui/internal/tui/tabs/help"
	"github.com/madicen/jj-tui/internal/tui/tabs/help/commandhistory"
	initrepotab "github.com/madicen/jj-tui/internal/tui/tabs/initrepo"
	prformtab "github.com/madicen/jj-tui/internal/tui/tabs/prform"
	prstab "github.com/madicen/jj-tui/internal/tui/tabs/prs"
	settingstab "github.com/madicen/jj-tui/internal/tui/tabs/settings"
	ticketformtab "github.com/madicen/jj-tui/internal/tui/tabs/ticketform"
	ticketstab "github.com/madicen/jj-tui/internal/tui/tabs/tickets"
	warningtab "github.com/madicen/jj-tui/internal/tui/tabs/warning"
	"github.com/madicen/jj-tui/internal/tui/util"
)

// Model is the main TUI model using bubblezone for mouse handling.
// All clickable elements are wrapped with zone.Mark() in the View.
// Mouse events are handled via zone.MsgZoneInBounds messages.
type Model struct {
	ctx         context.Context
	zoneManager *zone.Manager
	appState    state.AppState // Shared state and services; submodels receive &appState

	// Dimensions (main only)
	width  int
	height int
	// Selection state lives in tab models: graph (commit/file), prs, tickets, branches
	redoOperationID string

	// Tab-specific models (own all tab/modal state; main model does not duplicate)
	graphTabModel    graphtab.GraphModel
	prsTabModel      prstab.Model
	branchesTabModel branchestab.Model
	ticketsTabModel  ticketstab.Model
	settingsTabModel settingstab.Model
	helpTabModel     helptab.Model

	// Modal models (dialogs and modals)
	initRepoModel    initrepotab.Model
	errorModal       errortab.Model
	warningModal     warningtab.Model
	conflictModal    conflicttab.Model
	divergentModal   divergenttab.Model
	bookmarkModal    bookmarktab.Model
	prFormModal      prformtab.Model
	ticketFormModal  ticketformtab.Model
	desceditModal    descedittab.Model
	githubLoginModel githublogintab.Model

	busySpinner spinner.Model
}

// doPollMsg is a message used to trigger a GitHub token poll.
type doPollMsg struct{}

// estimatedContentHeight returns height available for tab content (excluding header/status).
// Used in Update() when delegating to tabs so viewport/list dimensions are correct for scroll handling.
func (m *Model) estimatedContentHeight() int {
	return max(m.height-4, 1)
}

// buildSettingsViewOpts builds ViewOpts for the settings tab (used when entering settings or on resize).
func (m *Model) buildSettingsViewOpts() settingstab.ViewOpts {
	ticketName := ""
	if m.appState.TicketService != nil {
		ticketName = m.appState.TicketService.GetProviderName()
	}
	return settingstab.ViewOpts{
		GitHubAvailable:   m.isGitHubAvailable(),
		TicketServiceName: ticketName,
		Config:            m.appState.Config,
		ContentHeight:     m.estimatedContentHeight(),
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

// refreshRepository starts a refresh of the repository data.
func (m *Model) refreshRepository() tea.Cmd {
	m.appState.StatusMessage = "Refreshing..."
	var cmds []tea.Cmd
	if m.appState.JJService == nil {
		cmds = append(cmds, data.InitializeServices(m.appState.DemoMode))
	} else {
		cmds = append(cmds, data.LoadRepository(m.appState.JJService))
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
	if r.Checkout || r.Squash || r.Abandon || r.NewCommit || r.PerformRebase || r.ResolveDivergent != nil || r.CreateBookmark || r.DeleteBookmark || r.CreatePR || r.UpdatePR || r.MoveFileUp || r.MoveFileDown || r.RevertFile || r.MoveDeltaOntoOrigin {
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
	return m, m.settingsTabModel.EnterTab()
}

func (m *Model) handleNavigateToHelpTab() (tea.Model, tea.Cmd) {
	m.appState.ViewMode = state.ViewHelp
	m.helpTabModel.SetCommandHistoryEntries(helptab.BuildCommandHistoryEntries(m.appState.JJService))
	m.helpTabModel.SetSelectedCommand(0)
	m.appState.StatusMessage = "Loaded Help"
	return m, nil
}

func (m *Model) handleNavigateToBranchesTab() (tea.Model, tea.Cmd) {
	m.appState.ViewMode = state.ViewBranches
	status, cmd := branchestab.EnterTab(m)
	m.appState.StatusMessage = status
	return m, cmd
}

// handleNavigate performs view changes that only main can do (it owns modals and cross-tab state).
func (m *Model) handleNavigate(t state.NavigateTarget) (tea.Model, tea.Cmd) {
	if t.Kind == state.NavigateSaveDescription || t.Kind == state.NavigateSubmitBookmark || t.Kind == state.NavigateSubmitPR || t.Kind == state.NavigateSubmitTicket || t.Kind == state.NavigateResolveConflict || t.Kind == state.NavigateResolveDivergent || t.Kind == state.NavigateRunInit {
		m.redoOperationID = ""
	}
	switch t.Kind {
	case state.NavigateEditDescription:
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
		return m, branchestab.LoadBranchesCmd(m.appState.JJService, m.settingsTabModel.GetSettingsBranchLimit())
	case state.NavigateCreateBookmarkFromTicket:
		m.appState.ViewMode = state.ViewCreateBookmark
		m.appState.StatusMessage = bookmarktab.OpenCreateBookmarkFromTicket(&m.bookmarkModal, m.appState.Repository, t.TicketKey, t.TicketTitle, t.TicketDisplayKey, m.branchesTabModel.BuildBookmarkNameConflictSources(), m.appState.Config != nil && m.appState.Config.ShouldSanitizeBookmarkNames(), m.width-10)
		return m, nil
	case state.NavigateWarning:
		m.warningModal.Show(t.WarningTitle, t.WarningMessage, t.WarningCommits)
		return m, nil
	case state.NavigateCreatePR:
		m.startCreatePR()
		return m, nil
	case state.NavigateBackToGraph:
		m.appState.ViewMode = state.ViewCommitGraph
		if t.StatusMessage != "" {
			m.appState.StatusMessage = t.StatusMessage
		}
		return m, nil
	case state.NavigateBackToBranches:
		m.appState.ViewMode = state.ViewBranches
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
		m.appState.ViewMode = state.ViewCommitGraph
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
		m.appState.ViewMode = state.ViewSettings
		if t.StatusMessage != "" {
			m.appState.StatusMessage = t.StatusMessage
		}
		return m, nil
	case state.NavigateSaveDescription:
		if t.SaveCommitID != "" && m.appState.JJService != nil {
			return m, graphtab.SaveDescriptionCmd(m.appState.JJService, t.SaveCommitID, t.SaveDescription)
		}
		return m, nil
	case state.NavigateSubmitBookmark:
		if m.appState.JJService != nil {
			return m, m.submitBookmark()
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
	case state.NavigateWarningCancel:
		if t.StatusMessage != "" {
			m.appState.StatusMessage = t.StatusMessage
		}
		return m, nil
	case state.NavigateRunInit:
		m.appState.StatusMessage = "Initializing repository..."
		return m, data.RunJJInit()
	case state.NavigateDismissErrorAndRefresh:
		m.errorModal.ClearError()
		m.appState.ViewMode = state.ViewCommitGraph
		return m, m.refreshRepository()
	case state.NavigateBackFromPRForm:
		m.prFormModal.Hide()
		m.appState.ViewMode = state.ViewCommitGraph
		if t.StatusMessage != "" {
			m.appState.StatusMessage = t.StatusMessage
		}
		return m, nil
	case state.NavigateCreateTicket:
		m.startCreateTicket()
		return m, nil
	case state.NavigateBackFromTicketForm:
		m.ticketFormModal.Hide()
		m.appState.ViewMode = state.ViewTickets
		if t.StatusMessage != "" {
			m.appState.StatusMessage = t.StatusMessage
		}
		return m, nil
	case state.NavigateSubmitTicket:
		return m, m.submitTicket()
	default:
		return m, nil
	}
}

func (m *Model) handleUndo() (tea.Model, tea.Cmd) {
	if m.appState.JJService != nil {
		m.appState.StatusMessage = "Undoing..."
		return m, graphtab.UndoCmd(m.appState.JJService)
	}
	return m, nil
}

func (m *Model) handleRedo() (tea.Model, tea.Cmd) {
	if m.appState.JJService != nil && m.redoOperationID != "" {
		m.appState.StatusMessage = "Redoing..."
		return m, graphtab.RedoCmd(m.appState.JJService, m.redoOperationID)
	}
	return m, nil
}

func (m *Model) handleSelectCommit(index int) (tea.Model, tea.Cmd) {
	return m.processGraphRequest(graphtab.Request{SelectCommit: &index})
}

// startEditingDescription switches to description edit view and starts loading the description.
func (m *Model) startEditingDescription(commit internal.Commit) (tea.Model, tea.Cmd) {
	m.appState.ViewMode = state.ViewEditDescription
	m.desceditModal, m.appState.StatusMessage = descedittab.StartEditing(m.desceditModal, commit, m.width-10, m.height-12)
	return m, descedittab.LoadDescriptionCmd(m.appState.JJService, commit.ChangeID)
}

// startCreateBookmark opens the bookmark creation dialog for the selected commit.
func (m *Model) startCreateBookmark() {
	if !m.isSelectedCommitValid() {
		m.appState.StatusMessage = "No commit selected"
		return
	}
	idx := m.GetSelectedCommit()
	m.appState.ViewMode = state.ViewCreateBookmark
	m.appState.StatusMessage = bookmarktab.OpenCreateBookmark(&m.bookmarkModal, m.appState.Repository, idx, m.branchesTabModel.BuildBookmarkNameConflictSources(), m.appState.Config != nil && m.appState.Config.ShouldSanitizeBookmarkNames(), m.width-10)
}

// submitBookmark runs the bookmark create/move command.
func (m *Model) submitBookmark() tea.Cmd {
	cmd, status := bookmarktab.SubmitBookmark(&m.bookmarkModal, m.appState.Repository, m.appState.Config, m.appState.JJService)
	m.appState.StatusMessage = status
	return cmd
}

// startCreatePR opens the PR creation dialog for the selected commit's bookmark.
func (m *Model) startCreatePR() {
	if !m.isSelectedCommitValid() {
		m.appState.StatusMessage = "No commit selected"
		return
	}
	idx := m.GetSelectedCommit()
	contentHeight := m.estimatedContentHeight()
	res := prformtab.OpenCreatePR(&m.prFormModal, m.appState.Repository, idx, m.bookmarkModal.GetJiraBookmarkTitles(), m.width-10, contentHeight)
	if !res.Ok {
		m.appState.StatusMessage = res.StatusMessage
		return
	}
	m.appState.ViewMode = state.ViewCreatePR
	m.appState.StatusMessage = res.StatusMessage
}

// submitPR runs the PR creation command.
func (m *Model) submitPR() tea.Cmd {
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
	res := ticketformtab.OpenCreateTicket(&m.ticketFormModal, m.appState.TicketService, m.width-10, contentHeight)
	if !res.Ok {
		m.appState.StatusMessage = res.StatusMessage
		return
	}
	m.appState.ViewMode = state.ViewCreateTicket
	m.appState.StatusMessage = res.StatusMessage
}

// submitTicket runs the create-ticket command and closes the modal on success.
func (m *Model) submitTicket() tea.Cmd {
	res := ticketformtab.SubmitTicket(&m.ticketFormModal, m.appState.TicketService, m.appState.DemoMode)
	m.appState.StatusMessage = res.StatusMessage
	return res.Cmd
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
	switch msg := msg.(type) {
	case SetStatusMsg:
		m.appState.StatusMessage = msg.Status
		return m, nil

	case spinner.TickMsg:
		if !m.appState.Loading {
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

		// Resize text areas to fit new window width
		inputWidth := min(
			// Leave margin for borders/padding
			max(
				m.width-20, 30,
			),
			// Cap at reasonable max
			80,
		)

		m.desceditModal.SetDimensions(inputWidth, max(m.height-12, 3))
		m.prFormModal.GetBodyInput().SetWidth(inputWidth)
		m.prFormModal.GetTitleInput().Width = inputWidth
		m.ticketFormModal.GetBodyInput().SetWidth(inputWidth)
		m.ticketFormModal.GetTitleInput().Width = inputWidth
		// PR form body uses full content height when in create-PR view
		contentHeight := m.estimatedContentHeight()
		if m.appState.ViewMode == state.ViewCreatePR {
			const fixedFormLines = 9
			bodyH := contentHeight - fixedFormLines
			if bodyH < 3 {
				bodyH = 3
			}
			m.prFormModal.GetBodyInput().SetHeight(bodyH)
		}
		if m.appState.ViewMode == state.ViewCreateTicket {
			const fixedFormLines = 10
			bodyH := contentHeight - fixedFormLines
			if bodyH < 3 {
				bodyH = 3
			}
			m.ticketFormModal.GetBodyInput().SetHeight(bodyH)
		}
		m.bookmarkModal.GetNameInput().Width = inputWidth

		m.settingsTabModel.SetInputWidths(inputWidth - 10)

		if m.appState.ViewMode == state.ViewSettings {
			m.settingsTabModel.SetViewOpts(m.buildSettingsViewOpts())
		}

		// Propagate dimensions to tab models so they can render
		cmds := util.PropagateUpdate(msg, &m.graphTabModel, &m.prsTabModel, &m.branchesTabModel, &m.ticketsTabModel, &m.settingsTabModel, &m.helpTabModel)
		// Set content-area height on tabs so graph/files split fills the content area (not full window)
		m.graphTabModel.SetDimensions(m.width, contentHeight)
		m.prsTabModel.SetDimensions(m.width, contentHeight)
		m.branchesTabModel.SetDimensions(m.width, contentHeight)
		m.ticketsTabModel.SetDimensions(m.width, contentHeight)
		m.settingsTabModel.SetDimensions(m.width, contentHeight)
		m.helpTabModel.SetDimensions(m.width, contentHeight)
		if len(cmds) > 0 {
			return m, tea.Batch(cmds...)
		}
		return m, nil

	case tea.KeyMsg:
		// When an overlay or blocking modal is showing, route keys to handleKeyMsg (init, error, warning) or view modals.
		if m.initRepoModel.Path() != "" || m.errorModal.GetError() != nil || m.warningModal.IsShown() {
			return m.handleKeyMsg(msg)
		}
		// View-specific modals (divergent, bookmark conflict): route keys to handleKeyMsg so the modal gets them.
		if m.appState.ViewMode == state.ViewDivergentCommit || m.appState.ViewMode == state.ViewBookmarkConflict {
			return m.handleKeyMsg(msg)
		}
		// Esc in Settings: navigate back to graph (same as tab sending PerformCancelMsg).
		if m.appState.ViewMode == state.ViewSettings && msg.String() == "esc" {
			return m.handleNavigate(state.NavigateTarget{Kind: state.NavigateBackToGraph, StatusMessage: "Settings cancelled"})
		}
		// Delegate to tab models for their specific views (tabs own selection state)
		switch m.appState.ViewMode {
		case state.ViewCommitGraph:
			updated, cmd := m.graphTabModel.UpdateWithApp(msg, &m.appState)
			m.graphTabModel = updated
			if cmd != nil {
				return m, m.wrapGraphTabCmd(cmd)
			}
		case state.ViewPullRequests:
			updated, cmd := m.prsTabModel.UpdateWithApp(msg, &m.appState)
			m.prsTabModel = updated
			if cmd != nil {
				return m, cmd
			}
			// Fall through to handleKeyMsg for non-delegated keys
		case state.ViewBranches:
			updated, cmd := m.branchesTabModel.UpdateWithApp(msg, &m.appState)
			m.branchesTabModel = updated
			if cmd != nil {
				return m, m.wrapBranchFetchCmd(cmd)
			}
		case state.ViewTickets:
			updated, cmd := m.ticketsTabModel.UpdateWithApp(msg, &m.appState)
			m.ticketsTabModel = updated
			if cmd != nil {
				return m, cmd
			}
		case state.ViewSettings:
			cmds := util.PropagateUpdate(msg, &m.settingsTabModel)
			if len(cmds) > 0 && cmds[0] != nil {
				return m, cmds[0]
			}
			return m, nil
		case state.ViewHelp:
			cmds := util.PropagateUpdate(msg, &m.helpTabModel)
			if len(cmds) > 0 && cmds[0] != nil {
				return m, cmds[0]
			}
			// Tab/shift+tab switch help sub-tab; don't fall through to handleKeyMsg (which would switch to graph)
			if msg.String() == "tab" || msg.String() == "shift+tab" {
				return m, nil
			}
		}
		return m.handleKeyMsg(msg)

	case tea.MouseMsg:
		// Blocking overlays and modal views: run zone check on release first so clicks reach the modal, not the tab.
		if msg.Action == tea.MouseActionRelease &&
			(m.initRepoModel.Path() != "" || m.errorModal.GetError() != nil || m.warningModal.IsShown() ||
				m.appState.ViewMode == state.ViewCreatePR || m.appState.ViewMode == state.ViewCreateTicket || m.appState.ViewMode == state.ViewEditDescription || m.appState.ViewMode == state.ViewCreateBookmark || m.appState.ViewMode == state.ViewDivergentCommit || m.appState.ViewMode == state.ViewBookmarkConflict) {
			return m.zoneManager.AnyInBoundsAndUpdate(m, msg)
		}
		// Handle wheel: IsWheel() covers standard encodings; also accept raw X11 4/5
		isWheel := tea.MouseEvent(msg).IsWheel() || msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown
		if isWheel {
			contentHeight := m.estimatedContentHeight()
			switch m.appState.ViewMode {
			case state.ViewCommitGraph:
				m.graphTabModel.SetDimensions(m.width, contentHeight)
				updated, cmd := m.graphTabModel.UpdateWithApp(msg, &m.appState)
				m.graphTabModel = updated
				if cmd != nil {
					return m, m.wrapGraphTabCmd(cmd)
				}
			case state.ViewPullRequests:
				m.prsTabModel.SetDimensions(m.width, contentHeight)
				updated, cmd := m.prsTabModel.UpdateWithApp(msg, &m.appState)
				m.prsTabModel = updated
				if cmd != nil {
					return m, cmd
				}
			case state.ViewBranches:
				m.branchesTabModel.SetDimensions(m.width, contentHeight)
				cmds := util.PropagateUpdate(msg, &m.branchesTabModel)
				if len(cmds) > 0 && cmds[0] != nil {
					return m, cmds[0]
				}
			case state.ViewTickets:
				m.ticketsTabModel.SetDimensions(m.width, contentHeight)
				updated, cmd := m.ticketsTabModel.UpdateWithApp(msg, &m.appState)
				m.ticketsTabModel = updated
				if cmd != nil {
					return m, cmd
				}
			case state.ViewSettings:
				m.settingsTabModel.SetDimensions(m.width, contentHeight)
				cmds := util.PropagateUpdate(msg, &m.settingsTabModel)
				if len(cmds) > 0 && cmds[0] != nil {
					return m, cmds[0]
				}
			case state.ViewHelp:
				m.helpTabModel.SetDimensions(m.width, contentHeight)
				cmds := util.PropagateUpdate(msg, &m.helpTabModel)
				if len(cmds) > 0 && cmds[0] != nil {
					return m, cmds[0]
				}
			}
			return m, nil
		}
		// Delegate other mouse to active tab (same as KeyMsg) for any other scroll/click handling
		// Set dimensions for list tabs so wheel/scroll works even when isWheel wasn't true (e.g. terminal encoding)
		contentHeight := m.estimatedContentHeight()
		switch m.appState.ViewMode {
		case state.ViewCommitGraph:
			updated, cmd := m.graphTabModel.UpdateWithApp(msg, &m.appState)
			m.graphTabModel = updated
			if cmd != nil {
				return m, m.wrapGraphTabCmd(cmd)
			}
		case state.ViewPullRequests:
			m.prsTabModel.SetDimensions(m.width, contentHeight)
			updated, cmd := m.prsTabModel.UpdateWithApp(msg, &m.appState)
			m.prsTabModel = updated
			if cmd != nil {
				return m, cmd
			}
		case state.ViewBranches:
			m.branchesTabModel.SetDimensions(m.width, contentHeight)
			updated, cmd := m.branchesTabModel.UpdateWithApp(msg, &m.appState)
			m.branchesTabModel = updated
			if cmd != nil {
				return m, m.wrapBranchFetchCmd(cmd)
			}
		case state.ViewTickets:
			m.ticketsTabModel.SetDimensions(m.width, contentHeight)
			cmds := util.PropagateUpdate(msg, &m.ticketsTabModel)
			if len(cmds) > 0 && cmds[0] != nil {
				return m, cmds[0]
			}
		case state.ViewSettings:
			cmds := util.PropagateUpdate(msg, &m.settingsTabModel)
			if len(cmds) > 0 && cmds[0] != nil {
				return m, cmds[0]
			}
		case state.ViewHelp:
			cmds := util.PropagateUpdate(msg, &m.helpTabModel)
			if len(cmds) > 0 && cmds[0] != nil {
				return m, cmds[0]
			}
		}
		if msg.Action == tea.MouseActionRelease {
			return m.zoneManager.AnyInBoundsAndUpdate(m, msg)
		}
		return m, nil

	case zone.MsgZoneInBounds:
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
		// Delegate to tab when in that view so it can return requests
		if m.appState.ViewMode == state.ViewCommitGraph {
			updated, cmd := m.graphTabModel.UpdateWithApp(msg, &m.appState)
			m.graphTabModel = updated
			if cmd != nil {
				return m, m.wrapGraphTabCmd(cmd)
			}
		}
		if m.appState.ViewMode == state.ViewPullRequests {
			updated, cmd := m.prsTabModel.UpdateWithApp(msg, &m.appState)
			m.prsTabModel = updated
			if cmd != nil {
				return m, cmd
			}
		}
		if m.appState.ViewMode == state.ViewBranches {
			updated, cmd := m.branchesTabModel.UpdateWithApp(msg, &m.appState)
			m.branchesTabModel = updated
			if cmd != nil {
				return m, m.wrapBranchFetchCmd(cmd)
			}
		}
		if m.appState.ViewMode == state.ViewTickets {
			updated, cmd := m.ticketsTabModel.UpdateWithApp(msg, &m.appState)
			m.ticketsTabModel = updated
			if cmd != nil {
				return m, cmd
			}
		}
		return m.handleZoneClick(msg)

	case state.NavigateMsg:
		return m.handleNavigate(msg.Target)

	case commandhistory.Request:
		return m.handleHelpRequest(msg)

	case settingstab.Request:
		return m.handleSettingsRequest(msg)
	case settingstab.SaveSettingsEffect:
		return m, m.saveSettings()
	case settingstab.SaveSettingsLocalEffect:
		return m, m.saveSettingsLocal()
	case settingstab.PerformCancelMsg:
		return m.handleNavigate(state.NavigateTarget{Kind: state.NavigateBackToGraph, StatusMessage: "Settings cancelled"})

	case ticketstab.OpenURLEffect:
		return m, util.OpenURL(msg.URL)
	case ticketstab.ToggleModeEffect:
		mode := !m.ticketsTabModel.IsStatusChangeMode()
		m.ticketsTabModel.SetStatusChangeMode(mode)
		m.appState.StatusMessage = msg.Status
		return m, nil
	case ticketstab.OpenCreateBookmarkFromTicketEffect:
		return m.handleNavigate(state.NavigateTarget{
			Kind:             state.NavigateCreateBookmarkFromTicket,
			TicketKey:        msg.TicketKey,
			TicketTitle:      msg.Title,
			TicketDisplayKey: msg.DisplayKey,
		})

	case descedittab.SaveRequestedMsg, descedittab.CancelRequestedMsg:
		updated, cmd := m.desceditModal.Update(msg)
		m.desceditModal = updated
		return m, cmd

	case bookmarktab.CancelRequestedMsg, bookmarktab.SubmitRequestedMsg:
		updated, cmd := m.bookmarkModal.Update(msg)
		m.bookmarkModal = updated
		m.bookmarkModal.UpdateNameExistsFromInput(m.appState.Config != nil && m.appState.Config.ShouldSanitizeBookmarkNames())
		return m, cmd

	case prformtab.CancelRequestedMsg, prformtab.SubmitRequestedMsg:
		updated, cmd := m.prFormModal.Update(msg)
		m.prFormModal = updated
		return m, cmd

	case ticketformtab.CancelRequestedMsg, ticketformtab.SubmitRequestedMsg:
		updated, cmd := m.ticketFormModal.Update(msg)
		m.ticketFormModal = updated
		return m, cmd

	case settingstab.RequestConfirmCleanupMsg:
		return m, m.confirmCleanup()
	case settingstab.RequestCancelCleanupMsg:
		m.appState.StatusMessage = settingstab.CancelCleanupStatus
		return m, nil
	case settingstab.RequestSetStatusMsg:
		m.appState.StatusMessage = msg.Status
		return m, nil

	case errortab.RequestCopyMsg:
		if m.errorModal.GetError() != nil {
			m.errorModal.SetCopied(true)
			return m, util.CopyToClipboard(m.errorModal.GetError().Error())
		}
		return m, nil

	case warningtab.EditCommitRequestedMsg:
		updated, cmd := m.warningModal.Update(msg)
		m.warningModal = updated
		return m, cmd

	case graphtab.EditCompletedMsg:
		// Preserve PRs from previous repository
		var oldPRs []internal.GitHubPR
		if m.appState.Repository != nil {
			oldPRs = m.appState.Repository.PRs
		}
		m.appState.Repository = msg.Repository
		m.appState.Repository.PRs = oldPRs // Restore PRs temporarily
		// Push fresh graph into tab models before clearing loading so the overlay stays up until
		// the UI can render the new @ / tree (appState alone does not update GraphModel).
		m.graphTabModel.UpdateRepository(m.appState.Repository)
		m.prsTabModel.UpdateRepository(m.appState.Repository)
		m.prsTabModel.SetGithubService(m.isGitHubAvailable())
		m.branchesTabModel.UpdateRepository(m.appState.Repository)
		m.ticketsTabModel.UpdateRepository(m.appState.Repository)
		m.settingsTabModel.UpdateRepository(m.appState.Repository)
		m.helpTabModel.UpdateRepository(m.appState.Repository)
		// Don't clear error modal here - let errors persist until dismissed
		var workingChangeID string
		for i, commit := range msg.Repository.Graph.Commits {
			if commit.IsWorking {
				m.graphTabModel.SelectCommit(i)
				workingChangeID = commit.ChangeID
				break
			}
		}
		m.appState.Loading = false
		m.appState.StatusMessage = "Now editing working copy"

		var cmds []tea.Cmd
		cmds = append(cmds, m.tickCmd())
		if workingChangeID != "" && m.appState.JJService != nil {
			cmds = append(cmds, graphtab.LoadChangedFilesCmd(m.appState.JJService, workingChangeID))
		}

		// Also refresh PRs when GitHub is connected (needed for Update PR button)
		if m.appState.GitHubService != nil {
			existingPRs := 0
			if m.appState.Repository != nil {
				existingPRs = len(m.appState.Repository.PRs)
			}
			cmds = append(cmds, m.wrapFirstPRLoadCmd(prstab.LoadPRsCmd(m.appState.GitHubService, m.appState.GithubInfo, m.appState.DemoMode, existingPRs)))
		}

		return m, tea.Batch(cmds...)

	case errorMsg:
		cmd, info := errortab.HandleError(errortab.ErrorInput{NotJJRepo: msg.NotJJRepo, CurrentPath: msg.CurrentPath, Err: msg.Err}, &m.appState)
		if info != nil {
			if info.NotJJRepo {
				m.initRepoModel.SetPath(info.CurrentPath)
			} else {
				m.errorModal.SetError(info.Err, false, "")
			}
		}
		return m, cmd
	case data.InitErrorMsg:
		cmd, info := initrepotab.HandleInitError(msg, &m.appState)
		if info != nil {
			if info.NotJJRepo {
				m.initRepoModel.SetPath(info.CurrentPath)
			} else {
				m.errorModal.SetError(info.Err, false, "")
			}
		}
		return m, cmd
	case data.JJInitSuccessMsg:
		m.initRepoModel.SetPath("")
		m.errorModal.SetError(nil, false, "")
		return m, initrepotab.HandleJJInitSuccess(msg, &m.appState)
	case data.RepoReadyMsg:
		return m.handleRepoReadyMsg(msg)
	case data.AuxServicesReadyMsg:
		return m.handleAuxServicesReadyMsg(msg)
	case data.ServicesInitializedMsg:
		return m.handleDataServicesInitializedMsg(msg)
	case data.RepositoryLoadedMsg:
		return m.handleDataRepositoryLoadedMsg(msg)
	case graphtab.RepositoryLoadedMsg:
		return m.handleActionsRepositoryLoadedMsg(msg)
	case data.SilentRepositoryLoadedMsg:
		return m.handleDataSilentRepositoryLoadedMsg(msg)

	case prstab.PrsLoadedMsg:
		m.appState.PRsLoadedOnce = true
		m.appState.Loading = false
		updated, cmd := m.prsTabModel.UpdateWithApp(msg, &m.appState)
		m.prsTabModel = updated
		m.prsTabModel.UpdateRepository(m.appState.Repository)
		return m, cmd
	case prstab.PrMergedMsg, prstab.PrClosedMsg:
		updated, cmd := m.prsTabModel.UpdateWithApp(msg, &m.appState)
		m.prsTabModel = updated
		var err error
		switch mmsg := msg.(type) {
		case prstab.PrMergedMsg:
			err = mmsg.Err
		case prstab.PrClosedMsg:
			err = mmsg.Err
		}
		if err != nil {
			m.errorModal.SetError(err, false, "")
			return m, nil
		}
		return m, cmd
	case prstab.LoadErrorMsg:
		m.appState.PRsLoadedOnce = true
		m.appState.Loading = false
		updated, _ := m.prsTabModel.UpdateWithApp(msg, &m.appState)
		m.prsTabModel = updated
		m.errorModal.SetError(msg.Err, false, "")
		return m, nil
	case prstab.ReauthNeededMsg:
		updated, _ := m.prsTabModel.UpdateWithApp(msg, &m.appState)
		m.prsTabModel = updated
		return m.handleReauthNeededEffect(prstab.ApplyReauthNeededEffect(msg))
	case prstab.PrTickMsg:
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
		updated, cmd := m.prsTabModel.UpdateWithApp(prInput, &m.appState)
		m.prsTabModel = updated
		return m, cmd

	case ticketstab.TicketsLoadedMsg:
		m.appState.TicketsLoadedOnce = true
		m.appState.Loading = false
		input := ticketstab.TicketsLoadedInput{
			Tickets:      msg.Tickets,
			ProviderName: "",
			HasService:   m.appState.TicketService != nil,
			CanCreate:    m.appState.TicketService != nil && m.appState.TicketService.CanCreateTicket(),
		}
		if m.appState.TicketService != nil {
			input.ProviderName = m.appState.TicketService.GetProviderName()
		}
		updated, cmd := m.ticketsTabModel.UpdateWithApp(input, &m.appState)
		m.ticketsTabModel = updated
		return m, cmd
	case ticketstab.TransitionsLoadedMsg:
		updated, cmd := m.ticketsTabModel.UpdateWithApp(msg, &m.appState)
		m.ticketsTabModel = updated
		return m, cmd
	case ticketstab.TransitionCompletedMsg:
		updated, cmd := m.ticketsTabModel.UpdateWithApp(msg, &m.appState)
		m.ticketsTabModel = updated
		if msg.Err != nil {
			m.errorModal.SetError(msg.Err, false, "")
			return m, nil
		}
		return m, cmd
	case ticketstab.LoadErrorMsg:
		m.appState.TicketsLoadedOnce = true
		m.appState.Loading = false
		updated, _ := m.ticketsTabModel.UpdateWithApp(msg, &m.appState)
		m.ticketsTabModel = updated
		m.errorModal.SetError(msg.Err, false, "")
		m.appState.StatusMessage = fmt.Sprintf("Error: %v", msg.Err)
		return m, nil

	case branchestab.BranchesLoadedMsg:
		input := branchestab.BranchesLoadedInput{
			BranchesLoadedMsg:    msg,
			InCreateBookmarkView: m.appState.ViewMode == state.ViewCreateBookmark,
			HasError:             m.errorModal.GetError() != nil,
		}
		updated, cmd := m.branchesTabModel.UpdateWithApp(input, &m.appState)
		m.branchesTabModel = updated
		if input.InCreateBookmarkView {
			m.bookmarkModal.SetNameConflictSources(m.branchesTabModel.BuildBookmarkNameConflictSources())
			m.bookmarkModal.UpdateNameExistsFromInput(m.appState.Config != nil && m.appState.Config.ShouldSanitizeBookmarkNames())
		}
		return m, cmd
	case branchestab.BranchActionMsg:
		updated, _ := m.branchesTabModel.UpdateWithApp(msg, &m.appState)
		m.branchesTabModel = updated
		if msg.Action == "fetch" {
			m.appState.BranchRemoteFetchPending = false
			if msg.Err != nil {
				m.appState.Loading = false
			}
		}
		if msg.Err != nil {
			m.errorModal.SetError(msg.Err, false, "")
			return m, nil
		}
		return m, tea.Batch(
			branchestab.LoadBranchesCmd(m.appState.JJService, m.settingsTabModel.GetSettingsBranchLimit()),
			data.LoadRepository(m.appState.JJService),
		)

	case settingstab.SettingsSavedMsg:
		wasSettings := m.appState.ViewMode == state.ViewSettings
		cmd, errInfo := settingstab.HandleSettingsSavedMsg(msg, &m.appState)
		if errInfo != nil {
			m.errorModal.SetError(errInfo.Err, false, "")
			return m, nil
		}
		if wasSettings {
			m.settingsTabModel.SetViewOpts(m.buildSettingsViewOpts())
		}
		return m, cmd

	case settingstab.GitHubDeviceFlowStartedMsg:
		m.githubLoginModel.SetDeviceFlow(msg.DeviceCode, msg.UserCode, msg.VerificationURL, msg.Interval)
		m.appState.ViewMode = state.ViewGitHubLogin
		m.appState.StatusMessage = "Waiting for GitHub authorization..."
		// Do not auto-open the browser; user can press Enter or click "Copy Code & Open Browser" on the login screen.
		return m, settingstab.PollGitHubTokenCmd(m.githubLoginModel.GetDeviceCode())

	case settingstab.GitHubLoginPollMsg:
		if m.githubLoginModel.GetPolling() {
			if msg.Interval > 0 {
				m.githubLoginModel.SetPollInterval(m.githubLoginModel.GetPollInterval() + msg.Interval)
			}
			return m, tea.Tick(time.Duration(m.githubLoginModel.GetPollInterval())*time.Second, func(t time.Time) tea.Msg {
				return doPollMsg{}
			})
		}
		return m, nil

	case doPollMsg:
		if m.githubLoginModel.GetPolling() {
			return m, settingstab.PollGitHubTokenCmd(m.githubLoginModel.GetDeviceCode())
		}
		return m, nil

	case settingstab.GitHubLoginSuccessMsg:
		m.githubLoginModel.ClearFlow()
		m.appState.ViewMode = state.ViewSettings
		m.settingsTabModel.SetViewOpts(m.buildSettingsViewOpts())
		m.appState.StatusMessage = "GitHub login successful!"
		cfg, _ := config.Load()
		cfg.SetGitHubToken(msg.Token, config.GitHubAuthDeviceFlow)
		_ = cfg.Save()
		_ = os.Setenv("GITHUB_TOKEN", msg.Token)
		m.settingsTabModel.SetSettingInputValue(0, msg.Token)
		return m, data.InitializeServices(m.appState.DemoMode)

	case settingstab.GitHubLoginErrorMsg:
		m.githubLoginModel.ClearFlow()
		m.appState.ViewMode = state.ViewSettings
		m.appState.StatusMessage = fmt.Sprintf("GitHub login error: %v", msg.Err)
		m.errorModal.SetError(msg.Err, false, "")
		return m, nil

	case prformtab.PRCreatedMsg:
		return m, prformtab.HandlePRCreatedMsg(prformtab.PRCreatedInput{PRCreatedMsg: msg, DemoMode: m.appState.DemoMode}, &m.appState)
	case ticketformtab.TicketCreatedMsg:
		m.ticketFormModal.Hide()
		m.appState.ViewMode = state.ViewTickets
		if msg.Ticket != nil {
			m.appState.StatusMessage = fmt.Sprintf("Created %s: %s", msg.Ticket.DisplayKey, msg.Ticket.Summary)
			cmd := ticketformtab.HandleTicketCreatedMsg(msg.Ticket, m.appState.TicketService, m.appState.DemoMode)
			if cmd != nil {
				return m, tea.Batch(cmd, ticketstab.LoadTicketsCmd(m.appState.TicketService, m.appState.DemoMode))
			}
			return m, ticketstab.LoadTicketsCmd(m.appState.TicketService, m.appState.DemoMode)
		}
		return m, ticketstab.LoadTicketsCmd(m.appState.TicketService, m.appState.DemoMode)
	case prstab.BranchPushedMsg:
		return m, branchestab.HandleBranchPushedMsg(msg, &m.appState)
	case bookmarktab.BookmarkCreatedMsg:
		return m, bookmarktab.HandleBookmarkCreatedMsg(msg, &m.appState)
	case bookmarktab.BookmarkDeletedMsg:
		return m, branchestab.HandleBookmarkDeletedMsg(msg, &m.appState)
	case branchestab.BookmarkConflictInfoMsg:
		cmd, info := conflicttab.HandleBookmarkConflictInfoMsg(msg, &m.appState)
		if info != nil {
			m.conflictModal.Show(info.BookmarkName, info.LocalID, info.RemoteID, info.LocalSummary, info.RemoteSummary)
			m.appState.ViewMode = state.ViewBookmarkConflict
		}
		return m, cmd
	case conflicttab.BookmarkConflictResolvedMsg:
		return m, conflicttab.HandleBookmarkConflictResolvedMsg(msg, &m.appState, m.settingsTabModel.GetSettingsBranchLimit())
	case graphtab.DivergentCommitInfoMsg:
		cmd, info := divergenttab.HandleDivergentCommitInfoMsg(msg, &m.appState)
		if info != nil {
			m.divergentModal.Show(info.ChangeID, info.CommitIDs, info.Summaries)
			m.appState.ViewMode = state.ViewDivergentCommit
		}
		return m, cmd
	case divergenttab.DivergentCommitResolvedMsg:
		return m, divergenttab.HandleDivergentCommitResolvedMsg(msg, &m.appState)
	case graphtab.FileMoveCompletedMsg:
		graphtab.HandleFileMoveCompletedMsg(graphtab.FileMoveInput{
			FileMoveCompletedMsg: msg,
			ChangedFilesCommitID: m.graphTabModel.GetChangedFilesCommitID(),
		}, &m.appState)
		m.graphTabModel.UpdateRepository(m.appState.Repository)
		if m.appState.Repository != nil {
			for i, commit := range m.appState.Repository.Graph.Commits {
				if commit.ChangeID == m.graphTabModel.GetChangedFilesCommitID() {
					m.graphTabModel.SelectCommit(i)
					break
				}
			}
			// Load changed files for the currently selected commit so the files pane updates.
			idx := m.graphTabModel.GetSelectedCommit()
			commits := m.appState.Repository.Graph.Commits
			if idx >= 0 && idx < len(commits) && m.appState.JJService != nil {
				return m, graphtab.LoadChangedFilesCmd(m.appState.JJService, commits[idx].ChangeID)
			}
		}
		return m, nil
	case graphtab.FileRevertedMsg:
		graphtab.HandleFileRevertedMsg(graphtab.FileRevertedInput{
			FileRevertedMsg:      msg,
			ChangedFilesCommitID: m.graphTabModel.GetChangedFilesCommitID(),
		}, &m.appState)
		m.graphTabModel.UpdateRepository(m.appState.Repository)
		if m.appState.Repository != nil {
			for i, commit := range m.appState.Repository.Graph.Commits {
				if commit.ChangeID == m.graphTabModel.GetChangedFilesCommitID() {
					m.graphTabModel.SelectCommit(i)
					break
				}
			}
			idx := m.graphTabModel.GetSelectedCommit()
			commits := m.appState.Repository.Graph.Commits
			if idx >= 0 && idx < len(commits) && m.appState.JJService != nil {
				return m, graphtab.LoadChangedFilesCmd(m.appState.JJService, commits[idx].ChangeID)
			}
		}
		return m, nil
	case descedittab.DescriptionSavedMsg:
		cmd := descedittab.HandleDescriptionSavedMsg(msg, &m.appState)
		m.desceditModal.Hide()
		return m, cmd
	case descedittab.DescriptionLoadedMsg:
		if m.appState.ViewMode != state.ViewEditDescription || m.desceditModal.GetEditingCommitID() != msg.CommitID {
			return m, nil
		}
		finalDesc := descedittab.SuggestDescriptionForLoad(descedittab.DescriptionLoadedInput{
			CommitID:       msg.CommitID,
			Description:    msg.Description,
			Repository:     m.appState.Repository,
			CommitIdx:      commitIdxForChangeID(m.appState.Repository, msg.CommitID),
			TicketKeys:     m.bookmarkModal.GetTicketBookmarkDisplayKeys(),
			FindBookmarkFn: bookmarktab.FindBookmarkForCommit,
		})
		if finalDesc == "" {
			finalDesc = msg.Description
			if finalDesc == "(no description)" {
				finalDesc = ""
			}
		}
		m.desceditModal.SetDescription(finalDesc)
		m.appState.StatusMessage = "Editing description (Ctrl+S to save, Esc to cancel)"
		return m, nil
	case util.ClipboardCopiedMsg:
		return m.handleClipboardCopiedMsg(msg)
	case settingstab.CleanupCompletedMsg:
		return m, settingstab.HandleCleanupCompletedMsg(msg, &m.appState)

	case graphtab.ChangedFilesLoadedMsg:
		updated, cmd := m.graphTabModel.Update(msg)
		if g, ok := updated.(*graphtab.GraphModel); ok {
			m.graphTabModel = *g
		}
		return m, cmd
	case loadChangedFilesTriggerMsg:
		if m.appState.JJService != nil && m.appState.Repository != nil {
			commits := m.appState.Repository.Graph.Commits
			idx := m.graphTabModel.GetSelectedCommit()
			if idx >= 0 && idx < len(commits) {
				return m, graphtab.LoadChangedFilesCmd(m.appState.JJService, commits[idx].ChangeID)
			}
		}
		return m, nil
	case tickMsg:
		return m.handleTickMsg()
	case graphtab.UndoCompletedMsg:
		cmd, errInfo := graphtab.HandleUndoCompletedMsg(msg, &m.appState)
		if errInfo != nil {
			m.errorModal.SetError(errInfo.Err, false, "")
			return m, nil
		}
		if msg.Message == "Undo completed" {
			m.redoOperationID = msg.RedoOpID
		} else {
			m.redoOperationID = ""
		}
		return m, cmd

	// Handle our custom messages
	case TabSelectedMsg:
		m.appState.ViewMode = msg.Tab
		if msg.Tab == state.ViewSettings {
			m.settingsTabModel.SetViewOpts(m.buildSettingsViewOpts())
		}
		if msg.Tab == state.ViewHelp {
			m.helpTabModel.SetCommandHistoryEntries(helptab.BuildCommandHistoryEntries(m.appState.JJService))
		}
		return m, nil

	// Theme color picker: close picker and update color when user confirms or cancels
	case bubblepicker.ColorChosenMsg, bubblepicker.ColorCanceledMsg:
		if m.appState.ViewMode == state.ViewSettings {
			cmds := util.PropagateUpdate(msg, &m.settingsTabModel)
			if len(cmds) > 0 && cmds[0] != nil {
				return m, cmds[0]
			}
		}
		return m, nil

	case ActionMsg:
		return m.handleAction(msg.Action)

	// Handle messages from actions package
	case util.ErrorMsg:
		return m.Update(errorMsg{Err: msg.Err})
	}

	return m, nil
}
