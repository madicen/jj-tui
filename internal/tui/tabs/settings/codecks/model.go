package codecks

import (
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/tui/form"
)

// Field indices into the shared form (also the parent's local focus order).
const (
	fieldSubdomain = iota
	fieldToken
	fieldProject
	fieldExcluded
)

// Model represents the Codecks settings sub-tab. Focus/navigation/width plumbing
// lives in the embedded form.Model; the getters/setters below map config fields
// onto form field indices.
type Model struct {
	form form.Model
}

// NewModel creates a new Codecks settings model
func NewModel() Model {
	subdomainInput := textinput.New()
	subdomainInput.Placeholder = "your-team (from your-team.codecks.io)"
	subdomainInput.CharLimit = 100
	subdomainInput.Width = 50

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
		form: form.New(subdomainInput, tokenInput, projectInput, excludedInput),
	}
}

// NewModelFromConfig creates a model initialized from config and env.
func NewModelFromConfig(cfg *config.Config) Model {
	m := NewModel()
	m.SetSubdomain(os.Getenv("CODECKS_SUBDOMAIN"))
	m.SetToken(os.Getenv("CODECKS_TOKEN"))
	m.SetProject(os.Getenv("CODECKS_PROJECT"))
	if cfg != nil {
		m.SetExcludedStatuses(cfg.CodecksExcludedStatuses)
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

// GetSubdomain returns the Codecks subdomain
func (m *Model) GetSubdomain() string { return m.form.Value(fieldSubdomain) }

// SetSubdomain sets the Codecks subdomain
func (m *Model) SetSubdomain(s string) { m.form.SetValue(fieldSubdomain, s) }

// GetToken returns the Codecks token
func (m *Model) GetToken() string { return m.form.Value(fieldToken) }

// SetToken sets the Codecks token
func (m *Model) SetToken(s string) { m.form.SetValue(fieldToken, s) }

// GetProject returns the Codecks project filter
func (m *Model) GetProject() string { return m.form.Value(fieldProject) }

// SetProject sets the Codecks project filter
func (m *Model) SetProject(s string) { m.form.SetValue(fieldProject, s) }

// GetExcludedStatuses returns the Codecks excluded statuses
func (m *Model) GetExcludedStatuses() string { return m.form.Value(fieldExcluded) }

// SetExcludedStatuses sets the Codecks excluded statuses
func (m *Model) SetExcludedStatuses(s string) { m.form.SetValue(fieldExcluded, s) }

// GetAPIKey returns the Codecks API key (alias for GetToken for compatibility)
func (m *Model) GetAPIKey() string { return m.form.Value(fieldToken) }

// SetAPIKey sets the Codecks API key (alias for SetToken)
func (m *Model) SetAPIKey(key string) { m.form.SetValue(fieldToken, key) }

// GetInputViews returns the view strings for all 4 inputs
func (m *Model) GetInputViews() []string { return m.form.Views() }

// GetFocusedField returns the focused input index (0-3)
func (m *Model) GetFocusedField() int { return m.form.Focused() }

// SetFocusedField sets the focused input index (0-3)
func (m *Model) SetFocusedField(i int) { m.form.SetFocused(i) }

// SetInputWidth sets the width of all inputs
func (m *Model) SetInputWidth(w int) { m.form.SetWidth(w) }

// P2.8: Codecks settings don't depend on the repository; no-op hook removed.
