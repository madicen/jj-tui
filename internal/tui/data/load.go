package data

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/integrations/jj"
)

// LoadRepository loads or refreshes repository data. Returns a cmd that sends RepositoryLoadedMsg.
// Caller should use InitializeServices if jjService is nil.
func LoadRepository(jjService *jj.Service) tea.Cmd {
	if jjService == nil {
		return nil
	}
	return func() tea.Msg {
		repo, err := jjService.GetRepository(context.Background())
		if err != nil {
			return InitErrorMsg{Err: err}
		}
		return RepositoryLoadedMsg{Repository: repo}
	}
}

// LoadRepositorySilent loads repository without surfacing errors (for background refresh).
// Returns nil on error; sends SilentRepositoryLoadedMsg on success.
func LoadRepositorySilent(jjService *jj.Service) tea.Cmd {
	if jjService == nil {
		return nil
	}
	return func() tea.Msg {
		repo, err := jjService.GetRepository(context.Background())
		if err != nil {
			return nil
		}
		return SilentRepositoryLoadedMsg{Repository: repo}
	}
}
