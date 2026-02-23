package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v66/github"
	"github.com/madicen/jj-tui/internal/tickets"
)

// IssuesService handles GitHub Issues as a ticket provider
type IssuesService struct {
	client   *github.Client
	owner    string
	repo     string
	username string // cached authenticated username
}

// NewIssuesService creates a new GitHub Issues service from an existing GitHub service
func NewIssuesService(ghService *Service) (*IssuesService, error) {
	if ghService == nil {
		return nil, fmt.Errorf("GitHub service is required")
	}

	return &IssuesService{
		client:   ghService.client,
		owner:    ghService.owner,
		repo:     ghService.repo,
		username: ghService.username,
	}, nil
}

// NewIssuesServiceWithToken creates a new GitHub Issues service with a provided token
func NewIssuesServiceWithToken(owner, repo, token string) (*IssuesService, error) {
	ghService, err := NewServiceWithToken(owner, repo, token)
	if err != nil {
		return nil, err
	}
	return NewIssuesService(ghService)
}

// GetAssignedTickets returns GitHub issues assigned to the current user
func (s *IssuesService) GetAssignedTickets(ctx context.Context) ([]tickets.Ticket, error) {
	// Get authenticated username if not cached
	if s.username == "" {
		user, _, err := s.client.Users.Get(ctx, "")
		if err != nil {
			return nil, fmt.Errorf("failed to get authenticated user: %w", err)
		}
		s.username = user.GetLogin()
	}

	// List issues assigned to the current user
	opts := &github.IssueListByRepoOptions{
		Assignee:  s.username,
		State:     "open", // Default to open issues
		Sort:      "updated",
		Direction: "desc",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var allTickets []tickets.Ticket
	for {
		issues, resp, err := s.client.Issues.ListByRepo(ctx, s.owner, s.repo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list issues: %w", err)
		}

		for _, issue := range issues {
			// Skip pull requests (GitHub API returns PRs as issues)
			if issue.IsPullRequest() {
				continue
			}

			ticket := s.issueToTicket(issue)
			allTickets = append(allTickets, ticket)
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allTickets, nil
}

// GetTicket returns a single issue by number
func (s *IssuesService) GetTicket(ctx context.Context, key string) (*tickets.Ticket, error) {
	// Parse the issue number from the key (e.g., "#123" or "123")
	key = strings.TrimPrefix(key, "#")
	var issueNumber int
	if _, err := fmt.Sscanf(key, "%d", &issueNumber); err != nil {
		return nil, fmt.Errorf("invalid issue number: %s", key)
	}

	issue, _, err := s.client.Issues.Get(ctx, s.owner, s.repo, issueNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue #%d: %w", issueNumber, err)
	}

	ticket := s.issueToTicket(issue)
	return &ticket, nil
}

// GetTicketURL returns the browser URL for an issue
func (s *IssuesService) GetTicketURL(ticket tickets.Ticket) string {
	return fmt.Sprintf("https://github.com/%s/%s/issues/%s", s.owner, s.repo, strings.TrimPrefix(ticket.Key, "#"))
}

// GetProviderName returns the name of this provider
func (s *IssuesService) GetProviderName() string {
	return "GitHub Issues"
}

// GetAvailableTransitions returns available state transitions for an issue
// GitHub issues only have two states: open and closed
func (s *IssuesService) GetAvailableTransitions(ctx context.Context, ticketKey string) ([]tickets.Transition, error) {
	// Parse the issue number to check current state
	key := strings.TrimPrefix(ticketKey, "#")
	var issueNumber int
	if _, err := fmt.Sscanf(key, "%d", &issueNumber); err != nil {
		return nil, fmt.Errorf("invalid issue number: %s", ticketKey)
	}

	issue, _, err := s.client.Issues.Get(ctx, s.owner, s.repo, issueNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}

	var transitions []tickets.Transition
	state := issue.GetState()

	if state == "open" {
		transitions = append(transitions, tickets.Transition{
			ID:   "closed",
			Name: "Close Issue",
		})
	} else {
		transitions = append(transitions, tickets.Transition{
			ID:   "open",
			Name: "Reopen Issue",
		})
	}

	return transitions, nil
}

// TransitionTicket changes the issue's state
func (s *IssuesService) TransitionTicket(ctx context.Context, ticketKey string, transitionID string) error {
	// Parse the issue number
	key := strings.TrimPrefix(ticketKey, "#")
	var issueNumber int
	if _, err := fmt.Sscanf(key, "%d", &issueNumber); err != nil {
		return fmt.Errorf("invalid issue number: %s", ticketKey)
	}

	// Validate transition ID
	if transitionID != "open" && transitionID != "closed" {
		return fmt.Errorf("invalid transition: %s (must be 'open' or 'closed')", transitionID)
	}

	// Update the issue state
	issueRequest := &github.IssueRequest{
		State: github.String(transitionID),
	}

	_, _, err := s.client.Issues.Edit(ctx, s.owner, s.repo, issueNumber, issueRequest)
	if err != nil {
		return fmt.Errorf("failed to transition issue: %w", err)
	}

	return nil
}

// issueToTicket converts a GitHub issue to a tickets.Ticket
func (s *IssuesService) issueToTicket(issue *github.Issue) tickets.Ticket {
	// Capitalize status: "open" -> "Open", "closed" -> "Closed"
	state := issue.GetState()
	if len(state) > 0 {
		state = strings.ToUpper(state[:1]) + state[1:]
	}

	ticket := tickets.Ticket{
		Key:        fmt.Sprintf("#%d", issue.GetNumber()),
		DisplayKey: fmt.Sprintf("#%d", issue.GetNumber()),
		Summary:    issue.GetTitle(),
		Status:     state,
		Type:       "Issue",
	}

	// Get priority from labels (if any label contains "priority")
	for _, label := range issue.Labels {
		name := strings.ToLower(label.GetName())
		if strings.Contains(name, "priority") || strings.Contains(name, "p0") ||
			strings.Contains(name, "p1") || strings.Contains(name, "p2") ||
			strings.Contains(name, "critical") || strings.Contains(name, "high") ||
			strings.Contains(name, "medium") || strings.Contains(name, "low") {
			ticket.Priority = label.GetName()
			break
		}
	}

	// Get type from labels (bug, feature, enhancement, etc.)
	for _, label := range issue.Labels {
		name := strings.ToLower(label.GetName())
		if strings.Contains(name, "bug") {
			ticket.Type = "Bug"
			break
		} else if strings.Contains(name, "feature") || strings.Contains(name, "enhancement") {
			ticket.Type = "Feature"
			break
		} else if strings.Contains(name, "documentation") || strings.Contains(name, "docs") {
			ticket.Type = "Documentation"
			break
		}
	}

	// Description (body)
	if issue.Body != nil {
		ticket.Description = *issue.Body
	}

	return ticket
}

// Ensure IssuesService implements tickets.Service
var _ tickets.Service = (*IssuesService)(nil)

