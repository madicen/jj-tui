package model

// tab_queries.go holds small *Model helpers that reach into concrete tab
// packages on behalf of the generic dispatch in model.go. Keeping the
// package-qualified calls here (rather than inline in Update) lets model.go
// avoid importing those concrete tab packages directly — the call sites use a
// plain method call on *Model.

import (
	helptab "github.com/madicen/jj-tui/internal/tui/tabs/help"
)

// refreshHelpCommandHistory rebuilds the help tab's command-history list from
// the jj service's recorded commands. Called when entering/selecting the Help
// tab so the list reflects the latest run commands.
func (m *Model) refreshHelpCommandHistory() {
	m.helpTabModel.SetCommandHistoryEntries(helptab.BuildCommandHistoryEntries(m.appState.JJService))
}
