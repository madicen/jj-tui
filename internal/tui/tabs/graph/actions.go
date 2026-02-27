package graph

import (
	"context"
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/data"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/util"
	bookmarktab "github.com/madicen/jj-tui/internal/tui/tabs/bookmark"
	descedittab "github.com/madicen/jj-tui/internal/tui/tabs/descedit"
	prstab "github.com/madicen/jj-tui/internal/tui/tabs/prs"
)

// HandleRequest runs the requested graph action using the given context.
func HandleRequest(r Request, ctx *RequestContext) Result {
	if ctx == nil {
		if r.Checkout {
			return Result{Status: "Cannot edit: repository not loaded"}
		}
		return Result{}
	}
	if r.LoadChangedFiles != nil {
		return Result{FollowUp: FollowUpLoadChangedFiles, ChangeID: *r.LoadChangedFiles, CommitIndex: -1}
	}
	if r.SelectCommit != nil {
		idx := *r.SelectCommit
		if ctx.Repository == nil || idx < 0 || idx >= len(ctx.Repository.Graph.Commits) {
			return Result{}
		}
		commit := ctx.Repository.Graph.Commits[idx]
		return Result{FollowUp: FollowUpLoadChangedFiles, ChangeID: commit.ChangeID, CommitIndex: idx}
	}
	if ctx.JJService == nil && !r.StartEditDescription && !r.StartRebaseMode && r.ResolveDivergent == nil {
		if r.Checkout {
			return Result{Status: "Cannot edit: not in a jj repository"}
		}
		return Result{}
	}
	if r.Checkout {
		cmd, status := executeCheckout(ctx)
		return Result{Cmd: cmd, Status: status}
	}
	if r.Squash {
		cmd, status := executeSquash(ctx)
		return Result{Cmd: cmd, Status: status}
	}
	if r.Abandon {
		if ctx.IsSelectedCommitValid() {
			commit := ctx.Repository.Graph.Commits[ctx.SelectedCommit]
			if commit.Divergent {
				return Result{FollowUp: FollowUpResolveDivergent, ChangeID: commit.ChangeID}
			}
		}
		cmd, status := executeAbandon(ctx)
		return Result{Cmd: cmd, Status: status}
	}
	if r.PerformRebase {
		cmd, status := executePerformRebase(r.RebaseDestIndex, ctx)
		if status != "" {
			return Result{Status: status}
		}
		if cmd != nil && ctx.RebaseSourceCommit >= 0 && ctx.RebaseSourceCommit < len(ctx.Repository.Graph.Commits) &&
			r.RebaseDestIndex >= 0 && r.RebaseDestIndex < len(ctx.Repository.Graph.Commits) {
			src := ctx.Repository.Graph.Commits[ctx.RebaseSourceCommit]
			dst := ctx.Repository.Graph.Commits[r.RebaseDestIndex]
			return Result{Cmd: cmd, SuccessStatus: fmt.Sprintf("Rebasing %s onto %s...", src.ShortID, dst.ShortID), PerformRebase: true}
		}
		return Result{Cmd: cmd, PerformRebase: true}
	}
	if r.DeleteBookmark {
		cmd, status := executeDeleteBookmark(ctx)
		return Result{Cmd: cmd, Status: status}
	}
	if r.MoveFileUp {
		cmd, status := executeMoveFileUp(ctx)
		return Result{Cmd: cmd, Status: status}
	}
	if r.MoveFileDown {
		cmd, status := executeMoveFileDown(ctx)
		return Result{Cmd: cmd, Status: status}
	}
	if r.RevertFile {
		cmd, status := executeRevertFile(ctx)
		return Result{Cmd: cmd, Status: status}
	}
	if r.NewCommit {
		cmd, _ := executeNewCommit(ctx)
		status := "Creating new commit..."
		if ctx.IsSelectedCommitValid() {
			commit := ctx.Repository.Graph.Commits[ctx.SelectedCommit]
			status = fmt.Sprintf("Creating new commit from %s...", commit.ShortID)
		}
		return Result{Cmd: cmd, NewCommitStatus: status}
	}
	if r.ResolveDivergent != nil {
		if !ctx.IsSelectedCommitValid() {
			return Result{}
		}
		commit := ctx.Repository.Graph.Commits[ctx.SelectedCommit]
		if !commit.Divergent {
			return Result{Status: "This commit is not divergent"}
		}
		return Result{FollowUp: FollowUpResolveDivergent, ChangeID: *r.ResolveDivergent}
	}
	if r.StartEditDescription {
		if !ctx.IsSelectedCommitValid() {
			return Result{}
		}
		commit := ctx.Repository.Graph.Commits[ctx.SelectedCommit]
		if commit.Immutable {
			return Result{Status: "Cannot edit description: commit is immutable"}
		}
		return Result{FollowUp: FollowUpStartEditDescription, CommitIndex: ctx.SelectedCommit}
	}
	if r.StartRebaseMode {
		if !ctx.IsSelectedCommitValid() {
			return Result{}
		}
		commit := ctx.Repository.Graph.Commits[ctx.SelectedCommit]
		if commit.Immutable {
			return Result{Status: "Cannot rebase: commit is immutable"}
		}
		return Result{FollowUp: FollowUpStartRebaseMode}
	}
	if r.CreateBookmark {
		if !ctx.IsSelectedCommitValid() || ctx.JJService == nil {
			return Result{}
		}
		commit := ctx.Repository.Graph.Commits[ctx.SelectedCommit]
		if commit.Immutable {
			return Result{Status: "Cannot create bookmark: commit is immutable"}
		}
		return Result{FollowUp: FollowUpCreateBookmark}
	}
	if r.CreatePR {
		if !ctx.GitHubAvailable {
			return Result{Status: "GitHub not connected. Configure in Settings (,)"}
		}
		if !ctx.IsSelectedCommitValid() || ctx.JJService == nil {
			return Result{}
		}
		emptyDescCommits := FindCommitsWithEmptyDescriptions(ctx.Repository, ctx.SelectedCommit)
		if len(emptyDescCommits) > 0 {
			return Result{
				FollowUp:       FollowUpShowEmptyDescWarning,
				WarningTitle:   "Commits Need Descriptions",
				WarningMessage: "GitHub requires commit descriptions. Please add descriptions before creating a PR.",
				WarningCommits: emptyDescCommits,
			}
		}
		return Result{FollowUp: FollowUpCreatePR}
	}
	if r.UpdatePR {
		if !ctx.IsSelectedCommitValid() || ctx.JJService == nil {
			return Result{}
		}
		emptyDescCommits := FindCommitsWithEmptyDescriptions(ctx.Repository, ctx.SelectedCommit)
		if len(emptyDescCommits) > 0 {
			return Result{
				FollowUp:       FollowUpShowEmptyDescWarning,
				WarningTitle:   "Commits Need Descriptions",
				WarningMessage: "GitHub requires commit descriptions. Please add descriptions before updating the PR.",
				WarningCommits: emptyDescCommits,
			}
		}
		return Result{FollowUp: FollowUpUpdatePR}
	}
	return Result{}
}

// ExecuteRequest is deprecated: use HandleRequest and Result instead.
func ExecuteRequest(r Request, ctx *RequestContext) (cmd tea.Cmd, statusMsg string) {
	res := HandleRequest(r, ctx)
	return res.Cmd, res.Status
}

func executeCheckout(ctx *RequestContext) (tea.Cmd, string) {
	if !ctx.IsSelectedCommitValid() {
		return nil, ""
	}
	if ctx.JJService == nil {
		return nil, "Cannot edit: not in a jj repository"
	}
	commit := ctx.Repository.Graph.Commits[ctx.SelectedCommit]
	if commit.IsWorking {
		return nil, "Already editing this commit"
	}
	if commit.Immutable {
		return nil, "Cannot edit: commit is immutable"
	}
	return Checkout(ctx.JJService, commit.ChangeID), ""
}

func executeSquash(ctx *RequestContext) (tea.Cmd, string) {
	if !ctx.IsSelectedCommitValid() {
		return nil, ""
	}
	commit := ctx.Repository.Graph.Commits[ctx.SelectedCommit]
	if commit.Immutable {
		return nil, "Cannot squash: commit is immutable"
	}
	return Squash(ctx.JJService, commit.ChangeID), ""
}

func executeAbandon(ctx *RequestContext) (tea.Cmd, string) {
	if !ctx.IsSelectedCommitValid() {
		return nil, ""
	}
	commit := ctx.Repository.Graph.Commits[ctx.SelectedCommit]
	if commit.Immutable {
		return nil, "Cannot abandon: commit is immutable"
	}
	if commit.Divergent {
		return nil, "__divergent__"
	}
	return Abandon(ctx.JJService, commit.ChangeID), ""
}

func executePerformRebase(destIndex int, ctx *RequestContext) (tea.Cmd, string) {
	if !ctx.IsSelectedCommitValid() || ctx.RebaseSourceCommit < 0 ||
		ctx.RebaseSourceCommit >= len(ctx.Repository.Graph.Commits) ||
		destIndex < 0 || destIndex >= len(ctx.Repository.Graph.Commits) {
		return nil, ""
	}
	if ctx.RebaseSourceCommit == destIndex {
		return nil, "Cannot rebase commit onto itself"
	}
	sourceCommit := ctx.Repository.Graph.Commits[ctx.RebaseSourceCommit]
	destCommit := ctx.Repository.Graph.Commits[destIndex]
	return Rebase(ctx.JJService, sourceCommit.ChangeID, destCommit.ChangeID), ""
}

func executeDeleteBookmark(ctx *RequestContext) (tea.Cmd, string) {
	if !ctx.IsSelectedCommitValid() {
		return nil, ""
	}
	commit := ctx.Repository.Graph.Commits[ctx.SelectedCommit]
	if len(commit.Branches) == 0 {
		return nil, "No bookmark on this commit to delete"
	}
	return bookmarktab.DeleteBookmarkCmd(ctx.JJService, commit.Branches[0]), ""
}

func executeMoveFileUp(ctx *RequestContext) (tea.Cmd, string) {
	if ctx.GraphFocused || len(ctx.ChangedFiles) == 0 {
		return nil, ""
	}
	if ctx.SelectedFile < 0 || ctx.SelectedFile >= len(ctx.ChangedFiles) {
		return nil, ""
	}
	if !ctx.IsSelectedCommitValid() {
		return nil, ""
	}
	commit := ctx.Repository.Graph.Commits[ctx.SelectedCommit]
	if commit.Immutable {
		return nil, "Cannot move file: commit is immutable"
	}
	return SplitFileToParent(ctx.JJService, commit.ChangeID, ctx.ChangedFiles[ctx.SelectedFile].Path), ""
}

func executeMoveFileDown(ctx *RequestContext) (tea.Cmd, string) {
	if ctx.GraphFocused || len(ctx.ChangedFiles) == 0 {
		return nil, ""
	}
	if ctx.SelectedFile < 0 || ctx.SelectedFile >= len(ctx.ChangedFiles) {
		return nil, ""
	}
	if !ctx.IsSelectedCommitValid() {
		return nil, ""
	}
	commit := ctx.Repository.Graph.Commits[ctx.SelectedCommit]
	if commit.Immutable {
		return nil, "Cannot move file: commit is immutable"
	}
	return MoveFileToChild(ctx.JJService, commit.ChangeID, ctx.ChangedFiles[ctx.SelectedFile].Path), ""
}

func executeRevertFile(ctx *RequestContext) (tea.Cmd, string) {
	if ctx.GraphFocused || len(ctx.ChangedFiles) == 0 {
		return nil, ""
	}
	if ctx.SelectedFile < 0 || ctx.SelectedFile >= len(ctx.ChangedFiles) {
		return nil, ""
	}
	if !ctx.IsSelectedCommitValid() {
		return nil, ""
	}
	commit := ctx.Repository.Graph.Commits[ctx.SelectedCommit]
	if commit.Immutable {
		return nil, "Cannot revert file: commit is immutable"
	}
	return RevertFile(ctx.JJService, commit.ChangeID, ctx.ChangedFiles[ctx.SelectedFile].Path), ""
}

func executeNewCommit(ctx *RequestContext) (tea.Cmd, string) {
	parentCommitID := ""
	if ctx.IsSelectedCommitValid() {
		parentCommitID = ctx.Repository.Graph.Commits[ctx.SelectedCommit].ChangeID
	}
	return NewCommit(ctx.JJService, parentCommitID), ""
}

// SaveDescriptionCmd returns a command to save the description for the given commit.
func SaveDescriptionCmd(jjService *jj.Service, commitID, body string) tea.Cmd {
	return descedittab.SaveDescriptionCmd(jjService, commitID, strings.TrimSpace(body))
}

// CreateBookmarkCmd returns a command to create a bookmark.
func CreateBookmarkCmd(jjService *jj.Service, bookmarkName, commitID string) tea.Cmd {
	return bookmarktab.CreateBookmarkCmd(jjService, bookmarkName, commitID)
}

// ApplyResult applies the result: updates the graph model, mutates app state, and returns the Cmd to run.
// For follow-ups that require main to open a modal (edit description, create bookmark, warning, create PR),
// it returns a state.NavigateMsg cmd. For load/update PR it sets app status and returns the cmd directly.
func ApplyResult(res Result, graphModel *GraphModel, ctx *RequestContext, app *state.AppState) tea.Cmd {
	if res.Status != "" {
		app.StatusMessage = res.Status
	}
	switch res.FollowUp {
	case FollowUpLoadChangedFiles:
		if res.CommitIndex >= 0 {
			graphModel.SelectCommit(res.CommitIndex)
		}
		if ctx != nil && ctx.JJService != nil {
			return LoadChangedFilesCmd(ctx.JJService, res.ChangeID)
		}
		return nil
	case FollowUpResolveDivergent:
		if ctx != nil && ctx.JJService != nil {
			app.StatusMessage = "Loading divergent commit info..."
			return LoadDivergentCommitInfoCmd(ctx.JJService, res.ChangeID)
		}
		return nil
	case FollowUpStartEditDescription:
		if ctx != nil && ctx.Repository != nil && res.CommitIndex >= 0 && res.CommitIndex < len(ctx.Repository.Graph.Commits) {
			return state.NavigateTarget{Kind: state.NavigateEditDescription, Commit: ctx.Repository.Graph.Commits[res.CommitIndex]}.Cmd()
		}
		return nil
	case FollowUpStartRebaseMode:
		if ctx != nil && ctx.Repository != nil && ctx.SelectedCommit >= 0 && ctx.SelectedCommit < len(ctx.Repository.Graph.Commits) {
			graphModel.StartRebaseMode(ctx.SelectedCommit)
			app.StatusMessage = RebaseModeStartMessage(ctx.Repository.Graph.Commits[ctx.SelectedCommit].ShortID)
		}
		return nil
	case FollowUpCreateBookmark:
		return state.NavigateTarget{Kind: state.NavigateCreateBookmark}.Cmd()
	case FollowUpShowEmptyDescWarning:
		return state.NavigateTarget{
			Kind:          state.NavigateWarning,
			WarningTitle:  res.WarningTitle,
			WarningMessage: res.WarningMessage,
			WarningCommits: res.WarningCommits,
		}.Cmd()
	case FollowUpCreatePR:
		return state.NavigateTarget{Kind: state.NavigateCreatePR}.Cmd()
	case FollowUpUpdatePR:
		if ctx == nil || ctx.Repository == nil || !ctx.IsSelectedCommitValid() {
			return nil
		}
		prBranch := FindPRBranchForCommit(ctx.Repository, ctx.SelectedCommit)
		if prBranch == "" {
			app.StatusMessage = "No open PR found for this commit or its ancestors"
			return nil
		}
		commit := ctx.Repository.Graph.Commits[ctx.SelectedCommit]
		needsMoveBookmark := !slices.Contains(commit.Branches, prBranch)
		if needsMoveBookmark {
			app.StatusMessage = fmt.Sprintf("Moving %s and pushing...", prBranch)
		} else {
			app.StatusMessage = fmt.Sprintf("Pushing %s...", prBranch)
		}
		return prstab.PushToPRCmd(ctx.JJService, prBranch, commit.ChangeID, needsMoveBookmark)
	}
	if res.Cmd != nil {
		if res.PerformRebase {
			graphModel.CancelRebaseMode()
		}
		if res.NewCommitStatus != "" {
			app.StatusMessage = res.NewCommitStatus
		}
		if res.SuccessStatus != "" {
			app.StatusMessage = res.SuccessStatus
		}
		return res.Cmd
	}
	return nil
}

// NewCommit creates a new commit as a child of the given parent.
func NewCommit(svc *jj.Service, parentCommitID string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.NewCommit(context.Background(), parentCommitID); err != nil {
			return util.ErrorMsg{Err: fmt.Errorf("failed to create commit: %w", err)}
		}
		repo, err := svc.GetRepository(context.Background(), "")
		if err != nil {
			return util.ErrorMsg{Err: err}
		}
		return RepositoryLoadedMsg{Repository: repo}
	}
}

// Checkout checks out (edits) the specified commit.
func Checkout(svc *jj.Service, changeID string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.CheckoutCommit(context.Background(), changeID); err != nil {
			return util.ErrorMsg{Err: fmt.Errorf("failed to checkout: %w", err)}
		}
		repo, err := svc.GetRepository(context.Background(), "")
		if err != nil {
			return util.ErrorMsg{Err: err}
		}
		return EditCompletedMsg{Repository: repo}
	}
}

// Squash squashes the specified commit into its parent.
func Squash(svc *jj.Service, changeID string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.SquashCommit(context.Background(), changeID); err != nil {
			return util.ErrorMsg{Err: fmt.Errorf("failed to squash: %w", err)}
		}
		repo, err := svc.GetRepository(context.Background(), "")
		if err != nil {
			return util.ErrorMsg{Err: err}
		}
		return RepositoryLoadedMsg{Repository: repo}
	}
}

// Abandon abandons the specified commit.
func Abandon(svc *jj.Service, changeID string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.AbandonCommit(context.Background(), changeID); err != nil {
			return util.ErrorMsg{Err: fmt.Errorf("failed to abandon: %w", err)}
		}
		repo, err := svc.GetRepository(context.Background(), "")
		if err != nil {
			return util.ErrorMsg{Err: err}
		}
		return RepositoryLoadedMsg{Repository: repo}
	}
}

// Rebase rebases the source commit onto the destination.
func Rebase(svc *jj.Service, sourceChangeID, destChangeID string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.RebaseCommit(context.Background(), sourceChangeID, destChangeID); err != nil {
			return util.ErrorMsg{Err: fmt.Errorf("failed to rebase: %w", err)}
		}
		repo, err := svc.GetRepository(context.Background(), "")
		if err != nil {
			return util.ErrorMsg{Err: err}
		}
		return RepositoryLoadedMsg{Repository: repo}
	}
}

// SplitFileToParent moves a file from a commit to a new parent commit.
func SplitFileToParent(svc *jj.Service, commitID, filePath string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.SplitFileToParent(context.Background(), commitID, filePath); err != nil {
			return util.ErrorMsg{Err: fmt.Errorf("failed to move file to parent: %w", err)}
		}
		repo, err := svc.GetRepository(context.Background(), "")
		if err != nil {
			return util.ErrorMsg{Err: err}
		}
		return FileMoveCompletedMsg{Repository: repo, FilePath: filePath, Direction: "up"}
	}
}

// MoveFileToChild moves a file from a commit to a new child commit.
func MoveFileToChild(svc *jj.Service, commitID, filePath string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.MoveFileToChild(context.Background(), commitID, filePath); err != nil {
			return util.ErrorMsg{Err: fmt.Errorf("failed to move file to child: %w", err)}
		}
		repo, err := svc.GetRepository(context.Background(), "")
		if err != nil {
			return util.ErrorMsg{Err: err}
		}
		return FileMoveCompletedMsg{Repository: repo, FilePath: filePath, Direction: "down"}
	}
}

// RevertFile reverts all changes to a file in a commit.
func RevertFile(svc *jj.Service, commitID, filePath string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.RevertFile(context.Background(), commitID, filePath); err != nil {
			return util.ErrorMsg{Err: fmt.Errorf("failed to revert file: %w", err)}
		}
		repo, err := svc.GetRepository(context.Background(), "")
		if err != nil {
			return util.ErrorMsg{Err: err}
		}
		return FileRevertedMsg{Repository: repo, FilePath: filePath}
	}
}

// FindPRBranchForCommit finds the PR branch for a commit (BFS over ancestors for an open PR head branch).
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

// FileMoveInput is the context main sends when forwarding FileMoveCompletedMsg.
type FileMoveInput struct {
	FileMoveCompletedMsg
	ChangedFilesCommitID string
}

// FileRevertedInput is the context main sends when forwarding FileRevertedMsg.
type FileRevertedInput struct {
	FileRevertedMsg
	ChangedFilesCommitID string
}

// HandleFileMoveCompletedMsg mutates app and returns the Cmd to run.
// Caller (main model) should run LoadChangedFiles for the currently selected commit after updating the graph;
// we do not run LoadRepository here to avoid RepositoryLoadedMsg re-entering and overwriting graph/selection state.
func HandleFileMoveCompletedMsg(input FileMoveInput, app *state.AppState) tea.Cmd {
	var oldPRs []internal.GitHubPR
	if app.Repository != nil {
		oldPRs = app.Repository.PRs
	}
	app.Repository = input.Repository
	if app.Repository != nil {
		app.Repository.PRs = oldPRs
	}
	directionText := "new parent commit"
	if input.Direction == "down" {
		directionText = "new child commit"
	}
	app.StatusMessage = fmt.Sprintf("Moved %s to %s", input.FilePath, directionText)
	return nil
}

// HandleFileRevertedMsg mutates app and returns the Cmd to run.
// Caller (main model) should run LoadChangedFiles for the currently selected commit after updating the graph.
func HandleFileRevertedMsg(input FileRevertedInput, app *state.AppState) tea.Cmd {
	var oldPRs []internal.GitHubPR
	if app.Repository != nil {
		oldPRs = app.Repository.PRs
	}
	app.Repository = input.Repository
	if app.Repository != nil {
		app.Repository.PRs = oldPRs
	}
	app.StatusMessage = fmt.Sprintf("Reverted changes to %s", input.FilePath)
	return nil
}

// UndoErrorInfo is returned when undo completed with an error.
type UndoErrorInfo struct {
	Err error
}

// HandleUndoCompletedMsg mutates app (StatusMessage). On error returns (nil, *UndoErrorInfo); on success returns (refresh cmd, nil).
func HandleUndoCompletedMsg(msg UndoCompletedMsg, app *state.AppState) (tea.Cmd, *UndoErrorInfo) {
	if msg.Err != nil {
		app.StatusMessage = "Error: " + msg.Err.Error()
		return nil, &UndoErrorInfo{Err: msg.Err}
	}
	app.StatusMessage = msg.Message
	return data.LoadRepository(app.JJService), nil
}

// Effect types used by ApplyResult (only for FollowUps that require main; others set app and return cmd).
// These are still sent when app is nil (e.g. tests). NavigateTarget.Cmd() is used when app is non-nil.
type SetStatusEffect struct{ Status string }
type StartEditDescriptionEffect struct{ Commit internal.Commit }
type StartRebaseModeEffect struct{ Status string }
type CreateBookmarkEffect struct{}
type ShowEmptyDescWarningEffect struct {
	Title   string
	Message string
	Commits []internal.Commit
}
type StartCreatePREffect struct{}
type UpdatePREffect struct {
	PrBranch          string
	CommitID          string
	NeedsMoveBookmark bool
}
type LoadChangedFilesEffect struct{ Cmd tea.Cmd }
type LoadDivergentEffect struct {
	Status string
	Cmd    tea.Cmd
}

func StartEditDescriptionCmd(commit internal.Commit) tea.Cmd {
	return func() tea.Msg { return StartEditDescriptionEffect{Commit: commit} }
}

func CreateBookmarkEffectCmd() tea.Cmd {
	return func() tea.Msg { return CreateBookmarkEffect{} }
}

func ShowEmptyDescWarningCmd(title, message string, commits []internal.Commit) tea.Cmd {
	return func() tea.Msg { return ShowEmptyDescWarningEffect{Title: title, Message: message, Commits: commits} }
}

func StartCreatePRCmd() tea.Cmd {
	return func() tea.Msg { return StartCreatePREffect{} }
}

func UpdatePREffectCmd(prBranch, commitID string, needsMoveBookmark bool) tea.Cmd {
	return func() tea.Msg { return UpdatePREffect{PrBranch: prBranch, CommitID: commitID, NeedsMoveBookmark: needsMoveBookmark} }
}

func LoadChangedFilesEffectCmd(cmd tea.Cmd) tea.Cmd {
	return func() tea.Msg { return LoadChangedFilesEffect{Cmd: cmd} }
}

func LoadDivergentEffectCmd(status string, cmd tea.Cmd) tea.Cmd {
	return func() tea.Msg { return LoadDivergentEffect{Status: status, Cmd: cmd} }
}

func SetStatusCmd(status string) tea.Cmd {
	return func() tea.Msg { return SetStatusEffect{Status: status} }
}

func StartRebaseModeCmd(status string) tea.Cmd {
	return func() tea.Msg { return StartRebaseModeEffect{Status: status} }
}
