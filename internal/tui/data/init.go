package data

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/codecks"
	"github.com/madicen/jj-tui/internal/integrations/github"
	"github.com/madicen/jj-tui/internal/integrations/jira"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/mock"
	"github.com/madicen/jj-tui/internal/tickets"
)

// InitializeServices sets up the jj service and loads repository data first (RepoReadyMsg),
// so the UI can show the graph immediately. The model then runs LoadAuxServicesCmd to load
// GitHub and ticket services in the background (AuxServicesReadyMsg).
// Returns a cmd that sends RepoReadyMsg or InitErrorMsg.
func InitializeServices(demoMode bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		cwd, _ := os.Getwd()

		jjSvc, err := jj.NewService("")
		if err != nil {
			notJJRepo := strings.Contains(err.Error(), "not a jujutsu repository")
			return InitErrorMsg{Err: err, NotJJRepo: notJJRepo, CurrentPath: cwd}
		}

		cfg, _ := config.Load()
		revset := ""
		if cfg != nil {
			revset = cfg.GraphRevset
		}

		// Run the two slow jj operations in parallel so we can show the UI as soon as both complete.
		var repo *internal.Repository
		var repoErr error
		var remoteURL string
		var remoteErr error
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			repo, repoErr = jjSvc.GetRepository(ctx, revset)
		}()
		go func() {
			defer wg.Done()
			remoteURL, remoteErr = jjSvc.GetGitRemoteURL(ctx)
		}()
		wg.Wait()

		if repoErr != nil {
			return InitErrorMsg{Err: repoErr}
		}

		owner, repoName := "", ""
		githubInfoFromURL := "no remote configured"
		if remoteErr == nil && remoteURL != "" {
			owner, repoName, err = github.ParseGitHubURL(remoteURL)
			if err == nil {
				githubInfoFromURL = fmt.Sprintf("repo=%s/%s", owner, repoName)
			} else {
				githubInfoFromURL = fmt.Sprintf("remote=%s (not GitHub)", remoteURL)
			}
		}

		return RepoReadyMsg{
			JJService:         jjSvc,
			Repository:        repo,
			DemoMode:          demoMode,
			Owner:             owner,
			RepoName:          repoName,
			GitHubInfoFromURL: githubInfoFromURL,
		}
	}
}

// LoadAuxServicesCmd returns a cmd that loads GitHub and ticket services (after RepoReadyMsg).
// Run this after handling RepoReadyMsg so the graph is already visible; GitHub/ticket load in the background.
func LoadAuxServicesCmd(demoMode bool, owner, repoName, githubInfoFromURL string) tea.Cmd {
	return func() tea.Msg {
		if demoMode {
			cfg, _ := config.Load()
			ticketProvider := "jira"
			if cfg != nil && cfg.TicketProvider != "" {
				ticketProvider = cfg.TicketProvider
			}
			return AuxServicesReadyMsg{
				GitHubService: nil,
				TicketService: mock.NewTicketService(ticketProvider),
				TicketError:   nil,
				GitHubInfo:    "demo mode (mock services)",
			}
		}

		var ghSvc *github.Service
		githubInfo := githubInfoFromURL
		if owner != "" && repoName != "" {
			tokenSource := ""
			token := os.Getenv("GITHUB_TOKEN")
			if token != "" {
				tokenSource = "env:GITHUB_TOKEN"
			}
			if token == "" {
				cfg, _ := config.Load()
				if cfg != nil && cfg.GitHubToken != "" {
					token = cfg.GitHubToken
					if cfg.LoadedFrom() != "" {
						tokenSource = fmt.Sprintf("config:%s", cfg.LoadedFrom())
					} else {
						tokenSource = "config"
					}
				}
			}
			if token != "" {
				tokenPreview := token[:min(8, len(token))] + "..."
				githubInfo = fmt.Sprintf("repo=%s/%s token=%s(%s)", owner, repoName, tokenPreview, tokenSource)
				ghSvc, _ = github.NewServiceWithToken(owner, repoName, token)
			} else {
				githubInfo = fmt.Sprintf("repo=%s/%s (no token)", owner, repoName)
			}
		}

		ticketSvc, ticketErr := CreateTicketService(owner, repoName)
		return AuxServicesReadyMsg{
			GitHubService: ghSvc,
			TicketService: ticketSvc,
			TicketError:   ticketErr,
			GitHubInfo:    githubInfo,
		}
	}
}

// CreateTicketService creates the appropriate ticket service based on configuration.
func CreateTicketService(owner, repo string) (tickets.Service, error) {
	cfg, _ := config.Load()
	if cfg != nil {
		if os.Getenv("JIRA_URL") == "" && cfg.JiraURL != "" {
			os.Setenv("JIRA_URL", cfg.JiraURL)
		}
		if os.Getenv("JIRA_USER") == "" && cfg.JiraUser != "" {
			os.Setenv("JIRA_USER", cfg.JiraUser)
		}
		if os.Getenv("JIRA_TOKEN") == "" && cfg.JiraToken != "" {
			os.Setenv("JIRA_TOKEN", cfg.JiraToken)
		}
		if os.Getenv("CODECKS_SUBDOMAIN") == "" && cfg.CodecksSubdomain != "" {
			os.Setenv("CODECKS_SUBDOMAIN", cfg.CodecksSubdomain)
		}
		if os.Getenv("CODECKS_TOKEN") == "" && cfg.CodecksToken != "" {
			os.Setenv("CODECKS_TOKEN", cfg.CodecksToken)
		}
		if os.Getenv("CODECKS_PROJECT") == "" && cfg.CodecksProject != "" {
			os.Setenv("CODECKS_PROJECT", cfg.CodecksProject)
		}
		if os.Getenv("TICKET_PROVIDER") == "" && cfg.TicketProvider != "" {
			os.Setenv("TICKET_PROVIDER", cfg.TicketProvider)
		}
	}

	provider := os.Getenv("TICKET_PROVIDER")
	switch provider {
	case "codecks":
		if codecks.IsConfigured() {
			svc, err := codecks.NewService()
			if err != nil {
				return nil, fmt.Errorf("codecks: %w", err)
			}
			return svc, nil
		}
		return nil, fmt.Errorf("TICKET_PROVIDER=codecks but CODECKS_SUBDOMAIN or CODECKS_TOKEN not set")
	case "jira":
		if jira.IsConfigured() {
			svc, err := jira.NewService()
			if err != nil {
				return nil, fmt.Errorf("jira: %w", err)
			}
			return svc, nil
		}
		return nil, fmt.Errorf("TICKET_PROVIDER=jira but Jira env vars not set")
	case "github_issues":
		token := os.Getenv("GITHUB_TOKEN")
		if token == "" && cfg != nil {
			token = cfg.GitHubToken
		}
		if token == "" {
			return nil, fmt.Errorf("TICKET_PROVIDER=github_issues but GITHUB_TOKEN not set")
		}
		if owner == "" || repo == "" {
			return nil, fmt.Errorf("TICKET_PROVIDER=github_issues but not in a GitHub repository")
		}
		svc, err := github.NewIssuesServiceWithToken(owner, repo, token)
		if err != nil {
			return nil, fmt.Errorf("github_issues: %w", err)
		}
		return svc, nil
	default:
		if codecks.IsConfigured() {
			svc, err := codecks.NewService()
			if err != nil {
				return nil, fmt.Errorf("codecks: %w", err)
			}
			return svc, nil
		}
		if jira.IsConfigured() {
			svc, err := jira.NewService()
			if err != nil {
				return nil, fmt.Errorf("jira: %w", err)
			}
			return svc, nil
		}
	}
	return nil, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
