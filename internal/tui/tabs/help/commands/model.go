package commands

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
)

// CommandInfo represents information about a command
type CommandInfo struct {
	Name  string
	Desc  string
	Usage string
}

// Model represents the Help Commands tab
type Model struct {
	commandHistory []CommandInfo
	selectedIdx    int
	statusMessage  string
}

// NewModel creates a new Help Commands model
func NewModel() Model {
	return Model{
		selectedIdx: 0,
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
	return m, nil
}

// View renders the model
func (m Model) View() string {
	return "" // Rendered by parent
}

// handleKeyMsg handles keyboard input
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.selectedIdx < len(m.commandHistory)-1 {
			m.selectedIdx++
		}
		return m, nil
	case "k", "up":
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
		return m, nil
	}
	return m, nil
}

// Accessors

// GetCommandHistory returns the command history
func (m *Model) GetCommandHistory() []CommandInfo {
	return m.commandHistory
}

// SetCommandHistory sets the command history
func (m *Model) SetCommandHistory(history []CommandInfo) {
	m.commandHistory = history
	if m.selectedIdx >= len(history) {
		m.selectedIdx = len(history) - 1
	}
}

// GetSelectedIdx returns the selected command index
func (m *Model) GetSelectedIdx() int {
	return m.selectedIdx
}

// SetSelectedIdx sets the selected command index
func (m *Model) SetSelectedIdx(idx int) {
	if idx >= 0 && idx < len(m.commandHistory) {
		m.selectedIdx = idx
	}
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
	// Help doesn't depend on repository
}
