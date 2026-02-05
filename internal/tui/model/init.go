package model

import (
	"context"
	"os"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/github"
	"github.com/madicen/jj-tui/internal/jj"
)

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
	settingsInputs := createSettingsInputs(cfg)

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

// createSettingsInputs creates and initializes all settings input fields
func createSettingsInputs(cfg *config.Config) []textinput.Model {
	settingsInputs := make([]textinput.Model, 9)

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

	return settingsInputs
}

// NewWithServices creates a new Model with pre-configured services
func NewWithServices(ctx context.Context, jjSvc *jj.Service, ghSvc *github.Service) *Model {
	m := New(ctx)
	m.jjService = jjSvc
	m.githubService = ghSvc
	return m
}
