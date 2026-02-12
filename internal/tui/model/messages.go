package model

import (
	"time"

	"github.com/madicen/jj-tui/internal/github"
	"github.com/madicen/jj-tui/internal/jj"
	"github.com/madicen/jj-tui/internal/models"
	"github.com/madicen/jj-tui/internal/tickets"
)

// tickMsg is sent on each timer tick for auto-refresh (jj repository)
type tickMsg time.Time

// prTickMsg is sent on each timer tick for PR auto-refresh
type prTickMsg time.Time

// repositoryLoadedMsg is sent when repository data is loaded
type repositoryLoadedMsg struct {
	repository *models.Repository
}

// editCompletedMsg is sent when an edit operation completes
type editCompletedMsg struct {
	repository *models.Repository
}

// servicesInitializedMsg is sent when all services are initialized
type servicesInitializedMsg struct {
	jjService     *jj.Service
	githubService *github.Service
	ticketService tickets.Service
	ticketError   error  // Error from ticket service initialization (for debugging)
	repository    *models.Repository
	githubInfo    string // Diagnostic info about GitHub connection (token source, repo)
	demoMode      bool   // True if running in demo mode with mock services
}

// prsLoadedMsg is sent when PRs are loaded from GitHub
type prsLoadedMsg struct {
	prs []models.GitHubPR
}

// ticketsLoadedMsg is sent when tickets are loaded
type ticketsLoadedMsg struct {
	tickets []tickets.Ticket
}

// transitionsLoadedMsg is sent when available transitions are loaded for a ticket
type transitionsLoadedMsg struct {
	transitions []tickets.Transition
}

// transitionCompletedMsg is sent when a ticket status transition completes
type transitionCompletedMsg struct {
	ticketKey string
	newStatus string
	err       error
}

// settingsSavedMsg is sent when settings are saved
type settingsSavedMsg struct {
	githubConnected bool
	ticketService   tickets.Service
	ticketProvider  string // "jira", "codecks", or ""
	savedLocal      bool   // true if saved to local .jj-tui.json
	err             error  // error if save failed
}

// ErrorMsgType is the error message type (exported for testing)
type ErrorMsgType struct {
	Err         error
	NotJJRepo   bool   // true if the error is "not a jj repository"
	CurrentPath string // the path where we tried to find a jj repo
}

// errorMsg is the internal alias for ErrorMsgType
type errorMsg = ErrorMsgType

// ErrorMsg creates an error message for testing purposes
func ErrorMsg(err error) ErrorMsgType {
	return ErrorMsgType{Err: err}
}

// jjInitSuccessMsg is sent when jj init succeeds
type jjInitSuccessMsg struct{}

// GitHub Device Flow messages

// githubDeviceFlowStartedMsg is sent when device flow authentication starts
type githubDeviceFlowStartedMsg struct {
	deviceCode      string
	userCode        string
	verificationURL string
	interval        int
}

// githubLoginSuccessMsg is sent when GitHub login succeeds
type githubLoginSuccessMsg struct {
	token string
}

// githubLoginPollMsg is sent to continue polling for GitHub token
type githubLoginPollMsg struct {
	interval int // Polling interval in seconds
}

// descriptionSavedMsg is sent when a commit description is saved
type descriptionSavedMsg struct {
	commitID string
}

// prCreatedMsg is sent when a PR is successfully created
type prCreatedMsg struct {
	pr *models.GitHubPR
}

// prMergedMsg is sent when a PR is successfully merged
type prMergedMsg struct {
	prNumber int
	err      error
}

// prClosedMsg is sent when a PR is successfully closed
type prClosedMsg struct {
	prNumber int
	err      error
}

// branchPushedMsg is sent when a branch is pushed to remote
type branchPushedMsg struct {
	branch     string
	pushOutput string
}

// bookmarkCreatedOnCommitMsg is sent when a bookmark is created or moved on a commit
type bookmarkCreatedOnCommitMsg struct {
	bookmarkName string
	commitID     string
	wasMoved     bool   // true if bookmark was moved, false if newly created
	ticketKey    string // set when creating from a ticket (for auto-transition)
}

// bookmarkDeletedMsg is sent when a bookmark is deleted
type bookmarkDeletedMsg struct {
	bookmarkName string
}

// bookmarkConflictInfoMsg contains info about a conflicted bookmark
type bookmarkConflictInfoMsg struct {
	bookmarkName  string
	localID       string
	remoteID      string
	localSummary  string
	remoteSummary string
	err           error
}

// bookmarkConflictResolvedMsg is sent when a bookmark conflict is resolved
type bookmarkConflictResolvedMsg struct {
	bookmarkName string
	resolution   string // "keep_local" or "reset_remote"
	err          error
}

// divergentCommitInfoMsg contains info about divergent commits
type divergentCommitInfoMsg struct {
	changeID  string
	commitIDs []string
	summaries []string
	err       error
}

// divergentCommitResolvedMsg is sent when a divergent commit is resolved
type divergentCommitResolvedMsg struct {
	changeID     string
	keptCommitID string
	err          error
}

// changedFilesLoadedMsg is sent when changed files for a commit are loaded
type changedFilesLoadedMsg struct {
	commitID string
	files    []jj.ChangedFile
}

// silentRepositoryLoadedMsg is for background refreshes that don't update the status
type silentRepositoryLoadedMsg struct {
	repository *models.Repository
}

// descriptionLoadedMsg contains the full description fetched from jj
type descriptionLoadedMsg struct {
	commitID    string
	description string
}

// cleanupCompletedMsg is sent when a cleanup operation completes
type cleanupCompletedMsg struct {
	success bool
	message string
	err     error
}

// undoCompletedMsg is sent when an undo/redo operation completes
type undoCompletedMsg struct {
	message string
}

// fileMoveCompletedMsg is sent when a file is moved to a new commit
type fileMoveCompletedMsg struct {
	repository *models.Repository
	filePath   string
	direction  string // "up" or "down"
}

// fileRevertedMsg is sent when a file's changes are reverted
type fileRevertedMsg struct {
	repository *models.Repository
	filePath   string
}

// branchesLoadedMsg is sent when branches are loaded
type branchesLoadedMsg struct {
	branches []models.Branch
	err      error
}

// branchActionMsg is sent when a branch action completes (track, untrack, push, fetch)
type branchActionMsg struct {
	action string // "track", "untrack", "push", "fetch"
	branch string
	err    error
}

// githubReauthNeededMsg is sent when GitHub authentication has expired and reauth is needed
type githubReauthNeededMsg struct {
	reason string // Human-readable reason for reauth
}
