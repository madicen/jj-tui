package actions

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/github"
	"github.com/madicen/jj-tui/internal/integrations/jj"
)

// PRCreateParams contains parameters for PR creation
type PRCreateParams struct {
	Title             string
	Body              string
	HeadBranch        string
	BaseBranch        string
	NeedsMoveBookmark bool
	CommitChangeID    string
}

// CreatePR pushes a branch and creates a PR
func CreatePR(jjSvc *jj.Service, ghSvc *github.Service, params PRCreateParams) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		if params.NeedsMoveBookmark && params.CommitChangeID != "" {
			if err := jjSvc.MoveBookmark(ctx, params.HeadBranch, params.CommitChangeID); err != nil {
				return ErrorMsg{Err: fmt.Errorf("failed to move bookmark %s: %w", params.HeadBranch, err)}
			}
		}

		pushOutput, err := jjSvc.PushToGit(ctx, params.HeadBranch)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to push branch: %w\nOutput: %s", err, pushOutput)}
		}

		time.Sleep(3 * time.Second)

		var pr *internal.GitHubPR
		var lastErr error
		for range 5 {
			pr, lastErr = ghSvc.CreatePullRequest(ctx, &internal.CreatePRRequest{
				Title:      params.Title,
				Body:       params.Body,
				HeadBranch: params.HeadBranch,
				BaseBranch: params.BaseBranch,
			})
			if lastErr == nil {
				break
			}
			if strings.Contains(lastErr.Error(), "not all refs") || strings.Contains(lastErr.Error(), "422") {
				time.Sleep(3 * time.Second)
				continue
			}
			break
		}
		if lastErr != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to create PR: %w\nPush output: %s", lastErr, pushOutput)}
		}
		return PRCreatedMsg{PR: pr}
	}
}

// PushToPR pushes updates to a PR
func PushToPR(svc *jj.Service, branch, commitID string, moveBookmark bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		if moveBookmark {
			if err := svc.MoveBookmark(ctx, branch, commitID); err != nil {
				return ErrorMsg{Err: fmt.Errorf("failed to move bookmark %s: %w", branch, err)}
			}
		}

		pushOutput, err := svc.PushToGit(ctx, branch)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to push: %w\nOutput: %s", err, pushOutput)}
		}
		return BranchPushedMsg{Branch: branch, PushOutput: pushOutput}
	}
}

// FindPRBranchForCommit finds the PR branch for a commit
func FindPRBranchForCommit(repo *internal.Repository, commitIndex int) string {
	if repo == nil || commitIndex < 0 || commitIndex >= len(repo.Graph.Commits) {
		return ""
	}

	openPRBranches := make(map[string]bool)
	for _, pr := range repo.PRs {
		if pr.State == "open" {
			openPRBranches[pr.HeadBranch] = true
		}
	}

	commitIDToIndex := make(map[string]int)
	for i, commit := range repo.Graph.Commits {
		commitIDToIndex[commit.ID] = i
		commitIDToIndex[commit.ChangeID] = i
	}

	visited := make(map[int]bool)
	queue := []int{commitIndex}

	for len(queue) > 0 {
		idx := queue[0]
		queue = queue[1:]

		if visited[idx] {
			continue
		}
		visited[idx] = true

		commit := repo.Graph.Commits[idx]
		for _, branch := range commit.Branches {
			if openPRBranches[branch] {
				return branch
			}
		}

		for _, parentID := range commit.Parents {
			if parentIdx, ok := commitIDToIndex[parentID]; ok && !visited[parentIdx] {
				queue = append(queue, parentIdx)
			}
		}
	}
	return ""
}
