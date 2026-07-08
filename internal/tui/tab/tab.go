// Package tab defines the interfaces the root TUI model uses to treat its
// content tabs uniformly, decoupling the model from concrete tab packages.
//
// PLAN(P2.3): this package is the seam for the Tab-interface migration. The
// root model currently holds each primary tab as a concrete field and
// dispatches to it via per-ViewMode switches. Renderer is the first, safe slice
// of that contract (render + size) and is wired through the model's tab
// registry today. Tab, RepositoryAware, and Activatable are the target
// contracts the remaining migration (generic message forwarding, concrete-field
// removal) grows into; they are documented here so the direction is explicit.
package tab

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tui/state"
)

// Renderer is the minimal render+size contract shared by every primary content
// tab. The root model's registry stores tabs behind this interface so the
// window-resize fan-out and the content-render dispatch iterate uniformly
// instead of naming each concrete field.
type Renderer interface {
	// View renders the tab's content for the current frame.
	View() string
	// SetDimensions sets the tab's content area to width x height.
	SetDimensions(width, height int)
}

// Tab is the target interface the P2.3 migration moves the concrete tabs onto.
// Once every primary tab satisfies it (via message handlers that return
// cross-cutting effects instead of mutating the root model), the model can
// replace its concrete fields with a map[state.ViewMode]Tab and forward
// messages generically.
//
// PLAN(P2.3): not yet implemented by the concrete tabs; the existing tabs use
// UpdateWithApp returning their own type and a no-argument View. Adapting them
// is the remaining work.
type Tab interface {
	Update(msg tea.Msg, app *state.AppState) (Tab, tea.Cmd)
	View(app *state.AppState) string
	SetDimensions(width, height int)
}

// RepositoryAware is an optional hook a tab implements when it needs to recompute
// derived state after a repository load. Checked by type assertion.
type RepositoryAware interface {
	OnRepositoryLoaded(*internal.Repository)
}

// Activatable is an optional hook a tab implements when it needs to react to
// becoming (or ceasing to be) the active view. Checked by type assertion.
type Activatable interface {
	OnActivated()
	OnDeactivated()
}
