package model

import (
	"fmt"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/data"
	"github.com/madicen/jj-tui/internal/tui/state"
	bookmarktab "github.com/madicen/jj-tui/internal/tui/tabs/bookmark"
	branchestab "github.com/madicen/jj-tui/internal/tui/tabs/branches"
	descedittab "github.com/madicen/jj-tui/internal/tui/tabs/descedit"
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
//
// The NavigateMsg dispatcher and its per-domain handlers live in the
// navigate*.go sibling files (P2.7).

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

// propagateRepository pushes the single source of truth (m.appState.Repository)
// to the tabs that keep derived state or a render cache. It replaces the old
// per-site UpdateRepository fan-out (P2.8): tabs that never consumed the repository
// (tickets/settings/help/…) no longer implement or receive a repository hook, and
// the three that do (graph selection, prs selection, branches render cache) satisfy
// tab.RepositoryAware via OnRepositoryLoaded.
func (m *Model) propagateRepository() {
	repo := m.appState.Repository
	m.graphTabModel.OnRepositoryLoaded(repo)
	m.prsTabModel.OnRepositoryLoaded(repo)
	m.branchesTabModel.OnRepositoryLoaded(repo)
}

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
	m.propagateRepository()
	m.prsTabModel.SetGithubService(m.isGitHubAvailable())
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
