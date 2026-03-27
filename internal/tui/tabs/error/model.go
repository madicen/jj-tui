package error

import (
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/termmouse"
)

// Model represents the error modal (generic errors only; "not a jj repo" is handled by initrepo tab).
type Model struct {
	err         error
	copied      bool
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
	return renderModal(m.zoneManager, w, m.height, errStr, m.copied)
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+q", "ctrl+c":
		termmouse.Flush()
		return m, tea.Quit
	case "ctrl+r":
		return m, state.NavigateTarget{Kind: state.NavigateDismissErrorAndRefresh}.Cmd()
	case "esc", "enter", " ":
		return m, state.NavigateTarget{Kind: state.NavigateDismissError, StatusMessage: "Error dismissed"}.Cmd()
	case "c":
		return m, RequestCopyCmd()
	}
	return m, nil
}

// ZoneIDs returns the zone IDs used when rendering this modal's buttons.
func (m Model) ZoneIDs() []string {
	return []string{
		mouse.ZoneActionCopyError, mouse.ZoneActionDismissError,
		mouse.ZoneActionRetry, mouse.ZoneActionQuit,
	}
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
		return m, state.NavigateTarget{Kind: state.NavigateDismissErrorAndRefresh}.Cmd()
	case mouse.ZoneActionQuit:
		termmouse.Flush()
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

// SetError sets the error (path is ignored; init-repo screen uses initrepo tab).
func (m *Model) SetError(err error, _ bool, _ string) {
	m.err = err
	m.copied = false
}

// ClearError clears the error.
func (m *Model) ClearError() {
	m.err = nil
	m.copied = false
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
