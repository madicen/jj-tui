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
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen-utilities/jj-tui/v2/internal/github"
	"github.com/madicen-utilities/jj-tui/v2/internal/jira"
	"github.com/madicen-utilities/jj-tui/v2/internal/jj"
	"github.com/madicen-utilities/jj-tui/v2/internal/models"
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
	jiraService   *jira.Service
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

	// Jira state
	jiraTickets []jira.Ticket

	// Description editing
	descriptionInput textarea.Model
	editingCommitID  string // Change ID of commit being edited

	// Settings inputs
	settingsInputs       []textinput.Model
	settingsFocusedField int
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
	jiraService   *jira.Service
	repository    *models.Repository
}

type prsLoadedMsg struct {
	prs []models.GitHubPR
}

type jiraTicketsLoadedMsg struct {
	tickets []jira.Ticket
}

type bookmarkCreatedMsg struct {
	ticketKey  string
	branchName string
}

type settingsSavedMsg struct {
	githubConnected bool
	jiraConnected   bool
}

type errorMsg struct {
	err error
}

type descriptionSavedMsg struct {
	commitID string
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

	// Create settings inputs
	settingsInputs := make([]textinput.Model, 4)

	// GitHub Token
	settingsInputs[0] = textinput.New()
	settingsInputs[0].Placeholder = "GitHub Personal Access Token"
	settingsInputs[0].CharLimit = 100
	settingsInputs[0].Width = 50
	settingsInputs[0].EchoMode = textinput.EchoPassword
	settingsInputs[0].EchoCharacter = '•'
	settingsInputs[0].SetValue(os.Getenv("GITHUB_TOKEN"))

	// Jira URL
	settingsInputs[1] = textinput.New()
	settingsInputs[1].Placeholder = "https://your-domain.atlassian.net"
	settingsInputs[1].CharLimit = 100
	settingsInputs[1].Width = 50
	settingsInputs[1].SetValue(os.Getenv("JIRA_URL"))

	// Jira User
	settingsInputs[2] = textinput.New()
	settingsInputs[2].Placeholder = "your-email@example.com"
	settingsInputs[2].CharLimit = 100
	settingsInputs[2].Width = 50
	settingsInputs[2].SetValue(os.Getenv("JIRA_USER"))

	// Jira Token
	settingsInputs[3] = textinput.New()
	settingsInputs[3].Placeholder = "Jira API Token"
	settingsInputs[3].CharLimit = 100
	settingsInputs[3].Width = 50
	settingsInputs[3].EchoMode = textinput.EchoPassword
	settingsInputs[3].EchoCharacter = '•'
	settingsInputs[3].SetValue(os.Getenv("JIRA_TOKEN"))

	return &Model{
		ctx:              ctx,
		zone:             zone.New(),
		viewMode:         ViewCommitGraph,
		selectedCommit:   -1,
		statusMessage:    "Initializing...",
		loading:          true,
		descriptionInput: ta,
		settingsInputs:   settingsInputs,
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
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.MouseMsg:
		// Use AnyInBoundsAndUpdate to detect which zone was clicked
		if msg.Action == tea.MouseActionRelease {
			return m.zone.AnyInBoundsAndUpdate(m, msg)
		}
		return m, nil

	case zone.MsgZoneInBounds:
		return m.handleZoneClick(msg.Zone)

	case repositoryLoadedMsg:
		m.repository = msg.repository
		m.loading = false
		m.err = nil
		if m.jjService == nil {
			// First load - set the service
			jjSvc, _ := jj.NewService("")
			m.jjService = jjSvc
		}
		m.statusMessage = fmt.Sprintf("Loaded %d commits", len(msg.repository.Graph.Commits))
		// Auto-select first commit if none selected
		if m.selectedCommit == -1 && len(msg.repository.Graph.Commits) > 0 {
			m.selectedCommit = 0
		}
		return m, m.tickCmd() // Continue auto-refresh timer

	case editCompletedMsg:
		m.repository = msg.repository
		m.loading = false
		m.err = nil
		// Find and select the working copy commit
		for i, commit := range msg.repository.Graph.Commits {
			if commit.IsWorking {
				m.selectedCommit = i
				break
			}
		}
		m.statusMessage = "Now editing working copy"
		return m, m.tickCmd()

	case silentRepositoryLoadedMsg:
		// Background refresh - update data without changing status
		if msg.repository != nil {
			oldCount := 0
			if m.repository != nil {
				oldCount = len(m.repository.Graph.Commits)
			}
			m.repository = msg.repository
			m.err = nil
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
		m.loading = false
		m.statusMessage = fmt.Sprintf("Error: %v", msg.err)
		return m, m.tickCmd() // Continue auto-refresh even on error

	case servicesInitializedMsg:
		m.jjService = msg.jjService
		m.githubService = msg.githubService
		m.jiraService = msg.jiraService
		m.repository = msg.repository
		m.loading = false
		m.err = nil
		m.statusMessage = fmt.Sprintf("Loaded %d commits", len(msg.repository.Graph.Commits))
		if m.githubService != nil {
			m.statusMessage += " (GitHub connected)"
		}
		if m.jiraService != nil {
			m.statusMessage += " (Jira connected)"
		}
		// Auto-select first commit if none selected
		if m.selectedCommit == -1 && len(msg.repository.Graph.Commits) > 0 {
			m.selectedCommit = 0
		}
		return m, m.tickCmd()

	case prsLoadedMsg:
		if m.repository != nil {
			m.repository.PRs = msg.prs
		}
		m.statusMessage = fmt.Sprintf("Loaded %d PRs", len(msg.prs))
		return m, nil

	case jiraTicketsLoadedMsg:
		m.jiraTickets = msg.tickets
		m.statusMessage = fmt.Sprintf("Loaded %d Jira tickets", len(msg.tickets))
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
		var status []string
		if msg.githubConnected {
			status = append(status, "GitHub")
		}
		if msg.jiraConnected {
			status = append(status, "Jira")
		}
		if len(status) > 0 {
			m.statusMessage = fmt.Sprintf("Settings saved. Connected: %s", strings.Join(status, ", "))
		} else {
			m.statusMessage = "Settings saved"
		}
		// Reinitialize services with new credentials
		return m, m.initializeServices()

	case tickMsg:
		// Auto-refresh: reload repository data silently (but not while editing)
		if !m.loading && m.jjService != nil && m.viewMode != ViewEditDescription {
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
			m.descriptionInput.SetValue(description)
			m.descriptionInput.Focus()
			m.statusMessage = fmt.Sprintf("Editing description (Ctrl+S to save, Esc to cancel)")
		}
		return m, nil

	// Handle our custom messages
	case TabSelectedMsg:
		m.viewMode = msg.Tab
		return m, nil

	case CommitSelectedMsg:
		m.selectedCommit = msg.Index
		return m, nil

	case ActionMsg:
		return m.handleAction(msg.Action)
	}

	return m, nil
}
