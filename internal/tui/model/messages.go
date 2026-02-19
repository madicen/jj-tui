package model

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/state"
)

// SetStatusMsg sets the footer status; any component can send it (e.g. via SetStatusCmd).
type SetStatusMsg struct {
	Status string
}

// SetStatusCmd returns a command that sets the main model's status message.
func SetStatusCmd(status string) tea.Cmd {
	return func() tea.Msg { return SetStatusMsg{Status: status} }
}

// Public messages and view types (used by tests and external packages).

// TabSelectedMsg is emitted when a tab is clicked
type TabSelectedMsg struct {
	Tab state.ViewMode
}

// ActionMsg is emitted when an action button is clicked
type ActionMsg struct {
	Action ActionType
}

// ActionType represents the type of action triggered
type ActionType string

const (
	ActionQuit     ActionType = "quit"
	ActionRefresh  ActionType = "refresh"
	ActionNewPR    ActionType = "new_pr"
	ActionCheckout ActionType = "checkout"
	ActionEdit     ActionType = "edit"
	ActionSquash   ActionType = "squash"
	ActionRebase   ActionType = "rebase"
	ActionHelp     ActionType = "help"
)

// SelectionMode indicates what the user is selecting commits for
type SelectionMode int

const (
	SelectionNormal            SelectionMode = iota // Normal selection
	SelectionRebaseDestination                      // Selecting destination for rebase
)

// Internal message types (not exported).

// tickMsg is sent on each timer tick for auto-refresh (jj repository)
type tickMsg time.Time

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
