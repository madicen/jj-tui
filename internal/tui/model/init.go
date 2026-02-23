package model

import (
	"context"
	"os"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/github"
	"github.com/madicen/jj-tui/internal/integrations/jj"
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

// New creates a new Model
func New(ctx context.Context) *Model {
	// Load config for initial values
	cfg, _ := config.Load()

	// Create description textarea for graph tab (commit description editing)
	ta := textarea.New()
	ta.Placeholder = "Enter commit description..."
	ta.ShowLineNumbers = false
	ta.SetWidth(60)
	ta.SetHeight(5)

	zm := zone.New()
	graphTabModel := graphtab.NewGraphModel(zm)
	graphTabModel.SetDescriptionInput(ta)

	settingsTabModel := settingstab.NewModel()
	settingsInputs := createSettingsInputs(cfg)
	settingsTabModel.SetInputs(settingsInputs)
	if cfg != nil {
		settingsTabModel.SetSettingsShowMerged(cfg.ShowMergedPRs())
		settingsTabModel.SetSettingsShowClosed(cfg.ShowClosedPRs())
		settingsTabModel.SetSettingsOnlyMine(cfg.OnlyMyPRs())
		settingsTabModel.SetSettingsPRLimit(cfg.PRLimit())
		settingsTabModel.SetSettingsPRRefreshInterval(cfg.PRRefreshInterval())
		settingsTabModel.SetSettingsAutoInProgress(cfg.AutoInProgressOnBranch())
		settingsTabModel.SetSettingsBranchLimit(cfg.BranchLimit())
		settingsTabModel.SetSettingsSanitizeBookmarks(cfg.ShouldSanitizeBookmarkNames())
		settingsTabModel.SetSettingsTicketProvider(cfg.TicketProvider)
	}

	return &Model{
		ctx:             ctx,
		zoneManager:     zm,
		viewMode:        ViewCommitGraph,
		statusMessage:   "Initializing...",
		loading:         true,
		graphTabModel:   graphTabModel,
		prsTabModel:     prstab.NewModel(zm),
		branchesTabModel: branchestab.NewModel(zm),
		ticketsTabModel: ticketstab.NewModel(zm),
		settingsTabModel: settingsTabModel,
		helpTabModel:    helptab.NewModel(zm),
		errorModal:      errortab.NewModel(),
		warningModal:    warningtab.NewModel(),
		conflictModal:   conflicttab.NewModel(zm),
		divergentModal:  divergenttab.NewModel(zm),
		bookmarkModal:   bookmarktab.NewModel(zm),
		prFormModal:     prformtab.NewModel(zm),
	}
}

// createSettingsInputs creates and initializes all settings input fields
func createSettingsInputs(cfg *config.Config) []textinput.Model {
	settingsInputs := make([]textinput.Model, 12)

	// GitHub Token (index 0)
	settingsInputs[0] = textinput.New()
	settingsInputs[0].Placeholder = "GitHub Personal Access Token"
	settingsInputs[0].CharLimit = 256 // GitHub PATs can be long
	settingsInputs[0].Width = 50
	settingsInputs[0].EchoMode = textinput.EchoPassword
	settingsInputs[0].EchoCharacter = '•'
	// Load from env var first, then fall back to config
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" && cfg != nil && cfg.GitHubToken != "" {
		githubToken = cfg.GitHubToken
	}
	settingsInputs[0].SetValue(githubToken)

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

	// Jira Project filter (index 4)
	settingsInputs[4] = textinput.New()
	settingsInputs[4].Placeholder = "PROJ or PROJ,TEAM (comma-separated, optional)"
	settingsInputs[4].CharLimit = 200
	settingsInputs[4].Width = 50
	jiraProject := os.Getenv("JIRA_PROJECT")
	if jiraProject == "" && cfg != nil {
		jiraProject = cfg.JiraProject
	}
	settingsInputs[4].SetValue(jiraProject)

	// Jira JQL filter (index 5)
	settingsInputs[5] = textinput.New()
	settingsInputs[5].Placeholder = "sprint in openSprints() (optional custom JQL)"
	settingsInputs[5].CharLimit = 500
	settingsInputs[5].Width = 50
	jiraJQL := os.Getenv("JIRA_JQL")
	if jiraJQL == "" && cfg != nil {
		jiraJQL = cfg.JiraJQL
	}
	settingsInputs[5].SetValue(jiraJQL)

	// Jira Excluded Statuses (index 6)
	settingsInputs[6] = textinput.New()
	settingsInputs[6].Placeholder = "Done, Won't Do, Cancelled (comma-separated)"
	settingsInputs[6].CharLimit = 200
	settingsInputs[6].Width = 50
	if cfg != nil {
		settingsInputs[6].SetValue(cfg.JiraExcludedStatuses)
	}

	// Codecks Subdomain (index 7)
	settingsInputs[7] = textinput.New()
	settingsInputs[7].Placeholder = "your-team (from your-team.codecks.io)"
	settingsInputs[7].CharLimit = 100
	settingsInputs[7].Width = 50
	settingsInputs[7].SetValue(os.Getenv("CODECKS_SUBDOMAIN"))

	// Codecks Token (index 8)
	settingsInputs[8] = textinput.New()
	settingsInputs[8].Placeholder = "Codecks API Token (from browser cookie 'at')"
	settingsInputs[8].CharLimit = 256
	settingsInputs[8].Width = 50
	settingsInputs[8].EchoMode = textinput.EchoPassword
	settingsInputs[8].EchoCharacter = '•'
	settingsInputs[8].SetValue(os.Getenv("CODECKS_TOKEN"))

	// Codecks Project (index 9)
	settingsInputs[9] = textinput.New()
	settingsInputs[9].Placeholder = "Project name (optional, filters cards)"
	settingsInputs[9].CharLimit = 100
	settingsInputs[9].Width = 50
	settingsInputs[9].SetValue(os.Getenv("CODECKS_PROJECT"))

	// Codecks Excluded Statuses (index 10)
	settingsInputs[10] = textinput.New()
	settingsInputs[10].Placeholder = "done, archived (comma-separated)"
	settingsInputs[10].CharLimit = 200
	settingsInputs[10].Width = 50
	if cfg != nil {
		settingsInputs[10].SetValue(cfg.CodecksExcludedStatuses)
	}

	// GitHub Issues Excluded Statuses (index 11)
	settingsInputs[11] = textinput.New()
	settingsInputs[11].Placeholder = "closed (comma-separated)"
	settingsInputs[11].CharLimit = 200
	settingsInputs[11].Width = 50
	if cfg != nil {
		settingsInputs[11].SetValue(cfg.GitHubIssuesExcludedStatuses)
	}

	return settingsInputs
}

// NewWithServices creates a new Model with pre-configured services
func NewWithServices(ctx context.Context, jjSvc *jj.Service, ghSvc *github.Service) *Model {
	m := New(ctx)
	m.jjService = jjSvc
	m.githubService = ghSvc
	return m
}

// NewDemo creates a new Model in demo mode with mock services
// This is used for VHS screenshots and visual testing
func NewDemo(ctx context.Context) *Model {
	m := New(ctx)
	m.demoMode = true
	return m
}
