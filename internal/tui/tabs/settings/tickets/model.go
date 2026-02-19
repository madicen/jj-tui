package tickets

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/config"
)

// Model represents the Tickets settings sub-tab (provider selection, auto-in-progress, GitHub Issues excluded statuses).
type Model struct {
	ticketProvider         string
	autoInProgress         bool
	githubIssuesExcluded   textinput.Model
	focusedField           int
}

// NewModel creates a new Tickets settings model with default state.
func NewModel() Model {
	excluded := textinput.New()
	excluded.Placeholder = "closed (comma-separated)"
	excluded.CharLimit = 200
	excluded.Width = 50
	return Model{
		ticketProvider: "",
		autoInProgress:  true,
		githubIssuesExcluded: excluded,
		focusedField:   0,
	}
}

// NewModelFromConfig creates a model initialized from config and env.
func NewModelFromConfig(cfg *config.Config) Model {
	m := NewModel()
	if cfg != nil {
		m.ticketProvider = cfg.GetTicketProvider()
		m.autoInProgress = cfg.AutoInProgressOnBranch()
		if cfg.GitHubIssuesExcludedStatuses != "" {
			m.githubIssuesExcluded.SetValue(cfg.GitHubIssuesExcludedStatuses)
		}
	}
	return m
}

// Update forwards messages to the focused input when applicable.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if m.ticketProvider != "github_issues" {
		return m, nil
	}
	var cmd tea.Cmd
	m.githubIssuesExcluded, cmd = m.githubIssuesExcluded.Update(msg)
	return m, cmd
}

// GetTicketProvider returns the selected ticket provider ("", "jira", "codecks", "github_issues").
func (m *Model) GetTicketProvider() string {
	return m.ticketProvider
}

// SetTicketProvider sets the ticket provider.
func (m *Model) SetTicketProvider(s string) {
	m.ticketProvider = s
}

// GetAutoInProgress returns whether to auto-set "In Progress" when creating a branch.
func (m *Model) GetAutoInProgress() bool {
	return m.autoInProgress
}

// SetAutoInProgress sets the auto-in-progress flag.
func (m *Model) SetAutoInProgress(v bool) {
	m.autoInProgress = v
}

// GetGitHubIssuesExcludedStatuses returns the excluded statuses for GitHub Issues.
func (m *Model) GetGitHubIssuesExcludedStatuses() string {
	return m.githubIssuesExcluded.Value()
}

// SetGitHubIssuesExcludedStatuses sets the excluded statuses string.
func (m *Model) SetGitHubIssuesExcludedStatuses(s string) {
	m.githubIssuesExcluded.SetValue(s)
}

// GetInputViews returns the view strings for inputs (one element when provider is github_issues).
func (m *Model) GetInputViews() []string {
	if m.ticketProvider != "github_issues" {
		return nil
	}
	return []string{m.githubIssuesExcluded.View()}
}

// GetFocusedField returns the focused input index (0 for the single input when applicable).
func (m *Model) GetFocusedField() int {
	return m.focusedField
}

// SetFocusedField sets the focused input index.
func (m *Model) SetFocusedField(i int) {
	if i < 0 {
		i = 0
	}
	m.focusedField = i
	if m.focusedField == 0 {
		m.githubIssuesExcluded.Focus()
	} else {
		m.githubIssuesExcluded.Blur()
	}
}

// SetInputWidth sets the width of the excluded statuses input.
func (m *Model) SetInputWidth(w int) {
	m.githubIssuesExcluded.Width = w
}
