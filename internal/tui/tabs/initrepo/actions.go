package initrepo

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/data"
	"github.com/madicen/jj-tui/internal/tui/state"
)

// InitErrorApplyInfo is for main to apply (set init path or error modal).
type InitErrorApplyInfo struct {
	NotJJRepo   bool
	CurrentPath string
	Err         error
}

// HandleInitError mutates app (Loading, StatusMessage) and returns (nil, *InitErrorApplyInfo) for main to apply.
func HandleInitError(msg data.InitErrorMsg, app *state.AppState) (tea.Cmd, *InitErrorApplyInfo) {
	app.Loading = false
	if msg.NotJJRepo {
		app.StatusMessage = "Press 'i' to initialize a repository"
	} else {
		app.StatusMessage = fmt.Sprintf("Error: %v", msg.Err)
	}
	return nil, &InitErrorApplyInfo{
		NotJJRepo:   msg.NotJJRepo,
		CurrentPath: msg.CurrentPath,
		Err:         msg.Err,
	}
}

// HandleJJInitSuccess mutates app (StatusMessage) and returns the Cmd to run. Main should clear init path and error modal.
func HandleJJInitSuccess(msg data.JJInitSuccessMsg, app *state.AppState) tea.Cmd {
	_ = msg
	app.StatusMessage = "Repository initialized! Loading..."
	return data.InitializeServices(app.DemoMode)
}
