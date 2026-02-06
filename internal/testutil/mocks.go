// Package testutil provides mock implementations for testing
package testutil

import (
	"context"

	"github.com/madicen/jj-tui/internal/models"
	"github.com/madicen/jj-tui/internal/tickets"
)

// MockGitHubService mocks GitHub API interactions
type MockGitHubService struct {
	PRs          []models.GitHubPR
	CreatePRFunc func(ctx context.Context, req *models.CreatePRRequest) (*models.GitHubPR, error)
	GetPRsFunc   func(ctx context.Context) ([]models.GitHubPR, error)
	Owner        string
	Repo         string
}

// GetPullRequests returns mock PRs
func (m *MockGitHubService) GetPullRequests(ctx context.Context) ([]models.GitHubPR, error) {
	if m.GetPRsFunc != nil {
		return m.GetPRsFunc(ctx)
	}
	return m.PRs, nil
}

// CreatePullRequest creates a mock PR
func (m *MockGitHubService) CreatePullRequest(ctx context.Context, req *models.CreatePRRequest) (*models.GitHubPR, error) {
	if m.CreatePRFunc != nil {
		return m.CreatePRFunc(ctx, req)
	}
	pr := &models.GitHubPR{
		Number:     len(m.PRs) + 1,
		Title:      req.Title,
		URL:        "https://github.com/" + m.Owner + "/" + m.Repo + "/pull/" + string(rune(len(m.PRs)+1)),
		State:      "open",
		BaseBranch: req.BaseBranch,
		HeadBranch: req.HeadBranch,
	}
	m.PRs = append(m.PRs, *pr)
	return pr, nil
}

// NewMockGitHubService creates a mock GitHub service with sample data
func NewMockGitHubService() *MockGitHubService {
	return &MockGitHubService{
		Owner: "testowner",
		Repo:  "testrepo",
		PRs: []models.GitHubPR{
			{Number: 1, Title: "Feature: Add login", State: "open", BaseBranch: "main", HeadBranch: "feature/login", URL: "https://github.com/testowner/testrepo/pull/1"},
			{Number: 2, Title: "Fix: Bug in parser", State: "merged", BaseBranch: "main", HeadBranch: "fix/parser", URL: "https://github.com/testowner/testrepo/pull/2"},
			{Number: 3, Title: "Chore: Update deps", State: "closed", BaseBranch: "main", HeadBranch: "chore/deps", URL: "https://github.com/testowner/testrepo/pull/3"},
		},
	}
}

// MockTicketService mocks ticket provider interactions (Jira/Codecks)
type MockTicketService struct {
	Tickets      []tickets.Ticket
	ProviderName string
	BaseURL      string
	GetTicketsFunc func(ctx context.Context) ([]tickets.Ticket, error)
}

// GetAssignedTickets returns mock tickets
func (m *MockTicketService) GetAssignedTickets(ctx context.Context) ([]tickets.Ticket, error) {
	if m.GetTicketsFunc != nil {
		return m.GetTicketsFunc(ctx)
	}
	return m.Tickets, nil
}

// GetTicket returns a specific mock ticket
func (m *MockTicketService) GetTicket(ctx context.Context, key string) (*tickets.Ticket, error) {
	for _, t := range m.Tickets {
		if t.Key == key {
			return &t, nil
		}
	}
	return nil, nil
}

// GetTicketURL returns a mock URL for a ticket
func (m *MockTicketService) GetTicketURL(ticket tickets.Ticket) string {
	return m.BaseURL + "/ticket/" + ticket.Key
}

// GetProviderName returns the provider name
func (m *MockTicketService) GetProviderName() string {
	return m.ProviderName
}

// GetAvailableTransitions returns mock transitions
func (m *MockTicketService) GetAvailableTransitions(ctx context.Context, ticketKey string) ([]tickets.Transition, error) {
	// Return common transitions for testing
	return []tickets.Transition{
		{ID: "21", Name: "In Progress"},
		{ID: "31", Name: "Done"},
	}, nil
}

// TransitionTicket mocks transitioning a ticket
func (m *MockTicketService) TransitionTicket(ctx context.Context, ticketKey string, transitionID string) error {
	// Mock successful transition
	return nil
}

// NewMockJiraService creates a mock Jira service with sample data
func NewMockJiraService() *MockTicketService {
	return &MockTicketService{
		ProviderName: "Jira",
		BaseURL:      "https://test.atlassian.net",
		Tickets: []tickets.Ticket{
			{Key: "PROJ-123", DisplayKey: "PROJ-123", Summary: "Implement user authentication", Status: "In Progress", Type: "Story", Priority: "High"},
			{Key: "PROJ-124", DisplayKey: "PROJ-124", Summary: "Fix login button styling", Status: "To Do", Type: "Bug", Priority: "Medium"},
			{Key: "PROJ-125", DisplayKey: "PROJ-125", Summary: "Add unit tests", Status: "Done", Type: "Task", Priority: "Low"},
		},
	}
}

// NewMockCodecksService creates a mock Codecks service with sample data
func NewMockCodecksService() *MockTicketService {
	return &MockTicketService{
		ProviderName: "Codecks",
		BaseURL:      "https://test.codecks.io",
		Tickets: []tickets.Ticket{
			{Key: "card-guid-1", DisplayKey: "$12u", Summary: "Add codecks support", Status: "started", Type: "feature", Priority: "high", DeckID: "deck-1"},
			{Key: "card-guid-2", DisplayKey: "$12v", Summary: "Fix card display", Status: "open", Type: "bug", Priority: "medium", DeckID: "deck-1"},
			{Key: "card-guid-3", DisplayKey: "$12w", Summary: "Update documentation", Status: "done", Type: "chore", Priority: "low", DeckID: "deck-2"},
		},
	}
}

// MockJJService mocks jj command execution
type MockJJService struct {
	RepoPath    string
	Commits     []models.Commit
	Bookmarks   []string
	RemoteURL   string
	
	// Function overrides for custom behavior
	NewCommitFunc     func(parentID string) error
	DescribeFunc      func(changeID, description string) error
	SquashFunc        func(sourceID, destID string) error
	RebaseFunc        func(sourceID, destID string) error
	AbandonFunc       func(changeID string) error
	CreateBookmarkFunc func(name, commitID string) error
}

// GetRepository returns a mock repository
func (m *MockJJService) GetRepository(ctx context.Context) (*models.Repository, error) {
	if len(m.Commits) == 0 {
		m.Commits = DefaultMockCommits()
	}
	return &models.Repository{
		Path: m.RepoPath,
		Graph: models.CommitGraph{
			Commits: m.Commits,
		},
	}, nil
}

// NewCommit creates a mock new commit
func (m *MockJJService) NewCommit(parentID string) error {
	if m.NewCommitFunc != nil {
		return m.NewCommitFunc(parentID)
	}
	return nil
}

// Describe updates a mock commit description
func (m *MockJJService) Describe(changeID, description string) error {
	if m.DescribeFunc != nil {
		return m.DescribeFunc(changeID, description)
	}
	return nil
}

// Squash mocks squashing commits
func (m *MockJJService) Squash(sourceID, destID string) error {
	if m.SquashFunc != nil {
		return m.SquashFunc(sourceID, destID)
	}
	return nil
}

// Rebase mocks rebasing a commit
func (m *MockJJService) Rebase(sourceID, destID string) error {
	if m.RebaseFunc != nil {
		return m.RebaseFunc(sourceID, destID)
	}
	return nil
}

// Abandon mocks abandoning a commit
func (m *MockJJService) Abandon(changeID string) error {
	if m.AbandonFunc != nil {
		return m.AbandonFunc(changeID)
	}
	return nil
}

// CreateBookmark creates a mock bookmark
func (m *MockJJService) CreateBookmark(name, commitID string) error {
	if m.CreateBookmarkFunc != nil {
		return m.CreateBookmarkFunc(name, commitID)
	}
	m.Bookmarks = append(m.Bookmarks, name)
	return nil
}

// GetBookmarks returns mock bookmarks
func (m *MockJJService) GetBookmarks() []string {
	return m.Bookmarks
}

// GetGitRemoteURL returns mock remote URL
func (m *MockJJService) GetGitRemoteURL(ctx context.Context) (string, error) {
	if m.RemoteURL == "" {
		return "https://github.com/testowner/testrepo.git", nil
	}
	return m.RemoteURL, nil
}

// DefaultMockCommits returns a set of default mock commits for testing
func DefaultMockCommits() []models.Commit {
	return []models.Commit{
		{
			ID:          "abc123456789",
			ShortID:     "abc1",
			ChangeID:    "change-abc1",
			Summary:     "Add new feature",
			Description: "This commit adds a new feature to the application",
			Author:      "Test User",
			IsWorking:   true,
			Immutable:   false,
			Parents:     []string{"def456789012"},
		},
		{
			ID:          "def456789012",
			ShortID:     "def4",
			ChangeID:    "change-def4",
			Summary:     "Fix bug in parser",
			Description: "Fixed a critical bug in the parser module",
			Author:      "Test User",
			IsWorking:   false,
			Immutable:   false,
			Parents:     []string{"ghi789012345"},
		},
		{
			ID:          "ghi789012345",
			ShortID:     "ghi7",
			ChangeID:    "change-ghi7",
			Summary:     "Initial commit",
			Description: "Initial project setup",
			Author:      "Test User",
			IsWorking:   false,
			Immutable:   true,
			Parents:     []string{},
		},
	}
}

// NewMockJJService creates a mock jj service with default data
func NewMockJJService() *MockJJService {
	return &MockJJService{
		RepoPath:  "/test/repo",
		Commits:   DefaultMockCommits(),
		Bookmarks: []string{"main", "feature/test"},
		RemoteURL: "https://github.com/testowner/testrepo.git",
	}
}

