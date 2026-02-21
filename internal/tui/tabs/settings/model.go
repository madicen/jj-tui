package settings

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tickets"
)

// Model represents the state of the Settings tab
type Model struct {
	settingsTab               int
	settingsInputs            []textinput.Model
	settingsFocusedField      int
	settingsShowMerged        bool
	settingsShowClosed        bool
	settingsOnlyMine          bool
	settingsPRLimit           int
	settingsPRRefreshInterval int
	settingsAutoInProgress    bool
	settingsBranchLimit       int
	settingsSanitizeBookmarks bool
	settingsTicketProvider    string
	confirmingCleanup         string
	ticketService             tickets.Service
	statusMessage             string
	loading                   bool
	err                       error
}

// NewModel creates a new Settings tab model
func NewModel() Model {
	return Model{
		settingsTab:               0,
		settingsFocusedField:      0,
		settingsInputs:            make([]textinput.Model, 5),
		settingsPRLimit:           50,
		settingsPRRefreshInterval: 60,
		settingsBranchLimit:       100,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the Settings tab
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}
	return m, nil
}

// View renders the Settings tab
func (m Model) View() string {
	return ""
}

// handleKeyMsg handles keyboard input specific to the Settings tab
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		if m.settingsFocusedField < len(m.settingsInputs) {
			m.settingsInputs[m.settingsFocusedField].Blur()
		}
		m.settingsTab = (m.settingsTab + 1) % 4
		m.settingsFocusedField = 0
		if len(m.settingsInputs) > 0 {
			m.settingsInputs[0].Focus()
		}
		return m, nil
	case "j", "down":
		if m.settingsFocusedField < len(m.settingsInputs)-1 {
			if m.settingsFocusedField < len(m.settingsInputs) {
				m.settingsInputs[m.settingsFocusedField].Blur()
			}
			m.settingsFocusedField++
			if m.settingsFocusedField < len(m.settingsInputs) {
				m.settingsInputs[m.settingsFocusedField].Focus()
			}
		}
		return m, nil
	case "k", "up":
		if m.settingsFocusedField > 0 {
			m.settingsInputs[m.settingsFocusedField].Blur()
			m.settingsFocusedField--
			m.settingsInputs[m.settingsFocusedField].Focus()
		}
		return m, nil
	}

	if m.settingsFocusedField < len(m.settingsInputs) {
		var cmd tea.Cmd
		m.settingsInputs[m.settingsFocusedField], cmd = m.settingsInputs[m.settingsFocusedField].Update(msg)
		return m, cmd
	}

	return m, nil
}

// Accessors

// GetSettingsTab returns the currently selected settings tab
func (m *Model) GetSettingsTab() int {
	return m.settingsTab
}

// SetSettingsTab sets the settings tab
func (m *Model) SetSettingsTab(tab int) {
	m.settingsTab = tab % 4
}

// GetFocusedField returns the currently focused input field
func (m *Model) GetFocusedField() int {
	return m.settingsFocusedField
}

// SetFocusedField sets the focused input field
func (m *Model) SetFocusedField(idx int) {
	if idx >= 0 && idx < len(m.settingsInputs) {
		if m.settingsFocusedField < len(m.settingsInputs) {
			m.settingsInputs[m.settingsFocusedField].Blur()
		}
		m.settingsFocusedField = idx
		m.settingsInputs[idx].Focus()
	}
}

// GetSettingsInputs returns the settings input fields
func (m *Model) GetSettingsInputs() []textinput.Model {
	return m.settingsInputs
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
	// Repository state is not directly used in settings
	// This is a no-op for settings but required for interface consistency
}
