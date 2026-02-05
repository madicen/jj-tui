package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
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

// openURL opens a URL in the default browser
func openURL(url string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "linux":
			cmd = exec.Command("xdg-open", url)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		default:
			return nil
		}
		_ = cmd.Start()
		return nil
	}
}

// Auto-refresh interval
const autoRefreshInterval = 2 * time.Second

// isSelectedCommitValid returns true if selectedCommit points to a valid commit
func (m *Model) isSelectedCommitValid() bool {
	return m.repository != nil &&
		m.selectedCommit >= 0 &&
		m.selectedCommit < len(m.repository.Graph.Commits)
}

// tickMsg is sent on each timer tick for auto-refresh
type tickMsg time.Time

// Model is the main TUI model using bubblezone for mouse handling.
// All clickable elements are wrapped with zone.Mark() in the View.
// Mouse events are handled via zone.MsgZoneInBounds messages.
type Model struct {
	ctx           context.Context
	zone          *zone.Manager
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
	notJJRepo      bool   // true if error is "not a jj repository"
	currentPath    string // path where we're running (for jj init)

	// Changed files for selected commit
	changedFiles         []jj.ChangedFile
	changedFilesCommitID string // Which commit the files are for

	// Viewports for scrollable content
	viewport       viewport.Model // Main viewport (graph or other content)
	filesViewport  viewport.Model // Secondary viewport for changed files in graph view
	viewportReady  bool
	graphFocused   bool // True if graph viewport has focus, false if files viewport

	// Rebase mode state
	selectionMode      SelectionMode
	rebaseSourceCommit int // Index of commit being rebased

	// Ticket state (Jira, Codecks, etc.)
	ticketList []tickets.Ticket

	// Description editing
	descriptionInput textarea.Model
	editingCommitID  string // Change ID of commit being edited

	// Settings inputs
	settingsInputs       []textinput.Model
	settingsFocusedField int
	settingsTab          int // 0=GitHub, 1=Jira, 2=Codecks

	// Settings toggle states (for GitHub filters)
	settingsShowMerged bool
	settingsShowClosed bool

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
}

// Messages for async operations
type repositoryLoadedMsg struct {
	repository *models.Repository
}

type editCompletedMsg struct {
	repository *models.Repository
}

type servicesInitializedMsg struct {
	jjService     *jj.Service
	githubService *github.Service
	ticketService tickets.Service
	ticketError   error // Error from ticket service initialization (for debugging)
	repository    *models.Repository
}

type prsLoadedMsg struct {
	prs []models.GitHubPR
}

type ticketsLoadedMsg struct {
	tickets []tickets.Ticket
}

type bookmarkCreatedMsg struct {
	ticketKey  string
	branchName string
}

type settingsSavedMsg struct {
	githubConnected bool
	ticketService   tickets.Service
	ticketProvider  string // "jira", "codecks", or ""
	savedLocal      bool   // true if saved to local .jj-tui.json
	err             error  // error if save failed
}

type errorMsg struct {
	err         error
	notJJRepo   bool   // true if the error is "not a jj repository"
	currentPath string // the path where we tried to find a jj repo
}

// ErrorMsg creates an error message for testing purposes
func ErrorMsg(err error) errorMsg {
	return errorMsg{err: err}
}

type jjInitSuccessMsg struct{}

// GitHub Device Flow messages
type githubDeviceFlowStartedMsg struct {
	deviceCode      string
	userCode        string
	verificationURL string
	interval        int
}

type githubLoginSuccessMsg struct {
	token string
}

type githubLoginPollMsg struct {
	interval int // Polling interval in seconds
}

type descriptionSavedMsg struct {
	commitID string
}

// prCreatedMsg is sent when a PR is successfully created
type prCreatedMsg struct {
	pr *models.GitHubPR
}

// branchPushedMsg is sent when a branch is pushed to remote
type branchPushedMsg struct {
	branch     string
	pushOutput string
}

// bookmarkCreatedOnCommitMsg is sent when a bookmark is created or moved on a commit
type bookmarkCreatedOnCommitMsg struct {
	bookmarkName string
	commitID     string
	wasMoved     bool // true if bookmark was moved, false if newly created
}

// bookmarkDeletedMsg is sent when a bookmark is deleted
type bookmarkDeletedMsg struct {
	bookmarkName string
}

// changedFilesLoadedMsg is sent when changed files for a commit are loaded
type changedFilesLoadedMsg struct {
	commitID string
	files    []jj.ChangedFile
}

// silentRepositoryLoadedMsg is for background refreshes that don't update the status
type silentRepositoryLoadedMsg struct {
	repository *models.Repository
}

// descriptionLoadedMsg contains the full description fetched from jj
type descriptionLoadedMsg struct {
	commitID    string
	description string
}

// New creates a new Model
func New(ctx context.Context) *Model {
	// Create textarea for description editing
	ta := textarea.New()
	ta.Placeholder = "Enter commit description..."
	ta.ShowLineNumbers = false
	ta.SetWidth(60)
	ta.SetHeight(5)

	// Load config for initial values
	cfg, _ := config.Load()

	// Create settings inputs
	settingsInputs := make([]textinput.Model, 9)

	// GitHub Token (index 0)
	settingsInputs[0] = textinput.New()
	settingsInputs[0].Placeholder = "GitHub Personal Access Token"
	settingsInputs[0].CharLimit = 256 // GitHub PATs can be long
	settingsInputs[0].Width = 50
	settingsInputs[0].EchoMode = textinput.EchoPassword
	settingsInputs[0].EchoCharacter = '•'
	settingsInputs[0].SetValue(os.Getenv("GITHUB_TOKEN"))

	// Jira URL (index 1)
	settingsInputs[1] = textinput.New()
	settingsInputs[1].Placeholder = "https://your-domain.atlassian.net"
	settingsInputs[1].CharLimit = 100
	settingsInputs[1].Width = 50
	settingsInputs[1].SetValue(os.Getenv("JIRA_URL"))

	// Jira User (index 2)
	settingsInputs[2] = textinput.New()
	settingsInputs[2].Placeholder = "your-email@example.com"
	settingsInputs[2].CharLimit = 100
	settingsInputs[2].Width = 50
	settingsInputs[2].SetValue(os.Getenv("JIRA_USER"))

	// Jira Token (index 3)
	settingsInputs[3] = textinput.New()
	settingsInputs[3].Placeholder = "Jira API Token"
	settingsInputs[3].CharLimit = 256 // Atlassian tokens can be 150+ chars
	settingsInputs[3].Width = 50
	settingsInputs[3].EchoMode = textinput.EchoPassword
	settingsInputs[3].EchoCharacter = '•'
	settingsInputs[3].SetValue(os.Getenv("JIRA_TOKEN"))

	// Jira Excluded Statuses (index 4)
	settingsInputs[4] = textinput.New()
	settingsInputs[4].Placeholder = "Done, Won't Do, Cancelled (comma-separated)"
	settingsInputs[4].CharLimit = 200
	settingsInputs[4].Width = 50
	if cfg != nil {
		settingsInputs[4].SetValue(cfg.JiraExcludedStatuses)
	}

	// Codecks Subdomain (index 5)
	settingsInputs[5] = textinput.New()
	settingsInputs[5].Placeholder = "your-team (from your-team.codecks.io)"
	settingsInputs[5].CharLimit = 100
	settingsInputs[5].Width = 50
	settingsInputs[5].SetValue(os.Getenv("CODECKS_SUBDOMAIN"))

	// Codecks Token (index 6)
	settingsInputs[6] = textinput.New()
	settingsInputs[6].Placeholder = "Codecks API Token (from browser cookie 'at')"
	settingsInputs[6].CharLimit = 256
	settingsInputs[6].Width = 50
	settingsInputs[6].EchoMode = textinput.EchoPassword
	settingsInputs[6].EchoCharacter = '•'
	settingsInputs[6].SetValue(os.Getenv("CODECKS_TOKEN"))

	// Codecks Project (index 7)
	settingsInputs[7] = textinput.New()
	settingsInputs[7].Placeholder = "Project name (optional, filters cards)"
	settingsInputs[7].CharLimit = 100
	settingsInputs[7].Width = 50
	settingsInputs[7].SetValue(os.Getenv("CODECKS_PROJECT"))

	// Codecks Excluded Statuses (index 8)
	settingsInputs[8] = textinput.New()
	settingsInputs[8].Placeholder = "done, archived (comma-separated)"
	settingsInputs[8].CharLimit = 200
	settingsInputs[8].Width = 50
	if cfg != nil {
		settingsInputs[8].SetValue(cfg.CodecksExcludedStatuses)
	}

	// Initialize toggle states from config
	showMerged := true
	showClosed := true
	if cfg != nil {
		showMerged = cfg.ShowMergedPRs()
		showClosed = cfg.ShowClosedPRs()
	}

	// PR title input
	prTitle := textinput.New()
	prTitle.Placeholder = "Pull request title"
	prTitle.CharLimit = 200
	prTitle.Width = 60

	// PR body textarea
	prBody := textarea.New()
	prBody.Placeholder = "Describe your changes..."
	prBody.ShowLineNumbers = false
	prBody.SetWidth(60)
	prBody.SetHeight(8)

	// Bookmark name input
	bookmarkName := textinput.New()
	bookmarkName.Placeholder = "bookmark-name"
	bookmarkName.CharLimit = 100
	bookmarkName.Width = 50

	return &Model{
		ctx:                       ctx,
		zone:                      zone.New(),
		viewMode:                  ViewCommitGraph,
		selectedCommit:            -1,
		statusMessage:             "Initializing...",
		loading:                   true,
		descriptionInput:          ta,
		settingsInputs:            settingsInputs,
		settingsShowMerged:        showMerged,
		settingsShowClosed:        showClosed,
		prTitleInput:              prTitle,
		prBodyInput:               prBody,
		prBaseBranch:              "main",
		prCommitIndex:             -1,
		bookmarkNameInput:         bookmarkName,
		bookmarkCommitIdx:         -1,
		selectedBookmarkIdx:       -1,
		jiraBookmarkTitles:        make(map[string]string),
		ticketBookmarkDisplayKeys: make(map[string]string),
	}
}

// NewWithServices creates a new Model with services
func NewWithServices(ctx context.Context, jjSvc *jj.Service, ghSvc *github.Service) *Model {
	m := New(ctx)
	m.jjService = jjSvc
	m.githubService = ghSvc
	return m
}

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
			maxOffset := totalLines - contentHeight
			if maxOffset < 0 {
				maxOffset = 0
			}
			if m.viewport.YOffset > maxOffset {
				m.viewport.YOffset = maxOffset
			}
		}

		// Resize text areas to fit new window width
		inputWidth := m.width - 20 // Leave margin for borders/padding
		if inputWidth < 30 {
			inputWidth = 30
		}
		if inputWidth > 80 {
			inputWidth = 80 // Cap at reasonable max
		}

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
			return m.zone.AnyInBoundsAndUpdate(m, msg)
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
			// Only update status if commit count changed
			newCount := len(msg.repository.Graph.Commits)
			if newCount != oldCount {
				m.statusMessage = fmt.Sprintf("Updated: %d commits", newCount)
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
		m.err = msg.err
		m.notJJRepo = msg.notJJRepo
		m.currentPath = msg.currentPath
		m.loading = false
		m.statusMessage = fmt.Sprintf("Error: %v", msg.err)
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
		m.loading = false
		// Don't clear m.err here - let errors persist until user dismisses them
		m.statusMessage = fmt.Sprintf("Loaded %d commits", len(msg.repository.Graph.Commits))
		if m.githubService != nil {
			m.statusMessage += " (GitHub connected)"
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
		if m.githubService != nil {
			cmds = append(cmds, m.loadPRs())
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
			m.statusMessage = fmt.Sprintf("Loaded %d PRs", len(msg.prs))
		} else {
			m.statusMessage = fmt.Sprintf("Loaded %d PRs (warning: repository is nil)", len(msg.prs))
		}
		return m, nil

	case ticketsLoadedMsg:
		m.ticketList = msg.tickets
		providerName := "tickets"
		if m.ticketService != nil {
			providerName = m.ticketService.GetProviderName() + " tickets"
		}
		m.statusMessage = fmt.Sprintf("Loaded %d %s", len(msg.tickets), providerName)
		if len(msg.tickets) > 0 && m.selectedTicket < 0 {
			m.selectedTicket = 0
		}
		return m, nil

	case bookmarkCreatedMsg:
		m.statusMessage = fmt.Sprintf("Created branch '%s' from %s", msg.branchName, msg.ticketKey)
		// Switch to commit graph view to see the new bookmark
		m.viewMode = ViewCommitGraph
		// Reload repository to show the new bookmark
		return m, m.loadRepository()

	case settingsSavedMsg:
		m.viewMode = ViewCommitGraph
		m.ticketService = msg.ticketService

		// Handle save error
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error saving settings: %v", msg.err)
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
		m.viewMode = ViewGitHubLogin
		m.statusMessage = "Waiting for GitHub authorization..."
		// Start polling for the token
		return m, tea.Batch(
			openURL(msg.verificationURL),
			m.pollGitHubToken(msg.interval),
		)

	case githubLoginPollMsg:
		if m.githubLoginPolling {
			return m, m.pollGitHubToken(msg.interval)
		}
		return m, nil

	case githubLoginSuccessMsg:
		m.githubLoginPolling = false
		m.githubDeviceCode = ""
		m.githubUserCode = ""
		m.viewMode = ViewSettings
		m.statusMessage = "GitHub login successful!"
		// Save the token to config
		cfg, _ := config.Load()
		cfg.GitHubToken = msg.token
		_ = cfg.Save()
		cfg.ApplyToEnvironment()
		// Reinitialize services
		return m, m.initializeServices()

	case prCreatedMsg:
		m.viewMode = ViewCommitGraph
		m.statusMessage = fmt.Sprintf("PR #%d created: %s", msg.pr.Number, msg.pr.Title)
		// Open the PR in browser
		return m, openURL(msg.pr.URL)

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

	case bookmarkDeletedMsg:
		m.viewMode = ViewCommitGraph
		m.statusMessage = fmt.Sprintf("Bookmark '%s' deleted", msg.bookmarkName)
		// Reload repository to update the view
		return m, tea.Batch(m.loadRepository(), m.loadPRs())

	case changedFilesLoadedMsg:
		// Only update if the files are for the currently selected commit
		if msg.commitID == m.changedFilesCommitID {
			m.changedFiles = msg.files
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

	case CommitSelectedMsg:
		m.selectedCommit = msg.Index
		// Load changed files for the selected commit
		if m.repository != nil && msg.Index >= 0 && msg.Index < len(m.repository.Graph.Commits) {
			commit := m.repository.Graph.Commits[msg.Index]
			m.changedFilesCommitID = commit.ChangeID
			m.changedFiles = nil // Clear old files while loading
			return m, m.loadChangedFiles(commit.ChangeID)
		}
		return m, nil

	case ActionMsg:
		return m.handleAction(msg.Action)

	// Handle messages from actions package
	case actions.RepositoryLoadedMsg:
		return m.Update(repositoryLoadedMsg{repository: msg.Repository})

	case actions.EditCompletedMsg:
		return m.Update(editCompletedMsg{repository: msg.Repository})

	case actions.ErrorMsg:
		return m.Update(errorMsg{err: msg.Err})

	case actions.DescriptionLoadedMsg:
		return m.Update(descriptionLoadedMsg{commitID: msg.CommitID, description: msg.Description})

	case actions.DescriptionSavedMsg:
		return m.Update(descriptionSavedMsg{commitID: msg.CommitID})

	case actions.PRCreatedMsg:
		return m.Update(prCreatedMsg{pr: msg.PR})

	case actions.BranchPushedMsg:
		return m.Update(branchPushedMsg{branch: msg.Branch, pushOutput: msg.PushOutput})

	case actions.BookmarkCreatedMsg:
		return m.Update(bookmarkCreatedOnCommitMsg{bookmarkName: msg.BookmarkName, commitID: msg.CommitID, wasMoved: msg.WasMoved})

	case actions.BookmarkDeletedMsg:
		return m.Update(bookmarkDeletedMsg{bookmarkName: msg.BookmarkName})

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
	}

	return m, nil
}
