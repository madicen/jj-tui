package tui

import (
	"context"
	"fmt"
	"regexp"
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
		var ghErr error
		remoteURL, err := jjSvc.GetGitRemoteURL(ctx)
		if err == nil {
			owner, repoName, err := github.ParseGitHubURL(remoteURL)
			if err == nil {
				ghSvc, ghErr = github.NewService(owner, repoName)
				// ghErr will be non-nil if GITHUB_TOKEN is not set
				_ = ghErr // Suppress unused error - GitHub is optional
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
			// Return a status message to show GitHub isn't connected
			return prsLoadedMsg{prs: []models.GitHubPR{}}
		}
	}

	// Capture service reference for the closure
	ghSvc := m.githubService

	return func() tea.Msg {
		prs, err := ghSvc.GetPullRequests(context.Background())
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

// startBookmarkFromJiraTicket opens the bookmark creation screen pre-populated with the Jira ticket key
func (m *Model) startBookmarkFromJiraTicket(ticket jira.Ticket) {
	// Format bookmark name as "KEY-Title" with spaces replaced by hyphens
	// and invalid characters removed
	bookmarkName := formatBookmarkName(ticket.Key, ticket.Summary)
	m.bookmarkNameInput.SetValue(bookmarkName)
	m.bookmarkNameInput.Focus()
	m.bookmarkNameInput.Width = m.width - 10

	// Mark that this is coming from Jira (will create new branch from main)
	m.bookmarkFromJira = true
	m.bookmarkJiraTicketKey = ticket.Key
	m.bookmarkJiraTicketTitle = ticket.Summary // Store the ticket summary for PR title
	m.bookmarkCommitIdx = -1                   // -1 means create new branch from main
	m.existingBookmarks = nil                  // Don't show existing bookmarks for Jira flow
	m.selectedBookmarkIdx = -1

	m.viewMode = ViewCreateBookmark
	m.statusMessage = fmt.Sprintf("Create bookmark for %s (will create new branch from main)", ticket.Key)
}

// formatBookmarkName creates a valid bookmark name from a Jira ticket key and summary
// Format: "KEY-Title" with spaces replaced by hyphens and invalid chars removed
func formatBookmarkName(key, summary string) string {
	// Replace spaces with hyphens
	title := strings.ReplaceAll(summary, " ", "-")

	// Remove any characters that aren't valid for bookmark names
	// Valid: a-z, A-Z, 0-9, -, _, /
	invalidChars := regexp.MustCompile(`[^a-zA-Z0-9\-_/]`)
	title = invalidChars.ReplaceAllString(title, "")

	// Remove multiple consecutive hyphens
	multipleHyphens := regexp.MustCompile(`-+`)
	title = multipleHyphens.ReplaceAllString(title, "-")

	// Trim leading/trailing hyphens from title
	title = strings.Trim(title, "-")

	// Combine key and title
	if title != "" {
		return key + "-" + title
	}
	return key
}

