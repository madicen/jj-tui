package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/go-github/v66/github"
	"github.com/madicen/jj-tui/internal"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

// GitHubClientID is the OAuth App Client ID for jj-tui
const GitHubClientID = "Iv23liEpah7dINFx13j6"

// DeviceCodeResponse represents the response from GitHub's device code endpoint
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// TokenResponse represents the response from GitHub's token endpoint
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

// AuthError represents a GitHub authentication error that may require reauthorization
type AuthError struct {
	Err        error
	StatusCode int
}

func (e *AuthError) Error() string {
	return e.Err.Error()
}

func (e *AuthError) Unwrap() error {
	return e.Err
}

// IsAuthError checks if an error is an authentication error that needs reauth
func IsAuthError(err error) bool {
	var authErr *AuthError
	return err != nil && (strings.Contains(err.Error(), "401") ||
		strings.Contains(err.Error(), "Bad credentials") ||
		strings.Contains(err.Error(), "authentication") ||
		errors.As(err, &authErr))
}

// NewAuthError creates a new AuthError
func NewAuthError(err error, statusCode int) *AuthError {
	return &AuthError{Err: err, StatusCode: statusCode}
}

// PRFilterOptions contains options for filtering PRs
type PRFilterOptions struct {
	OnlyMine   bool // Only show PRs created by the authenticated user
	Limit      int  // Maximum number of PRs to fetch (0 = no limit)
	ShowMerged bool // Include merged PRs
	ShowClosed bool // Include closed PRs
}

// Service handles GitHub API interactions
type Service struct {
	client        *github.Client
	graphqlClient *githubv4.Client
	owner         string
	repo          string
	token         string
	username      string // cached authenticated username
	// Cached repo metadata (filled on first lookup so PR-form open / preflight checks don't
	// re-hit the API). Empty defaultBranch means "not fetched yet" — callers should fall back
	// to a sensible default (usually "main") if a fetch fails.
	defaultBranch string
}

// CreatePullRequest creates a new pull request
func (s *Service) CreatePullRequest(ctx context.Context, req *internal.CreatePRRequest) (*internal.GitHubPR, error) {
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
		// Decode 422 validation errors into a single user-friendly string so callers can
		// distinguish base- vs head-related failures. Without this, the only signal is the
		// raw `[{Resource:PullRequest Field:base Code:invalid Message:}]` blob, which loses
		// the field/code distinction inside fmt.Errorf and trips the head-ref retry path on
		// errors that retrying can't fix.
		if resp != nil && resp.StatusCode == 422 {
			detail := summarize422(err)
			// Only retry the head-ref/owner-prefix dance for head-related issues — retrying
			// a missing-or-invalid base branch will just 422 again every time.
			if isHeadRefRetryable(err, detail) {
				newPR.Head = github.String(s.owner + ":" + req.HeadBranch)
				pr, _, err = s.client.PullRequests.Create(ctx, s.owner, s.repo, newPR)
				if err != nil {
					return nil, fmt.Errorf("failed to create pull request (tried both head formats): %s", summarize422(err))
				}
			} else {
				return nil, fmt.Errorf("failed to create pull request: %s", detail)
			}
		} else {
			return nil, fmt.Errorf("failed to create pull request: %w", err)
		}
	}

	return &internal.GitHubPR{
		Number:     pr.GetNumber(),
		Title:      pr.GetTitle(),
		URL:        pr.GetHTMLURL(),
		State:      pr.GetState(),
		BaseBranch: pr.GetBase().GetRef(),
		HeadBranch: pr.GetHead().GetRef(),
		CommitIDs:  req.CommitIDs,
	}, nil
}

// summarize422 turns a go-github *ErrorResponse into a single human-readable line that
// preserves field/code information from the API's structured error array. Falls back to the
// underlying error string when err isn't a go-github ErrorResponse (so this is safe to call
// on any 4xx error). Used by CreatePullRequest to give callers something they can actually
// regex/match on (e.g. detecting a base-related failure to skip a doomed retry loop).
func summarize422(err error) string {
	var errResp *github.ErrorResponse
	if !errors.As(err, &errResp) || len(errResp.Errors) == 0 {
		return err.Error()
	}
	parts := make([]string, 0, len(errResp.Errors))
	for _, e := range errResp.Errors {
		seg := fmt.Sprintf("%s.%s=%s", e.Resource, e.Field, e.Code)
		if msg := strings.TrimSpace(e.Message); msg != "" {
			seg += " (" + msg + ")"
		}
		parts = append(parts, seg)
	}
	base := errResp.Message
	if base == "" {
		base = "validation failed"
	}
	return fmt.Sprintf("%s: %s", base, strings.Join(parts, "; "))
}

// isHeadRefRetryable returns true for the "tried head as plain branch, GitHub wants
// owner:branch" pattern that warrants one retry. We deliberately exclude any base-field
// failure: an invalid or missing base ref isn't fixed by changing the head format, so
// retrying just wastes the user's time and produces a confusing "(tried both head formats)"
// message for an unrelated cause.
func isHeadRefRetryable(err error, detail string) bool {
	var errResp *github.ErrorResponse
	if errors.As(err, &errResp) {
		for _, e := range errResp.Errors {
			if strings.EqualFold(e.Field, "base") {
				return false
			}
		}
	}
	lower := strings.ToLower(detail + " " + err.Error())
	if strings.Contains(lower, "field=base") || strings.Contains(lower, ".base=") {
		return false
	}
	// Match the legacy heuristic so we don't regress the working same-repo PR path.
	return strings.Contains(lower, "refs") || strings.Contains(lower, "head")
}

// GetDefaultBranch returns the repository's default branch name (e.g. "main", "master",
// "trunk"). The result is cached on the Service so subsequent calls (e.g. preflight checks
// when opening the PR-create form repeatedly) don't re-hit the API. Returns an error only
// when the API call itself fails; callers that want a graceful fallback should treat any
// error as "no opinion" and use their own default (typically "main").
func (s *Service) GetDefaultBranch(ctx context.Context) (string, error) {
	if s == nil {
		return "", fmt.Errorf("github service unavailable")
	}
	if s.defaultBranch != "" {
		return s.defaultBranch, nil
	}
	repo, resp, err := s.client.Repositories.Get(ctx, s.owner, s.repo)
	if err != nil {
		if resp != nil && (resp.StatusCode == 401 || resp.StatusCode == 403) {
			return "", NewAuthError(fmt.Errorf("failed to read default branch: %w", err), resp.StatusCode)
		}
		return "", fmt.Errorf("failed to read default branch for %s/%s: %w", s.owner, s.repo, err)
	}
	branch := strings.TrimSpace(repo.GetDefaultBranch())
	if branch == "" {
		// GitHub always populates default_branch for non-empty repos; an empty string here
		// means the repo has no commits at all yet (fresh `gh repo create`). Don't cache the
		// empty so the next call retries once the user has pushed something.
		return "", nil
	}
	s.defaultBranch = branch
	return branch, nil
}

// UpdatePullRequest updates an existing pull request
func (s *Service) UpdatePullRequest(ctx context.Context, prNumber int, req *internal.UpdatePRRequest) (*internal.GitHubPR, error) {
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

	return &internal.GitHubPR{
		Number:     pr.GetNumber(),
		Title:      pr.GetTitle(),
		URL:        pr.GetHTMLURL(),
		State:      pr.GetState(),
		BaseBranch: pr.GetBase().GetRef(),
		HeadBranch: pr.GetHead().GetRef(),
		CommitIDs:  req.CommitIDs,
	}, nil
}

// MergePullRequest merges a pull request using the default merge method
func (s *Service) MergePullRequest(ctx context.Context, prNumber int) error {
	// Use default merge commit method
	options := &github.PullRequestOptions{
		MergeMethod: "merge",
	}

	_, _, err := s.client.PullRequests.Merge(ctx, s.owner, s.repo, prNumber, "", options)
	if err != nil {
		if errResp, ok := err.(*github.ErrorResponse); ok {
			// If the error is a GitHub API error, read the body for more context.
			bodyBytes, readErr := io.ReadAll(errResp.Response.Body)
			if readErr != nil {
				// If we can't read the body, just return the original error.
				return fmt.Errorf("failed to merge pull request: %w (and failed to read error body)", err)
			}
			defer errResp.Response.Body.Close()
			return fmt.Errorf("failed to merge pull request: %v (body: %s)", err, string(bodyBytes))
		}
		return fmt.Errorf("failed to merge pull request: %w", err)
	}

	return nil
}

// ClosePullRequest closes a pull request without merging
func (s *Service) ClosePullRequest(ctx context.Context, prNumber int) error {
	updatePR := &github.PullRequest{
		State: github.String("closed"),
	}

	_, _, err := s.client.PullRequests.Edit(ctx, s.owner, s.repo, prNumber, updatePR)
	if err != nil {
		return fmt.Errorf("failed to close pull request: %w", err)
	}

	return nil
}

// GetAuthenticatedUsername returns the username of the authenticated user
func (s *Service) GetAuthenticatedUsername(ctx context.Context) (string, error) {
	// Return cached username if available
	if s.username != "" {
		return s.username, nil
	}

	user, resp, err := s.client.Users.Get(ctx, "")
	if err != nil {
		// Check for authentication errors (401 Bad credentials, etc.)
		if resp != nil && (resp.StatusCode == 401 || resp.StatusCode == 403) {
			return "", NewAuthError(fmt.Errorf("GitHub authentication failed: %w", err), resp.StatusCode)
		}
		return "", fmt.Errorf("failed to get authenticated user: %w", err)
	}

	s.username = user.GetLogin()
	return s.username, nil
}

// GetPullRequests retrieves pull requests for the repository with optional filtering
func (s *Service) GetPullRequests(ctx context.Context) ([]internal.GitHubPR, error) {
	return s.GetPullRequestsWithOptions(ctx, PRFilterOptions{
		Limit:      100,
		ShowMerged: true,
		ShowClosed: true,
	})
}

// GetPullRequestsWithOptions retrieves pull requests with the specified filter options
// Uses GraphQL to fetch PRs with check status and reviews in a single API call
// Falls back to REST API if GraphQL fails due to permission issues
func (s *Service) GetPullRequestsWithOptions(ctx context.Context, filterOpts PRFilterOptions) ([]internal.GitHubPR, error) {
	// Try GraphQL first (includes check status and reviews)
	prs, err := s.getPullRequestsGraphQL(ctx, filterOpts)
	if err != nil {
		// Check if this is a permission/access error - fall back to REST API
		errStr := err.Error()
		if strings.Contains(errStr, "Resource not accessible") ||
			strings.Contains(errStr, "Could not resolve to a Repository") ||
			strings.Contains(errStr, "403") ||
			strings.Contains(errStr, "insufficient") {
			// Fall back to REST API (no check status or reviews, but basic PR info works)
			return s.getPullRequestsREST(ctx, filterOpts)
		}
		return nil, err
	}
	return prs, nil
}

// getPullRequestsGraphQL fetches PRs using GraphQL (includes check status and reviews).
//
// Open PRs are always fetched in full (no limit); merged/closed PRs are then fetched up to the
// configured limit. A single newest-first query capped by Limit would, in busy repos with
// thousands of PRs, push older still-open PRs out of the result. That broke the graph's
// "Update PR" vs "Create PR" detection, which depends on every open PR's head branch being
// present so it can be matched against local bookmarks.
func (s *Service) getPullRequestsGraphQL(ctx context.Context, filterOpts PRFilterOptions) ([]internal.GitHubPR, error) {
	// Get authenticated username if filtering by user
	var username string
	if filterOpts.OnlyMine {
		var err error
		username, err = s.GetAuthenticatedUsername(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get username for filtering: %w", err)
		}
	}

	// Always retrieve every open PR so a local bookmark can be reliably matched to its PR.
	openPRs, err := s.queryPullRequestsGraphQL(ctx, []githubv4.PullRequestState{githubv4.PullRequestStateOpen}, 0, filterOpts, username)
	if err != nil {
		return nil, err
	}

	var otherStates []githubv4.PullRequestState
	if filterOpts.ShowMerged {
		otherStates = append(otherStates, githubv4.PullRequestStateMerged)
	}
	if filterOpts.ShowClosed {
		otherStates = append(otherStates, githubv4.PullRequestStateClosed)
	}
	if len(otherStates) == 0 {
		return openPRs, nil
	}

	otherPRs, err := s.queryPullRequestsGraphQL(ctx, otherStates, filterOpts.Limit, filterOpts, username)
	if err != nil {
		return nil, err
	}
	return append(openPRs, otherPRs...), nil
}

// queryPullRequestsGraphQL fetches PRs for the given states, paginating until the results are
// exhausted or until limit PRs have been collected (limit <= 0 means no limit). When
// filterOpts.OnlyMine is set, results are filtered to PRs authored by username.
func (s *Service) queryPullRequestsGraphQL(ctx context.Context, states []githubv4.PullRequestState, limit int, filterOpts PRFilterOptions, username string) ([]internal.GitHubPR, error) {
	// GraphQL query structure
	var query struct {
		Repository struct {
			PullRequests struct {
				PageInfo struct {
					HasNextPage bool
					EndCursor   githubv4.String
				}
				Nodes []struct {
					Number      int
					Title       string
					Body        string
					Url         string
					State       string
					BaseRefName string
					HeadRefName string
					Merged      bool
					Author      struct {
						Login string
					}
					Commits struct {
						Nodes []struct {
							Commit struct {
								StatusCheckRollup struct {
									State string
								}
							}
						}
					} `graphql:"commits(last: 1)"`
					Reviews struct {
						Nodes []struct {
							State  string
							Author struct {
								Login string
							}
						}
					} `graphql:"reviews(last: 20)"`
				}
			} `graphql:"pullRequests(first: $first, after: $after, states: $states, orderBy: {field: CREATED_AT, direction: DESC})"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	// Set query limit
	first := 100
	if limit > 0 && limit < first {
		first = limit
	}

	variables := map[string]any{
		"owner":  githubv4.String(s.owner),
		"name":   githubv4.String(s.repo),
		"first":  githubv4.Int(first),
		"after":  (*githubv4.String)(nil),
		"states": states,
	}

	var allPRs []internal.GitHubPR
	for {
		err := s.graphqlClient.Query(ctx, &query, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to query pull requests: %w", err)
		}

		for _, pr := range query.Repository.PullRequests.Nodes {
			// Filter by author if OnlyMine is set
			if filterOpts.OnlyMine && pr.Author.Login != username {
				continue
			}

			// Determine state - GraphQL returns OPEN, CLOSED, MERGED
			state := strings.ToLower(pr.State)
			if state == "closed" && pr.Merged {
				state = "merged"
			}

			// Parse check status from statusCheckRollup
			checkStatus := internal.CheckStatusNone
			if len(pr.Commits.Nodes) > 0 {
				rollupState := pr.Commits.Nodes[0].Commit.StatusCheckRollup.State
				switch rollupState {
				case "SUCCESS":
					checkStatus = internal.CheckStatusSuccess
				case "FAILURE", "ERROR":
					checkStatus = internal.CheckStatusFailure
				case "PENDING", "EXPECTED":
					checkStatus = internal.CheckStatusPending
				}
			}

			// Parse review status
			reviewStatus := parseReviewStatus(pr.Reviews.Nodes)

			allPRs = append(allPRs, internal.GitHubPR{
				Number:       pr.Number,
				Title:        pr.Title,
				Body:         pr.Body,
				URL:          pr.Url,
				State:        state,
				BaseBranch:   pr.BaseRefName,
				HeadBranch:   pr.HeadRefName,
				CheckStatus:  checkStatus,
				ReviewStatus: reviewStatus,
			})

			// Check limit
			if limit > 0 && len(allPRs) >= limit {
				return allPRs, nil
			}
		}

		// Check for more pages
		if !query.Repository.PullRequests.PageInfo.HasNextPage {
			break
		}
		variables["after"] = githubv4.NewString(query.Repository.PullRequests.PageInfo.EndCursor)

		// Early exit if we've hit the limit
		if limit > 0 && len(allPRs) >= limit {
			break
		}
	}

	return allPRs, nil
}

// getPullRequestsREST fetches PRs using REST API (fallback when GraphQL permissions are insufficient).
// This provides basic PR info but no check status or review status.
//
// Like the GraphQL path, open PRs are fetched in full so branch->PR matching is reliable, while
// merged/closed PRs honor the configured limit.
func (s *Service) getPullRequestsREST(ctx context.Context, filterOpts PRFilterOptions) ([]internal.GitHubPR, error) {
	// Get authenticated username if filtering by user
	var username string
	if filterOpts.OnlyMine {
		var err error
		username, err = s.GetAuthenticatedUsername(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get username for filtering: %w", err)
		}
	}

	// Always retrieve every open PR (no limit) so a local bookmark can be matched to its PR.
	openPRs, err := s.queryPullRequestsREST(ctx, "open", 0, filterOpts, username)
	if err != nil {
		return nil, err
	}
	if !filterOpts.ShowMerged && !filterOpts.ShowClosed {
		return openPRs, nil
	}

	// "closed" returns both closed and merged PRs; classification/skip is handled in the helper.
	closedPRs, err := s.queryPullRequestsREST(ctx, "closed", filterOpts.Limit, filterOpts, username)
	if err != nil {
		return nil, err
	}
	return append(openPRs, closedPRs...), nil
}

// queryPullRequestsREST fetches PRs for the given REST state filter ("open", "closed", or "all"),
// paginating until exhausted or until limit PRs are collected (limit <= 0 means no limit).
func (s *Service) queryPullRequestsREST(ctx context.Context, stateFilter string, limit int, filterOpts PRFilterOptions, username string) ([]internal.GitHubPR, error) {
	var allPRs []internal.GitHubPR
	opts := &github.PullRequestListOptions{
		State:     stateFilter,
		Sort:      "created",
		Direction: "desc",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		prs, resp, err := s.client.PullRequests.List(ctx, s.owner, s.repo, opts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, fmt.Errorf("failed to list pull requests for %s/%s: not found or access denied (GitHub returns 404 when the repository does not exist or the token cannot read it; for private org repos approve the OAuth app for the organization and complete SSO authorization if required): %w", s.owner, s.repo, err)
			}
			return nil, fmt.Errorf("failed to list pull requests: %w", err)
		}

		for _, pr := range prs {
			// Filter by author if OnlyMine is set
			if filterOpts.OnlyMine && pr.GetUser().GetLogin() != username {
				continue
			}

			// Determine state - check both GetMerged() and MergedAt for reliability
			state := pr.GetState()
			isMerged := pr.GetMerged() || !pr.GetMergedAt().IsZero()
			if state == "closed" && isMerged {
				state = "merged"
				if !filterOpts.ShowMerged {
					continue
				}
			} else if state == "closed" && !filterOpts.ShowClosed {
				continue
			}

			allPRs = append(allPRs, internal.GitHubPR{
				Number:       pr.GetNumber(),
				Title:        pr.GetTitle(),
				Body:         pr.GetBody(),
				URL:          pr.GetHTMLURL(),
				State:        state,
				BaseBranch:   pr.GetBase().GetRef(),
				HeadBranch:   pr.GetHead().GetRef(),
				CheckStatus:  internal.CheckStatusNone,  // Not available with REST fallback
				ReviewStatus: internal.ReviewStatusNone, // Not available with REST fallback
			})

			// Check limit
			if limit > 0 && len(allPRs) >= limit {
				return allPRs, nil
			}
		}

		// Check for more pages
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage

		// Early exit if we've hit the limit
		if limit > 0 && len(allPRs) >= limit {
			break
		}
	}

	return allPRs, nil
}

// parseReviewStatus aggregates review states and returns the overall status
func parseReviewStatus(reviews []struct {
	State  string
	Author struct {
		Login string
	}
}) internal.ReviewStatus {
	if len(reviews) == 0 {
		return internal.ReviewStatusNone
	}

	// Get the latest review state from each reviewer
	latestReviews := make(map[string]string)
	for _, review := range reviews {
		reviewer := review.Author.Login
		state := review.State
		// Only track meaningful states
		if state == "APPROVED" || state == "CHANGES_REQUESTED" || state == "DISMISSED" {
			latestReviews[reviewer] = state
		}
	}

	if len(latestReviews) == 0 {
		return internal.ReviewStatusPending
	}

	// Check for any changes requested or approvals
	hasApproval := false
	hasChangesRequested := false
	for _, state := range latestReviews {
		if state == "APPROVED" {
			hasApproval = true
		}
		if state == "CHANGES_REQUESTED" {
			hasChangesRequested = true
		}
	}

	// Changes requested takes priority over approval
	if hasChangesRequested {
		return internal.ReviewStatusChangesRequested
	}
	if hasApproval {
		return internal.ReviewStatusApproved
	}
	return internal.ReviewStatusPending
}

// GetOpenPRForBranch looks up a single open PR by its head branch name, independent of the bulk
// PR list. The bulk list can omit older open PRs in busy repos (it fetches newest-first across all
// states up to a limit), which breaks the graph's "Update PR" vs "Create PR" detection. This
// targeted query always resolves the branch's open PR if one exists. Returns nil when there is none.
func (s *Service) GetOpenPRForBranch(ctx context.Context, branch string) (*internal.GitHubPR, error) {
	if branch == "" {
		return nil, nil
	}
	if s.graphqlClient != nil {
		pr, err := s.getOpenPRForBranchGraphQL(ctx, branch)
		if err == nil {
			return pr, nil
		}
		errStr := err.Error()
		if !strings.Contains(errStr, "Resource not accessible") &&
			!strings.Contains(errStr, "Could not resolve to a Repository") &&
			!strings.Contains(errStr, "403") &&
			!strings.Contains(errStr, "insufficient") {
			return nil, err
		}
		// Permission/access error: fall back to REST.
	}
	return s.getOpenPRForBranchREST(ctx, branch)
}

func (s *Service) getOpenPRForBranchGraphQL(ctx context.Context, branch string) (*internal.GitHubPR, error) {
	var query struct {
		Repository struct {
			PullRequests struct {
				Nodes []struct {
					Number      int
					Title       string
					Body        string
					Url         string
					State       string
					BaseRefName string
					HeadRefName string
					Commits     struct {
						Nodes []struct {
							Commit struct {
								StatusCheckRollup struct {
									State string
								}
							}
						}
					} `graphql:"commits(last: 1)"`
					Reviews struct {
						Nodes []struct {
							State  string
							Author struct {
								Login string
							}
						}
					} `graphql:"reviews(last: 20)"`
				}
			} `graphql:"pullRequests(headRefName: $head, states: [OPEN], first: 1, orderBy: {field: CREATED_AT, direction: DESC})"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}
	variables := map[string]any{
		"owner": githubv4.String(s.owner),
		"name":  githubv4.String(s.repo),
		"head":  githubv4.String(branch),
	}
	if err := s.graphqlClient.Query(ctx, &query, variables); err != nil {
		return nil, fmt.Errorf("failed to query open PR for branch %q: %w", branch, err)
	}
	if len(query.Repository.PullRequests.Nodes) == 0 {
		return nil, nil
	}
	pr := query.Repository.PullRequests.Nodes[0]
	checkStatus := internal.CheckStatusNone
	if len(pr.Commits.Nodes) > 0 {
		switch pr.Commits.Nodes[0].Commit.StatusCheckRollup.State {
		case "SUCCESS":
			checkStatus = internal.CheckStatusSuccess
		case "FAILURE", "ERROR":
			checkStatus = internal.CheckStatusFailure
		case "PENDING", "EXPECTED":
			checkStatus = internal.CheckStatusPending
		}
	}
	return &internal.GitHubPR{
		Number:       pr.Number,
		Title:        pr.Title,
		Body:         pr.Body,
		URL:          pr.Url,
		State:        strings.ToLower(pr.State),
		BaseBranch:   pr.BaseRefName,
		HeadBranch:   pr.HeadRefName,
		CheckStatus:  checkStatus,
		ReviewStatus: parseReviewStatus(pr.Reviews.Nodes),
	}, nil
}

func (s *Service) getOpenPRForBranchREST(ctx context.Context, branch string) (*internal.GitHubPR, error) {
	opts := &github.PullRequestListOptions{
		State:       "open",
		Head:        s.owner + ":" + branch,
		ListOptions: github.ListOptions{PerPage: 1},
	}
	prs, resp, err := s.client.PullRequests.List(ctx, s.owner, s.repo, opts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list open PR for branch %q: %w", branch, err)
	}
	if len(prs) == 0 {
		return nil, nil
	}
	pr := prs[0]
	return &internal.GitHubPR{
		Number:       pr.GetNumber(),
		Title:        pr.GetTitle(),
		Body:         pr.GetBody(),
		URL:          pr.GetHTMLURL(),
		State:        pr.GetState(),
		BaseBranch:   pr.GetBase().GetRef(),
		HeadBranch:   pr.GetHead().GetRef(),
		CheckStatus:  internal.CheckStatusNone,
		ReviewStatus: internal.ReviewStatusNone,
	}, nil
}

// GetPullRequest retrieves a specific pull request
func (s *Service) GetPullRequest(ctx context.Context, prNumber int) (*internal.GitHubPR, error) {
	pr, _, err := s.client.PullRequests.Get(ctx, s.owner, s.repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request: %w", err)
	}

	// Get commits for this PR
	commits, err := s.getPullRequestCommits(ctx, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR commits: %w", err)
	}

	return &internal.GitHubPR{
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
func ParseGitHubURL(remoteURL string) (owner, repo string, err error) {
	// Handle various GitHub URL formats
	remoteURL = strings.TrimSpace(remoteURL)

	// Remove .git suffix
	remoteURL = strings.TrimSuffix(remoteURL, ".git")

	// Handle HTTPS URLs (including those with username like https://user@github.com/...)
	if strings.Contains(remoteURL, "github.com/") {
		// Find the github.com/ part and extract what comes after
		parts := strings.Split(remoteURL, "github.com/")
		if len(parts) == 2 {
			path := parts[1]
			pathParts := strings.Split(path, "/")
			if len(pathParts) >= 2 {
				return pathParts[0], pathParts[1], nil
			}
		}
	}

	// Handle SSH URLs
	if after, ok := strings.CutPrefix(remoteURL, "git@github.com:"); ok {
		path := after
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			return parts[0], parts[1], nil
		}
	}

	return "", "", fmt.Errorf("invalid GitHub URL: %s", remoteURL)
}

// StartDeviceFlow initiates the GitHub Device Flow authentication
// Returns the device code response containing the user code and verification URL
func StartDeviceFlow() (*DeviceCodeResponse, error) {
	data := url.Values{}
	data.Set("client_id", GitHubClientID)
	data.Set("scope", "repo")

	req, err := http.NewRequest("POST", "https://github.com/login/device/code", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to start device flow: %w", err)
	}
	defer resp.Body.Close()

	var deviceResp DeviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&deviceResp); err != nil {
		return nil, fmt.Errorf("failed to parse device code response: %w", err)
	}

	return &deviceResp, nil
}

// PollForToken polls GitHub for the access token after user authorization
// Returns the access token on success, or an error
// This should be called in a loop with the interval from DeviceCodeResponse
func PollForToken(deviceCode string) (string, error) {
	data := url.Values{}
	data.Set("client_id", GitHubClientID)
	data.Set("device_code", deviceCode)
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to poll for token: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	// Check for errors
	if tokenResp.Error != "" {
		switch tokenResp.Error {
		case "authorization_pending":
			// User hasn't authorized yet, keep polling
			return "", nil
		case "slow_down":
			// We're polling too fast, caller should increase interval
			return "", fmt.Errorf("slow_down")
		case "expired_token":
			return "", fmt.Errorf("device code expired, please try again")
		case "access_denied":
			return "", fmt.Errorf("access denied by user")
		default:
			return "", fmt.Errorf("auth error: %s - %s", tokenResp.Error, tokenResp.ErrorDesc)
		}
	}

	if tokenResp.AccessToken != "" {
		return tokenResp.AccessToken, nil
	}

	return "", nil
}

// GetOwner returns the repository owner
func (s *Service) GetOwner() string {
	return s.owner
}

// GetRepo returns the repository name
func (s *Service) GetRepo() string {
	return s.repo
}

// NewServiceWithToken creates a new GitHub service with a provided token
func NewServiceWithToken(owner, repo, token string) (*Service, error) {
	if token == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}

	// Create OAuth2 token source
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)

	// Create GitHub REST client
	client := github.NewClient(tc)

	// Create GitHub GraphQL client
	graphqlClient := githubv4.NewClient(tc)

	return &Service{
		client:        client,
		graphqlClient: graphqlClient,
		owner:         owner,
		repo:          repo,
		token:         token,
	}, nil
}
