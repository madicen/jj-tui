package github

import (
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
)

// Model represents the GitHub settings sub-tab
type Model struct {
tokenInput           textinput.Model
showMerged           bool
showClosed           bool
onlyMine             bool
prLimit              int
prRefreshInterval int
focusedField      int
}

// NewModel creates a new GitHub settings model
func NewModel() Model {
	tokenInput := textinput.New()
	tokenInput.Placeholder = "GitHub Personal Access Token"
	tokenInput.CharLimit = 256
	tokenInput.Width = 50
	tokenInput.EchoMode = textinput.EchoPassword
	tokenInput.EchoCharacter = '•'
	tokenInput.Focus()

	return Model{
		tokenInput:        tokenInput,
		showMerged:        true,
		showClosed:        true,
		onlyMine:          false,
		prLimit:           100,
		prRefreshInterval: 120,
		focusedField:      0,
	}
}

// NewModelFromConfig creates a model initialized from config and env.
func NewModelFromConfig(cfg *config.Config) Model {
	m := NewModel()
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" && cfg != nil && cfg.GitHubToken != "" {
		token = cfg.GitHubToken
	}
	m.tokenInput.SetValue(token)
	if cfg != nil {
		m.showMerged = cfg.ShowMergedPRs()
		m.showClosed = cfg.ShowClosedPRs()
		m.onlyMine = cfg.OnlyMyPRs()
		m.prLimit = cfg.PRLimit()
		m.prRefreshInterval = cfg.PRRefreshInterval()
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
		// Only handle nav and space (toggles) here; other keys go to token input when focused
		switch msg.String() {
		case "j", "down", "k", "up", " ":
			return m.handleKeyMsg(msg)
		}
	}

	var cmd tea.Cmd
	if m.focusedField == 0 {
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
if m.focusedField < 4 {
if m.focusedField > 0 {
m.tokenInput.Blur()
}
m.focusedField++
if m.focusedField == 0 {
m.tokenInput.Focus()
}
}
return m, nil
case "k", "up":
if m.focusedField > 0 {
m.focusedField--
if m.focusedField == 0 {
m.tokenInput.Focus()
}
}
return m, nil
case " ":
// Toggle boolean options
if m.focusedField == 1 {
m.showMerged = !m.showMerged
} else if m.focusedField == 2 {
m.showClosed = !m.showClosed
} else if m.focusedField == 3 {
m.onlyMine = !m.onlyMine
}
return m, nil
}
return m, nil
}

// Accessors

// GetToken returns the GitHub token
func (m *Model) GetToken() string {
return m.tokenInput.Value()
}

// SetToken sets the GitHub token
func (m *Model) SetToken(token string) {
m.tokenInput.SetValue(token)
}

// GetShowMerged returns whether to show merged PRs
func (m *Model) GetShowMerged() bool {
return m.showMerged
}

// SetShowMerged sets whether to show merged PRs
func (m *Model) SetShowMerged(show bool) {
m.showMerged = show
}

// GetShowClosed returns whether to show closed PRs
func (m *Model) GetShowClosed() bool {
return m.showClosed
}

// SetShowClosed sets whether to show closed PRs
func (m *Model) SetShowClosed(show bool) {
m.showClosed = show
}

// GetOnlyMine returns whether to show only own PRs
func (m *Model) GetOnlyMine() bool {
return m.onlyMine
}

// SetOnlyMine sets whether to show only own PRs
func (m *Model) SetOnlyMine(only bool) {
m.onlyMine = only
}

// GetPRLimit returns the PR limit
func (m *Model) GetPRLimit() int {
return m.prLimit
}

// SetPRLimit sets the PR limit
func (m *Model) SetPRLimit(limit int) {
m.prLimit = limit
}

// GetRefreshInterval returns the refresh interval
func (m *Model) GetRefreshInterval() int {
	return m.prRefreshInterval
}

// SetRefreshInterval sets the refresh interval
func (m *Model) SetRefreshInterval(interval int) {
	m.prRefreshInterval = interval
}

// GetInputViews returns the view strings for inputs (token only).
func (m *Model) GetInputViews() []string {
	return []string{m.tokenInput.View()}
}

// GetFocusedField returns the focused input index (0 for token).
func (m *Model) GetFocusedField() int {
	return m.focusedField
}

// SetFocusedField sets the focused input index (0 = token).
func (m *Model) SetFocusedField(i int) {
	if i < 0 {
		i = 0
	}
	m.focusedField = i
	if m.focusedField == 0 {
		m.tokenInput.Focus()
	} else {
		m.tokenInput.Blur()
	}
}

// SetInputWidth sets the token input width.
func (m *Model) SetInputWidth(w int) {
	m.tokenInput.Width = w
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
// GitHub settings don't depend on repository
}
