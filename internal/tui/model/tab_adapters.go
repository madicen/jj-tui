package model

// tab_adapters.go bridges the six concrete primary-tab models onto the generic
// tab.Tab contract (Update(msg, app) / View(app) / SetDimensions). Each adapter
// holds a pointer to the concrete field on *Model so a reassignment made by
// Update (via `*a.m = updated`) stays visible to every other code path that
// still reads the concrete field directly. Keeping the adapters in their own
// file (with the concrete-package imports) is what lets model.go route the
// primary tabs generically without importing those packages itself.
//
// PLAN(P2.3): the adapters are the seam that reconciles the heterogeneous tab
// shapes the earlier workers flagged — value vs. pointer receivers,
// View() vs. View(app), UpdateWithApp vs. Update, and the graph/prs/branches
// command wrappers (wrapGraphTabCmd / wrapSpinnerStart) — behind one interface.

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/tab"
	branchestab "github.com/madicen/jj-tui/internal/tui/tabs/branches"
	graphtab "github.com/madicen/jj-tui/internal/tui/tabs/graph"
	helptab "github.com/madicen/jj-tui/internal/tui/tabs/help"
	prstab "github.com/madicen/jj-tui/internal/tui/tabs/prs"
	settingstab "github.com/madicen/jj-tui/internal/tui/tabs/settings"
	ticketstab "github.com/madicen/jj-tui/internal/tui/tabs/tickets"
)

// graphTabAdapter wraps *graphtab.GraphModel. Its Update applies wrapGraphTabCmd
// so the busy-spinner tick is attached exactly as the pre-registry inline path
// did (the graph tab sets Loading itself for synchronous "Update PR" pushes).
type graphTabAdapter struct {
	m    *graphtab.GraphModel
	root *Model
}

func (a graphTabAdapter) Update(msg tea.Msg, app *state.AppState) (tab.Tab, tea.Cmd) {
	updated, cmd := a.m.UpdateWithApp(msg, app)
	*a.m = updated
	return a, a.root.wrapGraphTabCmd(cmd)
}

func (a graphTabAdapter) View(*state.AppState) string { return a.m.View() }

func (a graphTabAdapter) SetDimensions(w, h int) { a.m.SetDimensions(w, h) }

// prsTabAdapter wraps *prstab.Model. Its Update applies wrapSpinnerStart so a
// slow remote op the PR tab kicked off (merge/close) starts the busy spinner
// exactly once, matching the inline delegation it replaces.
type prsTabAdapter struct {
	m    *prstab.Model
	root *Model
}

func (a prsTabAdapter) Update(msg tea.Msg, app *state.AppState) (tab.Tab, tea.Cmd) {
	updated, cmd := a.m.UpdateWithApp(msg, app)
	*a.m = updated
	return a, a.root.wrapSpinnerStart(cmd)
}

func (a prsTabAdapter) View(*state.AppState) string { return a.m.View() }

func (a prsTabAdapter) SetDimensions(w, h int) { a.m.SetDimensions(w, h) }

// branchesTabAdapter wraps *branchestab.Model. Like prs it applies
// wrapSpinnerStart for fetch/push-all.
type branchesTabAdapter struct {
	m    *branchestab.Model
	root *Model
}

func (a branchesTabAdapter) Update(msg tea.Msg, app *state.AppState) (tab.Tab, tea.Cmd) {
	updated, cmd := a.m.UpdateWithApp(msg, app)
	*a.m = updated
	return a, a.root.wrapSpinnerStart(cmd)
}

func (a branchesTabAdapter) View(*state.AppState) string { return a.m.View() }

func (a branchesTabAdapter) SetDimensions(w, h int) { a.m.SetDimensions(w, h) }

// ticketsTabAdapter wraps *ticketstab.Model. No command wrapper (the tickets
// tab manages its own spinner state via appState).
type ticketsTabAdapter struct {
	m *ticketstab.Model
}

func (a ticketsTabAdapter) Update(msg tea.Msg, app *state.AppState) (tab.Tab, tea.Cmd) {
	updated, cmd := a.m.UpdateWithApp(msg, app)
	*a.m = updated
	return a, cmd
}

func (a ticketsTabAdapter) View(*state.AppState) string { return a.m.View() }

func (a ticketsTabAdapter) SetDimensions(w, h int) { a.m.SetDimensions(w, h) }

// settingsTabAdapter wraps *settingstab.Model. Settings has no app-aware
// Update, so the adapter drops the app argument; the settings sub-model reads
// everything it needs from the ViewOpts the root pushes before rendering.
type settingsTabAdapter struct {
	m *settingstab.Model
}

func (a settingsTabAdapter) Update(msg tea.Msg, _ *state.AppState) (tab.Tab, tea.Cmd) {
	updated, cmd := a.m.Update(msg)
	*a.m = updated
	return a, cmd
}

func (a settingsTabAdapter) View(*state.AppState) string { return a.m.View() }

func (a settingsTabAdapter) SetDimensions(w, h int) { a.m.SetDimensions(w, h) }

// helpTabAdapter wraps *helptab.Model.
type helpTabAdapter struct {
	m *helptab.Model
}

func (a helpTabAdapter) Update(msg tea.Msg, _ *state.AppState) (tab.Tab, tea.Cmd) {
	updated, cmd := a.m.Update(msg)
	*a.m = updated
	return a, cmd
}

func (a helpTabAdapter) View(*state.AppState) string { return a.m.View() }

func (a helpTabAdapter) SetDimensions(w, h int) { a.m.SetDimensions(w, h) }
