package actions

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/codecks"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/jira"
	"github.com/madicen/jj-tui/internal/tickets"
)

// SettingsParams contains all settings values
type SettingsParams struct {
	GitHubToken             string
	JiraURL                 string
	JiraUser                string
	JiraToken               string
	JiraExcludedStatuses    string
	CodecksSubdomain        string
	CodecksToken            string
	CodecksProject          string
	CodecksExcludedStatuses string
	ShowMerged              bool
	ShowClosed              bool
	OnlyMine                bool
	PRLimit                 int
	PRRefreshInterval       int
	AutoInProgress          bool
	BranchLimit             int
	SanitizeBookmarks       bool
}

// SettingsSavedMsg indicates settings were saved
type SettingsSavedMsg struct {
	GitHubConnected bool
	TicketService   tickets.Service
	TicketProvider  string
	SavedLocal      bool
	Err             error
}

// SaveSettings saves settings to global config
func SaveSettings(params SettingsParams) tea.Cmd {
	return func() tea.Msg {
		setEnvParams(params)

		ticketProvider := determineTicketProvider(params)

		cfg, _ := config.Load()
		if cfg == nil {
			cfg = &config.Config{}
		}

		cfg.GitHubToken = params.GitHubToken
		cfg.GitHubShowMerged = &params.ShowMerged
		cfg.GitHubShowClosed = &params.ShowClosed
		cfg.GitHubOnlyMine = &params.OnlyMine
		cfg.GitHubPRLimit = &params.PRLimit
		cfg.GitHubRefreshInterval = &params.PRRefreshInterval
		cfg.TicketAutoInProgress = &params.AutoInProgress
		cfg.TicketProvider = ticketProvider
		cfg.JiraURL = params.JiraURL
		cfg.JiraUser = params.JiraUser
		cfg.JiraToken = params.JiraToken
		cfg.JiraExcludedStatuses = params.JiraExcludedStatuses
		cfg.CodecksSubdomain = params.CodecksSubdomain
		cfg.CodecksToken = params.CodecksToken
		cfg.CodecksProject = params.CodecksProject
		cfg.CodecksExcludedStatuses = params.CodecksExcludedStatuses
		cfg.BranchStatsLimit = &params.BranchLimit
		cfg.SanitizeBookmarkNames = &params.SanitizeBookmarks

		_ = cfg.Save()

		return buildSettingsSavedMsg(params.GitHubToken, ticketProvider, false)
	}
}

// SaveSettingsLocal saves settings to local config file
func SaveSettingsLocal(params SettingsParams) tea.Cmd {
	return func() tea.Msg {
		setEnvParams(params)

		ticketProvider := determineTicketProvider(params)

		cfg := &config.Config{
			TicketProvider:          ticketProvider,
			GitHubShowMerged:        &params.ShowMerged,
			GitHubShowClosed:        &params.ShowClosed,
			GitHubOnlyMine:          &params.OnlyMine,
			GitHubPRLimit:           &params.PRLimit,
			GitHubRefreshInterval:   &params.PRRefreshInterval,
			TicketAutoInProgress:    &params.AutoInProgress,
			BranchStatsLimit:        &params.BranchLimit,
			SanitizeBookmarkNames:   &params.SanitizeBookmarks,
			JiraExcludedStatuses:    params.JiraExcludedStatuses,
			CodecksProject:          params.CodecksProject,
			CodecksExcludedStatuses: params.CodecksExcludedStatuses,
		}

		cfg.GitHubToken = params.GitHubToken
		cfg.JiraURL = params.JiraURL
		cfg.JiraUser = params.JiraUser
		cfg.JiraToken = params.JiraToken
		cfg.CodecksSubdomain = params.CodecksSubdomain
		cfg.CodecksToken = params.CodecksToken

		if err := cfg.SaveLocal(); err != nil {
			return SettingsSavedMsg{Err: err}
		}

		return buildSettingsSavedMsg(params.GitHubToken, ticketProvider, true)
	}
}

func setEnvParams(params SettingsParams) {
	os.Setenv("GITHUB_TOKEN", params.GitHubToken)
	os.Setenv("JIRA_URL", params.JiraURL)
	os.Setenv("JIRA_USER", params.JiraUser)
	os.Setenv("JIRA_TOKEN", params.JiraToken)
	os.Setenv("CODECKS_SUBDOMAIN", params.CodecksSubdomain)
	os.Setenv("CODECKS_TOKEN", params.CodecksToken)
	if params.CodecksProject != "" {
		os.Setenv("CODECKS_PROJECT", params.CodecksProject)
	} else {
		os.Unsetenv("CODECKS_PROJECT")
	}
}

func determineTicketProvider(params SettingsParams) string {
	if params.CodecksSubdomain != "" && params.CodecksToken != "" {
		return "codecks"
	}
	if params.JiraURL != "" && params.JiraUser != "" && params.JiraToken != "" {
		return "jira"
	}
	return ""
}

func buildSettingsSavedMsg(githubToken, ticketProvider string, savedLocal bool) SettingsSavedMsg {
	var githubConnected bool
	var ticketSvc tickets.Service

	if githubToken != "" {
		githubConnected = true
	}

	// Important: Only assign to ticketSvc if the service was successfully created.
	// Assigning a typed nil pointer (e.g., (*jira.Service)(nil)) to an interface
	// makes the interface non-nil, which would bypass nil checks and cause panics.
	if ticketProvider == "codecks" && codecks.IsConfigured() {
		if svc, err := codecks.NewService(); err == nil && svc != nil {
			ticketSvc = svc
		}
	} else if ticketProvider == "jira" && jira.IsConfigured() {
		if svc, err := jira.NewService(); err == nil && svc != nil {
			ticketSvc = svc
		}
	}

	return SettingsSavedMsg{
		GitHubConnected: githubConnected,
		TicketService:   ticketSvc,
		TicketProvider:  ticketProvider,
		SavedLocal:      savedLocal,
	}
}
