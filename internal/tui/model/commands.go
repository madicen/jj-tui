package model

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/codecks"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/github"
	"github.com/madicen/jj-tui/internal/jira"
	"github.com/madicen/jj-tui/internal/jj"
	"github.com/madicen/jj-tui/internal/models"
	"github.com/madicen/jj-tui/internal/tickets"
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

		// Get current directory for error reporting
		cwd, _ := os.Getwd()

		// Try to create jj service
		jjSvc, err := jj.NewService("")
		if err != nil {
			// Check if this is a "not a jj repository" error
			notJJRepo := strings.Contains(err.Error(), "not a jujutsu repository")
			return errorMsg{Err: err, NotJJRepo: notJJRepo, CurrentPath: cwd}
		}

		// Load repository data
		repo, err := jjSvc.GetRepository(ctx)
		if err != nil {
			return errorMsg{Err: err}
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

		// Try to create ticket service based on configured provider
		ticketSvc, ticketErr := createTicketService()

		return servicesInitializedMsg{
			jjService:     jjSvc,
			githubService: ghSvc,
			ticketService: ticketSvc,
			ticketError:   ticketErr,
			repository:    repo,
		}
	}
}

// createTicketService creates the appropriate ticket service based on configuration
// Priority: explicit TICKET_PROVIDER env var, then Codecks if configured, then Jira if configured
func createTicketService() (tickets.Service, error) {
	provider := os.Getenv("TICKET_PROVIDER")

	switch provider {
	case "codecks":
		if codecks.IsConfigured() {
			return codecks.NewService()
		}
		return nil, fmt.Errorf("TICKET_PROVIDER=codecks but CODECKS_SUBDOMAIN or CODECKS_TOKEN not set")
	case "jira":
		if jira.IsConfigured() {
			return jira.NewService()
		}
		return nil, fmt.Errorf("TICKET_PROVIDER=jira but Jira env vars not set")
	default:
		// Auto-detect: try Codecks first (if configured), then Jira
		if codecks.IsConfigured() {
			svc, err := codecks.NewService()
			if err != nil {
				// Codecks configured but failed to connect - return the error
				return nil, fmt.Errorf("Codecks: %w", err)
			}
			return svc, nil
		}
		if jira.IsConfigured() {
			svc, err := jira.NewService()
			if err != nil {
				return nil, fmt.Errorf("Jira: %w", err)
			}
			return svc, nil
		}
	}
	return nil, nil // No ticket service configured
}

// loadRepository loads/refreshes repository data
func (m *Model) loadRepository() tea.Cmd {
	if m.jjService == nil {
		return m.initializeServices()
	}

	return func() tea.Msg {
		repo, err := m.jjService.GetRepository(context.Background())
		if err != nil {
			return errorMsg{Err: err}
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
			return errorMsg{Err: fmt.Errorf("failed to load PRs: %w", err)}
		}

		// Apply PR filters from config
		cfg, _ := config.Load()
		if cfg != nil {
			var filtered []models.GitHubPR
			showMerged := cfg.ShowMergedPRs()
			showClosed := cfg.ShowClosedPRs()

			for _, pr := range prs {
				// Skip merged PRs if not showing them
				if !showMerged && pr.State == "merged" {
					continue
				}
				// Skip closed PRs if not showing them
				if !showClosed && pr.State == "closed" {
					continue
				}
				filtered = append(filtered, pr)
			}
			prs = filtered
		}

		return prsLoadedMsg{prs: prs}
	}
}

// loadTickets loads tickets from the configured ticket service
func (m *Model) loadTickets() tea.Cmd {
	if m.ticketService == nil {
		return func() tea.Msg {
			return ticketsLoadedMsg{tickets: []tickets.Ticket{}}
		}
	}

	// Capture service reference for the closure
	svc := m.ticketService

	return func() tea.Msg {
		ticketList, err := svc.GetAssignedTickets(context.Background())
		if err != nil {
			return errorMsg{Err: fmt.Errorf("failed to load tickets: %w", err)}
		}

		// Apply status filters from config
		cfg, _ := config.Load()
		if cfg != nil {
			// Build excluded statuses set based on provider
			excludedStatuses := make(map[string]bool)
			var excludedStr string
			if svc.GetProviderName() == "Jira" {
				excludedStr = cfg.JiraExcludedStatuses
			} else if svc.GetProviderName() == "Codecks" {
				excludedStr = cfg.CodecksExcludedStatuses
			}

			if excludedStr != "" {
				for _, status := range strings.Split(excludedStr, ",") {
					status = strings.TrimSpace(strings.ToLower(status))
					if status != "" {
						excludedStatuses[status] = true
					}
				}
			}

			// Filter tickets
			if len(excludedStatuses) > 0 {
				var filtered []tickets.Ticket
				for _, ticket := range ticketList {
					statusLower := strings.ToLower(ticket.Status)
					if !excludedStatuses[statusLower] {
						filtered = append(filtered, ticket)
					}
				}
				ticketList = filtered
			}
		}

		// Sort tickets by DisplayKey descending (most recent first)
		sort.Slice(ticketList, func(i, j int) bool {
			return ticketList[i].DisplayKey > ticketList[j].DisplayKey
		})

		return ticketsLoadedMsg{tickets: ticketList}
	}
}

// loadChangedFiles loads the changed files for a commit
func (m *Model) loadChangedFiles(commitID string) tea.Cmd {
	if m.jjService == nil || commitID == "" {
		return nil
	}

	return func() tea.Msg {
		files, err := m.jjService.GetChangedFiles(context.Background(), commitID)
		if err != nil {
			// Silently ignore errors for changed files
			return changedFilesLoadedMsg{commitID: commitID, files: nil}
		}
		return changedFilesLoadedMsg{commitID: commitID, files: files}
	}
}

// startBookmarkFromTicket opens the bookmark creation screen pre-populated with the ticket key
func (m *Model) startBookmarkFromTicket(ticket tickets.Ticket) {
	// Use DisplayKey (short ID) if available, otherwise fall back to Key
	keyForBookmark := ticket.Key
	if ticket.DisplayKey != "" {
		keyForBookmark = ticket.DisplayKey
	}

	// Format bookmark name as "KEY-Title" with spaces replaced by hyphens
	// and invalid characters removed
	bookmarkName := formatBookmarkName(keyForBookmark, ticket.Summary)
	m.bookmarkNameInput.SetValue(bookmarkName)
	m.bookmarkNameInput.Focus()
	m.bookmarkNameInput.Width = m.width - 10

	// Mark that this is coming from ticket service (will create new branch from main)
	m.bookmarkFromJira = true // Reusing this flag for any ticket provider
	m.bookmarkJiraTicketKey = ticket.Key
	m.bookmarkJiraTicketTitle = ticket.Summary // Store the ticket summary for PR title
	m.bookmarkTicketDisplayKey = ticket.DisplayKey // Store short ID for commit messages
	m.bookmarkCommitIdx = -1                       // -1 means create new branch from main
	m.existingBookmarks = nil                      // Don't show existing bookmarks for ticket flow
	m.selectedBookmarkIdx = -1

	m.viewMode = ViewCreateBookmark
	m.statusMessage = fmt.Sprintf("Create bookmark for %s (will create new branch from main)", ticket.Key)
}

// formatBookmarkName creates a valid bookmark name from a ticket key and summary
// Format: "KEY-Title" with spaces replaced by hyphens and invalid chars removed
func formatBookmarkName(key, summary string) string {
	// Remove any characters that aren't valid for bookmark names
	// Valid: a-z, A-Z, 0-9, -, _, /
	invalidChars := regexp.MustCompile(`[^a-zA-Z0-9\-_/]`)

	// Sanitize the key (e.g., strip "$" from Codecks short IDs like "$12u")
	sanitizedKey := invalidChars.ReplaceAllString(key, "")
	sanitizedKey = strings.Trim(sanitizedKey, "-")

	// Replace spaces with hyphens in title
	title := strings.ReplaceAll(summary, " ", "-")

	// Remove invalid characters from title
	title = invalidChars.ReplaceAllString(title, "")

	// Remove multiple consecutive hyphens
	multipleHyphens := regexp.MustCompile(`-+`)
	title = multipleHyphens.ReplaceAllString(title, "-")

	// Trim leading/trailing hyphens from title
	title = strings.Trim(title, "-")

	// Combine key and title
	if sanitizedKey != "" && title != "" {
		return sanitizedKey + "-" + title
	} else if sanitizedKey != "" {
		return sanitizedKey
	} else if title != "" {
		return title
	}
	return "bookmark"
}

// startGitHubLogin initiates the GitHub Device Flow authentication
func (m *Model) startGitHubLogin() tea.Cmd {
	return func() tea.Msg {
		deviceResp, err := github.StartDeviceFlow()
		if err != nil {
			return errorMsg{Err: fmt.Errorf("failed to start GitHub login: %w", err)}
		}
		return githubDeviceFlowStartedMsg{
			deviceCode:      deviceResp.DeviceCode,
			userCode:        deviceResp.UserCode,
			verificationURL: deviceResp.VerificationURI,
			interval:        deviceResp.Interval,
		}
	}
}

// pollGitHubToken polls GitHub for the access token
func (m *Model) pollGitHubToken(interval int) tea.Cmd {
	return tea.Tick(time.Duration(interval)*time.Second, func(t time.Time) tea.Msg {
		if m.githubDeviceCode == "" {
			return nil
		}

		token, err := github.PollForToken(m.githubDeviceCode)
		if err != nil {
			if err.Error() == "slow_down" {
				// Increase interval and continue polling
				return githubLoginPollMsg{interval: interval + 5}
			}
			return errorMsg{Err: fmt.Errorf("GitHub login failed: %w", err)}
		}

		if token != "" {
			return githubLoginSuccessMsg{token: token}
		}

		// Still waiting for user authorization
		return githubLoginPollMsg{interval: interval}
	})
}

// runJJInit runs jj git init to initialize a new repository
func (m *Model) runJJInit() tea.Cmd {
	return func() tea.Msg {
		// Run jj git init in the current directory
		cmd := exec.Command("jj", "git", "init")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return errorMsg{
				Err:       fmt.Errorf("failed to initialize repository: %s", strings.TrimSpace(string(output))),
				NotJJRepo: true,
			}
		}

		// Try to track main@origin (common default branch)
		// This makes the repo more useful if it's a clone with remote tracking
		trackCmd := exec.Command("jj", "bookmark", "track", "main@origin")
		trackOutput, trackErr := trackCmd.CombinedOutput()
		_ = trackOutput // Ignore output - tracking may fail if main@origin doesn't exist
		_ = trackErr    // Ignore error - this is optional, repo init still succeeded

		return jjInitSuccessMsg{}
	}
}

