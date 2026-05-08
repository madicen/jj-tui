package error

import (
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/util"
)

// Model represents the error modal (generic errors only; "not a jj repo" is handled by initrepo tab).
type Model struct {
	err         error
	copied      bool
	hasRetry    bool // true => render Retry button and accept ctrl+r / ZoneActionRetry
	zoneManager *zone.Manager
	width       int
	height      int
}

// NewModel creates a new Error model.
func NewModel() Model {
	return Model{copied: false}
}

// Init is required by tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles keys and zone clicks; sends request messages to main.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if m.err == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case zone.MsgZoneInBounds:
		if m.zoneManager != nil {
			if zoneID := m.resolveClickedZone(msg); zoneID != "" {
				return m.handleZoneClick(zoneID)
			}
		}
		return m, nil
	}
	return m, nil
}

// View renders the error modal content. Main applies centered layout.
func (m Model) View() string {
	if m.err == nil {
		return ""
	}
	errStr := m.err.Error()
	w := m.width
	if w < 50 {
		w = 80
	}
	return renderModal(m.zoneManager, w, m.height, errStr, m.copied, m.hasRetry)
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+q", "ctrl+c":
		util.FlushMouse()
		return m, tea.Quit
	case "ctrl+r":
		if !m.hasRetry {
			return m, nil
		}
		return m, state.NavigateTarget{Kind: state.NavigateRetryError}.Cmd()
	case "esc", "enter", " ":
		return m, state.NavigateTarget{Kind: state.NavigateDismissError, StatusMessage: "Error dismissed"}.Cmd()
	case "c":
		return m, RequestCopyCmd()
	}
	return m, nil
}

// ZoneIDs returns the zone IDs used when rendering this modal's buttons. The Retry zone is
// only included when hasRetry is set so clicks in that area don't get swallowed when the button
// isn't actually drawn (see SetHasRetry).
func (m Model) ZoneIDs() []string {
	ids := []string{
		mouse.ZoneActionCopyError, mouse.ZoneActionDismissError, mouse.ZoneActionQuit,
	}
	if m.hasRetry {
		ids = append(ids, mouse.ZoneActionRetry)
	}
	return ids
}

func (m Model) resolveClickedZone(msg zone.MsgZoneInBounds) string {
	if msg.Zone == nil {
		return ""
	}
	for _, id := range m.ZoneIDs() {
		z := m.zoneManager.Get(id)
		if z != nil && z.InBounds(msg.Event) {
			return id
		}
	}
	return ""
}

func (m Model) handleZoneClick(zoneID string) (Model, tea.Cmd) {
	switch zoneID {
	case mouse.ZoneActionCopyError:
		return m, RequestCopyCmd()
	case mouse.ZoneActionDismissError:
		return m, state.NavigateTarget{Kind: state.NavigateDismissError, StatusMessage: "Error dismissed"}.Cmd()
	case mouse.ZoneActionRetry:
		if !m.hasRetry {
			return m, nil
		}
		return m, state.NavigateTarget{Kind: state.NavigateRetryError}.Cmd()
	case mouse.ZoneActionQuit:
		util.FlushMouse()
		return m, tea.Quit
	}
	return m, nil
}

// SetZoneManager sets the zone manager used to resolve clicks.
func (m *Model) SetZoneManager(zm *zone.Manager) {
	m.zoneManager = zm
}

// SetWidth sets the width for modal layout.
func (m *Model) SetWidth(w int) {
	m.width = w
}

// SetHeight sets the terminal height so the error body can be capped and buttons stay on-screen.
func (m *Model) SetHeight(h int) {
	m.height = h
}

// GetError returns the current error.
func (m *Model) GetError() error {
	return m.err
}

// SetError sets the error (path is ignored; init-repo screen uses initrepo tab). hasRetry resets
// to false; callers that want a Retry button (e.g. AI generation failures with a saved replay
// target on the main Model) must follow up with SetHasRetry(true).
func (m *Model) SetError(err error, _ bool, _ string) {
	m.err = err
	m.copied = false
	m.hasRetry = false
}

// ClearError clears the error.
func (m *Model) ClearError() {
	m.err = nil
	m.copied = false
	m.hasRetry = false
}

// SetHasRetry toggles whether the Retry (^r) button is rendered and the corresponding
// keybinding/zone are honored. Main sets this when an AI generation fails with a saved replay
// target so the user can rerun the same request without losing the open form modal.
func (m *Model) SetHasRetry(has bool) {
	m.hasRetry = has
}

// HasRetry reports whether Retry is currently offered for the active error.
func (m *Model) HasRetry() bool {
	return m.hasRetry
}

// IsCopied returns whether the error was just copied.
func (m *Model) IsCopied() bool {
	return m.copied
}

// SetCopied sets the copied flag.
func (m *Model) SetCopied(copied bool) {
	m.copied = copied
}

// UpdateRepository is a no-op; kept for compatibility.
func (m *Model) UpdateRepository(repo *internal.Repository) {}
