package shortcuts

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
)

// Model represents the Help Shortcuts tab
type Model struct {
	statusMessage string
}

// NewModel creates a new Help Shortcuts model
func NewModel() Model {
	return Model{}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		_ = msg
		// No navigation in shortcuts tab, just display
	}
	return m, nil
}

// View renders the model
func (m Model) View() string {
	return "" // Rendered by parent
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
	// Help doesn't depend on repository
}
