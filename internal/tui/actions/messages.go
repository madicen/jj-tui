// Package actions provides action functions for the TUI.
package actions

import "github.com/madicen/jj-tui/internal/models"

// Message types returned by action functions

// RepositoryLoadedMsg indicates the repository was loaded
type RepositoryLoadedMsg struct {
	Repository *models.Repository
}

// EditCompletedMsg indicates checkout/edit completed
type EditCompletedMsg struct {
	Repository *models.Repository
}

// ErrorMsg indicates an error occurred
type ErrorMsg struct {
	Err error
}

// DescriptionLoadedMsg contains loaded description
type DescriptionLoadedMsg struct {
	CommitID    string
	Description string
}

// DescriptionSavedMsg indicates description was saved
type DescriptionSavedMsg struct {
	CommitID string
}

// PRCreatedMsg indicates a PR was created
type PRCreatedMsg struct {
	PR *models.GitHubPR
}

// BranchPushedMsg indicates a branch was pushed
type BranchPushedMsg struct {
	Branch     string
	PushOutput string
}

// BookmarkCreatedMsg indicates bookmark was created/moved
type BookmarkCreatedMsg struct {
	BookmarkName string
	CommitID     string
	WasMoved     bool
	TicketKey    string // set when creating from a ticket (for auto-transition)
}

// BookmarkDeletedMsg indicates bookmark was deleted
type BookmarkDeletedMsg struct {
	BookmarkName string
}

// ClipboardCopiedMsg indicates clipboard operation result
type ClipboardCopiedMsg struct {
	Success bool
	Err     error
}

// FileMoveCompletedMsg indicates a file was moved to a new commit
type FileMoveCompletedMsg struct {
	Repository *models.Repository
	FilePath   string
	Direction  string // "up" or "down"
}

// FileRevertedMsg indicates a file's changes were reverted
type FileRevertedMsg struct {
	Repository *models.Repository
	FilePath   string
}

