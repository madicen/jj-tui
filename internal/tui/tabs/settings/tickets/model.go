package tickets

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	bubbledropdown "github.com/madicen/bubble-dropdown"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

// providerValues maps the ticket-provider dropdown indices to their config
// values; providerLabels are the user-facing strings shown in the dropdown.
var (
	providerValues = []string{"", "jira", "codecks", "github_issues"}
	providerLabels = []string{"None (Disabled)", "Jira", "Codecks", "GitHub Issues"}
)

// providerIndex returns the dropdown index for a provider value (0 = None when unknown).
func providerIndex(value string) int {
	for i, v := range providerValues {
		if v == value {
			return i
		}
	}
	return 0
}

// Model represents the Tickets settings sub-tab (provider selection, auto-in-progress, GitHub Issues excluded statuses).
type Model struct {
	ticketProvider       string
	autoInProgress       bool
	githubIssuesExcluded textinput.Model
	focusedField         int

	// providerDropdown replaces the old radio rows for the active ticket provider.
	providerDropdown *bubbledropdown.Dropdown
}

// NewModel creates a new Tickets settings model with default state.
func NewModel() Model {
	excluded := textinput.New()
	excluded.Placeholder = "closed (comma-separated)"
	excluded.CharLimit = 200
	excluded.Width = 50
	return Model{
		ticketProvider:       "",
		autoInProgress:       true,
		githubIssuesExcluded: excluded,
		focusedField:         0,
		providerDropdown: bubbledropdown.New(
			bubbledropdown.WithOptions(providerLabels),
			bubbledropdown.WithAccentColor(string(styles.ColorPrimary)),
		),
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
	m.providerDropdown.SetSelectedIndex(providerIndex(m.ticketProvider))
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
	if m.providerDropdown != nil {
		m.providerDropdown.SetSelectedIndex(providerIndex(s))
	}
}

// ProviderDropdown returns the active-provider dropdown (for rendering and
// overlay). It syncs the accent so the panel tracks the live theme primary color.
func (m *Model) ProviderDropdown() *bubbledropdown.Dropdown {
	if accent := string(styles.ColorPrimary); m.providerDropdown.AccentColor() != accent {
		m.providerDropdown.SetAccentColor(accent)
	}
	return m.providerDropdown
}

// DropdownOpen reports whether the provider dropdown panel is open.
func (m *Model) DropdownOpen() bool { return m.providerDropdown.Open() }

// SetZoneManager wires the bubblezone manager into the provider dropdown.
func (m *Model) SetZoneManager(zm *zone.Manager) {
	m.providerDropdown.SetZoneManager(zm)
}

// UpdateDropdown forwards a message to the provider dropdown and, on selection,
// applies the chosen provider value. Returns any tea.Cmd the dropdown emits.
func (m *Model) UpdateDropdown(msg tea.Msg) tea.Cmd {
	if m.providerDropdown == nil {
		return nil
	}
	wasOpen := m.providerDropdown.Open()
	dd, cmd := m.providerDropdown.Update(msg)
	m.providerDropdown = dd
	if chosen, ok := msg.(bubbledropdown.ItemChosenMsg); ok && wasOpen {
		if chosen.Index >= 0 && chosen.Index < len(providerValues) {
			m.SetTicketProvider(providerValues[chosen.Index])
		}
	}
	return cmd
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
