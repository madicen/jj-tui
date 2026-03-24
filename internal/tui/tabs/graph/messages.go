package graph

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/jj"
)

// RepositoryLoadedMsg indicates the repository was loaded.
type RepositoryLoadedMsg struct {
	Repository *internal.Repository
}

// EditCompletedMsg indicates checkout/edit completed.
type EditCompletedMsg struct {
	Repository *internal.Repository
}

// FileMoveCompletedMsg indicates a file was moved to a new commit.
type FileMoveCompletedMsg struct {
	Repository *internal.Repository
	FilePath   string
	Direction  string // "up" or "down"
}

// FileRevertedMsg indicates a file's changes were reverted.
type FileRevertedMsg struct {
	Repository *internal.Repository
	FilePath   string
}

// ChangedFilesLoadedMsg is sent when changed files for a commit have been loaded.
type ChangedFilesLoadedMsg struct {
	Files    []jj.ChangedFile
	CommitID string
}

// UndoCompletedMsg is sent when an undo/redo operation completes.
type UndoCompletedMsg struct {
	Message  string
	Err      error
	RedoOpID string
}

// DivergentCommitInfoMsg is sent when divergent commit info has been loaded (or failed).
type DivergentCommitInfoMsg struct {
	ChangeID  string
	CommitIDs []string
	Summaries []string
	Err       error
}

// LoadChangedFilesCmd returns a command that loads changed files for the commit and sends ChangedFilesLoadedMsg.
func LoadChangedFilesCmd(svc *jj.Service, commitID string) tea.Cmd {
	if svc == nil || commitID == "" {
		return nil
	}
	return func() tea.Msg {
		files, err := svc.GetChangedFiles(context.Background(), commitID)
		if err != nil {
			return ChangedFilesLoadedMsg{Files: nil, CommitID: commitID}
		}
		return ChangedFilesLoadedMsg{Files: files, CommitID: commitID}
	}
}

// LoadDivergentCommitInfoCmd returns a command that loads divergent commit info and sends DivergentCommitInfoMsg.
func LoadDivergentCommitInfoCmd(svc *jj.Service, changeID string) tea.Cmd {
	if svc == nil || changeID == "" {
		return nil
	}
	return func() tea.Msg {
		commitIDs, summaries, err := svc.GetDivergentCommitInfo(context.Background(), changeID)
		if err != nil {
			return DivergentCommitInfoMsg{ChangeID: changeID, Err: err}
		}
		return DivergentCommitInfoMsg{ChangeID: changeID, CommitIDs: commitIDs, Summaries: summaries}
	}
}

// UndoCmd returns a command that runs jj undo and sends UndoCompletedMsg.
func UndoCmd(svc *jj.Service) tea.Cmd {
	if svc == nil {
		return nil
	}
	return func() tea.Msg {
		opID, err := svc.Undo(context.Background())
		if err != nil {
			return UndoCompletedMsg{Err: err}
		}
		return UndoCompletedMsg{Message: "Undo completed", RedoOpID: opID}
	}
}

// RedoCmd returns a command that runs jj redo and sends UndoCompletedMsg.
func RedoCmd(svc *jj.Service, opID string) tea.Cmd {
	if svc == nil {
		return nil
	}
	return func() tea.Msg {
		err := svc.Redo(context.Background(), opID)
		if err != nil {
			return UndoCompletedMsg{Err: err}
		}
		return UndoCompletedMsg{Message: "Redo completed"}
	}
}

// Request is sent to the main model so it can run jj/git commands (main has jjService).
type Request struct {
	LoadChangedFiles     *string
	SelectCommit         *int
	Checkout             bool
	Squash               bool
	Abandon              bool
	StartEditDescription bool
	NewCommit            bool
	StartRebaseMode      bool
	PerformRebase        bool
	RebaseDestIndex      int
	ResolveDivergent     *string
	CreateBookmark       bool
	DeleteBookmark       bool
	CreatePR             bool
	UpdatePR             bool
	MoveFileUp           bool
	MoveFileDown         bool
	RevertFile           bool
	// MoveDeltaOntoOrigin: new commit on bookmark@origin with same tree as selection; avoids force-push after amending a pushed branch.
	MoveDeltaOntoOrigin bool
}

// Cmd returns a tea.Cmd that sends this request to the program.
func (r Request) Cmd() tea.Cmd {
	return func() tea.Msg { return r }
}

// FollowUp tells the main model what to do after validation (e.g. show dialog, change view).
type FollowUp int

const (
	FollowUpNone FollowUp = iota
	FollowUpResolveDivergent
	FollowUpStartEditDescription
	FollowUpStartRebaseMode
	FollowUpCreateBookmark
	FollowUpCreatePR
	FollowUpUpdatePR
	FollowUpCancelRebase
	FollowUpLoadChangedFiles
	FollowUpShowEmptyDescWarning
)

// Result is returned by HandleRequest. Main sets status from Status, runs Cmd if set, and performs the FollowUp action.
type Result struct {
	Cmd             tea.Cmd
	Status          string
	FollowUp        FollowUp
	ChangeID        string
	CommitIndex     int
	NewCommitStatus string
	SuccessStatus   string
	WarningTitle    string
	WarningMessage  string
	WarningCommits  []internal.Commit
	PerformRebase bool
}

// FocusMessage returns the status bar message for graph vs files pane focus.
func FocusMessage(graphFocused bool) string {
	if graphFocused {
		return "Graph pane focused"
	}
	return "Files pane focused"
}

// RebaseModeStartMessage returns the status message when entering rebase mode.
func RebaseModeStartMessage(shortID string) string {
	return fmt.Sprintf("Select destination for rebasing %s (Esc to cancel)", shortID)
}
