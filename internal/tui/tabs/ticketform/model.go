package ticketform

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/tickets"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

// Model represents the Create Ticket dialog
type Model struct {
	zoneManager  *zone.Manager
	shown        bool
	titleInput   textinput.Model
	bodyInput    textarea.Model
	focusedField int // 0=title, 1=body
	providerName string
}

// NewModel creates a new Create Ticket model. zoneManager may be nil.
func NewModel(zoneManager *zone.Manager) Model {
	titleInput := textinput.New()
	titleInput.Placeholder = "Ticket title / summary"
	titleInput.CharLimit = 300
	titleInput.Width = 60

	bodyInput := textarea.New()
	bodyInput.Placeholder = "Description (optional)..."
	bodyInput.ShowLineNumbers = false
	bodyInput.SetWidth(60)
	bodyInput.SetHeight(8)

	return Model{
		zoneManager:  zoneManager,
		shown:        false,
		titleInput:   titleInput,
		bodyInput:    bodyInput,
		focusedField: 0,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the Create Ticket view
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.shown {
		return m, nil
	}
	switch msg.(type) {
	case CancelRequestedMsg:
		m.shown = false
		m.Reset()
		return m, state.NavigateTarget{Kind: state.NavigateBackFromTicketForm, StatusMessage: "Create ticket cancelled"}.Cmd()
	case SubmitRequestedMsg:
		return m, state.NavigateTarget{Kind: state.NavigateSubmitTicket}.Cmd()
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
		return m.handleKeyMsg(msg)
	}

	if m.focusedField == 0 {
		var cmd tea.Cmd
		m.titleInput, cmd = m.titleInput.Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	m.bodyInput, cmd = m.bodyInput.Update(msg)
	return m, cmd
}

// View renders the Create Ticket dialog
func (m Model) View() string {
	if !m.shown {
		return ""
	}
	return m.renderForm()
}

func (m Model) renderForm() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8BE9FD"))
	subtitleStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	buttonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#30363d")).
		Padding(0, 1).
		Bold(true)

	mark := func(id, s string) string {
		if m.zoneManager == nil {
			return s
		}
		return m.zoneManager.Mark(id, s)
	}

	providerLine := ""
	if m.providerName != "" {
		providerLine = subtitleStyle.Render(fmt.Sprintf("Provider: %s", m.providerName))
	}
	titleInput := mark(mouse.ZoneTicketFormTitle, m.titleInput.View())
	bodyInput := mark(mouse.ZoneTicketFormBody, m.bodyInput.View())
	submitBtn := mark(mouse.ZoneTicketFormSubmit, buttonStyle.Render("Create (Ctrl+S)"))
	cancelBtn := mark(mouse.ZoneTicketFormCancel, buttonStyle.Render("Cancel (Esc)"))

	blocks := []string{
		titleStyle.Render("Create Ticket"),
		providerLine,
		"",
		"Title:",
		titleInput,
		"",
		"Description (optional):",
		bodyInput,
		"",
		lipgloss.JoinHorizontal(lipgloss.Left, submitBtn, "  ", cancelBtn),
	}
	return lipgloss.JoinVertical(lipgloss.Left, blocks...)
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, CancelRequestedCmd()
	case "ctrl+s", "ctrl+enter":
		return m, SubmitRequestedCmd()
	case "tab":
		m.focusedField = (m.focusedField + 1) % 2
		if m.focusedField == 0 {
			m.titleInput.Focus()
			m.bodyInput.Blur()
		} else {
			m.titleInput.Blur()
			m.bodyInput.Focus()
		}
		return m, nil
	}
	if m.focusedField == 0 {
		var cmd tea.Cmd
		m.titleInput, cmd = m.titleInput.Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	m.bodyInput, cmd = m.bodyInput.Update(msg)
	return m, cmd
}

// ZoneIDs returns the zone IDs this modal uses
func (m Model) ZoneIDs() []string {
	return []string{mouse.ZoneTicketFormTitle, mouse.ZoneTicketFormBody, mouse.ZoneTicketFormSubmit, mouse.ZoneTicketFormCancel}
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
	case mouse.ZoneTicketFormTitle:
		m.SetFocusedField(0)
		return m, nil
	case mouse.ZoneTicketFormBody:
		m.SetFocusedField(1)
		return m, nil
	case mouse.ZoneTicketFormSubmit:
		return m, SubmitRequestedCmd()
	case mouse.ZoneTicketFormCancel:
		return m, CancelRequestedCmd()
	}
	return m, nil
}

// IsShown returns whether the dialog is displayed
func (m *Model) IsShown() bool {
	return m.shown
}

// Show displays the Create Ticket dialog
func (m *Model) Show(providerName string) {
	m.shown = true
	m.providerName = providerName
	m.focusedField = 0
	m.titleInput.Focus()
	m.bodyInput.Blur()
	m.Reset()
}

// Hide hides the dialog
func (m *Model) Hide() {
	m.shown = false
	m.Reset()
}

// Reset clears the form
func (m *Model) Reset() {
	m.titleInput.SetValue("")
	m.bodyInput.SetValue("")
	m.focusedField = 0
}

// GetSummary returns the title/summary
func (m *Model) GetSummary() string {
	return m.titleInput.Value()
}

// GetDescription returns the description
func (m *Model) GetDescription() string {
	return m.bodyInput.Value()
}

// GetFocusedField returns the focused field (0=title, 1=body)
func (m *Model) GetFocusedField() int {
	return m.focusedField
}

// SetFocusedField sets the focused field
func (m *Model) SetFocusedField(i int) {
	if i != 0 && i != 1 {
		return
	}
	m.focusedField = i
	if i == 0 {
		m.titleInput.Focus()
		m.bodyInput.Blur()
	} else {
		m.titleInput.Blur()
		m.bodyInput.Focus()
	}
}

// GetTitleInput returns the title input
func (m *Model) GetTitleInput() *textinput.Model {
	return &m.titleInput
}

// GetBodyInput returns the body textarea
func (m *Model) GetBodyInput() *textarea.Model {
	return &m.bodyInput
}

// CreateTicketInput builds tickets.CreateTicketInput from the form
func (m *Model) CreateTicketInput() *tickets.CreateTicketInput {
	return &tickets.CreateTicketInput{
		Summary:     m.GetSummary(),
		Description: m.GetDescription(),
	}
}
