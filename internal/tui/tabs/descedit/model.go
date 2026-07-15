package descedit

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/tui/genmenu"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/render"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

// Model represents the commit description editing dialog
type Model struct {
	shown            bool
	descriptionInput textarea.Model
	editingCommitID  string
	commitShortID    string // For header display (e.g. "abc123")
	zoneManager *zone.Manager
	profileMenu genmenu.ProfileMenu
}

// NewModel creates a new description-edit model. zoneManager may be nil.
func NewModel(zoneManager *zone.Manager) Model {
	ta := textarea.New()
	ta.Placeholder = "Enter commit description..."
	ta.ShowLineNumbers = false
	ta.SetWidth(60)
	ta.SetHeight(3)
	return Model{
		shown:            false,
		descriptionInput: ta,
		zoneManager:      zoneManager,
		profileMenu:      genmenu.NewProfileMenu(mouse.ZoneDescGenerate),
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
			Kind:            state.NavigateSaveDescription,
			SaveCommitID:    m.editingCommitID,
			SaveDescription: m.descriptionInput.Value(),
		}.Cmd()
	case CancelRequestedMsg:
		m.shown = false
		m.editingCommitID = ""
		m.commitShortID = ""
		m.descriptionInput.SetValue("")
		m.profileMenu.Reset()
		return m, state.NavigateTarget{Kind: state.NavigateBackToGraph, StatusMessage: "Description edit cancelled"}.Cmd()
	}
	switch msg := msg.(type) {
	case genmenu.TickMsg:
		m.profileMenu.OpenIfMatches(msg)
		return m, nil
	case tea.MouseMsg:
		return m, m.profileMenu.HandleMouse(m.zoneManager, msg, true, func(name string) tea.Cmd {
			return state.NavigateTarget{Kind: state.NavigateGenerateCommitDescription, AIOverrideProfile: name}.Cmd()
		})
	case zone.MsgZoneInBounds:
		// While the popover is shown a release click on a menu row was already
		// resolved by HandleMouse with the raw MouseMsg; ignore the follow-up
		// zone dispatch so we don't double-fire.
		if m.profileMenu.IsShown() {
			return m, nil
		}
		// Quick click on the generate chip (press → release before the long-press
		// tick fires) — clear the pending armed state but still run the normal
		// zone-click handler so the user gets the active-profile generate.
		m.profileMenu.CancelPress()
		if m.zoneManager != nil {
			if zoneID := m.resolveClickedZone(msg); zoneID != "" {
				return m.handleZoneClick(zoneID)
			}
		}
		return m, nil
	case tea.KeyMsg:
		if m.profileMenu.IsShown() && msg.String() == "esc" {
			m.profileMenu.Close()
			return m, nil
		}
		switch msg.String() {
		case "ctrl+s":
			return m, SaveRequestedCmd()
		case "ctrl+g":
			return m, state.NavigateTarget{Kind: state.NavigateGenerateCommitDescription}.Cmd()
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
	return []string{mouse.ZoneDescSave, mouse.ZoneDescCancel, mouse.ZoneDescClear, mouse.ZoneDescGenerate}
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
	case mouse.ZoneDescGenerate:
		return m, state.NavigateTarget{Kind: state.NavigateGenerateCommitDescription}.Cmd()
	}
	return m, nil
}

func (m Model) clearDescription() (Model, tea.Cmd) {
	m.descriptionInput.SetValue("")
	m.descriptionInput.Focus()
	return m, nil
}

// View renders the edit-description dialog. The window title ("Edit
// description") lives in the chrome tab — see chromedSlot — so this view
// no longer carries its own header line. The AI generate chip rides on
// the commit-info row, right-aligned, so it stays a single click target.
func (m Model) View() string {
	if !m.shown {
		return ""
	}
	subtitleStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)

	commitInfo := m.editingCommitID
	if m.commitShortID != "" {
		changeIDShort := m.editingCommitID
		if len(changeIDShort) > 8 {
			changeIDShort = changeIDShort[:8]
		}
		commitInfo = fmt.Sprintf("%s (%s)", m.commitShortID, changeIDShort)
	}

	contentW := m.descriptionInput.Width()
	if contentW < 12 {
		contentW = 60
	}
	mark := func(id, s string) string {
		return render.Mark(m.zoneManager, id, s)
	}
	genChip := mark(mouse.ZoneDescGenerate, styles.AIGenerateChip())
	commitLine := styles.SpreadRow(contentW, subtitleStyle.Render(fmt.Sprintf("Commit: %s", commitInfo)), genChip)
	actionButtons := lipgloss.JoinHorizontal(
		lipgloss.Left,
		mark(mouse.ZoneDescSave, styles.ButtonStyle.Render("Save (Ctrl+S)")),
		mark(mouse.ZoneDescClear, styles.ButtonStyle.Render("Clear (Ctrl+Shift+U)")),
		mark(mouse.ZoneDescCancel, styles.ButtonStyle.Render("Cancel (Esc)")),
	)
	return lipgloss.JoinVertical(
		lipgloss.Left,
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

// GetCommitShortID returns the short commit id for display / AI context (may be empty).
func (m *Model) GetCommitShortID() string {
	return m.commitShortID
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

// SetAIProfiles updates the profile list shown by the long-press menu and the
// active profile mark. Main calls this when the modal opens (and after the
// user saves changes to AI profiles in settings while the modal stays open).
func (m *Model) SetAIProfiles(profiles []config.AIProfile, activeProfile string) {
	m.profileMenu.SetAIProfiles(profiles, activeProfile)
}

// MenuState returns a pointer to the long-press menu state so main can render
// the popover overlay or check IsShown when laying out the view.
func (m *Model) MenuState() *genmenu.State {
	return m.profileMenu.MenuState()
}

// MenuOverlay returns the rendered popover string (or empty when the menu is
// not visible) along with the (x, y) terminal anchor where it should be drawn.
func (m *Model) MenuOverlay() (string, int, int) {
	return m.profileMenu.Overlay(m.zoneManager)
}
