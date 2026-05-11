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
// Bookmarks that already have an open PR are skipped so that "Create PR" targets a fresh branch.
func PrepareCreatePR(repo *internal.Repository, commitIdx int, jiraTitles map[string]string) PrepareCreatePRResult {
	if repo == nil || commitIdx < 0 || commitIdx >= len(repo.Graph.Commits) {
		return PrepareCreatePRResult{}
	}
	openPRBranches := make(map[string]bool)
	for _, pr := range repo.PRs {
		if pr.State == "open" {
			openPRBranches[pr.HeadBranch] = true
		}
	}
	commit := repo.Graph.Commits[commitIdx]
	var headBranch string
	var needsMove bool
	if len(commit.Branches) > 0 {
		headBranch = firstBookmarkWithoutOpenPR(commit.Branches, openPRBranches)
		if headBranch == "" {
			headBranch = util.FirstOperableBookmarkName(commit.Branches)
		}
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
		demoPR := &internal.GitHubPR{
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
		}
		// Delay so demo/VHS shows the same loading overlay as real create (push + GitHub API).
		const demoPRCreateDelay = 2 * time.Second
		return tea.Tick(demoPRCreateDelay, func(time.Time) tea.Msg {
			return PRCreatedMsg{PR: demoPR}
		}), ""
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
//
// Before the GitHub create call we preflight that the base branch actually exists on the
// remote: a fresh `gh repo create --source=. --remote=origin` produces a repo whose default
// branch may not have any commits pushed yet, and the previous code path would issue the
// create, get a 422 with `Field:base Code:invalid`, and retry up to 5 times (15s of dead
// time) before surfacing a confusing error. The preflight short-circuits that with a clear
// actionable hint instead, and the retry loop now only kicks in for transient head-related
// failures (the case it was actually written for).
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
			return util.ErrorMsg{Err: fmt.Errorf("failed to push branch: %w\nOutput: %s%s", err, pushOutput, util.MissingOriginHint(err))}
		}
		// Preflight base-branch existence. We swallow the bool-side error (network blips, auth
		// hiccups) because the create call below will surface the same problem with richer
		// detail; the preflight only short-circuits the unambiguous "base doesn't exist" case.
		if exists, perr := ghSvc.BranchExists(ctx, params.BaseBranch); perr == nil && !exists {
			return util.ErrorMsg{Err: fmt.Errorf(
				"base branch %q does not exist on the remote (%s/%s).\n\n"+
					"This usually means the GitHub repo is fresh and that branch hasn't been pushed yet. Fixes:\n"+
					"  - Push your local %s bookmark to origin (Settings → GitHub → Push all bookmarks),\n"+
					"  - or change the repo's default branch on GitHub to one that does exist,\n"+
					"  - or pick a different base when creating the PR",
				params.BaseBranch, ghSvc.GetOwner(), ghSvc.GetRepo(), params.BaseBranch,
			)}
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
			// Only retry transient head-ref propagation issues. A base-related 422 is
			// permanent until the user changes the base or pushes the branch — retrying
			// just delays the actionable error 15 seconds with no chance of success.
			msg := lastErr.Error()
			lower := strings.ToLower(msg)
			baseRelated := strings.Contains(lower, "field=base") ||
				strings.Contains(lower, ".base=") ||
				strings.Contains(lower, "base branch") ||
				strings.Contains(lower, "base ref")
			if baseRelated {
				break
			}
			if strings.Contains(lower, "not all refs") || strings.Contains(lower, "422") {
				time.Sleep(3 * time.Second)
				continue
			}
			break
		}
		if lastErr != nil {
			detail := lastErr.Error()
			lower := strings.ToLower(detail)
			if strings.Contains(lower, "field=base") || strings.Contains(lower, ".base=") {
				detail += fmt.Sprintf(
					"\n\nHint: GitHub rejected the PR's base branch %q. The branch may not exist on %s/%s yet — push it via Settings → GitHub → Push all bookmarks, or open the PR against a different base.",
					params.BaseBranch, ghSvc.GetOwner(), ghSvc.GetRepo(),
				)
			}
			return util.ErrorMsg{Err: fmt.Errorf("failed to create PR: %s\nPush output: %s", detail, pushOutput)}
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
// height is the content area height (available lines). The body textarea uses the rest after fixed form lines.
// defaultBranch is the resolved GitHub default branch (e.g. "main", "master", "trunk"); when
// empty the form falls back to "main" to preserve the legacy behavior on repos where the
// lookup hasn't completed or the GitHub service is unavailable.
// Caller sets view mode and status message from the result.
func OpenCreatePR(modal *Model, repo *internal.Repository, commitIdx int, jiraTitles map[string]string, defaultBranch string, width, height int) OpenCreatePRResult {
	data := PrepareCreatePR(repo, commitIdx, jiraTitles)
	if !data.Ok {
		return OpenCreatePRResult{StatusMessage: "No bookmark found. Create one first with 'b'.", Ok: false}
	}
	baseBranch := strings.TrimSpace(defaultBranch)
	if baseBranch == "" {
		baseBranch = "main"
	}
	modal.Show(commitIdx, baseBranch, data.HeadBranch)
	modal.SetNeedsMoveBookmark(data.NeedsMoveBookmark)
	modal.SetTitle(data.DefaultTitle)
	modal.GetTitleInput().Focus()
	modal.GetBodyInput().Blur()
	modal.SetBody("")
	modal.GetTitleInput().Width = width
	modal.GetBodyInput().SetWidth(width)
	// Use full content height: fixed lines (title, branch, "Title:", title input, "Body:", buttons) ≈ 11
	const fixedFormLines = 11
	bodyHeight := height - fixedFormLines
	if bodyHeight < 3 {
		bodyHeight = 3
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
	cmd, errStr := SubmitPRCmd(input)
	if errStr != "" {
		return SubmitPRResult{StatusMessage: errStr}
	}
	var statusMessage string
	if demoMode {
		statusMessage = "Creating PR (demo)..."
	} else {
		statusMessage = fmt.Sprintf("%s %s and creating PR...", util.If(modal.NeedsMoveBookmark(), "Moving bookmark", "Pushing"), modal.GetHeadBranch())
	}
	return SubmitPRResult{Cmd: cmd, StatusMessage: statusMessage}
}

// HandlePRCreatedMsg mutates app (ViewMode, StatusMessage, Repository in demo) and returns the Cmd to run.
func HandlePRCreatedMsg(input PRCreatedInput, app *state.AppState) tea.Cmd {
	app.Loading = false
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

// firstBookmarkWithoutOpenPR returns the first operable bookmark name that does
// not match any open PR head branch, or "" if none found.
func firstBookmarkWithoutOpenPR(branches []string, openPRBranches map[string]bool) string {
	for _, b := range branches {
		b = strings.TrimSpace(b)
		if b == "" {
			continue
		}
		name, _ := util.NormalizeBookmarkListToken(b)
		if name == "" || strings.Contains(name, "@") {
			continue
		}
		local := util.LocalBookmarkName(name)
		if openPRBranches[name] || openPRBranches[local] {
			continue
		}
		return name
	}
	return ""
}
