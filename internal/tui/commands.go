package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen-utilities/jj-tui/v2/internal/github"
	"github.com/madicen-utilities/jj-tui/v2/internal/jira"
	"github.com/madicen-utilities/jj-tui/v2/internal/jj"
	"github.com/madicen-utilities/jj-tui/v2/internal/models"
)

// tickCmd returns a command that sends a tick after the refresh interval
func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(autoRefreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// initializeServices sets up the jj service, GitHub service, and loads initial data
func (m *Model) initializeServices() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Try to create jj service
		jjSvc, err := jj.NewService("")
		if err != nil {
			return errorMsg{err: err}
		}

		// Load repository data
		repo, err := jjSvc.GetRepository(ctx)
		if err != nil {
			return errorMsg{err: err}
		}

		// Try to create GitHub service (optional - won't fail if no token)
		var ghSvc *github.Service
		remoteURL, err := jjSvc.GetGitRemoteURL(ctx)
		if err == nil {
			owner, repoName, err := github.ParseGitHubURL(remoteURL)
			if err == nil {
				ghSvc, _ = github.NewService(owner, repoName) // Ignore error if no GITHUB_TOKEN
			}
		}

		// Try to create Jira service (optional - won't fail if env vars not set)
		var jiraSvc *jira.Service
		if jira.IsConfigured() {
			jiraSvc, _ = jira.NewService() // Ignore error
		}

		return servicesInitializedMsg{
			jjService:     jjSvc,
			githubService: ghSvc,
			jiraService:   jiraSvc,
			repository:    repo,
		}
	}
}

// loadRepository loads/refreshes repository data
func (m *Model) loadRepository() tea.Cmd {
	if m.jjService == nil {
		return m.initializeServices()
	}

	return func() tea.Msg {
		repo, err := m.jjService.GetRepository(context.Background())
		if err != nil {
			return errorMsg{err: err}
		}
		return repositoryLoadedMsg{repository: repo}
	}
}

// loadRepositorySilent loads repository data without updating status message
func (m *Model) loadRepositorySilent() tea.Cmd {
	if m.jjService == nil {
		return nil
	}

	return func() tea.Msg {
		repo, err := m.jjService.GetRepository(context.Background())
		if err != nil {
			// Silently ignore errors during background refresh
			return nil
		}
		return silentRepositoryLoadedMsg{repository: repo}
	}
}

// loadPRs loads pull requests from GitHub
func (m *Model) loadPRs() tea.Cmd {
	if m.githubService == nil {
		return func() tea.Msg {
			return prsLoadedMsg{prs: []models.GitHubPR{}}
		}
	}

	return func() tea.Msg {
		prs, err := m.githubService.GetPullRequests(context.Background())
		if err != nil {
			return errorMsg{err: fmt.Errorf("failed to load PRs: %w", err)}
		}
		return prsLoadedMsg{prs: prs}
	}
}

// loadJiraTickets loads Jira tickets assigned to the user
func (m *Model) loadJiraTickets() tea.Cmd {
	if m.jiraService == nil {
		return func() tea.Msg {
			return jiraTicketsLoadedMsg{tickets: []jira.Ticket{}}
		}
	}

	return func() tea.Msg {
		tickets, err := m.jiraService.GetAssignedTickets(context.Background())
		if err != nil {
			return errorMsg{err: fmt.Errorf("failed to load Jira tickets: %w", err)}
		}
		return jiraTicketsLoadedMsg{tickets: tickets}
	}
}

// createBranchFromTicket creates a new branch from a Jira ticket
func (m *Model) createBranchFromTicket(ticket jira.Ticket) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Create branch name from ticket key (e.g., "PROJ-123")
		branchName := strings.ToLower(ticket.Key)

		// Create a new commit with jj new
		if err := m.jjService.NewCommit(ctx); err != nil {
			return errorMsg{err: fmt.Errorf("failed to create new commit: %w", err)}
		}

		// Create a bookmark with the ticket key as the name
		if err := m.jjService.CreateNewBranch(ctx, branchName); err != nil {
			return errorMsg{err: fmt.Errorf("failed to create bookmark: %w", err)}
		}

		// Set the description to the ticket summary
		description := fmt.Sprintf("%s: %s", ticket.Key, ticket.Summary)
		if err := m.jjService.DescribeCommit(ctx, "@", description); err != nil {
			return errorMsg{err: fmt.Errorf("failed to set description: %w", err)}
		}

		return bookmarkCreatedMsg{
			ticketKey:  ticket.Key,
			branchName: branchName,
		}
	}
}

