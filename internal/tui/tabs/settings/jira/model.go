package jira

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
)

// Model represents the Jira settings sub-tab
type Model struct {
	urlInput      textinput.Model
	userInput     textinput.Model
	tokenInput    textinput.Model
	focusedField  int
	statusMessage string
}

// NewModel creates a new Jira settings model
func NewModel() Model {
	urlInput := textinput.New()
	urlInput.Placeholder = "https://your-domain.atlassian.net"
	urlInput.CharLimit = 100
	urlInput.Width = 50
	urlInput.Focus()

	userInput := textinput.New()
	userInput.Placeholder = "your-email@example.com"
	userInput.CharLimit = 100
	userInput.Width = 50

	tokenInput := textinput.New()
	tokenInput.Placeholder = "Jira API Token"
	tokenInput.CharLimit = 256
	tokenInput.Width = 50
	tokenInput.EchoMode = textinput.EchoPassword
	tokenInput.EchoCharacter = '•'

	return Model{
		urlInput:     urlInput,
		userInput:    userInput,
		tokenInput:   tokenInput,
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
	switch m.focusedField {
	case 0:
		m.urlInput, cmd = m.urlInput.Update(msg)
	case 1:
		m.userInput, cmd = m.userInput.Update(msg)
	case 2:
		m.tokenInput, cmd = m.tokenInput.Update(msg)
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
		if m.focusedField < 2 {
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
		m.urlInput.Focus()
	case 1:
		m.userInput.Focus()
	case 2:
		m.tokenInput.Focus()
	}
}

func (m *Model) unfocus() {
	m.urlInput.Blur()
	m.userInput.Blur()
	m.tokenInput.Blur()
}

// Accessors

// GetURL returns the Jira URL
func (m *Model) GetURL() string {
	return m.urlInput.Value()
}

// SetURL sets the Jira URL
func (m *Model) SetURL(url string) {
	m.urlInput.SetValue(url)
}

// GetUser returns the Jira user
func (m *Model) GetUser() string {
	return m.userInput.Value()
}

// SetUser sets the Jira user
func (m *Model) SetUser(user string) {
	m.userInput.SetValue(user)
}

// GetToken returns the Jira token
func (m *Model) GetToken() string {
	return m.tokenInput.Value()
}

// SetToken sets the Jira token
func (m *Model) SetToken(token string) {
	m.tokenInput.SetValue(token)
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
	// Jira settings don't depend on repository
}
