package settings

import (
	"context"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/codecks"
	"github.com/madicen/jj-tui/internal/integrations/github"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/integrations/jira"
	"github.com/madicen/jj-tui/internal/tickets"
	"github.com/madicen/jj-tui/internal/tui/data"
	"github.com/madicen/jj-tui/internal/tui/state"
)

// SettingsParams contains all settings values.
type SettingsParams struct {
	GitHubToken                  string
	JiraURL                      string
	JiraUser                     string
	JiraToken                    string
	JiraProject                  string
	JiraJQL                      string
	JiraExcludedStatuses         string
	CodecksSubdomain             string
	CodecksToken                 string
	CodecksProject               string
	CodecksExcludedStatuses      string
	GitHubIssuesExcludedStatuses string
	TicketProvider               string
	ShowMerged                   bool
	ShowClosed                   bool
	OnlyMine                     bool
	PRLimit                      int
	PRRefreshInterval            int
	AutoInProgress               bool
	BranchLimit                  int
	SanitizeBookmarks            bool
	GraphRevset                  string
	GitHubOwner                  string
	GitHubRepo                   string
}

// Status messages for cleanup flows.
const (
	StartDeleteBookmarksStatus   = "Press Y to confirm deletion of all bookmarks, or N to cancel"
	StartAbandonOldCommitsStatus = "Press Y to confirm abandoning commits before origin/main, or N to cancel"
	CancelCleanupStatus         = "Cleanup cancelled"
)

// HandleCleanupCompletedMsg mutates app and returns the Cmd to run.
func HandleCleanupCompletedMsg(msg CleanupCompletedMsg, app *state.AppState) tea.Cmd {
	app.Loading = false
	if msg.Success {
		app.StatusMessage = msg.Message
		if app.JJService != nil {
			return data.LoadRepository(app.JJService)
		}
		return nil
	}
	app.StatusMessage = fmt.Sprintf("Cleanup failed: %v", msg.Err)
	return nil
}

// SettingsSavedErrorInfo is returned when settings save had an error.
type SettingsSavedErrorInfo struct {
	Err error
}

// HandleSettingsSavedMsg mutates app and returns the Cmd to run.
func HandleSettingsSavedMsg(msg SettingsSavedMsg, app *state.AppState) (tea.Cmd, *SettingsSavedErrorInfo) {
	app.TicketService = msg.TicketService
	cfg, _ := config.Load()
	app.Config = cfg
	if msg.Err != nil {
		app.StatusMessage = fmt.Sprintf("Error saving settings: %v", msg.Err)
		return nil, &SettingsSavedErrorInfo{Err: msg.Err}
	}
	app.ViewMode = state.ViewCommitGraph
	app.StatusMessage = BuildSettingsSavedStatusFromMsg(msg, cfg)
	return data.InitializeServices(app.DemoMode), nil
}

// BuildSettingsSavedStatusFromMsg builds status string from settings save msg and config.
func BuildSettingsSavedStatusFromMsg(msg SettingsSavedMsg, cfg *config.Config) string {
	var status []string
	if msg.GitHubConnected {
		status = append(status, "GitHub")
	}
	if msg.TicketProvider != "" {
		status = append(status, msg.TicketProvider)
	}
	saveLocation := "globally"
	if msg.SavedLocal {
		saveLocation = "to .jj-tui.json (local)"
	}
	if len(status) > 0 {
		return fmt.Sprintf("Settings saved %s. Connected: %s", saveLocation, strings.Join(status, ", "))
	}
	return fmt.Sprintf("Settings saved %s", saveLocation)
}

// SaveSettingsEffect is sent when the user requests save (main runs saveSettings).
type SaveSettingsEffect struct{}

// SaveSettingsLocalEffect is sent when the user requests save to local file (main runs saveSettingsLocal).
type SaveSettingsLocalEffect struct{}

// SaveSettingsEffectCmd returns a command that sends SaveSettingsEffect.
func SaveSettingsEffectCmd() tea.Cmd {
	return func() tea.Msg { return SaveSettingsEffect{} }
}

// SaveSettingsLocalEffectCmd returns a command that sends SaveSettingsLocalEffect.
func SaveSettingsLocalEffectCmd() tea.Cmd {
	return func() tea.Msg { return SaveSettingsLocalEffect{} }
}

// ExecuteRequest validates the request and returns (statusMsg, cmd). Main sets statusMsg and returns the cmd.
func ExecuteRequest(r Request) (statusMsg string, cmd tea.Cmd) {
	if r.Cancel {
		return "", PerformCancelCmd()
	}
	if r.SaveSettings {
		return "", SaveSettingsEffectCmd()
	}
	if r.SaveSettingsLocal {
		return "", SaveSettingsLocalEffectCmd()
	}
	return "", nil
}

// BuildSettingsParams builds SettingsParams from the settings model state (sub-models).
func BuildSettingsParams(m *Model, githubOwner, githubRepo string) SettingsParams {
	gh := m.GetGitHubModel()
	jr := m.GetJiraModel()
	cc := m.GetCodecksModel()
	tk := m.GetTicketsModel()
	br := m.GetBranchesModel()
	adv := m.GetAdvancedModel()
	params := SettingsParams{
		TicketProvider:       tk.GetTicketProvider(),
		ShowMerged:           gh.GetShowMerged(),
		ShowClosed:           gh.GetShowClosed(),
		OnlyMine:             gh.GetOnlyMine(),
		PRLimit:              gh.GetPRLimit(),
		PRRefreshInterval:    gh.GetRefreshInterval(),
		AutoInProgress:       tk.GetAutoInProgress(),
		BranchLimit:          br.GetBranchLimit(),
		SanitizeBookmarks:    adv.GetSanitizeBookmarks(),
		GraphRevset:          strings.TrimSpace(adv.GetGraphRevset()),
		GitHubOwner:          githubOwner,
		GitHubRepo:           githubRepo,
	}
	params.GitHubToken = strings.TrimSpace(gh.GetToken())
	params.JiraURL = strings.TrimSpace(jr.GetURL())
	params.JiraUser = strings.TrimSpace(jr.GetUser())
	params.JiraToken = strings.TrimSpace(jr.GetToken())
	params.JiraProject = strings.TrimSpace(jr.GetProject())
	params.JiraJQL = strings.TrimSpace(jr.GetJQL())
	params.JiraExcludedStatuses = strings.TrimSpace(jr.GetExcludedStatuses())
	params.CodecksSubdomain = strings.TrimSpace(cc.GetSubdomain())
	params.CodecksToken = strings.TrimSpace(cc.GetToken())
	params.CodecksProject = strings.TrimSpace(cc.GetProject())
	params.CodecksExcludedStatuses = strings.TrimSpace(cc.GetExcludedStatuses())
	params.GitHubIssuesExcludedStatuses = strings.TrimSpace(tk.GetGitHubIssuesExcludedStatuses())
	return params
}

// SaveSettings builds params from the settings model and returns the save command.
func SaveSettings(m *Model, githubOwner, githubRepo string) tea.Cmd {
	params := BuildSettingsParams(m, githubOwner, githubRepo)
	return SaveSettingsCmd(params)
}

// SaveSettingsLocal builds params from the settings model and returns the local save command.
func SaveSettingsLocal(m *Model, githubOwner, githubRepo string) tea.Cmd {
	params := BuildSettingsParams(m, githubOwner, githubRepo)
	return SaveSettingsLocalCmd(params)
}

// ConfirmCleanup gets the current confirming type from the settings model, clears it, and returns the cleanup command.
func ConfirmCleanup(m *Model, jjSvc *jj.Service, repo *internal.Repository) tea.Cmd {
	confirmingType := m.GetConfirmingCleanup()
	m.SetConfirmingCleanup("")
	return ConfirmCleanupCmd(confirmingType, jjSvc, repo)
}

// ConfirmCleanupCmd returns the command for the given confirming cleanup type, or nil.
func ConfirmCleanupCmd(confirmingType string, jjSvc *jj.Service, repo *internal.Repository) tea.Cmd {
	switch confirmingType {
	case "delete_bookmarks":
		return DeleteAllBookmarksCmd(jjSvc, repo)
	case "abandon_old_commits":
		return AbandonOldCommitsCmd(jjSvc, repo)
	}
	return nil
}

// SaveSettingsCmd saves settings to global config.
func SaveSettingsCmd(params SettingsParams) tea.Cmd {
	return func() tea.Msg {
		setEnvParams(params)
		cfg, _ := config.Load()
		if cfg == nil {
			cfg = &config.Config{}
		}
		if params.GitHubToken != "" && params.GitHubToken != cfg.GitHubToken {
			cfg.SetGitHubToken(params.GitHubToken, config.GitHubAuthToken)
		} else if params.GitHubToken == "" {
			cfg.ClearGitHub()
		} else {
			cfg.GitHubToken = params.GitHubToken
		}
		cfg.GitHubShowMerged = &params.ShowMerged
		cfg.GitHubShowClosed = &params.ShowClosed
		cfg.GitHubOnlyMine = &params.OnlyMine
		cfg.GitHubPRLimit = &params.PRLimit
		cfg.GitHubRefreshInterval = &params.PRRefreshInterval
		cfg.TicketAutoInProgress = &params.AutoInProgress
		cfg.TicketProvider = params.TicketProvider
		cfg.JiraURL = params.JiraURL
		cfg.JiraUser = params.JiraUser
		cfg.JiraToken = params.JiraToken
		cfg.JiraProject = params.JiraProject
		cfg.JiraJQL = params.JiraJQL
		cfg.JiraExcludedStatuses = params.JiraExcludedStatuses
		cfg.CodecksSubdomain = params.CodecksSubdomain
		cfg.CodecksToken = params.CodecksToken
		cfg.CodecksProject = params.CodecksProject
		cfg.CodecksExcludedStatuses = params.CodecksExcludedStatuses
		cfg.GitHubIssuesExcludedStatuses = params.GitHubIssuesExcludedStatuses
		cfg.BranchStatsLimit = &params.BranchLimit
		cfg.SanitizeBookmarkNames = &params.SanitizeBookmarks
		cfg.GraphRevset = params.GraphRevset
		_ = cfg.Save()
		return buildSettingsSavedMsg(params, false)
	}
}

// SaveSettingsLocalCmd saves settings to local config file.
func SaveSettingsLocalCmd(params SettingsParams) tea.Cmd {
	return func() tea.Msg {
		setEnvParams(params)
		cfg := &config.Config{
			TicketProvider:               params.TicketProvider,
			GitHubShowMerged:             &params.ShowMerged,
			GitHubShowClosed:             &params.ShowClosed,
			GitHubOnlyMine:               &params.OnlyMine,
			GitHubPRLimit:                &params.PRLimit,
			GitHubRefreshInterval:        &params.PRRefreshInterval,
			TicketAutoInProgress:         &params.AutoInProgress,
			BranchStatsLimit:             &params.BranchLimit,
			SanitizeBookmarkNames:        &params.SanitizeBookmarks,
			GraphRevset:                  params.GraphRevset,
			JiraProject:                  params.JiraProject,
			JiraJQL:                      params.JiraJQL,
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
	if params.JiraProject != "" {
		os.Setenv("JIRA_PROJECT", params.JiraProject)
	} else {
		os.Unsetenv("JIRA_PROJECT")
	}
	if params.JiraJQL != "" {
		os.Setenv("JIRA_JQL", params.JiraJQL)
	} else {
		os.Unsetenv("JIRA_JQL")
	}
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

// DeleteAllBookmarksCmd returns a command that deletes all bookmarks present on commits in the repo.
func DeleteAllBookmarksCmd(jjSvc *jj.Service, repo *internal.Repository) tea.Cmd {
	if jjSvc == nil || repo == nil {
		return func() tea.Msg {
			return CleanupCompletedMsg{Err: fmt.Errorf("jj service or repository not initialized")}
		}
	}
	return func() tea.Msg {
		ctx := context.Background()
		bookmarkMap := make(map[string]bool)
		for _, commit := range repo.Graph.Commits {
			for _, branch := range commit.Branches {
				bookmarkMap[branch] = true
			}
		}
		if len(bookmarkMap) == 0 {
			return CleanupCompletedMsg{Success: true, Message: "No bookmarks to delete"}
		}
		var deletedCount int
		for bookmarkName := range bookmarkMap {
			if err := jjSvc.DeleteBookmark(ctx, bookmarkName); err == nil {
				deletedCount++
			}
		}
		return CleanupCompletedMsg{Success: true, Message: fmt.Sprintf("Deleted %d bookmarks", deletedCount)}
	}
}

// AbandonOldCommitsCmd returns a command that abandons all mutable commits that are not ancestors of main@origin.
func AbandonOldCommitsCmd(jjSvc *jj.Service, repo *internal.Repository) tea.Cmd {
	if jjSvc == nil || repo == nil {
		return func() tea.Msg {
			return CleanupCompletedMsg{Err: fmt.Errorf("jj service or repository not initialized")}
		}
	}
	return func() tea.Msg {
		ctx := context.Background()
		mainCommitID, err := jjSvc.GetRevisionChangeID(ctx, "main@origin")
		if err != nil || mainCommitID == "" {
			return CleanupCompletedMsg{Err: fmt.Errorf("could not find main@origin - make sure to track it first")}
		}
		var abandonedCount int
		for _, commit := range repo.Graph.Commits {
			if commit.IsWorking || commit.Immutable || commit.ChangeID == mainCommitID {
				continue
			}
			if err := jjSvc.AbandonCommit(ctx, commit.ChangeID); err == nil {
				abandonedCount++
			}
		}
		return CleanupCompletedMsg{Success: true, Message: fmt.Sprintf("Abandoned %d commits", abandonedCount)}
	}
}

// NewInputs creates and initializes all settings input fields from config and env.
func NewInputs(cfg *config.Config) []textinput.Model {
	inputs := make([]textinput.Model, 12)

	inputs[0] = textinput.New()
	inputs[0].Placeholder = "GitHub Personal Access Token"
	inputs[0].CharLimit = 256
	inputs[0].Width = 50
	inputs[0].EchoMode = textinput.EchoPassword
	inputs[0].EchoCharacter = '•'
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" && cfg != nil && cfg.GitHubToken != "" {
		githubToken = cfg.GitHubToken
	}
	inputs[0].SetValue(githubToken)

	inputs[1] = textinput.New()
	inputs[1].Placeholder = "https://your-domain.atlassian.net"
	inputs[1].CharLimit = 100
	inputs[1].Width = 50
	inputs[1].SetValue(os.Getenv("JIRA_URL"))

	inputs[2] = textinput.New()
	inputs[2].Placeholder = "your-email@example.com"
	inputs[2].CharLimit = 100
	inputs[2].Width = 50
	inputs[2].SetValue(os.Getenv("JIRA_USER"))

	inputs[3] = textinput.New()
	inputs[3].Placeholder = "Jira API Token"
	inputs[3].CharLimit = 256
	inputs[3].Width = 50
	inputs[3].EchoMode = textinput.EchoPassword
	inputs[3].EchoCharacter = '•'
	inputs[3].SetValue(os.Getenv("JIRA_TOKEN"))

	inputs[4] = textinput.New()
	inputs[4].Placeholder = "PROJ or PROJ,TEAM (comma-separated, optional)"
	inputs[4].CharLimit = 200
	inputs[4].Width = 50
	jiraProject := os.Getenv("JIRA_PROJECT")
	if jiraProject == "" && cfg != nil {
		jiraProject = cfg.JiraProject
	}
	inputs[4].SetValue(jiraProject)

	inputs[5] = textinput.New()
	inputs[5].Placeholder = "sprint in openSprints() (optional custom JQL)"
	inputs[5].CharLimit = 500
	inputs[5].Width = 50
	jiraJQL := os.Getenv("JIRA_JQL")
	if jiraJQL == "" && cfg != nil {
		jiraJQL = cfg.JiraJQL
	}
	inputs[5].SetValue(jiraJQL)

	inputs[6] = textinput.New()
	inputs[6].Placeholder = "Done, Won't Do, Cancelled (comma-separated)"
	inputs[6].CharLimit = 200
	inputs[6].Width = 50
	if cfg != nil {
		inputs[6].SetValue(cfg.JiraExcludedStatuses)
	}

	inputs[7] = textinput.New()
	inputs[7].Placeholder = "your-team (from your-team.codecks.io)"
	inputs[7].CharLimit = 100
	inputs[7].Width = 50
	inputs[7].SetValue(os.Getenv("CODECKS_SUBDOMAIN"))

	inputs[8] = textinput.New()
	inputs[8].Placeholder = "Codecks API Token (from browser cookie 'at')"
	inputs[8].CharLimit = 256
	inputs[8].Width = 50
	inputs[8].EchoMode = textinput.EchoPassword
	inputs[8].EchoCharacter = '•'
	inputs[8].SetValue(os.Getenv("CODECKS_TOKEN"))

	inputs[9] = textinput.New()
	inputs[9].Placeholder = "Project name (optional, filters cards)"
	inputs[9].CharLimit = 100
	inputs[9].Width = 50
	inputs[9].SetValue(os.Getenv("CODECKS_PROJECT"))

	inputs[10] = textinput.New()
	inputs[10].Placeholder = "done, archived (comma-separated)"
	inputs[10].CharLimit = 200
	inputs[10].Width = 50
	if cfg != nil {
		inputs[10].SetValue(cfg.CodecksExcludedStatuses)
	}

	inputs[11] = textinput.New()
	inputs[11].Placeholder = "closed (comma-separated)"
	inputs[11].CharLimit = 200
	inputs[11].Width = 50
	if cfg != nil {
		inputs[11].SetValue(cfg.GitHubIssuesExcludedStatuses)
	}

	return inputs
}

// StartGitHubLoginCmd returns a command that starts GitHub Device Flow and sends GitHubDeviceFlowStartedMsg or GitHubLoginErrorMsg.
func StartGitHubLoginCmd() tea.Cmd {
	return func() tea.Msg {
		deviceResp, err := github.StartDeviceFlow()
		if err != nil {
			return GitHubLoginErrorMsg{Err: fmt.Errorf("failed to start GitHub login: %w", err)}
		}
		return GitHubDeviceFlowStartedMsg{
			DeviceCode:      deviceResp.DeviceCode,
			UserCode:        deviceResp.UserCode,
			VerificationURL: deviceResp.VerificationURI,
			Interval:        deviceResp.Interval,
		}
	}
}

// PollGitHubTokenCmd returns a command that polls for the GitHub access token.
func PollGitHubTokenCmd(deviceCode string) tea.Cmd {
	if deviceCode == "" {
		return nil
	}
	return func() tea.Msg {
		token, err := github.PollForToken(deviceCode)
		if err != nil {
			if err.Error() == "slow_down" {
				return GitHubLoginPollMsg{Interval: 5}
			}
			return GitHubLoginErrorMsg{Err: fmt.Errorf("GitHub login failed: %w", err)}
		}
		if token != "" {
			return GitHubLoginSuccessMsg{Token: token}
		}
		return GitHubLoginPollMsg{Interval: 0}
	}
}
