package model

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"reflect"
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

// isNilInterface checks if an interface contains a nil concrete value.
// In Go, an interface is only nil if both its type and value are nil.
// An interface holding a nil pointer (e.g., (*Service)(nil)) is NOT nil.
func isNilInterface(i interface{}) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	return v.Kind() == reflect.Ptr && v.IsNil()
}

// tickCmd returns a command that sends a tick after the refresh interval
func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(autoRefreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// prTickCmd returns a command that sends a PR tick after the configured interval
// Returns nil if PR auto-refresh is disabled (interval = 0)
func (m *Model) prTickCmd() tea.Cmd {
	cfg, _ := config.Load()
	interval := 120 // Default: 2 minutes
	if cfg != nil {
		interval = cfg.PRRefreshInterval()
	}
	if interval <= 0 {
		return nil // Auto-refresh disabled
	}
	return tea.Tick(time.Duration(interval)*time.Second, func(t time.Time) tea.Msg {
		return prTickMsg(t)
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
		var githubInfo string
		remoteURL, err := jjSvc.GetGitRemoteURL(ctx)
		if err == nil {
			owner, repoName, err := github.ParseGitHubURL(remoteURL)
			if err == nil {
				// Track token source for diagnostics
				tokenSource := ""

				// Try to get GitHub token from environment variable first
				token := os.Getenv("GITHUB_TOKEN")
				if token != "" {
					tokenSource = "env:GITHUB_TOKEN"
				}

				// If not in env var, try to load from config
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

				// Build diagnostic info
				if token != "" {
					tokenPreview := token[:min(8, len(token))] + "..."
					githubInfo = fmt.Sprintf("repo=%s/%s token=%s(%s)", owner, repoName, tokenPreview, tokenSource)

					ghSvc, ghErr = github.NewServiceWithToken(owner, repoName, token)
					if ghErr != nil {
						// Service creation failed - this is still optional, so continue
						githubInfo += fmt.Sprintf(" error=%v", ghErr)
						ghSvc = nil
					}
				} else {
					githubInfo = fmt.Sprintf("repo=%s/%s (no token)", owner, repoName)
				}
			} else {
				githubInfo = fmt.Sprintf("remote=%s (not GitHub)", remoteURL)
			}
		} else {
			githubInfo = "no remote configured"
		}

		// Try to create ticket service based on configured provider
		ticketSvc, ticketErr := createTicketService()

		return servicesInitializedMsg{
			jjService:     jjSvc,
			githubService: ghSvc,
			ticketService: ticketSvc,
			ticketError:   ticketErr,
			githubInfo:    githubInfo,
			repository:    repo,
		}
	}
}

// createTicketService creates the appropriate ticket service based on configuration
// Priority: explicit TICKET_PROVIDER env var, then Codecks if configured, then Jira if configured
// Reads credentials from both environment variables AND config file (env vars take priority)
func createTicketService() (tickets.Service, error) {
	// Load config and set env vars from config if not already set
	// This allows config file credentials to work alongside env vars
	cfg, _ := config.Load()
	if cfg != nil {
		// Jira: set env vars from config if not already set
		if os.Getenv("JIRA_URL") == "" && cfg.JiraURL != "" {
			os.Setenv("JIRA_URL", cfg.JiraURL)
		}
		if os.Getenv("JIRA_USER") == "" && cfg.JiraUser != "" {
			os.Setenv("JIRA_USER", cfg.JiraUser)
		}
		if os.Getenv("JIRA_TOKEN") == "" && cfg.JiraToken != "" {
			os.Setenv("JIRA_TOKEN", cfg.JiraToken)
		}

		// Codecks: set env vars from config if not already set
		if os.Getenv("CODECKS_SUBDOMAIN") == "" && cfg.CodecksSubdomain != "" {
			os.Setenv("CODECKS_SUBDOMAIN", cfg.CodecksSubdomain)
		}
		if os.Getenv("CODECKS_TOKEN") == "" && cfg.CodecksToken != "" {
			os.Setenv("CODECKS_TOKEN", cfg.CodecksToken)
		}
		if os.Getenv("CODECKS_PROJECT") == "" && cfg.CodecksProject != "" {
			os.Setenv("CODECKS_PROJECT", cfg.CodecksProject)
		}

		// Ticket provider selection from config if not set in env
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
	default:
		// Auto-detect: try Codecks first (if configured), then Jira
		if codecks.IsConfigured() {
			svc, err := codecks.NewService()
			if err != nil {
				// Codecks configured but failed to connect - return the error
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

	// Capture service reference and diagnostic info for the closure
	ghSvc := m.githubService
	ghInfo := m.githubInfo

	return func() tea.Msg {
		// Build filter options from config
		cfg, _ := config.Load()
		filterOpts := github.PRFilterOptions{
			Limit:      100,
			ShowMerged: true,
			ShowClosed: true,
			OnlyMine:   false,
		}
		if cfg != nil {
			filterOpts.ShowMerged = cfg.ShowMergedPRs()
			filterOpts.ShowClosed = cfg.ShowClosedPRs()
			filterOpts.OnlyMine = cfg.OnlyMyPRs()
			filterOpts.Limit = cfg.PRLimit()
		}

		prs, err := ghSvc.GetPullRequestsWithOptions(context.Background(), filterOpts)
		if err != nil {
			// Include diagnostic info in the error for easier troubleshooting
			errMsg := fmt.Sprintf("failed to load PRs: %v", err)
			if ghInfo != "" {
				errMsg += fmt.Sprintf(" [%s]", ghInfo)
			}
			return errorMsg{Err: fmt.Errorf("%s", errMsg)}
		}

		return prsLoadedMsg{prs: prs}
	}
}

// mergePR merges the selected pull request
func (m *Model) mergePR(prNumber int) tea.Cmd {
	if m.githubService == nil {
		return nil
	}

	ghSvc := m.githubService

	return func() tea.Msg {
		err := ghSvc.MergePullRequest(context.Background(), prNumber)
		return prMergedMsg{prNumber: prNumber, err: err}
	}
}

// closePR closes the selected pull request without merging
func (m *Model) closePR(prNumber int) tea.Cmd {
	if m.githubService == nil {
		return nil
	}

	ghSvc := m.githubService

	return func() tea.Msg {
		err := ghSvc.ClosePullRequest(context.Background(), prNumber)
		return prClosedMsg{prNumber: prNumber, err: err}
	}
}

// loadTickets loads tickets from the configured ticket service
func (m *Model) loadTickets() tea.Cmd {
	// Check if ticketService is nil or contains a nil pointer
	// (Go interfaces can be non-nil while containing nil concrete values)
	if m.ticketService == nil || isNilInterface(m.ticketService) {
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

// loadTransitions loads the available transitions for the selected ticket
func (m *Model) loadTransitions() tea.Cmd {
	if m.ticketService == nil || m.selectedTicket < 0 || m.selectedTicket >= len(m.ticketList) {
		return func() tea.Msg {
			return transitionsLoadedMsg{transitions: nil}
		}
	}

	ticket := m.ticketList[m.selectedTicket]
	svc := m.ticketService

	return func() tea.Msg {
		transitions, err := svc.GetAvailableTransitions(context.Background(), ticket.Key)
		if err != nil {
			// Silently return empty transitions on error
			return transitionsLoadedMsg{transitions: nil}
		}
		return transitionsLoadedMsg{transitions: transitions}
	}
}

// transitionTicket executes a status transition on the selected ticket
func (m *Model) transitionTicket(transitionID string) tea.Cmd {
	if m.ticketService == nil || m.selectedTicket < 0 || m.selectedTicket >= len(m.ticketList) {
		return nil
	}

	ticket := m.ticketList[m.selectedTicket]
	svc := m.ticketService

	return func() tea.Msg {
		err := svc.TransitionTicket(context.Background(), ticket.Key, transitionID)
		if err != nil {
			return transitionCompletedMsg{ticketKey: ticket.Key, err: err}
		}
		// Get the transition name for status message
		transitions, _ := svc.GetAvailableTransitions(context.Background(), ticket.Key)
		var newStatus string
		for _, t := range transitions {
			if t.ID == transitionID {
				newStatus = t.Name
				break
			}
		}
		if newStatus == "" {
			newStatus = transitionID
		}
		return transitionCompletedMsg{ticketKey: ticket.Key, newStatus: newStatus}
	}
}

// transitionTicketToInProgress transitions a ticket to "In Progress" status
// This is used for auto-transition when creating a branch
func (m *Model) transitionTicketToInProgress(ticketKey string) tea.Cmd {
	if m.ticketService == nil {
		return nil
	}

	svc := m.ticketService

	return func() tea.Msg {
		// Get available transitions
		transitions, err := svc.GetAvailableTransitions(context.Background(), ticketKey)
		if err != nil {
			return transitionCompletedMsg{ticketKey: ticketKey, err: err}
		}

		// Find an "in progress" like transition
		// Must contain "progress" OR ("start" but NOT "not start")
		var inProgressID string
		for _, t := range transitions {
			lowerName := strings.ToLower(t.Name)
			isInProgress := strings.Contains(lowerName, "progress") ||
				(strings.Contains(lowerName, "start") && !strings.Contains(lowerName, "not start") && !strings.Contains(lowerName, "not_start"))
			if isInProgress {
				inProgressID = t.ID
				break
			}
		}

		if inProgressID == "" {
			// No "in progress" transition found - that's OK, just continue
			return transitionCompletedMsg{ticketKey: ticketKey, newStatus: ""}
		}

		err = svc.TransitionTicket(context.Background(), ticketKey, inProgressID)
		if err != nil {
			return transitionCompletedMsg{ticketKey: ticketKey, err: err}
		}

		return transitionCompletedMsg{ticketKey: ticketKey, newStatus: "In Progress"}
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

// undoOperation undoes the last jj operation and refreshes
func (m *Model) undoOperation() tea.Cmd {
	if m.jjService == nil {
		return nil
	}

	jjSvc := m.jjService
	return func() tea.Msg {
		err := jjSvc.Undo(context.Background())
		if err != nil {
			return errorMsg{Err: fmt.Errorf("undo failed: %w", err)}
		}
		return undoCompletedMsg{message: "Undo successful"}
	}
}

// redoOperation redoes the last undone operation (by undoing the undo)
func (m *Model) redoOperation() tea.Cmd {
	if m.jjService == nil {
		return nil
	}

	jjSvc := m.jjService
	return func() tea.Msg {
		err := jjSvc.Redo(context.Background())
		if err != nil {
			return errorMsg{Err: fmt.Errorf("redo failed: %w", err)}
		}
		return undoCompletedMsg{message: "Redo successful"}
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
	m.bookmarkJiraTicketTitle = ticket.Summary     // Store the ticket summary for PR title
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

// pollGitHubToken returns a command that polls for the GitHub access token in the background.
func (m *Model) pollGitHubToken() tea.Cmd {
	return func() tea.Msg {
		if m.githubDeviceCode == "" {
			return nil
		}

		token, err := github.PollForToken(m.githubDeviceCode)
		if err != nil {
			if err.Error() == "slow_down" {
				// Signal to handler to increase poll interval.
				return githubLoginPollMsg{interval: 5} // Use interval to signal slow down
			}
			return errorMsg{Err: fmt.Errorf("GitHub login failed: %w", err)}
		}

		if token != "" {
			return githubLoginSuccessMsg{token: token}
		}

		// Still waiting for user authorization. The handler will trigger the next poll.
		return githubLoginPollMsg{interval: 0}
	}
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

// loadBranches loads the list of local and remote branches
func (m *Model) loadBranches() tea.Cmd {
	if m.jjService == nil {
		return nil
	}

	jjSvc := m.jjService
	branchLimit := m.settingsBranchLimit
	return func() tea.Msg {
		// Get branches (limited by recency first, then stats calculated)
		// The limit is applied in ListBranches after sorting by commit timestamp
		branches, err := jjSvc.ListBranches(context.Background(), branchLimit)
		if err != nil {
			return branchesLoadedMsg{err: err}
		}

		// Sort branches for the tree visualization:
		// 1. Ahead branches first (most ahead at top) - these appear ABOVE trunk
		// 2. Behind/at-trunk branches next (least behind at top) - these appear BELOW trunk
		// Within each group: local before remote, then alphabetically
		sort.Slice(branches, func(i, j int) bool {
			iAhead := branches[i].Ahead > 0 && branches[i].Behind == 0
			jAhead := branches[j].Ahead > 0 && branches[j].Behind == 0

			// Ahead branches come before behind/at-trunk branches
			if iAhead != jAhead {
				return iAhead
			}

			if iAhead {
				// Both ahead: sort by ahead count descending (most ahead first)
				if branches[i].Ahead != branches[j].Ahead {
					return branches[i].Ahead > branches[j].Ahead
				}
			} else {
				// Both behind/at-trunk: local before remote
				if branches[i].IsLocal != branches[j].IsLocal {
					return branches[i].IsLocal
				}
				// Then by behind count ascending (closest to trunk first)
				if branches[i].Behind != branches[j].Behind {
					return branches[i].Behind < branches[j].Behind
				}
			}

			// Tiebreaker: alphabetically by name
			return branches[i].Name < branches[j].Name
		})
		return branchesLoadedMsg{branches: branches}
	}
}

// trackBranch starts tracking a remote branch
func (m *Model) trackBranch(branchName, remote string) tea.Cmd {
	if m.jjService == nil {
		return nil
	}

	jjSvc := m.jjService
	return func() tea.Msg {
		err := jjSvc.TrackBranch(context.Background(), branchName, remote)
		if err != nil {
			return branchActionMsg{action: "track", branch: branchName, err: err}
		}
		return branchActionMsg{action: "track", branch: branchName}
	}
}

// untrackBranch stops tracking a remote branch
func (m *Model) untrackBranch(branchName, remote string) tea.Cmd {
	if m.jjService == nil {
		return nil
	}

	jjSvc := m.jjService
	return func() tea.Msg {
		err := jjSvc.UntrackBranch(context.Background(), branchName, remote)
		if err != nil {
			return branchActionMsg{action: "untrack", branch: branchName, err: err}
		}
		return branchActionMsg{action: "untrack", branch: branchName}
	}
}

// restoreLocalBranch restores a deleted local branch from its tracked remote
func (m *Model) restoreLocalBranch(branchName, commitID string) tea.Cmd {
	if m.jjService == nil {
		return nil
	}

	jjSvc := m.jjService
	return func() tea.Msg {
		err := jjSvc.RestoreLocalBranch(context.Background(), branchName, commitID)
		if err != nil {
			return branchActionMsg{action: "restore", branch: branchName, err: err}
		}
		return branchActionMsg{action: "restore", branch: branchName}
	}
}

// deleteBranchBookmark deletes a local bookmark
func (m *Model) deleteBranchBookmark(branchName string) tea.Cmd {
	if m.jjService == nil {
		return nil
	}

	jjSvc := m.jjService
	return func() tea.Msg {
		err := jjSvc.DeleteBookmark(context.Background(), branchName)
		if err != nil {
			return branchActionMsg{action: "delete", branch: branchName, err: err}
		}
		return branchActionMsg{action: "delete", branch: branchName}
	}
}

// pushBranch pushes a local branch to remote
func (m *Model) pushBranch(branchName string) tea.Cmd {
	if m.jjService == nil {
		return nil
	}

	jjSvc := m.jjService
	return func() tea.Msg {
		err := jjSvc.PushBranch(context.Background(), branchName)
		if err != nil {
			return branchActionMsg{action: "push", branch: branchName, err: err}
		}
		return branchActionMsg{action: "push", branch: branchName}
	}
}

// fetchAllRemotes fetches from all remotes
func (m *Model) fetchAllRemotes() tea.Cmd {
	if m.jjService == nil {
		return nil
	}

	jjSvc := m.jjService
	return func() tea.Msg {
		err := jjSvc.FetchAllRemotes(context.Background())
		if err != nil {
			return branchActionMsg{action: "fetch", err: err}
		}
		return branchActionMsg{action: "fetch"}
	}
}
