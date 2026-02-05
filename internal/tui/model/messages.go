package model

import (
	"time"

	"github.com/madicen/jj-tui/internal/github"
	"github.com/madicen/jj-tui/internal/jj"
	"github.com/madicen/jj-tui/internal/models"
	"github.com/madicen/jj-tui/internal/tickets"
)

// tickMsg is sent on each timer tick for auto-refresh
type tickMsg time.Time

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
	ticketError   error // Error from ticket service initialization (for debugging)
	repository    *models.Repository
}

// prsLoadedMsg is sent when PRs are loaded from GitHub
type prsLoadedMsg struct {
	prs []models.GitHubPR
}

// ticketsLoadedMsg is sent when tickets are loaded
type ticketsLoadedMsg struct {
	tickets []tickets.Ticket
}

// bookmarkCreatedMsg is sent when a bookmark is created from a ticket
type bookmarkCreatedMsg struct {
	ticketKey  string
	branchName string
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

// branchPushedMsg is sent when a branch is pushed to remote
type branchPushedMsg struct {
	branch     string
	pushOutput string
}

// bookmarkCreatedOnCommitMsg is sent when a bookmark is created or moved on a commit
type bookmarkCreatedOnCommitMsg struct {
	bookmarkName string
	commitID     string
	wasMoved     bool // true if bookmark was moved, false if newly created
}

// bookmarkDeletedMsg is sent when a bookmark is deleted
type bookmarkDeletedMsg struct {
	bookmarkName string
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
