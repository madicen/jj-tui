package help

import (
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
)

// CommandInfo represents information about a command (legacy)
type CommandInfo struct {
	Name  string
	Desc  string
	Usage string
}

// CommandHistoryEntry is one entry for the command history list (from jj service)
type CommandHistoryEntry struct {
	Command   string
	Timestamp string
	Duration  string
	Success   bool
	Error     string
}

// Model represents the state of the Help tab
type Model struct {
	zoneManager           *zone.Manager
	helpTab               int   // 0=Shortcuts, 1=Commands
	helpSelectedCommand   int
	commandHistory        []CommandInfo          // legacy
	commandHistoryEntries []CommandHistoryEntry  // for rendering (set by main model)
	width                 int
	height                int
	helpYOffset           int   // scroll offset for wheel scrolling (no click required)
}

// NewModel creates a new Help tab model. zoneManager may be nil.
func NewModel(zoneManager *zone.Manager) Model {
	return Model{
		zoneManager: zoneManager,
		helpTab:     0,
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tea.MouseMsg:
		if tea.MouseEvent(msg).IsWheel() {
			delta := 3
			isUp := msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelLeft
			if isUp {
				m.helpYOffset -= delta
			} else {
				m.helpYOffset += delta
			}
			if m.helpYOffset < 0 {
				m.helpYOffset = 0
			}
			return m, nil
		}
	}
	return m, nil
}

// View renders the Help tab
func (m Model) View() string {
	return m.renderHelp()
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
			maxCmd := len(m.commandHistoryEntries) - 1
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
	case "y":
		if m.helpTab == 1 && len(m.commandHistoryEntries) > 0 && m.helpSelectedCommand >= 0 && m.helpSelectedCommand < len(m.commandHistoryEntries) {
			return m, Request{CopyCommand: m.commandHistoryEntries[m.helpSelectedCommand].Command}.Cmd()
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

// SetDimensions sets the content area size (used for scroll window height).
func (m *Model) SetDimensions(width, height int) {
	m.width = width
	m.height = height
}

// GetSelectedCommand returns the index of the selected command
func (m *Model) GetSelectedCommand() int {
	return m.helpSelectedCommand
}

// SetSelectedCommand sets the selected command index
func (m *Model) SetSelectedCommand(idx int) {
	if idx >= -1 && idx < len(m.commandHistoryEntries) {
		m.helpSelectedCommand = idx
	}
}

// SetCommandHistoryEntries sets the command history for the Commands sub-tab (called by main model)
func (m *Model) SetCommandHistoryEntries(entries []CommandHistoryEntry) {
	m.commandHistoryEntries = entries
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
