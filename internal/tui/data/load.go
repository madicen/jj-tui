package data

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/jj"
)

// ResolveGraphRevset returns the jj revset for graph loading. When filterRevset is
// non-empty it is used as-is (active search filter). Otherwise config graph_revset
// and the mine() wrapper apply as for a normal reload.
func ResolveGraphRevset(cfg *config.Config, filterRevset string) string {
	if strings.TrimSpace(filterRevset) != "" {
		return filterRevset
	}
	revset := ""
	if cfg != nil {
		revset = cfg.GraphRevset
		if cfg.GraphFilterToMine() {
			revset = jj.ApplyMineFilterToRevset(revset)
		}
	}
	return revset
}

// LoadRepository loads or refreshes repository data. Returns a cmd that sends RepositoryLoadedMsg.
// filterRevset is the active graph search revset from app state (empty when no filter).
func LoadRepository(jjService *jj.Service, filterRevset string) tea.Cmd {
	if jjService == nil {
		return nil
	}
	return func() tea.Msg {
		cfg, _ := config.Load()
		revset := ResolveGraphRevset(cfg, filterRevset)
		if cfg != nil {
			jjService.BookmarkListPreferTracked = cfg.BranchesFilterToTrackedAndMine()
		}
		repo, err := jjService.GetRepository(context.Background(), revset)
		if err != nil {
			return InitErrorMsg{Err: err}
		}
		return RepositoryLoadedMsg{Repository: repo}
	}
}

// GraphFilterLoadedMsg is sent after applying a graph search filter. On error the
// repository field is nil so the UI keeps the previous graph.
type GraphFilterLoadedMsg struct {
	Repository *internal.Repository
	Query      string
	Revset     string
	Err        error
}

// ApplyGraphFilterCmd loads the graph with a compiled search revset and sends GraphFilterLoadedMsg.
func ApplyGraphFilterCmd(jjService *jj.Service, query, revset string) tea.Cmd {
	if jjService == nil || strings.TrimSpace(revset) == "" {
		return nil
	}
	return func() tea.Msg {
		cfg, _ := config.Load()
		if cfg != nil {
			jjService.BookmarkListPreferTracked = cfg.BranchesFilterToTrackedAndMine()
		}
		repo, err := jjService.GetRepositoryStrict(context.Background(), revset)
		return GraphFilterLoadedMsg{Repository: repo, Query: query, Revset: revset, Err: err}
	}
}

// LoadRepositorySilent loads repository without surfacing errors (for background refresh).
// filterRevset is the active graph search revset (empty when no filter).
func LoadRepositorySilent(jjService *jj.Service, filterRevset string) tea.Cmd {
	if jjService == nil {
		return nil
	}
	return func() tea.Msg {
		cfg, _ := config.Load()
		revset := ResolveGraphRevset(cfg, filterRevset)
		repo, err := jjService.GetRepositoryQuiet(context.Background(), revset)
		if err != nil {
			return SilentRepositoryLoadedMsg{Repository: nil}
		}
		return SilentRepositoryLoadedMsg{Repository: repo}
	}
}
