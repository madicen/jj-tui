package model

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/github"
	"github.com/madicen/jj-tui/internal/jj"
	"github.com/madicen/jj-tui/internal/models"
	"github.com/madicen/jj-tui/internal/tickets"
	"github.com/madicen/jj-tui/internal/tui/actions"
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
	repository    *models.Repository

	// UI state
	viewMode       ViewMode
	width          int
	height         int
	selectedCommit int
	selectedPR     int
	selectedTicket int
	statusMessage  string
	err            error
	loading        bool
	githubInfo     string // Diagnostic info about GitHub connection

	// Ticket transitions
	availableTransitions []tickets.Transition
	transitionInProgress bool
	statusChangeMode     bool // whether status change buttons are expanded
	loadingTransitions   bool
	notJJRepo            bool   // true if error is "not a jj repository"
	currentPath          string // path where we're running (for jj init)

	// Changed files for selected commit
	changedFiles         []jj.ChangedFile
	changedFilesCommitID string // Which commit the files are for
	selectedFile         int    // Index of selected file in changed files list (-1 = none)

	// Viewports for scrollable content
	viewport      viewport.Model // Main viewport (graph or other content)
	filesViewport viewport.Model // Secondary viewport for changed files in graph view
	viewportReady bool
	graphFocused  bool // True if graph viewport has focus, false if files viewport

	// Rebase mode state
	selectionMode      SelectionMode
	rebaseSourceCommit int // Index of commit being rebased

	// Ticket state (Jira, Codecks, etc.)
	ticketList []tickets.Ticket

	// Branch state
	branchList     []models.Branch
	selectedBranch int

	// Description editing
	descriptionInput textarea.Model
	editingCommitID  string // Change ID of commit being edited

	// Settings inputs
	settingsInputs       []textinput.Model
	settingsFocusedField int
	settingsTab          int // 0=GitHub, 1=Jira, 2=Codecks, 3=Advanced

	// Settings toggle states (for GitHub filters)
	settingsShowMerged        bool
	settingsShowClosed        bool
	settingsOnlyMine          bool
	settingsPRLimit           int
	settingsPRRefreshInterval int  // in seconds, 0 = disabled
	settingsAutoInProgress    bool // auto-set ticket to "In Progress" when creating branch
	settingsBranchLimit       int  // max branches to calculate stats for
	settingsSanitizeBookmarks bool // auto-fix invalid bookmark names

	// Advanced settings state
	confirmingCleanup string // "" = not confirming, "delete_bookmarks", "abandon_old_commits"

	// PR creation state
	prTitleInput        textinput.Model
	prBodyInput         textarea.Model
	prBaseBranch        string
	prHeadBranch        string
	prFocusedField      int  // 0=title, 1=body
	prCommitIndex       int  // Index of commit PR is being created from
	prNeedsMoveBookmark bool // True if we need to move the bookmark to include all commits

	// Bookmark creation state
	bookmarkNameInput         textinput.Model
	bookmarkCommitIdx         int               // Index of commit to create bookmark on (-1 for new branch from main)
	existingBookmarks         []string          // List of existing bookmarks
	selectedBookmarkIdx       int               // Index of selected existing bookmark (-1 for new)
	bookmarkFromJira          bool              // True if creating bookmark from Jira ticket
	bookmarkJiraTicketKey     string            // Jira ticket key if creating from Jira
	bookmarkJiraTicketTitle   string            // Jira ticket summary if creating from Jira
	bookmarkTicketDisplayKey  string            // Short display key (e.g., "$12u" for Codecks) for commit messages
	jiraBookmarkTitles        map[string]string // Maps bookmark names to formatted PR titles ("KEY - Title")
	ticketBookmarkDisplayKeys map[string]string // Maps bookmark names to ticket short IDs for commit messages

	// GitHub Device Flow state
	githubDeviceCode      string // Device code for polling
	githubUserCode        string // Code user needs to enter
	githubVerificationURL string // URL user needs to visit
	githubLoginPolling    bool   // True if currently polling for token
	githubPollInterval    int    // Current polling interval in seconds
}

// doPollMsg is a message used to trigger a GitHub token poll.
type doPollMsg struct{}

// SetRepository sets the repository data
func (m *Model) SetRepository(repo *models.Repository) {
	m.repository = repo
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

		// Initialize or resize viewport
		// Header is 1 line, status bar is 1 line
		headerHeight := 1
		statusHeight := 1
		contentHeight := m.height - headerHeight - statusHeight
		if contentHeight < 1 {
			contentHeight = 1
		}

		if !m.viewportReady {
			m.viewport = viewport.New(m.width, contentHeight)
			m.viewport.MouseWheelEnabled = true
			m.filesViewport = viewport.New(m.width, contentHeight)
			m.filesViewport.MouseWheelEnabled = true
			m.graphFocused = true // Start with graph focused
			m.viewportReady = true
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = contentHeight
			m.filesViewport.Width = m.width
			m.filesViewport.Height = contentHeight

			// Reset scroll position if it's now beyond valid bounds
			totalLines := m.viewport.TotalLineCount()
			maxOffset := max(totalLines-contentHeight, 0)
			if m.viewport.YOffset > maxOffset {
				m.viewport.YOffset = maxOffset
			}
		}

		// Resize text areas to fit new window width
		inputWidth := min(
			// Leave margin for borders/padding
			max(
				m.width-20, 30,
			),
			// Cap at reasonable max
			80,
		)

		m.descriptionInput.SetWidth(inputWidth)
		m.prBodyInput.SetWidth(inputWidth)
		m.prTitleInput.Width = inputWidth
		m.bookmarkNameInput.Width = inputWidth

		// Resize settings inputs
		for i := range m.settingsInputs {
			m.settingsInputs[i].Width = inputWidth - 10
		}

		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.MouseMsg:
		// Handle mouse wheel scrolling
		if msg.Action == tea.MouseActionPress && (msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown) {
			// For graph view, scroll the focused pane
			if m.viewMode == ViewCommitGraph {
				if m.graphFocused {
					// Scroll graph pane
					var cmd tea.Cmd
					m.viewport, cmd = m.viewport.Update(msg)
					return m, cmd
				} else {
					// Scroll files pane
					var cmd tea.Cmd
					m.filesViewport, cmd = m.filesViewport.Update(msg)
					return m, cmd
				}
			}
			// For other views, let the viewport handle scrolling directly
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
		// Use AnyInBoundsAndUpdate to detect which zone was clicked
		if msg.Action == tea.MouseActionRelease {
			return m.zoneManager.AnyInBoundsAndUpdate(m, msg)
		}
		return m, nil

	case zone.MsgZoneInBounds:
		return m.handleZoneClick(msg.Zone)

	case repositoryLoadedMsg:
		// Preserve PRs from previous repository
		var oldPRs []models.GitHubPR
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

		// Build commands to run
		var cmds []tea.Cmd
		cmds = append(cmds, m.tickCmd())

		// Also refresh PRs when GitHub is connected (needed for Update PR button)
		if m.githubService != nil {
			cmds = append(cmds, m.loadPRs())
		}

		// Re-sync selection by ChangeID if we have one tracked
		if m.changedFilesCommitID != "" {
			found := false
			for i, commit := range msg.repository.Graph.Commits {
				if commit.ChangeID == m.changedFilesCommitID {
					m.selectedCommit = i
					found = true
					break
				}
			}
			if !found {
				// Commit no longer exists, reset selection
				m.selectedCommit = -1
				m.changedFilesCommitID = ""
			}
		}

		// Auto-select first commit if none selected
		if m.selectedCommit == -1 && len(msg.repository.Graph.Commits) > 0 {
			m.selectedCommit = 0
			// Load changed files for the auto-selected commit
			commit := msg.repository.Graph.Commits[0]
			m.changedFilesCommitID = commit.ChangeID
			cmds = append(cmds, m.loadChangedFiles(commit.ChangeID))
		}
		return m, tea.Batch(cmds...)

	case editCompletedMsg:
		// Preserve PRs from previous repository
		var oldPRs []models.GitHubPR
		if m.repository != nil {
			oldPRs = m.repository.PRs
		}
		m.repository = msg.repository
		m.repository.PRs = oldPRs // Restore PRs temporarily
		m.loading = false
		// Don't clear m.err here - let errors persist until dismissed
		// Find and select the working copy commit
		for i, commit := range msg.repository.Graph.Commits {
			if commit.IsWorking {
				m.selectedCommit = i
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
			var oldPRs []models.GitHubPR
			if m.repository != nil {
				oldCount = len(m.repository.Graph.Commits)
				oldPRs = m.repository.PRs // Preserve PRs from previous load
			}
			m.repository = msg.repository
			m.repository.PRs = oldPRs // Restore PRs
			// Don't clear m.err here - let errors persist until dismissed
			// Only update status if commit count changed AND there's no existing error
			newCount := len(msg.repository.Graph.Commits)
			if newCount != oldCount && m.err == nil {
				m.statusMessage = fmt.Sprintf("Updated: %d commits", newCount)
			}

			// Re-sync selection by ChangeID if we have one tracked
			if m.changedFilesCommitID != "" {
				found := false
				for i, commit := range msg.repository.Graph.Commits {
					if commit.ChangeID == m.changedFilesCommitID {
						m.selectedCommit = i
						found = true
						break
					}
				}
				if !found {
					// Commit no longer exists, reset selection
					m.selectedCommit = -1
					m.changedFilesCommitID = ""
				}
			}

			// Ensure selection is still valid
			if m.selectedCommit >= newCount {
				m.selectedCommit = newCount - 1
			}
			if m.selectedCommit == -1 && newCount > 0 {
				m.selectedCommit = 0
			}
		}
		return m, nil

	case errorMsg:
		m.err = msg.Err
		m.notJJRepo = msg.NotJJRepo
		m.currentPath = msg.CurrentPath
		m.loading = false
		m.statusMessage = fmt.Sprintf("Error: %v", msg.Err)
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
		m.loading = false
		// Don't clear m.err here - let errors persist until user dismisses them
		m.statusMessage = fmt.Sprintf("Loaded %d commits", len(msg.repository.Graph.Commits))
		if m.githubService != nil {
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

		// Load PRs on startup if GitHub is connected (needed for Update PR button)
		// Also start PR auto-refresh timer
		if m.githubService != nil {
			cmds = append(cmds, m.loadPRs())
			if prTickCmd := m.prTickCmd(); prTickCmd != nil {
				cmds = append(cmds, prTickCmd)
			}
		}

		// Auto-select first commit if none selected and load its changed files
		if m.selectedCommit == -1 && len(msg.repository.Graph.Commits) > 0 {
			m.selectedCommit = 0
			commit := msg.repository.Graph.Commits[0]
			m.changedFilesCommitID = commit.ChangeID
			cmds = append(cmds, m.loadChangedFiles(commit.ChangeID))
		}

		return m, tea.Batch(cmds...)

	case prsLoadedMsg:
		if m.repository != nil {
			m.repository.PRs = msg.prs
			// Only update status if there's no existing error
			if m.err == nil {
				m.statusMessage = fmt.Sprintf("Loaded %d PRs", len(msg.prs))
			}
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
		m.ticketList = msg.tickets
		// Only update status if there's no existing error
		if m.err == nil {
			providerName := "tickets"
			if m.ticketService != nil {
				providerName = m.ticketService.GetProviderName() + " tickets"
			}
			m.statusMessage = fmt.Sprintf("Loaded %d %s", len(msg.tickets), providerName)
		}
		if len(msg.tickets) > 0 && m.selectedTicket < 0 {
			m.selectedTicket = 0
		}
		// Load transitions for the selected ticket
		m.availableTransitions = nil
		m.loadingTransitions = true
		return m, m.loadTransitions()

	case transitionsLoadedMsg:
		m.loadingTransitions = false
		m.availableTransitions = msg.transitions
		return m, nil

	case transitionCompletedMsg:
		m.transitionInProgress = false
		m.statusChangeMode = false // Collapse status buttons after transition
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
			if m.err == nil {
				m.statusMessage = fmt.Sprintf("Loaded %d branches", len(msg.branches))
			}
			if len(msg.branches) > 0 && m.selectedBranch < 0 {
				m.selectedBranch = 0
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
		m.githubDeviceCode = msg.deviceCode
		m.githubUserCode = msg.userCode
		m.githubVerificationURL = msg.verificationURL
		m.githubLoginPolling = true
		m.githubPollInterval = msg.interval
		m.viewMode = ViewGitHubLogin
		m.statusMessage = "Waiting for GitHub authorization..."
		// Start polling for the token
		return m, tea.Batch(
			openURL(msg.verificationURL),
			m.pollGitHubToken(), // Start first poll immediately
		)

	case githubLoginPollMsg:
		if m.githubLoginPolling {
			// Check for slow_down signal
			if msg.interval > 0 {
				m.githubPollInterval += msg.interval
			}
			// Schedule the next poll
			return m, tea.Tick(time.Duration(m.githubPollInterval)*time.Second, func(t time.Time) tea.Msg {
				return doPollMsg{}
			})
		}
		return m, nil

	case doPollMsg:
		if m.githubLoginPolling {
			return m, m.pollGitHubToken()
		}
		return m, nil

	case githubLoginSuccessMsg:
		m.githubLoginPolling = false
		m.githubDeviceCode = ""
		m.githubUserCode = ""
		m.githubPollInterval = 0
		m.viewMode = ViewSettings
		m.statusMessage = "GitHub login successful!"
		// Save the token to config
		cfg, _ := config.Load()
		cfg.GitHubToken = msg.token
		_ = cfg.Save()
		// Explicitly set the token in env (override any existing value)
		_ = os.Setenv("GITHUB_TOKEN", msg.token)
		// Update the settings input field with the new token so it's ready to save
		if len(m.settingsInputs) > 0 {
			m.settingsInputs[0].SetValue(msg.token)
		}
		// Reinitialize services
		return m, m.initializeServices()

	case prCreatedMsg:
		m.viewMode = ViewCommitGraph
		m.statusMessage = fmt.Sprintf("PR #%d created: %s", msg.pr.Number, msg.pr.Title)
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

	case fileMoveCompletedMsg:
		// Save the ChangeID of the commit we were working on before updating
		originalCommitID := m.changedFilesCommitID

		// Update repository with new state
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

		// Find the original commit by ChangeID and update the selected index
		// This is important because the graph structure may have changed
		for i, commit := range msg.repository.Graph.Commits {
			if commit.ChangeID == originalCommitID {
				m.selectedCommit = i
				break
			}
		}

		// Reload repository and changed files
		var cmds []tea.Cmd
		cmds = append(cmds, m.loadRepository())

		// Reload changed files for the commit we were working on
		if originalCommitID != "" {
			m.changedFilesCommitID = originalCommitID
			m.changedFiles = nil // Clear old files
			cmds = append(cmds, m.loadChangedFiles(originalCommitID))
		}
		return m, tea.Batch(cmds...)

	case fileRevertedMsg:
		// Save the ChangeID of the commit we were working on
		originalCommitID := m.changedFilesCommitID

		// Update repository with new state
		if m.repository != nil {
			oldPRs := m.repository.PRs
			m.repository = msg.repository
			m.repository.PRs = oldPRs
		} else {
			m.repository = msg.repository
		}
		m.statusMessage = fmt.Sprintf("Reverted changes to %s", msg.filePath)

		// Reload repository and changed files
		var cmds []tea.Cmd
		cmds = append(cmds, m.loadRepository())

		// Reload changed files for the commit we were working on
		if originalCommitID != "" {
			m.changedFilesCommitID = originalCommitID
			m.changedFiles = nil // Clear old files
			cmds = append(cmds, m.loadChangedFiles(originalCommitID))
		}
		return m, tea.Batch(cmds...)

	case changedFilesLoadedMsg:
		// Only update if the files are for the currently selected commit
		if msg.commitID == m.changedFilesCommitID {
			m.changedFiles = msg.files
			m.selectedFile = 0 // Reset file selection when files are loaded
		}
		return m, nil

	case tickMsg:
		// Stop auto-refresh if there's an error - let user handle it
		if m.err != nil {
			return m, nil
		}
		// Auto-refresh: reload repository data silently (but not while editing, creating PR, or creating bookmark)
		if !m.loading && m.jjService != nil && m.viewMode != ViewEditDescription && m.viewMode != ViewCreatePR && m.viewMode != ViewCreateBookmark {
			return m, tea.Batch(m.loadRepositorySilent(), m.tickCmd())
		}
		return m, m.tickCmd()

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
		m.editingCommitID = ""
		m.statusMessage = fmt.Sprintf("Description updated for %s", msg.commitID)
		// Reload repository to show updated description
		return m, m.loadRepository()

	case descriptionLoadedMsg:
		// Full description loaded, populate the textarea
		if m.viewMode == ViewEditDescription && m.editingCommitID == msg.commitID {
			description := msg.description
			if description == "(no description)" {
				description = ""
			}

			// If description is empty, check if commit has a ticket-associated bookmark
			// and prepopulate with the short ID
			if description == "" && m.repository != nil {
				// Find the commit index
				commitIdx := -1
				for i, commit := range m.repository.Graph.Commits {
					if commit.ChangeID == msg.commitID {
						commitIdx = i
						break
					}
				}

				if commitIdx >= 0 {
					// First check bookmarks directly on this commit
					commit := m.repository.Graph.Commits[commitIdx]
					var foundShortID string
					for _, branch := range commit.Branches {
						if shortID, ok := m.ticketBookmarkDisplayKeys[branch]; ok {
							foundShortID = shortID
							break
						}
					}

					// If not found, check ancestor bookmarks
					if foundShortID == "" {
						ancestorBookmark := m.findBookmarkForCommit(commitIdx)
						if ancestorBookmark != "" {
							if shortID, ok := m.ticketBookmarkDisplayKeys[ancestorBookmark]; ok {
								foundShortID = shortID
							}
						}
					}

					if foundShortID != "" {
						description = foundShortID + " "
					}
				}
			}

			m.descriptionInput.SetValue(description)
			m.descriptionInput.Focus()
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
			m.statusMessage = "Error copied to clipboard!"
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
