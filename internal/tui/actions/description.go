package actions

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/integrations/jj"
)

// LoadDescription fetches the complete description for a commit
func LoadDescription(svc *jj.Service, commitID string) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return ErrorMsg{Err: fmt.Errorf("jj service not available")}
		}
		desc, err := svc.GetCommitDescription(context.Background(), commitID)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to load description: %w", err)}
		}
		return DescriptionLoadedMsg{CommitID: commitID, Description: desc}
	}
}

// SaveDescription saves a commit description
func SaveDescription(svc *jj.Service, commitID, description string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.DescribeCommit(context.Background(), commitID, description); err != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to update description: %w", err)}
		}
		return DescriptionSavedMsg{CommitID: commitID}
	}
}
