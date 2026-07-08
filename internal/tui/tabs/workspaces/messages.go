package workspaces

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/integrations/jj"
)

// WorkspacesLoadedMsg carries the result of loading `jj workspace list`.
type WorkspacesLoadedMsg struct {
	Workspaces []jj.Workspace
	Err        error
}

// WorkspaceChangedMsg is sent after an add/forget completes; StatusMessage is a
// human-readable summary and Err is non-nil on failure. The main model reloads
// the workspace list on success.
type WorkspaceChangedMsg struct {
	StatusMessage string
	Err           error
}

// LoadWorkspacesCmd loads the workspace list and sends WorkspacesLoadedMsg.
func LoadWorkspacesCmd(svc *jj.Service) tea.Cmd {
	if svc == nil {
		return nil
	}
	return func() tea.Msg {
		ws, err := svc.ListWorkspaces(context.Background())
		return WorkspacesLoadedMsg{Workspaces: ws, Err: err}
	}
}

// AddWorkspaceCmd runs `jj workspace add PATH` and sends WorkspaceChangedMsg.
func AddWorkspaceCmd(svc *jj.Service, path string) tea.Cmd {
	if svc == nil {
		return nil
	}
	return func() tea.Msg {
		if err := svc.AddWorkspace(context.Background(), path); err != nil {
			return WorkspaceChangedMsg{Err: err}
		}
		return WorkspaceChangedMsg{StatusMessage: "Added workspace at " + path}
	}
}

// ForgetWorkspaceCmd runs `jj workspace forget NAME` and sends WorkspaceChangedMsg.
func ForgetWorkspaceCmd(svc *jj.Service, name string) tea.Cmd {
	if svc == nil {
		return nil
	}
	return func() tea.Msg {
		if err := svc.ForgetWorkspace(context.Background(), name); err != nil {
			return WorkspaceChangedMsg{Err: err}
		}
		return WorkspaceChangedMsg{StatusMessage: "Forgot workspace " + name}
	}
}
