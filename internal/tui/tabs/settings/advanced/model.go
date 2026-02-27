package advanced

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
)

// Model represents the Advanced settings sub-tab (sanitize bookmarks, graph revset, cleanup confirmation).
type Model struct {
	sanitizeBookmarks bool
	confirmingCleanup string
	graphRevsetInput  textinput.Model
	focusedField      int // 0 = graph revset input
}

// NewModel creates a new Advanced settings model
func NewModel() Model {
	revsetInput := textinput.New()
	revsetInput.Placeholder = "e.g. trunk() | (ancestors(@) - ancestors(trunk()))"
	revsetInput.CharLimit = 500
	revsetInput.Width = 60

	return Model{
		sanitizeBookmarks: true,
		confirmingCleanup: "",
		graphRevsetInput:  revsetInput,
		focusedField:      0,
	}
}

// NewModelFromConfig creates a model initialized from config.
func NewModelFromConfig(cfg *config.Config) Model {
	m := NewModel()
	if cfg != nil {
		m.sanitizeBookmarks = cfg.ShouldSanitizeBookmarkNames()
		m.graphRevsetInput.SetValue(cfg.GraphRevset)
	}
	return m
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages (key handling for revset input; zones handled by parent)
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if m.focusedField == 0 {
		var cmd tea.Cmd
		m.graphRevsetInput, cmd = m.graphRevsetInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

// View renders the model
func (m Model) View() string {
	return "" // Rendered by parent
}

// Accessors

// GetSanitizeBookmarks returns whether to sanitize bookmark names
func (m *Model) GetSanitizeBookmarks() bool {
	return m.sanitizeBookmarks
}

// SetSanitizeBookmarks sets whether to sanitize bookmark names
func (m *Model) SetSanitizeBookmarks(sanitize bool) {
	m.sanitizeBookmarks = sanitize
}

// GetGraphRevset returns the graph revset string
func (m *Model) GetGraphRevset() string {
	return m.graphRevsetInput.Value()
}

// SetGraphRevset sets the graph revset string
func (m *Model) SetGraphRevset(s string) {
	m.graphRevsetInput.SetValue(s)
}

// GetConfirmingCleanup returns the current cleanup confirmation type ("", "delete_bookmarks", "abandon_old_commits")
func (m *Model) GetConfirmingCleanup() string {
	return m.confirmingCleanup
}

// SetConfirmingCleanup sets the cleanup confirmation type
func (m *Model) SetConfirmingCleanup(s string) {
	m.confirmingCleanup = s
}

// GetInputViews returns the view strings for the graph revset input
func (m *Model) GetInputViews() []string {
	return []string{m.graphRevsetInput.View()}
}

// GetFocusedField returns the focused input index (0 = graph revset)
func (m *Model) GetFocusedField() int {
	return m.focusedField
}

// SetFocusedField sets the focused input index (0 = graph revset).
// Returns the tea.Cmd from Focus() so the cursor is shown; caller must return it from Update.
func (m *Model) SetFocusedField(i int) tea.Cmd {
	if i < 0 {
		i = 0
	}
	m.focusedField = i
	if m.focusedField == 0 {
		return m.graphRevsetInput.Focus()
	}
	m.graphRevsetInput.Blur()
	return nil
}

// SetInputWidth sets the graph revset input width (minimum 40 so the field and cursor are visible).
func (m *Model) SetInputWidth(w int) {
	if w < 40 {
		w = 40
	}
	m.graphRevsetInput.Width = w
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {}
