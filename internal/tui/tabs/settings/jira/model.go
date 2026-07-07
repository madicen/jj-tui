package jira

import (
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/tui/form"
)

// Field indices into the shared form (also the parent's local focus order).
const (
	fieldURL = iota
	fieldUser
	fieldToken
	fieldProject
	fieldProjectFilter
	fieldIssueType
	fieldJQL
	fieldExcluded
)

// Model represents the Jira settings sub-tab. Focus/navigation/width plumbing
// lives in the embedded form.Model; the getters/setters below map config fields
// onto form field indices.
type Model struct {
	form form.Model
}

// NewModel creates a new Jira settings model
func NewModel() Model {
	urlInput := textinput.New()
	urlInput.Placeholder = "https://your-domain.atlassian.net"
	urlInput.CharLimit = 100
	urlInput.Width = 50

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
	projectInput.Placeholder = "PROJ (required for creating issues)"
	projectInput.CharLimit = 200
	projectInput.Width = 50

	projectFilterInput := textinput.New()
	projectFilterInput.Placeholder = "PROJ or PROJ,TEAM (optional; filters ticket list)"
	projectFilterInput.CharLimit = 200
	projectFilterInput.Width = 50

	issueTypeInput := textinput.New()
	issueTypeInput.Placeholder = "Task (optional; default when creating issues)"
	issueTypeInput.CharLimit = 64
	issueTypeInput.Width = 50

	jqlInput := textinput.New()
	jqlInput.Placeholder = "sprint in openSprints() (optional custom JQL)"
	jqlInput.CharLimit = 500
	jqlInput.Width = 50

	excludedInput := textinput.New()
	excludedInput.Placeholder = "Done, Won't Do, Cancelled (comma-separated)"
	excludedInput.CharLimit = 200
	excludedInput.Width = 50

	return Model{
		form: form.New(urlInput, userInput, tokenInput, projectInput,
			projectFilterInput, issueTypeInput, jqlInput, excludedInput),
	}
}

// NewModelFromConfig creates a model initialized from config and env.
func NewModelFromConfig(cfg *config.Config) Model {
	m := NewModel()
	m.SetURL(os.Getenv("JIRA_URL"))
	m.SetUser(os.Getenv("JIRA_USER"))
	m.SetToken(os.Getenv("JIRA_TOKEN"))
	jiraProject := os.Getenv("JIRA_PROJECT")
	if jiraProject == "" && cfg != nil {
		jiraProject = cfg.JiraProject
	}
	m.SetProject(jiraProject)
	jiraProjectFilter := os.Getenv("JIRA_PROJECT_FILTER")
	if jiraProjectFilter == "" && cfg != nil {
		jiraProjectFilter = cfg.JiraProjectFilter
	}
	m.SetProjectFilter(jiraProjectFilter)
	jiraIssueType := os.Getenv("JIRA_ISSUE_TYPE")
	if jiraIssueType == "" && cfg != nil && cfg.JiraIssueType != "" {
		jiraIssueType = cfg.JiraIssueType
	}
	m.SetIssueType(jiraIssueType)
	jiraJQL := os.Getenv("JIRA_JQL")
	if jiraJQL == "" && cfg != nil {
		jiraJQL = cfg.JiraJQL
	}
	m.SetJQL(jiraJQL)
	if cfg != nil {
		m.SetExcludedStatuses(cfg.JiraExcludedStatuses)
	}
	return m
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages: vertical navigation cycles focus; everything else is
// routed to the focused input.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		if m.form.HandleNavKey(key.String()) {
			return m, nil
		}
	}
	return m, m.form.Update(msg)
}

// View renders the model
func (m Model) View() string {
	return "" // Rendered by parent
}

// Accessors

// GetURL returns the Jira URL
func (m *Model) GetURL() string { return m.form.Value(fieldURL) }

// SetURL sets the Jira URL
func (m *Model) SetURL(url string) { m.form.SetValue(fieldURL, url) }

// GetUser returns the Jira user
func (m *Model) GetUser() string { return m.form.Value(fieldUser) }

// SetUser sets the Jira user
func (m *Model) SetUser(user string) { m.form.SetValue(fieldUser, user) }

// GetToken returns the Jira token
func (m *Model) GetToken() string { return m.form.Value(fieldToken) }

// SetToken sets the Jira token
func (m *Model) SetToken(token string) { m.form.SetValue(fieldToken, token) }

// GetProject returns the Jira project for creating new issues
func (m *Model) GetProject() string { return m.form.Value(fieldProject) }

// SetProject sets the Jira project for creating new issues
func (m *Model) SetProject(s string) { m.form.SetValue(fieldProject, s) }

// GetProjectFilter returns the Jira project filter for the ticket list
func (m *Model) GetProjectFilter() string { return m.form.Value(fieldProjectFilter) }

// SetProjectFilter sets the Jira project filter
func (m *Model) SetProjectFilter(s string) { m.form.SetValue(fieldProjectFilter, s) }

// GetIssueType returns the default Jira issue type when creating issues
func (m *Model) GetIssueType() string { return m.form.Value(fieldIssueType) }

// SetIssueType sets the default Jira issue type
func (m *Model) SetIssueType(s string) { m.form.SetValue(fieldIssueType, s) }

// GetJQL returns the Jira JQL filter
func (m *Model) GetJQL() string { return m.form.Value(fieldJQL) }

// SetJQL sets the Jira JQL filter
func (m *Model) SetJQL(s string) { m.form.SetValue(fieldJQL, s) }

// GetExcludedStatuses returns the Jira excluded statuses
func (m *Model) GetExcludedStatuses() string { return m.form.Value(fieldExcluded) }

// SetExcludedStatuses sets the Jira excluded statuses
func (m *Model) SetExcludedStatuses(s string) { m.form.SetValue(fieldExcluded, s) }

// GetInputViews returns the view strings for all 8 inputs
func (m *Model) GetInputViews() []string { return m.form.Views() }

// GetFocusedField returns the focused input index (0-7)
func (m *Model) GetFocusedField() int { return m.form.Focused() }

// SetFocusedField sets the focused input index (0-7)
func (m *Model) SetFocusedField(i int) { m.form.SetFocused(i) }

// SetInputWidth sets the width of all inputs
func (m *Model) SetInputWidth(w int) { m.form.SetWidth(w) }

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
	// Jira settings don't depend on repository
}
