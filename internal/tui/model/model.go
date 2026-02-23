package model

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/github"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tickets"
	"github.com/madicen/jj-tui/internal/tui/actions"
	bookmarktab "github.com/madicen/jj-tui/internal/tui/tabs/bookmark"
	branchestab "github.com/madicen/jj-tui/internal/tui/tabs/branches"
	conflicttab "github.com/madicen/jj-tui/internal/tui/tabs/conflict"
	divergenttab "github.com/madicen/jj-tui/internal/tui/tabs/divergent"
	errortab "github.com/madicen/jj-tui/internal/tui/tabs/error"
	graphtab "github.com/madicen/jj-tui/internal/tui/tabs/graph"
	helptab "github.com/madicen/jj-tui/internal/tui/tabs/help"
	prformtab "github.com/madicen/jj-tui/internal/tui/tabs/prform"
	prstab "github.com/madicen/jj-tui/internal/tui/tabs/prs"
	settingstab "github.com/madicen/jj-tui/internal/tui/tabs/settings"
	ticketstab "github.com/madicen/jj-tui/internal/tui/tabs/tickets"
	warningtab "github.com/madicen/jj-tui/internal/tui/tabs/warning"
)

// Model is the main TUI model using bubblezone for mouse handling.
// All clickable elements are wrapped with zone.Mark() in the View.
// Mouse events are handled via zone.MsgZoneInBounds messages.
type Model struct {
	ctx           context.Context
	zoneManager   *zone.Manager
	jjService     *jj.Service
	githubService *github.Service
	ticketService tickets.Service // Generic ticket service (Jira, Codecks, etc.)
	repository    *internal.Repository
	demoMode      bool // When true, uses mock services for screenshots/testing

	// UI state
	viewMode ViewMode
	width    int
	height   int
	// Selection state lives in tab models: graph (commit/file), prs, tickets, branches
	statusMessage string
	err           error
	loading       bool
	githubInfo    string // Diagnostic info about GitHub connection

	notJJRepo   bool   // true if error is "not a jj repository"
	currentPath          string // path where we're running (for jj init)
	errorCopied          bool   // true if error was just copied to clipboard

	// Warning modal state (for empty commit descriptions, etc.)
	showWarningModal   bool              // true if warning modal is displayed
	warningTitle       string            // title for warning modal
	warningMessage     string            // message for warning modal
	warningCommits     []internal.Commit // commits with issues (for display)
	warningSelectedIdx int               // selected commit index in warning modal

	graphFocused bool // True if graph viewport has focus, false if files viewport (graph tab only)

	// Branch state (selection in branches tab)
	branchList []internal.Branch

	// Tab-specific models (own all tab/modal state; main model does not duplicate)
	graphTabModel    graphtab.GraphModel
	prsTabModel      prstab.Model
	branchesTabModel branchestab.Model
	ticketsTabModel  ticketstab.Model
	settingsTabModel settingstab.Model
	helpTabModel     helptab.Model

	// Modal models (dialogs and modals)
	errorModal     errortab.Model
	warningModal   warningtab.Model
	conflictModal  conflicttab.Model
	divergentModal divergenttab.Model
	bookmarkModal  bookmarktab.Model
	prFormModal    prformtab.Model
}

// doPollMsg is a message used to trigger a GitHub token poll.
type doPollMsg struct{}

// estimatedContentHeight returns height available for tab content (excluding header/status).
// Used in Update() when delegating to tabs so viewport/list dimensions are correct for scroll handling.
func (m *Model) estimatedContentHeight() int {
	return max(m.height-4, 1)
}

// SetRepository sets the repository data and syncs to tab models (e.g. for tests)
func (m *Model) SetRepository(repo *internal.Repository) {
	m.repository = repo
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
	// Initialize jj service, load repository, and start auto-refresh timer
	return tea.Batch(
		m.initializeServices(),
		m.tickCmd(),
	)
}

// Update implements tea.Model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Resize text areas to fit new window width
		inputWidth := min(
			// Leave margin for borders/padding
			max(
				m.width-20, 30,
			),
			// Cap at reasonable max
			80,
		)

		m.graphTabModel.GetDescriptionInput().SetWidth(inputWidth)
		m.prFormModal.GetBodyInput().SetWidth(inputWidth)
		m.prFormModal.GetTitleInput().Width = inputWidth
		m.bookmarkModal.GetNameInput().Width = inputWidth

		m.settingsTabModel.SetInputWidths(inputWidth - 10)

		// Propagate dimensions to tab models so they can render
		cmds := PropagateUpdate(msg, &m.graphTabModel, &m.prsTabModel, &m.branchesTabModel, &m.ticketsTabModel, &m.settingsTabModel, &m.helpTabModel)
		if len(cmds) > 0 {
			return m, tea.Batch(cmds...)
		}
		return m, nil

	case tea.KeyMsg:
		// Esc in Settings: handle in main model so a single Update() returns to graph (tab would return Request cmd that needs a second Update).
		if m.viewMode == ViewSettings && msg.String() == "esc" {
			return m.handleSettingsCancel()
		}
		// Delegate to tab models for their specific views (tabs own selection state)
		switch m.viewMode {
		case ViewCommitGraph:
			cmds := PropagateUpdate(msg, &m.graphTabModel)
			// Graph tab returns requests (LoadChangedFiles, Checkout, etc.) as cmds; run them
			if len(cmds) > 0 && cmds[0] != nil {
				return m, tea.Batch(cmds...)
			}
		case ViewPullRequests:
			cmds := PropagateUpdate(msg, &m.prsTabModel)
			if len(cmds) > 0 && cmds[0] != nil {
				return m, cmds[0]
			}
			// Fall through to handleKeyMsg for non-delegated keys
		case ViewBranches:
			cmds := PropagateUpdate(msg, &m.branchesTabModel)
			if len(cmds) > 0 && cmds[0] != nil {
				return m, cmds[0]
			}
		case ViewTickets:
			cmds := PropagateUpdate(msg, &m.ticketsTabModel)
			if len(cmds) > 0 && cmds[0] != nil {
				return m, cmds[0]
			}
		case ViewSettings:
			cmds := PropagateUpdate(msg, &m.settingsTabModel)
			if len(cmds) > 0 && cmds[0] != nil {
				return m, cmds[0]
			}
		case ViewHelp:
			cmds := PropagateUpdate(msg, &m.helpTabModel)
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
		// Modal views: run zone check on release first so form clicks aren't consumed by the delegate switch.
		if (m.viewMode == ViewCreatePR || m.viewMode == ViewEditDescription || m.viewMode == ViewCreateBookmark) &&
			msg.Action == tea.MouseActionRelease {
			return m.zoneManager.AnyInBoundsAndUpdate(m, msg)
		}
		// Handle wheel: IsWheel() covers standard encodings; also accept raw X11 4/5
		isWheel := tea.MouseEvent(msg).IsWheel() || msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown
		if isWheel {
			contentHeight := m.estimatedContentHeight()
			switch m.viewMode {
			case ViewCommitGraph:
				m.graphTabModel.SetDimensions(m.width, contentHeight)
				cmds := PropagateUpdate(msg, &m.graphTabModel)
				if len(cmds) > 0 && cmds[0] != nil {
					return m, cmds[0]
				}
			case ViewPullRequests:
				m.prsTabModel.SetDimensions(m.width, contentHeight)
				cmds := PropagateUpdate(msg, &m.prsTabModel)
				if len(cmds) > 0 && cmds[0] != nil {
					return m, cmds[0]
				}
			case ViewBranches:
				m.branchesTabModel.SetDimensions(m.width, contentHeight)
				cmds := PropagateUpdate(msg, &m.branchesTabModel)
				if len(cmds) > 0 && cmds[0] != nil {
					return m, cmds[0]
				}
			case ViewTickets:
				m.ticketsTabModel.SetDimensions(m.width, contentHeight)
				cmds := PropagateUpdate(msg, &m.ticketsTabModel)
				if len(cmds) > 0 && cmds[0] != nil {
					return m, cmds[0]
				}
			case ViewSettings:
				m.settingsTabModel.SetDimensions(m.width, contentHeight)
				cmds := PropagateUpdate(msg, &m.settingsTabModel)
				if len(cmds) > 0 && cmds[0] != nil {
					return m, cmds[0]
				}
			case ViewHelp:
				m.helpTabModel.SetDimensions(m.width, contentHeight)
				cmds := PropagateUpdate(msg, &m.helpTabModel)
				if len(cmds) > 0 && cmds[0] != nil {
					return m, cmds[0]
				}
			}
			return m, nil
		}
		// Delegate other mouse to active tab (same as KeyMsg) for any other scroll/click handling
		// Set dimensions for list tabs so wheel/scroll works even when isWheel wasn't true (e.g. terminal encoding)
		contentHeight := m.estimatedContentHeight()
		switch m.viewMode {
		case ViewCommitGraph:
			cmds := PropagateUpdate(msg, &m.graphTabModel)
			if len(cmds) > 0 && cmds[0] != nil {
				return m, cmds[0]
			}
		case ViewPullRequests:
			m.prsTabModel.SetDimensions(m.width, contentHeight)
			cmds := PropagateUpdate(msg, &m.prsTabModel)
			if len(cmds) > 0 && cmds[0] != nil {
				return m, cmds[0]
			}
		case ViewBranches:
			m.branchesTabModel.SetDimensions(m.width, contentHeight)
			cmds := PropagateUpdate(msg, &m.branchesTabModel)
			if len(cmds) > 0 && cmds[0] != nil {
				return m, cmds[0]
			}
		case ViewTickets:
			m.ticketsTabModel.SetDimensions(m.width, contentHeight)
			cmds := PropagateUpdate(msg, &m.ticketsTabModel)
			if len(cmds) > 0 && cmds[0] != nil {
				return m, cmds[0]
			}
		case ViewSettings:
			cmds := PropagateUpdate(msg, &m.settingsTabModel)
			if len(cmds) > 0 && cmds[0] != nil {
				return m, cmds[0]
			}
		case ViewHelp:
			cmds := PropagateUpdate(msg, &m.helpTabModel)
			if len(cmds) > 0 && cmds[0] != nil {
				return m, cmds[0]
			}
		}
		if msg.Action == tea.MouseActionRelease {
			return m.zoneManager.AnyInBoundsAndUpdate(m, msg)
		}
		return m, nil

	case zone.MsgZoneInBounds:
		// Delegate to tab when in that view so it can return requests
		if m.viewMode == ViewCommitGraph {
			cmds := PropagateUpdate(msg, &m.graphTabModel)
			if len(cmds) > 0 && cmds[0] != nil {
				return m, tea.Batch(cmds...)
			}
		}
		if m.viewMode == ViewPullRequests {
			cmds := PropagateUpdate(msg, &m.prsTabModel)
			if len(cmds) > 0 && cmds[0] != nil {
				return m, cmds[0]
			}
		}
		if m.viewMode == ViewBranches {
			cmds := PropagateUpdate(msg, &m.branchesTabModel)
			if len(cmds) > 0 && cmds[0] != nil {
				return m, cmds[0]
			}
		}
		if m.viewMode == ViewTickets {
			cmds := PropagateUpdate(msg, &m.ticketsTabModel)
			if len(cmds) > 0 && cmds[0] != nil {
				return m, cmds[0]
			}
		}
		return m.handleZoneClick(msg)

	case graphtab.Request:
		return m.handleGraphRequest(msg)

	case prstab.Request:
		return m.handlePRsRequest(msg)

	case branchestab.Request:
		return m.handleBranchesRequest(msg)

	case helptab.Request:
		return m.handleHelpRequest(msg)

	case settingstab.Request:
		return m.handleSettingsRequest(msg)

	case ticketstab.Request:
		return m.handleTicketsRequest(msg)

	case repositoryLoadedMsg:
		// Preserve PRs from previous repository
		var oldPRs []internal.GitHubPR
		if m.repository != nil {
			oldPRs = m.repository.PRs
		}
		m.repository = msg.repository
		m.repository.PRs = oldPRs // Restore PRs temporarily
		m.loading = false
		// Don't clear m.err here - let errors persist until dismissed
		if m.jjService == nil {
			// First load - set the service
			jjSvc, _ := jj.NewService("")
			m.jjService = jjSvc
		}
		m.statusMessage = fmt.Sprintf("Loaded %d commits", len(msg.repository.Graph.Commits))

		// Sync repository to tab models (graph tab preserves selection by ChangeID in UpdateRepository)
		m.graphTabModel.UpdateRepository(m.repository)
		m.prsTabModel.UpdateRepository(m.repository)
		m.prsTabModel.SetGithubService(m.isGitHubAvailable())
		m.branchesTabModel.UpdateRepository(m.repository)
		m.ticketsTabModel.UpdateRepository(m.repository)
		m.settingsTabModel.UpdateRepository(m.repository)
		m.helpTabModel.UpdateRepository(m.repository)

		// Build commands to run
		var cmds []tea.Cmd
		cmds = append(cmds, m.tickCmd())

		if m.githubService != nil {
			cmds = append(cmds, m.loadPRs())
		}

		// Load changed files for the selected commit (initial or re-resolved by UpdateRepository).
		// Always call SelectCommit so changedFilesCommitID is set; otherwise when the async load
		// completes, SetChangedFiles will reject the result (commitID != changedFilesCommitID).
		commits := msg.repository.Graph.Commits
		if len(commits) > 0 {
			idx := m.graphTabModel.GetSelectedCommit()
			if idx < 0 {
				idx = 0
			}
			m.graphTabModel.SelectCommit(idx)
			cmds = append(cmds, m.loadChangedFiles(commits[idx].ChangeID))
		}
		return m, tea.Batch(cmds...)

	case editCompletedMsg:
		// Preserve PRs from previous repository
		var oldPRs []internal.GitHubPR
		if m.repository != nil {
			oldPRs = m.repository.PRs
		}
		m.repository = msg.repository
		m.repository.PRs = oldPRs // Restore PRs temporarily
		m.loading = false
		// Don't clear m.err here - let errors persist until dismissed
		// Find and select the working copy commit (graph tab owns selection)
		for i, commit := range msg.repository.Graph.Commits {
			if commit.IsWorking {
				m.graphTabModel.SelectCommit(i)
				break
			}
		}
		m.statusMessage = "Now editing working copy"

		// Build commands to run
		var cmds []tea.Cmd
		cmds = append(cmds, m.tickCmd())

		// Also refresh PRs when GitHub is connected (needed for Update PR button)
		if m.githubService != nil {
			cmds = append(cmds, m.loadPRs())
		}

		return m, tea.Batch(cmds...)

	case silentRepositoryLoadedMsg:
		// Background refresh - update data without changing status
		if msg.repository != nil {
			oldCount := 0
			var oldPRs []internal.GitHubPR
			if m.repository != nil {
				oldCount = len(m.repository.Graph.Commits)
				oldPRs = m.repository.PRs // Preserve PRs from previous load
			}
			m.repository = msg.repository
			m.repository.PRs = oldPRs // Restore PRs
			// Sync repository to tab models (graph tab preserves selection in UpdateRepository)
			m.graphTabModel.UpdateRepository(m.repository)
			m.prsTabModel.UpdateRepository(m.repository)
			m.prsTabModel.SetGithubService(m.isGitHubAvailable())
			m.branchesTabModel.UpdateRepository(m.repository)
			m.ticketsTabModel.UpdateRepository(m.repository)
			m.settingsTabModel.UpdateRepository(m.repository)
			m.helpTabModel.UpdateRepository(m.repository)
			// Don't clear m.err here - let errors persist until dismissed
			// Only update status if commit count changed AND there's no existing error
			newCount := len(msg.repository.Graph.Commits)
			if newCount != oldCount && m.err == nil {
				m.statusMessage = fmt.Sprintf("Updated: %d commits", newCount)
			}

		}
		return m, nil

	case errorMsg:
		m.err = msg.Err
		m.notJJRepo = msg.NotJJRepo
		m.currentPath = msg.CurrentPath
		m.loading = false
		// Use friendly message for welcome screen, error message for real errors
		if msg.NotJJRepo {
			m.statusMessage = "Press 'i' to initialize a repository"
		} else {
			m.statusMessage = fmt.Sprintf("Error: %v", msg.Err)
		}
		// Don't continue auto-refresh on error - let user dismiss or manually refresh
		return m, nil

	case jjInitSuccessMsg:
		m.notJJRepo = false
		// Don't clear m.err here - let errors persist until user dismisses them
		m.statusMessage = "Repository initialized! Loading..."
		return m, m.initializeServices()

	case servicesInitializedMsg:
		m.jjService = msg.jjService
		m.githubService = msg.githubService
		m.ticketService = msg.ticketService
		m.repository = msg.repository
		m.githubInfo = msg.githubInfo // Store diagnostic info
		m.demoMode = msg.demoMode     // Set demo mode from message
		m.loading = false
		// Don't clear m.err here - let errors persist until user dismisses them
		m.statusMessage = fmt.Sprintf("Loaded %d commits", len(msg.repository.Graph.Commits))
		if m.demoMode {
			m.statusMessage += " (demo mode)"
		} else if m.githubService != nil {
			m.statusMessage += " (GitHub connected)"
		} else if msg.githubInfo != "" {
			// Show brief info when GitHub isn't connected
			m.statusMessage += fmt.Sprintf(" (GitHub: %s)", msg.githubInfo)
		}
		if m.ticketService != nil {
			m.statusMessage += fmt.Sprintf(" (%s connected)", m.ticketService.GetProviderName())
		} else if msg.ticketError != nil {
			m.statusMessage += fmt.Sprintf(" (Tickets error: %v)", msg.ticketError)
		}

		// Build commands to run after initialization
		var cmds []tea.Cmd
		cmds = append(cmds, m.tickCmd())

		// Load PRs on startup if GitHub is connected or in demo mode
		// Also start PR auto-refresh timer
		if m.isGitHubAvailable() {
			cmds = append(cmds, m.loadPRs())
			if prTickCmd := m.prTickCmd(); prTickCmd != nil {
				cmds = append(cmds, prTickCmd)
			}
		}

		// Auto-select first commit if none selected and load its changed files
		if m.graphTabModel.GetSelectedCommit() < 0 && len(msg.repository.Graph.Commits) > 0 {
			m.graphTabModel.SelectCommit(0)
			commit := msg.repository.Graph.Commits[0]
			cmds = append(cmds, m.loadChangedFiles(commit.ChangeID))
		}

		return m, tea.Batch(cmds...)

	case prsLoadedMsg:
		// nil prs signals to keep existing PRs (used in demo mode)
		if msg.prs == nil {
			if m.repository != nil && m.err == nil {
				m.statusMessage = fmt.Sprintf("PRs: %d", len(m.repository.PRs))
			}
			return m, nil
		}
		if m.repository != nil {
			m.repository.PRs = msg.prs
			// Only update status if there's no existing error
			if m.err == nil {
				m.statusMessage = fmt.Sprintf("Loaded %d PRs", len(msg.prs))
			}
			// Sync to PR tab so it can auto-select first PR when list loads
			m.prsTabModel.UpdateRepository(m.repository)
		} else if m.err == nil {
			m.statusMessage = fmt.Sprintf("Loaded %d PRs (warning: repository is nil)", len(msg.prs))
		}
		return m, nil

	case prMergedMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to merge PR #%d: %v", msg.prNumber, msg.err)
			m.err = msg.err
		} else {
			m.statusMessage = fmt.Sprintf("Merged PR #%d", msg.prNumber)
			// Reload PRs to update status
			return m, m.loadPRs()
		}
		return m, nil

	case prClosedMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to close PR #%d: %v", msg.prNumber, msg.err)
			m.err = msg.err
		} else {
			m.statusMessage = fmt.Sprintf("Closed PR #%d", msg.prNumber)
			// Reload PRs to update status
			return m, m.loadPRs()
		}
		return m, nil

	case ticketsLoadedMsg:
		m.ticketsTabModel.UpdateTickets(msg.tickets)
		providerName := ""
		if m.ticketService != nil {
			providerName = m.ticketService.GetProviderName()
		}
		m.ticketsTabModel.SetTicketServiceInfo(providerName, m.ticketService != nil)
		// Only update status if there's no existing error
		if m.err == nil {
			pName := "tickets"
			if m.ticketService != nil {
				pName = providerName + " tickets"
			}
			m.statusMessage = fmt.Sprintf("Loaded %d %s", len(msg.tickets), pName)
		}
		// Load transitions for the selected ticket (tab owns transition state)
		m.ticketsTabModel.SetAvailableTransitions(nil)
		m.ticketsTabModel.SetLoadingTransitions(true)
		return m, m.loadTransitions()

	case transitionsLoadedMsg:
		m.ticketsTabModel.SetLoadingTransitions(false)
		m.ticketsTabModel.SetAvailableTransitions(msg.transitions)
		return m, nil

	case transitionCompletedMsg:
		m.ticketsTabModel.SetTransitionInProgress(false)
		m.ticketsTabModel.SetStatusChangeMode(false)
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to transition %s: %v", msg.ticketKey, msg.err)
			m.err = msg.err
		} else if msg.newStatus != "" {
			m.statusMessage = fmt.Sprintf("Ticket %s transitioned to %s", msg.ticketKey, msg.newStatus)
			// Reload tickets to get updated status
			return m, m.loadTickets()
		}

	case branchesLoadedMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to load branches: %v", msg.err)
		} else {
			m.branchList = msg.branches
			m.branchesTabModel.UpdateBranches(msg.branches)
			if m.err == nil && m.viewMode != ViewCreateBookmark {
				m.statusMessage = fmt.Sprintf("Loaded %d branches", len(msg.branches))
			}
			// Re-check bookmark name duplicate if we're in bookmark creation view
			if m.viewMode == ViewCreateBookmark {
				m.updateBookmarkNameExists()
			}
		}
		return m, nil

	case branchActionMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to %s branch: %v", msg.action, msg.err)
			m.err = msg.err
		} else {
			switch msg.action {
			case "track":
				m.statusMessage = fmt.Sprintf("Now tracking branch %s", msg.branch)
			case "untrack":
				m.statusMessage = fmt.Sprintf("Stopped tracking branch %s", msg.branch)
			case "restore":
				m.statusMessage = fmt.Sprintf("Restored local branch %s", msg.branch)
			case "delete":
				m.statusMessage = fmt.Sprintf("Deleted bookmark %s", msg.branch)
			case "push":
				m.statusMessage = fmt.Sprintf("Pushed branch %s to remote", msg.branch)
			case "fetch":
				m.statusMessage = "Fetched from all remotes"
			}
			// Reload branches and repository (to see new commits in graph)
			return m, tea.Batch(m.loadBranches(), m.loadRepository())
		}
		return m, nil

	case settingsSavedMsg:
		m.viewMode = ViewCommitGraph
		m.ticketService = msg.ticketService

		// Handle save error
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error saving settings: %v", msg.err)
			m.err = msg.err
			return m, nil
		}

		var status []string
		if msg.githubConnected {
			status = append(status, "GitHub")
		}
		if msg.ticketProvider != "" {
			status = append(status, msg.ticketProvider)
		}

		saveLocation := "globally"
		if msg.savedLocal {
			saveLocation = "to .jj-tui.json (local)"
		}

		if len(status) > 0 {
			m.statusMessage = fmt.Sprintf("Settings saved %s. Connected: %s", saveLocation, strings.Join(status, ", "))
		} else {
			m.statusMessage = fmt.Sprintf("Settings saved %s", saveLocation)
		}
		// Reinitialize services with new credentials
		return m, m.initializeServices()

	case githubDeviceFlowStartedMsg:
		m.settingsTabModel.SetGitHubDeviceCode(msg.deviceCode)
		m.settingsTabModel.SetGitHubUserCode(msg.userCode)
		m.settingsTabModel.SetGitHubVerificationURL(msg.verificationURL)
		m.settingsTabModel.SetGitHubLoginPolling(true)
		m.settingsTabModel.SetGitHubPollInterval(msg.interval)
		m.viewMode = ViewGitHubLogin
		m.statusMessage = "Waiting for GitHub authorization..."
		// Start polling for the token
		return m, tea.Batch(
			openURL(msg.verificationURL),
			m.pollGitHubToken(), // Start first poll immediately
		)

	case githubLoginPollMsg:
		if m.settingsTabModel.GetGitHubLoginPolling() {
			if msg.interval > 0 {
				m.settingsTabModel.SetGitHubPollInterval(m.settingsTabModel.GetGitHubPollInterval() + msg.interval)
			}
			return m, tea.Tick(time.Duration(m.settingsTabModel.GetGitHubPollInterval())*time.Second, func(t time.Time) tea.Msg {
				return doPollMsg{}
			})
		}
		return m, nil

	case doPollMsg:
		if m.settingsTabModel.GetGitHubLoginPolling() {
			return m, m.pollGitHubToken()
		}
		return m, nil

	case githubLoginSuccessMsg:
		m.settingsTabModel.SetGitHubLoginPolling(false)
		m.settingsTabModel.SetGitHubDeviceCode("")
		m.settingsTabModel.SetGitHubUserCode("")
		m.settingsTabModel.SetGitHubPollInterval(0)
		m.viewMode = ViewSettings
		m.statusMessage = "GitHub login successful!"
		cfg, _ := config.Load()
		cfg.SetGitHubToken(msg.token, config.GitHubAuthDeviceFlow)
		_ = cfg.Save()
		_ = os.Setenv("GITHUB_TOKEN", msg.token)
		m.settingsTabModel.SetSettingInputValue(0, msg.token)
		return m, m.initializeServices()

	case githubReauthNeededMsg:
		// GitHub authentication expired - prompt for reauthorization
		m.statusMessage = msg.reason
		// Clear the old token and start a new Device Flow
		cfg, _ := config.Load()
		if cfg != nil {
			cfg.ClearGitHub()
			_ = cfg.Save()
		}
		// Clear env var too
		_ = os.Unsetenv("GITHUB_TOKEN")
		m.githubService = nil
		// Start the Device Flow login automatically
		return m, m.startGitHubLogin()

	case prCreatedMsg:
		m.viewMode = ViewCommitGraph
		m.statusMessage = fmt.Sprintf("PR #%d created: %s", msg.pr.Number, msg.pr.Title)

		if m.demoMode {
			// In demo mode, add the PR to the list directly without opening browser
			if m.repository != nil {
				m.repository.PRs = append([]internal.GitHubPR{*msg.pr}, m.repository.PRs...)
			}
			return m, nil
		}

		// Open the PR in browser and refresh PR list
		return m, tea.Batch(openURL(msg.pr.URL), m.loadPRs())

	case branchPushedMsg:
		m.statusMessage = fmt.Sprintf("Pushed %s to remote", msg.branch)
		// Reload repository and PRs to show updated state
		return m, tea.Batch(m.loadRepository(), m.loadPRs())

	case bookmarkCreatedOnCommitMsg:
		m.viewMode = ViewCommitGraph
		if msg.wasMoved {
			m.statusMessage = fmt.Sprintf("Bookmark '%s' moved", msg.bookmarkName)
		} else {
			m.statusMessage = fmt.Sprintf("Bookmark '%s' created", msg.bookmarkName)
		}
		// Trigger auto-transition to "In Progress" if enabled and created from a ticket
		if msg.ticketKey != "" && m.ticketService != nil {
			cfg, _ := config.Load()
			if cfg != nil && cfg.AutoInProgressOnBranch() {
				return m, tea.Batch(m.loadRepository(), m.transitionTicketToInProgress(msg.ticketKey))
			}
		}
		return m, m.loadRepository()

	case bookmarkDeletedMsg:
		m.viewMode = ViewCommitGraph
		m.statusMessage = fmt.Sprintf("Bookmark '%s' deleted", msg.bookmarkName)
		// Reload repository to update the view
		return m, tea.Batch(m.loadRepository(), m.loadPRs())

	case bookmarkConflictInfoMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error loading conflict info: %v", msg.err)
			m.viewMode = ViewBranches
			return m, nil
		}
		m.conflictModal.Show(msg.bookmarkName, msg.localID, msg.remoteID, msg.localSummary, msg.remoteSummary)
		m.viewMode = ViewBookmarkConflict
		return m, nil

	case bookmarkConflictResolvedMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error resolving conflict: %v", msg.err)
		} else {
			resolutionDesc := "kept local version"
			if msg.resolution == "reset_remote" {
				resolutionDesc = "reset to remote"
			}
			m.statusMessage = fmt.Sprintf("Bookmark '%s' conflict resolved (%s)", msg.bookmarkName, resolutionDesc)
		}
		m.viewMode = ViewBranches
		// Reload branches to reflect the change
		return m, tea.Batch(m.loadRepository(), m.loadBranches())

	case divergentCommitInfoMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error loading divergent info: %v", msg.err)
			m.viewMode = ViewCommitGraph
			return m, nil
		}
		m.divergentModal.Show(msg.changeID, msg.commitIDs, msg.summaries)
		m.viewMode = ViewDivergentCommit
		return m, nil

	case divergentCommitResolvedMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error resolving divergent commit: %v", msg.err)
		} else {
			m.statusMessage = fmt.Sprintf("Divergent commit resolved (kept %s)", msg.keptCommitID)
		}
		m.viewMode = ViewCommitGraph
		// Reload repository to reflect the change
		return m, m.loadRepository()

	case fileMoveCompletedMsg:
		originalCommitID := m.graphTabModel.GetChangedFilesCommitID()

		if m.repository != nil {
			oldPRs := m.repository.PRs
			m.repository = msg.repository
			m.repository.PRs = oldPRs
		} else {
			m.repository = msg.repository
		}
		directionText := "new parent commit"
		if msg.direction == "down" {
			directionText = "new child commit"
		}
		m.statusMessage = fmt.Sprintf("Moved %s to %s", msg.filePath, directionText)

		m.graphTabModel.UpdateRepository(m.repository)
		for i, commit := range msg.repository.Graph.Commits {
			if commit.ChangeID == originalCommitID {
				m.graphTabModel.SelectCommit(i)
				break
			}
		}

		var cmds []tea.Cmd
		cmds = append(cmds, m.loadRepository())
		if originalCommitID != "" {
			cmds = append(cmds, m.loadChangedFiles(originalCommitID))
		}
		return m, tea.Batch(cmds...)

	case fileRevertedMsg:
		originalCommitID := m.graphTabModel.GetChangedFilesCommitID()

		if m.repository != nil {
			oldPRs := m.repository.PRs
			m.repository = msg.repository
			m.repository.PRs = oldPRs
		} else {
			m.repository = msg.repository
		}
		m.statusMessage = fmt.Sprintf("Reverted changes to %s", msg.filePath)

		m.graphTabModel.UpdateRepository(m.repository)
		for i, commit := range msg.repository.Graph.Commits {
			if commit.ChangeID == originalCommitID {
				m.graphTabModel.SelectCommit(i)
				break
			}
		}

		var cmds []tea.Cmd
		cmds = append(cmds, m.loadRepository())
		if originalCommitID != "" {
			cmds = append(cmds, m.loadChangedFiles(originalCommitID))
		}
		return m, tea.Batch(cmds...)

	case changedFilesLoadedMsg:
		m.graphTabModel.SetChangedFiles(msg.files, msg.commitID)
		return m, nil

	case tickMsg:
		// Stop auto-refresh if there's an error - let user handle it
		if m.err != nil {
			return m, nil
		}
		var cmds []tea.Cmd
		// When on graph view, ensure changed files are loaded for the selected commit (covers initial load / late-arriving result)
		if m.viewMode == ViewCommitGraph && m.repository != nil && m.jjService != nil {
			commits := m.repository.Graph.Commits
			idx := m.graphTabModel.GetSelectedCommit()
			if idx >= 0 && idx < len(commits) {
				wantCommitID := commits[idx].ChangeID
				if m.graphTabModel.GetChangedFilesCommitID() != wantCommitID {
					cmds = append(cmds, m.loadChangedFiles(wantCommitID))
				}
			}
		}
		// Auto-refresh: reload repository data silently (but not while editing, creating PR, creating bookmark, or selecting rebase destination)
		if !m.loading && m.jjService != nil && m.viewMode != ViewEditDescription && m.viewMode != ViewCreatePR && m.viewMode != ViewCreateBookmark && !m.graphTabModel.IsInRebaseMode() {
			cmds = append(cmds, m.loadRepositorySilent())
		}
		cmds = append(cmds, m.tickCmd())
		return m, tea.Batch(cmds...)

	case prTickMsg:
		// PR auto-refresh: only refresh when GitHub is connected and viewing PR tab
		if m.err != nil || m.githubService == nil {
			return m, nil
		}
		var cmds []tea.Cmd
		// Only actually refresh PRs if we're on the PR tab
		if m.viewMode == ViewPullRequests && !m.loading {
			cmds = append(cmds, m.loadPRs())
		}
		// Always restart the timer
		if prTickCmd := m.prTickCmd(); prTickCmd != nil {
			cmds = append(cmds, prTickCmd)
		}
		if len(cmds) > 0 {
			return m, tea.Batch(cmds...)
		}
		return m, nil

	case descriptionSavedMsg:
		// Description saved successfully, go back to graph view
		m.viewMode = ViewCommitGraph
		m.graphTabModel.SetEditingCommitID("")
		m.statusMessage = fmt.Sprintf("Description updated for %s", msg.commitID)
		// Reload repository to show updated description
		return m, m.loadRepository()

	case descriptionLoadedMsg:
		// Full description loaded, populate the textarea
		if m.viewMode == ViewEditDescription && m.graphTabModel.GetEditingCommitID() == msg.commitID {
			description := msg.description
			if description == "(no description)" {
				description = ""
			}

			// If description is empty, check if commit has a ticket-associated bookmark
			// and prepopulate with the short ID
			if description == "" && m.repository != nil {
				commitIdx := -1
				for i, commit := range m.repository.Graph.Commits {
					if commit.ChangeID == msg.commitID {
						commitIdx = i
						break
					}
				}

				if commitIdx >= 0 {
					commit := m.repository.Graph.Commits[commitIdx]
					ticketKeys := m.bookmarkModal.GetTicketBookmarkDisplayKeys()
					var foundShortID string
					for _, branch := range commit.Branches {
						if shortID, ok := ticketKeys[branch]; ok {
							foundShortID = shortID
							break
						}
					}

					if foundShortID == "" {
						ancestorBookmark := m.findBookmarkForCommit(commitIdx)
						if ancestorBookmark != "" {
							if shortID, ok := ticketKeys[ancestorBookmark]; ok {
								foundShortID = shortID
							}
						}
					}

					if foundShortID != "" {
						description = foundShortID + " "
					}
				}
			}

			descInput := m.graphTabModel.GetDescriptionInput()
			descInput.SetValue(description)
			descInput.Focus()
			m.statusMessage = "Editing description (Ctrl+S to save, Esc to cancel)"
		}
		return m, nil

	// Handle our custom messages
	case TabSelectedMsg:
		m.viewMode = msg.Tab
		return m, nil

	case ActionMsg:
		return m.handleAction(msg.Action)

	// Handle messages from actions package
	case actions.RepositoryLoadedMsg:
		return m.Update(repositoryLoadedMsg{repository: msg.Repository})

	case actions.EditCompletedMsg:
		return m.Update(editCompletedMsg{repository: msg.Repository})

	case actions.ErrorMsg:
		return m.Update(errorMsg{Err: msg.Err})

	case actions.DescriptionLoadedMsg:
		return m.Update(descriptionLoadedMsg{commitID: msg.CommitID, description: msg.Description})

	case actions.DescriptionSavedMsg:
		return m.Update(descriptionSavedMsg{commitID: msg.CommitID})

	case actions.PRCreatedMsg:
		return m.Update(prCreatedMsg{pr: msg.PR})

	case actions.BranchPushedMsg:
		return m.Update(branchPushedMsg{branch: msg.Branch, pushOutput: msg.PushOutput})

	case actions.BookmarkCreatedMsg:
		return m.Update(bookmarkCreatedOnCommitMsg{bookmarkName: msg.BookmarkName, commitID: msg.CommitID, wasMoved: msg.WasMoved, ticketKey: msg.TicketKey})

	case actions.BookmarkDeletedMsg:
		return m.Update(bookmarkDeletedMsg{bookmarkName: msg.BookmarkName})

	case actions.FileMoveCompletedMsg:
		return m.Update(fileMoveCompletedMsg{repository: msg.Repository, filePath: msg.FilePath, direction: msg.Direction})

	case actions.FileRevertedMsg:
		return m.Update(fileRevertedMsg{repository: msg.Repository, filePath: msg.FilePath})

	case actions.ClipboardCopiedMsg:
		if msg.Success {
			// Different message depending on context
			if m.viewMode == ViewGitHubLogin {
				m.statusMessage = "Code copied to clipboard! Paste it in your browser."
			} else if m.err != nil {
				// In error modal, set flag to show "Copied!" in the modal
				m.errorCopied = true
				m.statusMessage = "Error copied to clipboard!"
			} else {
				m.statusMessage = "Copied to clipboard!"
			}
		} else {
			m.statusMessage = fmt.Sprintf("Failed to copy: %v", msg.Err)
		}
		return m, nil

	case actions.SettingsSavedMsg:
		return m.Update(settingsSavedMsg{
			githubConnected: msg.GitHubConnected,
			ticketService:   msg.TicketService,
			ticketProvider:  msg.TicketProvider,
			savedLocal:      msg.SavedLocal,
			err:             msg.Err,
		})

	case cleanupCompletedMsg:
		m.loading = false
		if msg.success {
			m.statusMessage = msg.message
			// Reload repository after successful cleanup
			return m, m.loadRepository()
		}
		m.statusMessage = fmt.Sprintf("Cleanup failed: %v", msg.err)
		return m, nil

	case undoCompletedMsg:
		m.statusMessage = msg.message
		// Reload repository after undo/redo
		return m, m.refreshRepository()
	}

	return m, nil
}
