package descedit

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

// Model represents the commit description editing dialog
type Model struct {
	shown           bool
	descriptionInput textarea.Model
	editingCommitID  string
	commitShortID    string // For header display (e.g. "abc123")
	zoneManager      *zone.Manager
}

// NewModel creates a new description-edit model. zoneManager may be nil.
func NewModel(zoneManager *zone.Manager) Model {
	ta := textarea.New()
	ta.Placeholder = "Enter commit description..."
	ta.ShowLineNumbers = false
	ta.SetWidth(60)
	ta.SetHeight(5)
	return Model{
		shown:            false,
		descriptionInput: ta,
		zoneManager:      zoneManager,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages; when shown, handles Save/Cancel keys and request messages, delegates rest to the textarea.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.shown {
		return m, nil
	}
	// Handle request messages (main forwards these to us)
	switch msg.(type) {
	case SaveRequestedMsg:
		return m, state.NavigateTarget{
			Kind:           state.NavigateSaveDescription,
			SaveCommitID:   m.editingCommitID,
			SaveDescription: m.descriptionInput.Value(),
		}.Cmd()
	case CancelRequestedMsg:
		m.shown = false
		m.editingCommitID = ""
		m.commitShortID = ""
		m.descriptionInput.SetValue("")
		return m, state.NavigateTarget{Kind: state.NavigateBackToGraph, StatusMessage: "Description edit cancelled"}.Cmd()
	}
	switch msg := msg.(type) {
	case zone.MsgZoneInBounds:
		if m.zoneManager != nil {
			if zoneID := m.resolveClickedZone(msg); zoneID != "" {
				return m.handleZoneClick(zoneID)
			}
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+s":
			return m, SaveRequestedCmd()
		case "esc":
			return m, CancelRequestedCmd()
		case "ctrl+shift+u":
			return m.clearDescription()
		}
	}
	var cmd tea.Cmd
	m.descriptionInput, cmd = m.descriptionInput.Update(msg)
	return m, cmd
}

// ZoneIDs returns the zone IDs this modal uses when rendering. Used to resolve clicks.
func (m Model) ZoneIDs() []string {
	return []string{mouse.ZoneDescSave, mouse.ZoneDescCancel, mouse.ZoneDescClear}
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

// handleZoneClick handles a zone click by zone ID (called from Update after resolve). Returns (updated model, cmd).
func (m Model) handleZoneClick(zoneID string) (Model, tea.Cmd) {
	switch zoneID {
	case mouse.ZoneDescSave:
		return m, SaveRequestedCmd()
	case mouse.ZoneDescCancel:
		return m, CancelRequestedCmd()
	case mouse.ZoneDescClear:
		return m.clearDescription()
	}
	return m, nil
}

func (m Model) clearDescription() (Model, tea.Cmd) {
	m.descriptionInput.SetValue("")
	m.descriptionInput.Focus()
	return m, nil
}

// View renders the edit-description dialog
func (m Model) View() string {
	if !m.shown {
		return ""
	}
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8BE9FD"))
	subtitleStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)

	commitInfo := m.editingCommitID
	if m.commitShortID != "" {
		changeIDShort := m.editingCommitID
		if len(changeIDShort) > 8 {
			changeIDShort = changeIDShort[:8]
		}
		commitInfo = fmt.Sprintf("%s (%s)", m.commitShortID, changeIDShort)
	}

	header := titleStyle.Render("Edit Commit Description")
	commitLine := subtitleStyle.Render(fmt.Sprintf("Commit: %s", commitInfo))
	mark := func(id, s string) string {
		if m.zoneManager == nil {
			return s
		}
		return m.zoneManager.Mark(id, s)
	}
	actionButtons := lipgloss.JoinHorizontal(
		lipgloss.Left,
		mark(mouse.ZoneDescSave, styles.ButtonStyle.Render("Save (Ctrl+S)")),
		mark(mouse.ZoneDescClear, styles.ButtonStyle.Render("Clear (Ctrl+Shift+U)")),
		mark(mouse.ZoneDescCancel, styles.ButtonStyle.Render("Cancel (Esc)")),
	)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		commitLine,
		"",
		m.descriptionInput.View(),
		"",
		actionButtons,
	)
}

// Show displays the dialog for the given commit
func (m *Model) Show(commitID, shortID string) {
	m.shown = true
	m.editingCommitID = commitID
	m.commitShortID = shortID
	m.descriptionInput.SetValue("")
	m.descriptionInput.Focus()
}

// PrepareForCommit prepares the modal for editing the given commit (show, set dimensions). Caller sets viewMode and runs load-description cmd.
func (m Model) PrepareForCommit(commit internal.Commit, width, height int) (Model, tea.Cmd) {
	m.Show(commit.ChangeID, commit.ShortID)
	m.SetDimensions(width, height)
	return m, nil
}

// Hide closes the dialog
func (m *Model) Hide() {
	m.shown = false
	m.editingCommitID = ""
	m.commitShortID = ""
	m.descriptionInput.SetValue("")
}

// IsShown returns whether the dialog is visible
func (m *Model) IsShown() bool {
	return m.shown
}

// SetDescription sets the textarea value (e.g. after loading from jj)
func (m *Model) SetDescription(value string) {
	m.descriptionInput.SetValue(value)
	m.descriptionInput.Focus()
}

// GetEditingCommitID returns the commit ID being edited
func (m *Model) GetEditingCommitID() string {
	return m.editingCommitID
}

// GetDescriptionValue returns the current description text
func (m *Model) GetDescriptionValue() string {
	return m.descriptionInput.Value()
}

// SetDimensions updates the textarea size (call when window resizes or when showing)
func (m *Model) SetDimensions(width, height int) {
	if width > 0 {
		m.descriptionInput.SetWidth(width)
	}
	if height > 0 {
		m.descriptionInput.SetHeight(height)
	}
}
