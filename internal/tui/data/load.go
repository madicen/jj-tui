package data

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/jj"
)

// LoadRepository loads or refreshes repository data. Returns a cmd that sends RepositoryLoadedMsg.
// Uses config.GraphRevset when set; otherwise jj.DefaultGraphRevset (mutable in @ cone | bookmarks | main@origin).
// Caller should use InitializeServices if jjService is nil.
func LoadRepository(jjService *jj.Service) tea.Cmd {
	if jjService == nil {
		return nil
	}
	return func() tea.Msg {
		cfg, _ := config.Load()
		revset := ""
		if cfg != nil {
			revset = cfg.GraphRevset
		}
		repo, err := jjService.GetRepository(context.Background(), revset)
		if err != nil {
			return InitErrorMsg{Err: err}
		}
		return RepositoryLoadedMsg{Repository: repo}
	}
}

// LoadRepositorySilent loads repository without surfacing errors (for background refresh).
// revset is the graph revset to use (e.g. from app config); empty uses jj default.
// Pass revset from app state to avoid reading config from disk every tick.
// Always returns SilentRepositoryLoadedMsg so the UI can clear in-flight refresh state;
// Repository is nil when GetRepository fails.
func LoadRepositorySilent(jjService *jj.Service, revset string) tea.Cmd {
	if jjService == nil {
		return nil
	}
	return func() tea.Msg {
		// Quiet refresh: same graph load as GetRepository but do not spam command history every tick.
		repo, err := jjService.GetRepositoryQuiet(context.Background(), revset)
		if err != nil {
			return SilentRepositoryLoadedMsg{Repository: nil}
		}
		return SilentRepositoryLoadedMsg{Repository: repo}
	}
}
