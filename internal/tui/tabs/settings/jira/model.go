package jira

import (
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
)

// Model represents the Jira settings sub-tab
type Model struct {
	urlInput      textinput.Model
	userInput     textinput.Model
	tokenInput   textinput.Model
	projectInput textinput.Model
	jqlInput     textinput.Model
	excludedInput textinput.Model
	focusedField int
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

	projectInput := textinput.New()
	projectInput.Placeholder = "PROJ or PROJ,TEAM (comma-separated, optional)"
	projectInput.CharLimit = 200
	projectInput.Width = 50

	jqlInput := textinput.New()
	jqlInput.Placeholder = "sprint in openSprints() (optional custom JQL)"
	jqlInput.CharLimit = 500
	jqlInput.Width = 50

	excludedInput := textinput.New()
	excludedInput.Placeholder = "Done, Won't Do, Cancelled (comma-separated)"
	excludedInput.CharLimit = 200
	excludedInput.Width = 50

	return Model{
		urlInput:      urlInput,
		userInput:     userInput,
		tokenInput:    tokenInput,
		projectInput:  projectInput,
		jqlInput:      jqlInput,
		excludedInput: excludedInput,
		focusedField:  0,
	}
}

// NewModelFromConfig creates a model initialized from config and env.
func NewModelFromConfig(cfg *config.Config) Model {
	m := NewModel()
	m.urlInput.SetValue(os.Getenv("JIRA_URL"))
	m.userInput.SetValue(os.Getenv("JIRA_USER"))
	m.tokenInput.SetValue(os.Getenv("JIRA_TOKEN"))
	jiraProject := os.Getenv("JIRA_PROJECT")
	if jiraProject == "" && cfg != nil {
		jiraProject = cfg.JiraProject
	}
	m.projectInput.SetValue(jiraProject)
	jiraJQL := os.Getenv("JIRA_JQL")
	if jiraJQL == "" && cfg != nil {
		jiraJQL = cfg.JiraJQL
	}
	m.jqlInput.SetValue(jiraJQL)
	if cfg != nil {
		m.excludedInput.SetValue(cfg.JiraExcludedStatuses)
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
	case 3:
		m.projectInput, cmd = m.projectInput.Update(msg)
	case 4:
		m.jqlInput, cmd = m.jqlInput.Update(msg)
	case 5:
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
		if m.focusedField < 5 {
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
	case 3:
		m.projectInput.Focus()
	case 4:
		m.jqlInput.Focus()
	case 5:
		m.excludedInput.Focus()
	}
}

func (m *Model) unfocus() {
	m.urlInput.Blur()
	m.userInput.Blur()
	m.tokenInput.Blur()
	m.projectInput.Blur()
	m.jqlInput.Blur()
	m.excludedInput.Blur()
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

// GetProject returns the Jira project filter
func (m *Model) GetProject() string {
	return m.projectInput.Value()
}

// SetProject sets the Jira project filter
func (m *Model) SetProject(s string) {
	m.projectInput.SetValue(s)
}

// GetJQL returns the Jira JQL filter
func (m *Model) GetJQL() string {
	return m.jqlInput.Value()
}

// SetJQL sets the Jira JQL filter
func (m *Model) SetJQL(s string) {
	m.jqlInput.SetValue(s)
}

// GetExcludedStatuses returns the Jira excluded statuses
func (m *Model) GetExcludedStatuses() string {
	return m.excludedInput.Value()
}

// SetExcludedStatuses sets the Jira excluded statuses
func (m *Model) SetExcludedStatuses(s string) {
	m.excludedInput.SetValue(s)
}

// GetInputViews returns the view strings for all 6 inputs
func (m *Model) GetInputViews() []string {
	return []string{
		m.urlInput.View(),
		m.userInput.View(),
		m.tokenInput.View(),
		m.projectInput.View(),
		m.jqlInput.View(),
		m.excludedInput.View(),
	}
}

// GetFocusedField returns the focused input index (0-5)
func (m *Model) GetFocusedField() int {
	return m.focusedField
}

// SetFocusedField sets the focused input index (0-5)
func (m *Model) SetFocusedField(i int) {
	if i < 0 {
		i = 0
	}
	if i > 5 {
		i = 5
	}
	m.focusedField = i
	m.unfocus()
	m.focus()
}

// SetInputWidth sets the width of all inputs
func (m *Model) SetInputWidth(w int) {
	m.urlInput.Width = w
	m.userInput.Width = w
	m.tokenInput.Width = w
	m.projectInput.Width = w
	m.jqlInput.Width = w
	m.excludedInput.Width = w
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
	// Jira settings don't depend on repository
}
