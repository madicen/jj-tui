package github

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v66/github"
	"github.com/madicen-utilities/jj-tui/v2/internal/models"
	"golang.org/x/oauth2"
)

// Service handles GitHub API interactions
type Service struct {
	client    *github.Client
	owner     string
	repo      string
	token     string
}

// NewService creates a new GitHub service
func NewService(owner, repo string) (*Service, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN environment variable is required")
	}

	// Create OAuth2 token source
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)

	// Create GitHub client
	client := github.NewClient(tc)

	return &Service{
		client: client,
		owner:  owner,
		repo:   repo,
		token:  token,
	}, nil
}

// CreatePullRequest creates a new pull request
func (s *Service) CreatePullRequest(ctx context.Context, req *models.CreatePRRequest) (*models.GitHubPR, error) {
	// For same-repo PRs, head can be just the branch name
	// But some GitHub configurations require owner:branch format
	headRef := req.HeadBranch
	
	newPR := &github.NewPullRequest{
		Title:               github.String(req.Title),
		Head:                github.String(headRef),
		Base:                github.String(req.BaseBranch),
		Body:                github.String(req.Body),
		MaintainerCanModify: github.Bool(true),
		Draft:               github.Bool(req.Draft),
	}

	pr, resp, err := s.client.PullRequests.Create(ctx, s.owner, s.repo, newPR)
	if err != nil {
		// If we get a "not all refs" error, try with owner:branch format
		if resp != nil && resp.StatusCode == 422 && strings.Contains(err.Error(), "refs") {
			newPR.Head = github.String(s.owner + ":" + req.HeadBranch)
			pr, _, err = s.client.PullRequests.Create(ctx, s.owner, s.repo, newPR)
			if err != nil {
				return nil, fmt.Errorf("failed to create pull request (tried both formats): %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to create pull request: %w", err)
		}
	}

	return &models.GitHubPR{
		Number:     pr.GetNumber(),
		Title:      pr.GetTitle(),
		URL:        pr.GetHTMLURL(),
		State:      pr.GetState(),
		BaseBranch: pr.GetBase().GetRef(),
		HeadBranch: pr.GetHead().GetRef(),
		CommitIDs:  req.CommitIDs,
	}, nil
}

// UpdatePullRequest updates an existing pull request
func (s *Service) UpdatePullRequest(ctx context.Context, prNumber int, req *models.UpdatePRRequest) (*models.GitHubPR, error) {
	updatePR := &github.PullRequest{}
	
	if req.Title != "" {
		updatePR.Title = github.String(req.Title)
	}
	if req.Body != "" {
		updatePR.Body = github.String(req.Body)
	}

	pr, _, err := s.client.PullRequests.Edit(ctx, s.owner, s.repo, prNumber, updatePR)
	if err != nil {
		return nil, fmt.Errorf("failed to update pull request: %w", err)
	}

	return &models.GitHubPR{
		Number:     pr.GetNumber(),
		Title:      pr.GetTitle(),
		URL:        pr.GetHTMLURL(),
		State:      pr.GetState(),
		BaseBranch: pr.GetBase().GetRef(),
		HeadBranch: pr.GetHead().GetRef(),
		CommitIDs:  req.CommitIDs,
	}, nil
}

// GetPullRequests retrieves all pull requests for the repository
func (s *Service) GetPullRequests(ctx context.Context) ([]models.GitHubPR, error) {
	opts := &github.PullRequestListOptions{
		State: "all",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var allPRs []models.GitHubPR
	for {
		prs, resp, err := s.client.PullRequests.List(ctx, s.owner, s.repo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list pull requests: %w", err)
		}

		for _, pr := range prs {
			allPRs = append(allPRs, models.GitHubPR{
				Number:     pr.GetNumber(),
				Title:      pr.GetTitle(),
				URL:        pr.GetHTMLURL(),
				State:      pr.GetState(),
				BaseBranch: pr.GetBase().GetRef(),
				HeadBranch: pr.GetHead().GetRef(),
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allPRs, nil
}

// GetPullRequest retrieves a specific pull request
func (s *Service) GetPullRequest(ctx context.Context, prNumber int) (*models.GitHubPR, error) {
	pr, _, err := s.client.PullRequests.Get(ctx, s.owner, s.repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request: %w", err)
	}

	// Get commits for this PR
	commits, err := s.getPullRequestCommits(ctx, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR commits: %w", err)
	}

	return &models.GitHubPR{
		Number:     pr.GetNumber(),
		Title:      pr.GetTitle(),
		URL:        pr.GetHTMLURL(),
		State:      pr.GetState(),
		BaseBranch: pr.GetBase().GetRef(),
		HeadBranch: pr.GetHead().GetRef(),
		CommitIDs:  commits,
	}, nil
}

// getPullRequestCommits retrieves commit IDs for a pull request
func (s *Service) getPullRequestCommits(ctx context.Context, prNumber int) ([]string, error) {
	opts := &github.ListOptions{PerPage: 100}
	
	var commitIDs []string
	for {
		commits, resp, err := s.client.PullRequests.ListCommits(ctx, s.owner, s.repo, prNumber, opts)
		if err != nil {
			return nil, err
		}

		for _, commit := range commits {
			commitIDs = append(commitIDs, commit.GetSHA())
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return commitIDs, nil
}

// BranchExists checks if a branch exists on GitHub
func (s *Service) BranchExists(ctx context.Context, branch string) (bool, error) {
	_, resp, err := s.client.Repositories.GetBranch(ctx, s.owner, s.repo, branch, 0)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return false, nil
		}
		// Log more details for debugging
		return false, fmt.Errorf("branch check failed for %s/%s branch %s: %w", s.owner, s.repo, branch, err)
	}
	return true, nil
}

// ParseGitHubURL extracts owner and repo from a GitHub URL
func ParseGitHubURL(url string) (owner, repo string, err error) {
	// Handle various GitHub URL formats
	url = strings.TrimSpace(url)
	
	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")
	
	// Handle HTTPS URLs
	if strings.HasPrefix(url, "https://github.com/") {
		path := strings.TrimPrefix(url, "https://github.com/")
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			return parts[0], parts[1], nil
		}
	}
	
	// Handle SSH URLs
	if strings.HasPrefix(url, "git@github.com:") {
		path := strings.TrimPrefix(url, "git@github.com:")
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			return parts[0], parts[1], nil
		}
	}
	
	return "", "", fmt.Errorf("invalid GitHub URL: %s", url)
}
