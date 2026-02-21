package help

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

// Model represents the state of the Help tab
type Model struct {
	helpTab             int
	helpSelectedCommand int
	commandHistory      []CommandInfo
}

// NewModel creates a new Help tab model
func NewModel() Model {
	return Model{
		helpTab:             0,
		helpSelectedCommand: -1,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the Help tab
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}
	return m, nil
}

// View renders the Help tab
func (m Model) View() string {
	return ""
}

// handleKeyMsg handles keyboard input specific to the Help tab
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.helpTab = (m.helpTab + 1) % 2
		m.helpSelectedCommand = 0
		return m, nil
	case "j", "down":
		if m.helpTab == 1 {
			maxCmd := len(m.commandHistory) - 1
			if maxCmd >= 0 && m.helpSelectedCommand < maxCmd {
				m.helpSelectedCommand++
			}
		}
		return m, nil
	case "k", "up":
		if m.helpTab == 1 && m.helpSelectedCommand > 0 {
			m.helpSelectedCommand--
		}
		return m, nil
	}
	return m, nil
}

// Accessors

// GetHelpTab returns the currently selected help tab
func (m *Model) GetHelpTab() int {
	return m.helpTab
}

// SetHelpTab sets the help tab
func (m *Model) SetHelpTab(tab int) {
	m.helpTab = tab % 2
}

// GetSelectedCommand returns the index of the selected command
func (m *Model) GetSelectedCommand() int {
	return m.helpSelectedCommand
}

// SetSelectedCommand sets the selected command index
func (m *Model) SetSelectedCommand(idx int) {
	if idx >= -1 && idx < len(m.commandHistory) {
		m.helpSelectedCommand = idx
	}
}

// GetCommandHistory returns the command history
func (m *Model) GetCommandHistory() []CommandInfo {
	return m.commandHistory
}

// UpdateCommandHistory updates the command history
func (m *Model) UpdateCommandHistory(history []CommandInfo) {
	m.commandHistory = history
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
	// Repository state is not directly used in help
	// This is a no-op for help but required for interface consistency
}
