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

	// GitHub Device Flow (login)
	githubDeviceCode      string
	githubUserCode        string
	githubVerificationURL string
	githubLoginPolling    bool
	githubPollInterval    int
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
	case "esc":
		return m, Request{Cancel: true}.Cmd()
	case "ctrl+s":
		if m.settingsTab != 5 {
			return m, Request{SaveSettings: true}.Cmd()
		}
		return m, nil
	case "ctrl+l":
		return m, Request{SaveSettingsLocal: true}.Cmd()
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

// SetInputs sets the settings input fields (called by main model at init)
func (m *Model) SetInputs(inputs []textinput.Model) {
	m.settingsInputs = inputs
}

// SetInputWidths sets the width of all settings inputs (called on window resize)
func (m *Model) SetInputWidths(width int) {
	for i := range m.settingsInputs {
		m.settingsInputs[i].Width = width
	}
}

// SetSettingInputValue sets the value of the settings input at index (e.g. after GitHub login)
func (m *Model) SetSettingInputValue(index int, value string) {
	if index >= 0 && index < len(m.settingsInputs) {
		m.settingsInputs[index].SetValue(value)
	}
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
	// Repository state is not directly used in settings
	// This is a no-op for settings but required for interface consistency
}

// Getters for toggle/state (used when saving config or rendering)
func (m *Model) GetSettingsShowMerged() bool        { return m.settingsShowMerged }
func (m *Model) GetSettingsShowClosed() bool        { return m.settingsShowClosed }
func (m *Model) GetSettingsOnlyMine() bool          { return m.settingsOnlyMine }
func (m *Model) GetSettingsPRLimit() int            { return m.settingsPRLimit }
func (m *Model) GetSettingsPRRefreshInterval() int  { return m.settingsPRRefreshInterval }
func (m *Model) GetSettingsAutoInProgress() bool   { return m.settingsAutoInProgress }
func (m *Model) GetSettingsBranchLimit() int        { return m.settingsBranchLimit }
func (m *Model) GetSettingsSanitizeBookmarks() bool { return m.settingsSanitizeBookmarks }
func (m *Model) GetSettingsTicketProvider() string  { return m.settingsTicketProvider }
func (m *Model) GetConfirmingCleanup() string      { return m.confirmingCleanup }

// Setters for init (load from config)
func (m *Model) SetSettingsShowMerged(v bool)        { m.settingsShowMerged = v }
func (m *Model) SetSettingsShowClosed(v bool)        { m.settingsShowClosed = v }
func (m *Model) SetSettingsOnlyMine(v bool)          { m.settingsOnlyMine = v }
func (m *Model) SetSettingsPRLimit(v int)            { m.settingsPRLimit = v }
func (m *Model) SetSettingsPRRefreshInterval(v int)  { m.settingsPRRefreshInterval = v }
func (m *Model) SetSettingsAutoInProgress(v bool)   { m.settingsAutoInProgress = v }
func (m *Model) SetSettingsBranchLimit(v int)      { m.settingsBranchLimit = v }
func (m *Model) SetSettingsSanitizeBookmarks(v bool) { m.settingsSanitizeBookmarks = v }
func (m *Model) SetSettingsTicketProvider(s string)  { m.settingsTicketProvider = s }
func (m *Model) SetConfirmingCleanup(s string)       { m.confirmingCleanup = s }

// GitHub Device Flow
func (m *Model) GetGitHubDeviceCode() string      { return m.githubDeviceCode }
func (m *Model) GetGitHubUserCode() string       { return m.githubUserCode }
func (m *Model) GetGitHubVerificationURL() string { return m.githubVerificationURL }
func (m *Model) GetGitHubLoginPolling() bool     { return m.githubLoginPolling }
func (m *Model) GetGitHubPollInterval() int      { return m.githubPollInterval }
func (m *Model) SetGitHubDeviceCode(s string)      { m.githubDeviceCode = s }
func (m *Model) SetGitHubUserCode(s string)       { m.githubUserCode = s }
func (m *Model) SetGitHubVerificationURL(s string) { m.githubVerificationURL = s }
func (m *Model) SetGitHubLoginPolling(v bool)     { m.githubLoginPolling = v }
func (m *Model) SetGitHubPollInterval(i int)      { m.githubPollInterval = i }
