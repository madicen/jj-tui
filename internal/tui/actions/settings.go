package actions

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/codecks"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/github"
	"github.com/madicen/jj-tui/internal/jira"
	"github.com/madicen/jj-tui/internal/tickets"
)

// SettingsParams contains all settings values
type SettingsParams struct {
	GitHubToken                  string
	JiraURL                      string
	JiraUser                     string
	JiraToken                    string
	JiraExcludedStatuses         string
	CodecksSubdomain             string
	CodecksToken                 string
	CodecksProject               string
	CodecksExcludedStatuses      string
	GitHubIssuesExcludedStatuses string
	TicketProvider               string // Explicit provider: "jira", "codecks", "github_issues", or ""
	ShowMerged                   bool
	ShowClosed                   bool
	OnlyMine                     bool
	PRLimit                      int
	PRRefreshInterval            int
	AutoInProgress               bool
	BranchLimit                  int
	SanitizeBookmarks            bool
	// GitHub repo info needed for GitHub Issues
	GitHubOwner string
	GitHubRepo  string
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

		cfg, _ := config.Load()
		if cfg == nil {
			cfg = &config.Config{}
		}

		// If a new token is being entered manually (not via Device Flow),
		// set the auth method to Token
		if params.GitHubToken != "" && params.GitHubToken != cfg.GitHubToken {
			// Token changed - set auth method to manual token
			cfg.SetGitHubToken(params.GitHubToken, config.GitHubAuthToken)
		} else if params.GitHubToken == "" {
			// Token cleared
			cfg.ClearGitHub()
		} else {
			// Token unchanged - keep existing auth method
			cfg.GitHubToken = params.GitHubToken
		}

		cfg.GitHubShowMerged = &params.ShowMerged
		cfg.GitHubShowClosed = &params.ShowClosed
		cfg.GitHubOnlyMine = &params.OnlyMine
		cfg.GitHubPRLimit = &params.PRLimit
		cfg.GitHubRefreshInterval = &params.PRRefreshInterval
		cfg.TicketAutoInProgress = &params.AutoInProgress
		cfg.TicketProvider = params.TicketProvider // Use explicit provider
		cfg.JiraURL = params.JiraURL
		cfg.JiraUser = params.JiraUser
		cfg.JiraToken = params.JiraToken
		cfg.JiraExcludedStatuses = params.JiraExcludedStatuses
		cfg.CodecksSubdomain = params.CodecksSubdomain
		cfg.CodecksToken = params.CodecksToken
		cfg.CodecksProject = params.CodecksProject
		cfg.CodecksExcludedStatuses = params.CodecksExcludedStatuses
		cfg.GitHubIssuesExcludedStatuses = params.GitHubIssuesExcludedStatuses
		cfg.BranchStatsLimit = &params.BranchLimit
		cfg.SanitizeBookmarkNames = &params.SanitizeBookmarks

		_ = cfg.Save()

		return buildSettingsSavedMsg(params, false)
	}
}

// SaveSettingsLocal saves settings to local config file
func SaveSettingsLocal(params SettingsParams) tea.Cmd {
	return func() tea.Msg {
		setEnvParams(params)

		cfg := &config.Config{
			TicketProvider:               params.TicketProvider, // Use explicit provider
			GitHubShowMerged:             &params.ShowMerged,
			GitHubShowClosed:             &params.ShowClosed,
			GitHubOnlyMine:               &params.OnlyMine,
			GitHubPRLimit:                &params.PRLimit,
			GitHubRefreshInterval:        &params.PRRefreshInterval,
			TicketAutoInProgress:         &params.AutoInProgress,
			BranchStatsLimit:             &params.BranchLimit,
			SanitizeBookmarkNames:        &params.SanitizeBookmarks,
			JiraExcludedStatuses:         params.JiraExcludedStatuses,
			CodecksProject:               params.CodecksProject,
			CodecksExcludedStatuses:      params.CodecksExcludedStatuses,
			GitHubIssuesExcludedStatuses: params.GitHubIssuesExcludedStatuses,
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

		return buildSettingsSavedMsg(params, true)
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

func buildSettingsSavedMsg(params SettingsParams, savedLocal bool) SettingsSavedMsg {
	var githubConnected bool
	var ticketSvc tickets.Service

	if params.GitHubToken != "" {
		githubConnected = true
	}

	// Important: Only assign to ticketSvc if the service was successfully created.
	// Assigning a typed nil pointer (e.g., (*jira.Service)(nil)) to an interface
	// makes the interface non-nil, which would bypass nil checks and cause panics.
	switch params.TicketProvider {
	case "codecks":
		if codecks.IsConfigured() {
			if svc, err := codecks.NewService(); err == nil && svc != nil {
				ticketSvc = svc
			}
		}
	case "jira":
		if jira.IsConfigured() {
			if svc, err := jira.NewService(); err == nil && svc != nil {
				ticketSvc = svc
			}
		}
	case "github_issues":
		if params.GitHubToken != "" && params.GitHubOwner != "" && params.GitHubRepo != "" {
			if svc, err := github.NewIssuesServiceWithToken(params.GitHubOwner, params.GitHubRepo, params.GitHubToken); err == nil && svc != nil {
				ticketSvc = svc
			}
		}
	}

	return SettingsSavedMsg{
		GitHubConnected: githubConnected,
		TicketService:   ticketSvc,
		TicketProvider:  params.TicketProvider,
		SavedLocal:      savedLocal,
	}
}
