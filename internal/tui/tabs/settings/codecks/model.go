package codecks

import (
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
)

// Model represents the Codecks settings sub-tab
type Model struct {
	subdomainInput textinput.Model
	tokenInput     textinput.Model
	projectInput   textinput.Model
	excludedInput  textinput.Model
	focusedField   int
}

// NewModel creates a new Codecks settings model
func NewModel() Model {
	subdomainInput := textinput.New()
	subdomainInput.Placeholder = "your-team (from your-team.codecks.io)"
	subdomainInput.CharLimit = 100
	subdomainInput.Width = 50
	subdomainInput.Focus()

	tokenInput := textinput.New()
	tokenInput.Placeholder = "Codecks API Token (from browser cookie 'at')"
	tokenInput.CharLimit = 256
	tokenInput.Width = 50
	tokenInput.EchoMode = textinput.EchoPassword
	tokenInput.EchoCharacter = '•'

	projectInput := textinput.New()
	projectInput.Placeholder = "Project name (optional, filters cards)"
	projectInput.CharLimit = 100
	projectInput.Width = 50

	excludedInput := textinput.New()
	excludedInput.Placeholder = "done, archived (comma-separated)"
	excludedInput.CharLimit = 200
	excludedInput.Width = 50

	return Model{
		subdomainInput: subdomainInput,
		tokenInput:     tokenInput,
		projectInput:   projectInput,
		excludedInput:  excludedInput,
		focusedField:   0,
	}
}

// NewModelFromConfig creates a model initialized from config and env.
func NewModelFromConfig(cfg *config.Config) Model {
	m := NewModel()
	m.subdomainInput.SetValue(os.Getenv("CODECKS_SUBDOMAIN"))
	m.tokenInput.SetValue(os.Getenv("CODECKS_TOKEN"))
	m.projectInput.SetValue(os.Getenv("CODECKS_PROJECT"))
	if cfg != nil {
		m.excludedInput.SetValue(cfg.CodecksExcludedStatuses)
	}
	return m
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Only handle nav keys here; all other keys go to the focused input below
		switch msg.String() {
		case "j", "down", "k", "up":
			return m.handleKeyMsg(msg)
		}
	}

	var cmd tea.Cmd
	switch m.focusedField {
	case 0:
		m.subdomainInput, cmd = m.subdomainInput.Update(msg)
	case 1:
		m.tokenInput, cmd = m.tokenInput.Update(msg)
	case 2:
		m.projectInput, cmd = m.projectInput.Update(msg)
	case 3:
		m.excludedInput, cmd = m.excludedInput.Update(msg)
	}
	return m, cmd
}

// View renders the model
func (m Model) View() string {
	return "" // Rendered by parent
}

// handleKeyMsg handles keyboard input
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.focusedField < 3 {
			m.unfocus()
			m.focusedField++
			m.focus()
		}
		return m, nil
	case "k", "up":
		if m.focusedField > 0 {
			m.unfocus()
			m.focusedField--
			m.focus()
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) focus() {
	switch m.focusedField {
	case 0:
		m.subdomainInput.Focus()
	case 1:
		m.tokenInput.Focus()
	case 2:
		m.projectInput.Focus()
	case 3:
		m.excludedInput.Focus()
	}
}

func (m *Model) unfocus() {
	m.subdomainInput.Blur()
	m.tokenInput.Blur()
	m.projectInput.Blur()
	m.excludedInput.Blur()
}

// Accessors

// GetSubdomain returns the Codecks subdomain
func (m *Model) GetSubdomain() string {
	return m.subdomainInput.Value()
}

// SetSubdomain sets the Codecks subdomain
func (m *Model) SetSubdomain(s string) {
	m.subdomainInput.SetValue(s)
}

// GetToken returns the Codecks token
func (m *Model) GetToken() string {
	return m.tokenInput.Value()
}

// SetToken sets the Codecks token
func (m *Model) SetToken(s string) {
	m.tokenInput.SetValue(s)
}

// GetProject returns the Codecks project filter
func (m *Model) GetProject() string {
	return m.projectInput.Value()
}

// SetProject sets the Codecks project filter
func (m *Model) SetProject(s string) {
	m.projectInput.SetValue(s)
}

// GetExcludedStatuses returns the Codecks excluded statuses
func (m *Model) GetExcludedStatuses() string {
	return m.excludedInput.Value()
}

// SetExcludedStatuses sets the Codecks excluded statuses
func (m *Model) SetExcludedStatuses(s string) {
	m.excludedInput.SetValue(s)
}

// GetAPIKey returns the Codecks API key (alias for GetToken for compatibility)
func (m *Model) GetAPIKey() string {
	return m.tokenInput.Value()
}

// SetAPIKey sets the Codecks API key (alias for SetToken)
func (m *Model) SetAPIKey(key string) {
	m.tokenInput.SetValue(key)
}

// GetInputViews returns the view strings for all 4 inputs
func (m *Model) GetInputViews() []string {
	return []string{
		m.subdomainInput.View(),
		m.tokenInput.View(),
		m.projectInput.View(),
		m.excludedInput.View(),
	}
}

// GetFocusedField returns the focused input index (0-3)
func (m *Model) GetFocusedField() int {
	return m.focusedField
}

// SetFocusedField sets the focused input index (0-3)
func (m *Model) SetFocusedField(i int) {
	if i < 0 {
		i = 0
	}
	if i > 3 {
		i = 3
	}
	m.focusedField = i
	m.unfocus()
	m.focus()
}

// SetInputWidth sets the width of all inputs
func (m *Model) SetInputWidth(w int) {
	m.subdomainInput.Width = w
	m.tokenInput.Width = w
	m.projectInput.Width = w
	m.excludedInput.Width = w
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
	// Codecks settings don't depend on repository
}
