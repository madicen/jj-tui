package advanced

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
)

// Model represents the Advanced settings sub-tab (sanitize bookmarks, cleanup confirmation).
type Model struct {
	sanitizeBookmarks bool
	confirmingCleanup string
}

// NewModel creates a new Advanced settings model
func NewModel() Model {
	return Model{
		sanitizeBookmarks: true,
		confirmingCleanup: "",
	}
}

// NewModelFromConfig creates a model initialized from config.
func NewModelFromConfig(cfg *config.Config) Model {
	m := NewModel()
	if cfg != nil {
		m.sanitizeBookmarks = cfg.ShouldSanitizeBookmarkNames()
	}
	return m
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages (no key handling; zones handled by parent)
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
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

// GetConfirmingCleanup returns the current cleanup confirmation type ("", "delete_bookmarks", "abandon_old_commits")
func (m *Model) GetConfirmingCleanup() string {
	return m.confirmingCleanup
}

// SetConfirmingCleanup sets the cleanup confirmation type
func (m *Model) SetConfirmingCleanup(s string) {
	m.confirmingCleanup = s
}

// GetInputViews returns nil (no text inputs in advanced panel)
func (m *Model) GetInputViews() []string {
	return nil
}

// GetFocusedField returns 0 (no focus in advanced)
func (m *Model) GetFocusedField() int {
	return 0
}

// SetFocusedField is a no-op for advanced
func (m *Model) SetFocusedField(i int) {}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {}
