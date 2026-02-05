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
		setEnvIfNotEmpty("GITHUB_TOKEN", params.GitHubToken)
		setEnvIfNotEmpty("JIRA_URL", params.JiraURL)
		setEnvIfNotEmpty("JIRA_USER", params.JiraUser)
		setEnvIfNotEmpty("JIRA_TOKEN", params.JiraToken)
		setEnvIfNotEmpty("CODECKS_SUBDOMAIN", params.CodecksSubdomain)
		setEnvIfNotEmpty("CODECKS_TOKEN", params.CodecksToken)
		if params.CodecksProject != "" {
			os.Setenv("CODECKS_PROJECT", params.CodecksProject)
		} else {
			os.Unsetenv("CODECKS_PROJECT")
		}

		ticketProvider := determineTicketProvider(params)

		cfg, _ := config.Load()
		if cfg == nil {
			cfg = &config.Config{}
		}

		if params.GitHubToken != "" {
			cfg.GitHubToken = params.GitHubToken
		}
		cfg.GitHubShowMerged = &params.ShowMerged
		cfg.GitHubShowClosed = &params.ShowClosed
		cfg.GitHubOnlyMine = &params.OnlyMine
		cfg.GitHubPRLimit = &params.PRLimit
		cfg.TicketProvider = ticketProvider
		cfg.JiraURL = params.JiraURL
		cfg.JiraUser = params.JiraUser
		cfg.JiraToken = params.JiraToken
		cfg.JiraExcludedStatuses = params.JiraExcludedStatuses
		cfg.CodecksSubdomain = params.CodecksSubdomain
		cfg.CodecksToken = params.CodecksToken
		cfg.CodecksProject = params.CodecksProject
		cfg.CodecksExcludedStatuses = params.CodecksExcludedStatuses

		_ = cfg.Save()

		return buildSettingsSavedMsg(params.GitHubToken, ticketProvider, false)
	}
}

// SaveSettingsLocal saves settings to local config file
func SaveSettingsLocal(params SettingsParams) tea.Cmd {
	return func() tea.Msg {
		setEnvIfNotEmpty("GITHUB_TOKEN", params.GitHubToken)
		setEnvIfNotEmpty("JIRA_URL", params.JiraURL)
		setEnvIfNotEmpty("JIRA_USER", params.JiraUser)
		setEnvIfNotEmpty("JIRA_TOKEN", params.JiraToken)
		setEnvIfNotEmpty("CODECKS_SUBDOMAIN", params.CodecksSubdomain)
		setEnvIfNotEmpty("CODECKS_TOKEN", params.CodecksToken)
		if params.CodecksProject != "" {
			os.Setenv("CODECKS_PROJECT", params.CodecksProject)
		} else {
			os.Unsetenv("CODECKS_PROJECT")
		}

		ticketProvider := determineTicketProvider(params)

		cfg := &config.Config{
			TicketProvider:          ticketProvider,
			GitHubShowMerged:        &params.ShowMerged,
			GitHubShowClosed:        &params.ShowClosed,
			GitHubOnlyMine:          &params.OnlyMine,
			GitHubPRLimit:           &params.PRLimit,
			JiraExcludedStatuses:    params.JiraExcludedStatuses,
			CodecksProject:          params.CodecksProject,
			CodecksExcludedStatuses: params.CodecksExcludedStatuses,
		}

		if params.GitHubToken != "" {
			cfg.GitHubToken = params.GitHubToken
		}
		if params.JiraURL != "" {
			cfg.JiraURL = params.JiraURL
		}
		if params.JiraUser != "" {
			cfg.JiraUser = params.JiraUser
		}
		if params.JiraToken != "" {
			cfg.JiraToken = params.JiraToken
		}
		if params.CodecksSubdomain != "" {
			cfg.CodecksSubdomain = params.CodecksSubdomain
		}
		if params.CodecksToken != "" {
			cfg.CodecksToken = params.CodecksToken
		}

		if err := cfg.SaveLocal(); err != nil {
			return SettingsSavedMsg{Err: err}
		}

		return buildSettingsSavedMsg(params.GitHubToken, ticketProvider, true)
	}
}

func setEnvIfNotEmpty(key, value string) {
	if value != "" {
		os.Setenv(key, value)
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

	if ticketProvider == "codecks" && codecks.IsConfigured() {
		ticketSvc, _ = codecks.NewService()
	} else if ticketProvider == "jira" && jira.IsConfigured() {
		ticketSvc, _ = jira.NewService()
	}

	return SettingsSavedMsg{
		GitHubConnected: githubConnected,
		TicketService:   ticketSvc,
		TicketProvider:  ticketProvider,
		SavedLocal:      savedLocal,
	}
}

