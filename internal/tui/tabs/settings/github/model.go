package github

import (
"github.com/charmbracelet/bubbles/textinput"
tea "github.com/charmbracelet/bubbletea"
"github.com/madicen/jj-tui/internal"
)

// Model represents the GitHub settings sub-tab
type Model struct {
tokenInput           textinput.Model
showMerged           bool
showClosed           bool
onlyMine             bool
prLimit              int
prRefreshInterval    int
focusedField         int
statusMessage        string
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

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
// GitHub settings don't depend on repository
}
