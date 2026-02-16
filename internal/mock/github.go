package mock

import (
	"context"
	"time"

	"github.com/google/go-github/v66/github"
	"github.com/madicen/jj-tui/internal"
)

// GitHubService is a mock GitHub service that returns demo PR data
type GitHubService struct {
	owner    string
	repo     string
	username string
}

// NewGitHubService creates a new mock GitHub service
func NewGitHubService() *GitHubService {
	return &GitHubService{
		owner:    "demo-org",
		repo:     "awesome-project",
		username: "demo-user",
	}
}

// GetOwner returns the repository owner
func (s *GitHubService) GetOwner() string {
	return s.owner
}

// GetRepo returns the repository name
func (s *GitHubService) GetRepo() string {
	return s.repo
}

// GetUsername returns the authenticated username
func (s *GitHubService) GetUsername() string {
	return s.username
}

// ListPullRequests returns demo pull requests
func (s *GitHubService) ListPullRequests(ctx context.Context, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error) {
	prs := demoPullRequests()
	return prs, &github.Response{}, nil
}

// GetPullRequest returns a single demo PR
func (s *GitHubService) GetPullRequest(ctx context.Context, number int) (*github.PullRequest, *github.Response, error) {
	for _, pr := range demoPullRequests() {
		if pr.GetNumber() == number {
			return pr, &github.Response{}, nil
		}
	}
	return nil, &github.Response{}, nil
}

// CreatePullRequest pretends to create a PR
func (s *GitHubService) CreatePullRequest(ctx context.Context, newPR *github.NewPullRequest) (*github.PullRequest, *github.Response, error) {
	pr := &github.PullRequest{
		Number:    github.Int(999),
		Title:     newPR.Title,
		Body:      newPR.Body,
		State:     github.String("open"),
		HTMLURL:   github.String("https://github.com/demo-org/awesome-project/pull/999"),
		CreatedAt: &github.Timestamp{Time: time.Now()},
		UpdatedAt: &github.Timestamp{Time: time.Now()},
		Head: &github.PullRequestBranch{
			Ref: newPR.Head,
		},
		Base: &github.PullRequestBranch{
			Ref: newPR.Base,
		},
		User: &github.User{
			Login: github.String(s.username),
		},
	}
	return pr, &github.Response{}, nil
}

// GetCombinedStatus returns demo CI status
func (s *GitHubService) GetCombinedStatus(ctx context.Context, ref string) (*github.CombinedStatus, *github.Response, error) {
	// Return varied statuses based on ref to make screenshots interesting
	var state string
	switch ref {
	case "feature/dark-mode", "fix/pagination":
		state = "success"
	case "feature/settings":
		state = "pending"
	case "fix/auth-bug":
		state = "failure"
	default:
		state = "success"
	}

	status := &github.CombinedStatus{
		State: github.String(state),
		Statuses: []*github.RepoStatus{
			{
				State:   github.String(state),
				Context: github.String("CI / Build"),
			},
			{
				State:   github.String(state),
				Context: github.String("CI / Test"),
			},
		},
	}
	return status, &github.Response{}, nil
}

// ListReviews returns demo review data
func (s *GitHubService) ListReviews(ctx context.Context, number int) ([]*github.PullRequestReview, *github.Response, error) {
	var reviews []*github.PullRequestReview

	// Return different review states based on PR number
	switch number {
	case 142:
		reviews = []*github.PullRequestReview{
			{
				State: github.String("APPROVED"),
				User:  &github.User{Login: github.String("reviewer1")},
			},
		}
	case 139:
		reviews = []*github.PullRequestReview{
			{
				State: github.String("CHANGES_REQUESTED"),
				User:  &github.User{Login: github.String("reviewer2")},
			},
		}
	case 135:
		// No reviews yet
	case 128:
		reviews = []*github.PullRequestReview{
			{
				State: github.String("APPROVED"),
				User:  &github.User{Login: github.String("reviewer1")},
			},
			{
				State: github.String("APPROVED"),
				User:  &github.User{Login: github.String("reviewer3")},
			},
		}
	}

	return reviews, &github.Response{}, nil
}

// Push is a no-op for the mock
func (s *GitHubService) Push(ctx context.Context, branch string) error {
	return nil
}

// demoPullRequests returns a set of demo PRs with varied states
func demoPullRequests() []*github.PullRequest {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	lastWeek := now.Add(-7 * 24 * time.Hour)

	return []*github.PullRequest{
		{
			Number:    github.Int(142),
			Title:     github.String("Add dark mode support to dashboard"),
			Body:      github.String("Implements dark mode theme with system preference detection.\n\nCloses PROJ-142"),
			State:     github.String("open"),
			HTMLURL:   github.String("https://github.com/demo-org/awesome-project/pull/142"),
			CreatedAt: &github.Timestamp{Time: yesterday},
			UpdatedAt: &github.Timestamp{Time: now},
			Head: &github.PullRequestBranch{
				Ref: github.String("feature/dark-mode"),
			},
			Base: &github.PullRequestBranch{
				Ref: github.String("main"),
			},
			User: &github.User{
				Login: github.String("demo-user"),
			},
			Additions:      github.Int(342),
			Deletions:      github.Int(28),
			MergeableState: github.String("clean"),
		},
		{
			Number:    github.Int(139),
			Title:     github.String("Fix pagination bug in search results"),
			Body:      github.String("Fixes the pagination issue when filtering by date.\n\nCloses PROJ-139"),
			State:     github.String("open"),
			HTMLURL:   github.String("https://github.com/demo-org/awesome-project/pull/139"),
			CreatedAt: &github.Timestamp{Time: lastWeek},
			UpdatedAt: &github.Timestamp{Time: yesterday},
			Head: &github.PullRequestBranch{
				Ref: github.String("fix/pagination"),
			},
			Base: &github.PullRequestBranch{
				Ref: github.String("main"),
			},
			User: &github.User{
				Login: github.String("demo-user"),
			},
			Additions:      github.Int(15),
			Deletions:      github.Int(8),
			MergeableState: github.String("blocked"),
		},
		{
			Number:    github.Int(135),
			Title:     github.String("Implement user profile settings page"),
			Body:      github.String("New settings page for user profile management."),
			State:     github.String("open"),
			HTMLURL:   github.String("https://github.com/demo-org/awesome-project/pull/135"),
			CreatedAt: &github.Timestamp{Time: lastWeek},
			UpdatedAt: &github.Timestamp{Time: lastWeek},
			Head: &github.PullRequestBranch{
				Ref: github.String("feature/settings"),
			},
			Base: &github.PullRequestBranch{
				Ref: github.String("main"),
			},
			User: &github.User{
				Login: github.String("demo-user"),
			},
			Additions:      github.Int(523),
			Deletions:      github.Int(12),
			MergeableState: github.String("unstable"),
		},
		{
			Number:    github.Int(128),
			Title:     github.String("Add export to CSV functionality"),
			Body:      github.String("Users can now export data to CSV format."),
			State:     github.String("merged"),
			HTMLURL:   github.String("https://github.com/demo-org/awesome-project/pull/128"),
			CreatedAt: &github.Timestamp{Time: lastWeek.Add(-3 * 24 * time.Hour)},
			UpdatedAt: &github.Timestamp{Time: lastWeek},
			MergedAt:  &github.Timestamp{Time: lastWeek},
			Head: &github.PullRequestBranch{
				Ref: github.String("feature/csv-export"),
			},
			Base: &github.PullRequestBranch{
				Ref: github.String("main"),
			},
			User: &github.User{
				Login: github.String("demo-user"),
			},
			Additions: github.Int(187),
			Deletions: github.Int(4),
		},
		{
			Number:    github.Int(121),
			Title:     github.String("Update authentication flow for SSO"),
			Body:      github.String("SSO integration with corporate identity provider."),
			State:     github.String("closed"),
			HTMLURL:   github.String("https://github.com/demo-org/awesome-project/pull/121"),
			CreatedAt: &github.Timestamp{Time: lastWeek.Add(-7 * 24 * time.Hour)},
			UpdatedAt: &github.Timestamp{Time: lastWeek.Add(-5 * 24 * time.Hour)},
			ClosedAt:  &github.Timestamp{Time: lastWeek.Add(-5 * 24 * time.Hour)},
			Head: &github.PullRequestBranch{
				Ref: github.String("fix/auth-bug"),
			},
			Base: &github.PullRequestBranch{
				Ref: github.String("main"),
			},
			User: &github.User{
				Login: github.String("other-dev"),
			},
			Additions: github.Int(89),
			Deletions: github.Int(234),
		},
	}
}

// DemoPullRequests returns demo PRs in the models.GitHubPR format
// This is used by the TUI's loadPRs function in demo mode
func DemoPullRequests() []internal.GitHubPR {
	return []internal.GitHubPR{
		{
			Number:       142,
			Title:        "Add dark mode support to dashboard",
			Body:         "Implements dark mode theme with system preference detection.\n\nCloses PROJ-142",
			URL:          "https://github.com/demo-org/awesome-project/pull/142",
			State:        "open",
			BaseBranch:   "main",
			HeadBranch:   "feature/dark-mode",
			CommitIDs:    []string{"abc123", "def456"},
			CheckStatus:  internal.CheckStatusSuccess,
			ReviewStatus: internal.ReviewStatusApproved,
		},
		{
			Number:       139,
			Title:        "Fix pagination bug in search results",
			Body:         "Fixes the pagination issue when filtering by date.\n\nCloses PROJ-139",
			URL:          "https://github.com/demo-org/awesome-project/pull/139",
			State:        "open",
			BaseBranch:   "main",
			HeadBranch:   "fix/pagination",
			CommitIDs:    []string{"ghi789"},
			CheckStatus:  internal.CheckStatusSuccess,
			ReviewStatus: internal.ReviewStatusChangesRequested,
		},
		{
			Number:       135,
			Title:        "Implement user profile settings page",
			Body:         "New settings page for user profile management.",
			URL:          "https://github.com/demo-org/awesome-project/pull/135",
			State:        "open",
			BaseBranch:   "main",
			HeadBranch:   "feature/settings",
			CommitIDs:    []string{"jkl012"},
			CheckStatus:  internal.CheckStatusPending,
			ReviewStatus: internal.ReviewStatusPending,
		},
		{
			Number:       128,
			Title:        "Add export to CSV functionality",
			Body:         "Users can now export data to CSV format.",
			URL:          "https://github.com/demo-org/awesome-project/pull/128",
			State:        "merged",
			BaseBranch:   "main",
			HeadBranch:   "feature/csv-export",
			CommitIDs:    []string{"mno345"},
			CheckStatus:  internal.CheckStatusSuccess,
			ReviewStatus: internal.ReviewStatusApproved,
		},
		{
			Number:       121,
			Title:        "Update authentication flow for SSO",
			Body:         "SSO integration with corporate identity provider.",
			URL:          "https://github.com/demo-org/awesome-project/pull/121",
			State:        "closed",
			BaseBranch:   "main",
			HeadBranch:   "fix/auth-bug",
			CommitIDs:    []string{"pqr678"},
			CheckStatus:  internal.CheckStatusFailure,
			ReviewStatus: internal.ReviewStatusNone,
		},
	}
}
