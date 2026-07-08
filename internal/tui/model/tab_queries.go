package model

// tab_queries.go holds small *Model helpers that reach into concrete tab
// packages on behalf of the generic dispatch in model.go. Keeping the
// package-qualified calls here (rather than inline in Update) lets model.go
// avoid importing those concrete tab packages directly — the call sites use a
// plain method call on *Model.

import (
	tea "github.com/charmbracelet/bubbletea"
	aitab "github.com/madicen/jj-tui/internal/tui/ai"
	graphtab "github.com/madicen/jj-tui/internal/tui/tabs/graph"
	helptab "github.com/madicen/jj-tui/internal/tui/tabs/help"
)

// refreshHelpCommandHistory rebuilds the help tab's command-history list from
// the jj service's recorded commands. Called when entering/selecting the Help
// tab so the list reflects the latest run commands.
func (m *Model) refreshHelpCommandHistory() {
	m.helpTabModel.SetCommandHistoryEntries(helptab.BuildCommandHistoryEntries(m.appState.JJService))
}

// applyEvologSplitDescriptionsCmd wraps the AI evolog-split apply command so the
// evolog-describe preview confirm handler in model.go's Update can trigger it
// without importing the ai tab package directly.
func (m *Model) applyEvologSplitDescriptionsCmd(parentDesc, childDesc string, skipParent bool) tea.Cmd {
	return aitab.ApplyEvologSplitDescriptionsCmd(0, m.appState.JJService, m.appState.Config, parentDesc, childDesc, skipParent)
}

// absorbApplyCmd wraps the graph absorb-apply command so the absorb-preview
// confirm handler in model.go's Update can trigger it without importing the
// graph tab package directly.
func (m *Model) absorbApplyCmd() tea.Cmd {
	return graphtab.AbsorbApplyCmd(m.appState.JJService)
}
