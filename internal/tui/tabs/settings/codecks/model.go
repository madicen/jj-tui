package codecks

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
)

// Model represents the Codecks settings sub-tab
type Model struct {
	apiKeyInput   textinput.Model
	focusedField  int
	statusMessage string
}

// NewModel creates a new Codecks settings model
func NewModel() Model {
	apiKeyInput := textinput.New()
	apiKeyInput.Placeholder = "Codecks API Key"
	apiKeyInput.CharLimit = 256
	apiKeyInput.Width = 50
	apiKeyInput.EchoMode = textinput.EchoPassword
	apiKeyInput.EchoCharacter = '•'
	apiKeyInput.Focus()

	return Model{
		apiKeyInput:  apiKeyInput,
		focusedField: 0,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	var cmd tea.Cmd
	m.apiKeyInput, cmd = m.apiKeyInput.Update(msg)
	return m, cmd
}

// View renders the model
func (m Model) View() string {
	return "" // Rendered by parent
}

// handleKeyMsg handles keyboard input (simplified, no navigation between fields)
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	return m, nil
}

// Accessors

// GetAPIKey returns the Codecks API key
func (m *Model) GetAPIKey() string {
	return m.apiKeyInput.Value()
}

// SetAPIKey sets the Codecks API key
func (m *Model) SetAPIKey(key string) {
	m.apiKeyInput.SetValue(key)
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
	// Codecks settings don't depend on repository
}
