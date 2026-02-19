package error

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/state"
)

// ErrorApplyInfo is for main to apply (set init path or error modal).
type ErrorApplyInfo struct {
	NotJJRepo   bool
	CurrentPath string
	Err         error
}

// HandleError mutates app (Loading, StatusMessage) and returns (nil, *ErrorApplyInfo) for main to apply.
func HandleError(input ErrorInput, app *state.AppState) (tea.Cmd, *ErrorApplyInfo) {
	app.Loading = false
	if input.NotJJRepo {
		app.StatusMessage = "Press 'i' to initialize a repository"
	} else {
		app.StatusMessage = fmt.Sprintf("Error: %v", input.Err)
	}
	return nil, &ErrorApplyInfo{
		NotJJRepo:   input.NotJJRepo,
		CurrentPath: input.CurrentPath,
		Err:         input.Err,
	}
}

// ErrorInput is the context main sends when handling an internal error (errorMsg).
type ErrorInput struct {
	NotJJRepo   bool
	CurrentPath string
	Err         error
}
