package prform

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/github"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/tabs/bookmark"
	"github.com/madicen/jj-tui/internal/tui/tabs/prs"
	"github.com/madicen/jj-tui/internal/tui/util"
)

// PrepareCreatePRResult is the result of PrepareCreatePR for opening the create-PR dialog.
type PrepareCreatePRResult struct {
	HeadBranch        string
	NeedsMoveBookmark bool
	DefaultTitle      string
	Ok                bool // false if no branch (e.g. no bookmark); caller should set status and not show
}

// PrepareCreatePR returns data needed to show the create-PR dialog (head branch, needs move, default title).
// jiraTitles can be nil; if provided, default title is taken from jiraTitles[headBranch] when set.
func PrepareCreatePR(repo *internal.Repository, commitIdx int, jiraTitles map[string]string) PrepareCreatePRResult {
	if repo == nil || commitIdx < 0 || commitIdx >= len(repo.Graph.Commits) {
		return PrepareCreatePRResult{}
	}
	commit := repo.Graph.Commits[commitIdx]
	var headBranch string
	var needsMove bool
	if len(commit.Branches) > 0 {
		headBranch = commit.Branches[0]
		needsMove = false
	} else {
		headBranch = bookmark.FindBookmarkForCommit(repo, commitIdx)
		if headBranch == "" {
			return PrepareCreatePRResult{Ok: false}
		}
		needsMove = true
	}
	defaultTitle := headBranch
	if jiraTitles != nil {
		if t, ok := jiraTitles[headBranch]; ok && t != "" {
			defaultTitle = t
		}
	}
	return PrepareCreatePRResult{
		HeadBranch:        headBranch,
		NeedsMoveBookmark: needsMove,
		DefaultTitle:      defaultTitle,
		Ok:                true,
	}
}

// SubmitPRInput contains everything needed to run the PR submit (create or demo).
type SubmitPRInput struct {
	Title             string
	Body              string
	HeadBranch        string
	BaseBranch        string
	NeedsMoveBookmark bool
	CommitChangeID    string
	CommitIDsForDemo  []string // optional; used in demo mode for PR.CommitIDs
	JJService         *jj.Service
	GitHubService     *github.Service
	DemoMode          bool
}

// SubmitPRCmd validates, then runs the appropriate command (CreatePRCmd or a demo PRCreatedMsg).
// Returns (cmd, ""); if validation fails returns (nil, validationError). Caller sets status from validationError.
func SubmitPRCmd(input SubmitPRInput) (tea.Cmd, string) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, "Title is required"
	}
	if input.DemoMode {
		body := strings.TrimSpace(input.Body)
		commitIDs := input.CommitIDsForDemo
		if commitIDs == nil {
			commitIDs = []string{}
		}
		return func() tea.Msg {
			return PRCreatedMsg{PR: &internal.GitHubPR{
				Number:       999,
				Title:        title,
				Body:         body,
				State:        "open",
				HeadBranch:   input.HeadBranch,
				BaseBranch:   input.BaseBranch,
				URL:          "https://github.com/example/repo/pull/999",
				CommitIDs:    commitIDs,
				CheckStatus:  internal.CheckStatusPending,
				ReviewStatus: internal.ReviewStatusNone,
			}}
		}, ""
	}
	return CreatePRCmd(input.JJService, input.GitHubService, PRCreateParams{
		Title:             title,
		Body:              strings.TrimSpace(input.Body),
		HeadBranch:        input.HeadBranch,
		BaseBranch:        input.BaseBranch,
		NeedsMoveBookmark: input.NeedsMoveBookmark,
		CommitChangeID:    input.CommitChangeID,
	}), ""
}

// PRCreateParams contains parameters for PR creation.
type PRCreateParams struct {
	Title             string
	Body              string
	HeadBranch        string
	BaseBranch        string
	NeedsMoveBookmark bool
	CommitChangeID    string
}

// CreatePRCmd pushes a branch and creates a PR.
func CreatePRCmd(jjSvc *jj.Service, ghSvc *github.Service, params PRCreateParams) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if params.NeedsMoveBookmark && params.CommitChangeID != "" {
			if err := jjSvc.MoveBookmark(ctx, params.HeadBranch, params.CommitChangeID); err != nil {
				return util.ErrorMsg{Err: fmt.Errorf("failed to move bookmark %s: %w", params.HeadBranch, err)}
			}
		}
		pushOutput, err := jjSvc.PushToGit(ctx, params.HeadBranch)
		if err != nil {
			return util.ErrorMsg{Err: fmt.Errorf("failed to push branch: %w\nOutput: %s", err, pushOutput)}
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
			return util.ErrorMsg{Err: fmt.Errorf("failed to create PR: %w\nPush output: %s", lastErr, pushOutput)}
		}
		return PRCreatedMsg{PR: pr}
	}
}

// OpenCreatePRResult is the result of OpenCreatePR.
type OpenCreatePRResult struct {
	StatusMessage string
	Ok            bool
}

// OpenCreatePR prepares and shows the PR creation dialog for the selected commit's bookmark.
// Caller sets view mode and status message from the result.
func OpenCreatePR(modal *Model, repo *internal.Repository, commitIdx int, jiraTitles map[string]string, width, height int) OpenCreatePRResult {
	data := PrepareCreatePR(repo, commitIdx, jiraTitles)
	if !data.Ok {
		return OpenCreatePRResult{StatusMessage: "No bookmark found. Create one first with 'b'.", Ok: false}
	}
	modal.Show(commitIdx, "main", data.HeadBranch)
	modal.SetNeedsMoveBookmark(data.NeedsMoveBookmark)
	modal.SetTitle(data.DefaultTitle)
	modal.GetTitleInput().Focus()
	modal.GetBodyInput().Blur()
	modal.SetBody("")
	modal.GetTitleInput().Width = width
	modal.GetBodyInput().SetWidth(width)
	bodyHeight := height - 20
	if bodyHeight < 3 {
		bodyHeight = 3
	}
	if bodyHeight > 8 {
		bodyHeight = 8
	}
	modal.GetBodyInput().SetHeight(bodyHeight)
	statusMessage := "Creating PR for " + data.HeadBranch
	if data.NeedsMoveBookmark {
		statusMessage = fmt.Sprintf("Creating PR for %s (will move bookmark)", data.HeadBranch)
	}
	return OpenCreatePRResult{StatusMessage: statusMessage, Ok: true}
}

// SubmitPRResult is the result of SubmitPR.
type SubmitPRResult struct {
	Cmd           tea.Cmd
	StatusMessage string
}

// SubmitPR builds submit input from modal and repo/services and runs the PR create command.
func SubmitPR(modal *Model, repo *internal.Repository, jjService *jj.Service, githubService *github.Service, demoMode bool) SubmitPRResult {
	var commitChangeID string
	var commitIDsForDemo []string
	if repo != nil {
		idx := modal.GetCommitIndex()
		if idx >= 0 && idx < len(repo.Graph.Commits) {
			commit := repo.Graph.Commits[idx]
			commitChangeID = commit.ChangeID
			if demoMode {
				commitIDsForDemo = []string{commit.ID}
			}
		}
	}
	input := SubmitPRInput{
		Title:             modal.GetTitle(),
		Body:              modal.GetBody(),
		HeadBranch:        modal.GetHeadBranch(),
		BaseBranch:        modal.GetBaseBranch(),
		NeedsMoveBookmark: modal.NeedsMoveBookmark(),
		CommitChangeID:    commitChangeID,
		CommitIDsForDemo:  commitIDsForDemo,
		JJService:         jjService,
		GitHubService:     githubService,
		DemoMode:          demoMode,
	}
	statusMessage := "Creating PR for " + modal.GetHeadBranch()
	if demoMode {
		statusMessage = "Creating PR (demo)..."
	} else {
		statusMessage = fmt.Sprintf("%s %s and creating PR...", util.If(modal.NeedsMoveBookmark(), "Moving bookmark", "Pushing"), modal.GetHeadBranch())
	}
	cmd, errStr := SubmitPRCmd(input)
	if errStr != "" {
		return SubmitPRResult{StatusMessage: errStr}
	}
	return SubmitPRResult{Cmd: cmd, StatusMessage: statusMessage}
}

// HandlePRCreatedMsg mutates app (ViewMode, StatusMessage, Repository in demo) and returns the Cmd to run.
func HandlePRCreatedMsg(input PRCreatedInput, app *state.AppState) tea.Cmd {
	app.ViewMode = state.ViewCommitGraph
	app.StatusMessage = fmt.Sprintf("PR #%d created: %s", input.PR.Number, input.PR.Title)
	if input.DemoMode {
		if app.Repository != nil && input.PR != nil {
			app.Repository.PRs = append([]internal.GitHubPR{*input.PR}, app.Repository.PRs...)
		}
		return nil
	}
	existing := 0
	if app.Repository != nil {
		existing = len(app.Repository.PRs)
	}
	return tea.Batch(util.OpenURL(input.PR.URL), prs.LoadPRsCmd(app.GitHubService, app.GithubInfo, app.DemoMode, existing))
}

// PRCreatedInput is the context main sends when forwarding PRCreatedMsg.
type PRCreatedInput struct {
	PRCreatedMsg
	DemoMode bool
}
