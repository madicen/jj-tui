package help

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/tabs/help/commandhistory"
	"github.com/madicen/jj-tui/internal/tui/tabs/help/shortcuts"
)

// CommandInfo represents information about a command (legacy)
type CommandInfo struct {
	Name  string
	Desc  string
	Usage string
}

// Model represents the state of the Help tab. It routes to Shortcuts or Command History sub-tab.
type Model struct {
	zoneManager *zone.Manager
	activeTab   int // 0=Shortcuts, 1=Commands
	width       int
	height      int

	shortcuts shortcuts.Model
	commands  commandhistory.Model
}

// NewModel creates a new Help tab model. zoneManager may be nil.
func NewModel(zoneManager *zone.Manager) Model {
	return Model{
		zoneManager: zoneManager,
		activeTab:   0,
		shortcuts:   shortcuts.NewModel(zoneManager),
		commands:    commandhistory.NewModel(zoneManager),
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update routes messages to the active sub-tab (Shortcuts or Command History).
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.shortcuts.SetDimensions(msg.Width, msg.Height)
		m.commands.SetDimensions(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+j":
			// Previous sub-tab (wrap: 0 -> 1)
			m.activeTab = (m.activeTab - 1 + 2) % 2
			m.commands.SetSelectedCommand(0)
			return m, nil
		case "ctrl+k":
			// Next sub-tab
			m.activeTab = (m.activeTab + 1) % 2
			m.commands.SetSelectedCommand(0)
			return m, nil
		case "tab":
			m.activeTab = (m.activeTab + 1) % 2
			m.commands.SetSelectedCommand(0)
			return m, nil
		}
		if m.activeTab == 0 {
			updated, cmd := m.shortcuts.Update(msg)
			m.shortcuts = updated
			return m, cmd
		}
		updated, cmd := m.commands.Update(msg)
		m.commands = updated
		return m, cmd

	case zone.MsgZoneInBounds:
		if m.zoneManager != nil {
			zoneID := m.resolveClickedZone(msg)
			if zoneID != "" {
				if zoneID == mouse.ZoneHelpTabShortcuts {
					m.activeTab = 0
					m.commands.SetSelectedCommand(0)
					return m, nil
				}
				if zoneID == mouse.ZoneHelpTabCommands {
					m.activeTab = 1
					m.commands.SetSelectedCommand(0)
					return m, nil
				}
				// Forward to commands (copy buttons)
				if m.activeTab == 1 {
					updated, cmd := m.commands.Update(msg)
					m.commands = updated
					return m, cmd
				}
			}
		}
		return m, nil

	case tea.MouseMsg:
		if tea.MouseEvent(msg).IsWheel() {
			if m.activeTab == 0 {
				updated, cmd := m.shortcuts.Update(msg)
				m.shortcuts = updated
				return m, cmd
			}
			updated, cmd := m.commands.Update(msg)
			m.commands = updated
			return m, cmd
		}
		return m, nil
	}
	return m, nil
}

// View renders the Help tab: tab bar + active sub-tab content with scroll.
func (m Model) View() string {
	tabBar := m.renderTabBar()
	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Italic(true).Render("^j/^k: switch tabs")
	visibleHeight := max(1, m.height-3)

	var lines []string
	var start int
	if m.activeTab == 0 {
		lines = m.shortcuts.Lines()
		start = m.shortcuts.YOffset()
	} else {
		lines = m.commands.Lines()
		start = m.commands.YOffset()
	}
	totalLines := len(lines)
	if start > totalLines-visibleHeight {
		start = max(0, totalLines-visibleHeight)
	}
	if start < 0 {
		start = 0
	}
	end := min(start+visibleHeight, totalLines)
	content := ""
	if end > start {
		content = strings.Join(lines[start:end], "\n")
	}
	return tabBar + "\n" + hint + "\n\n" + content
}

// ZoneIDs returns the zone IDs this tab uses when rendering (same IDs passed to Mark). Used to resolve clicks.
func (m Model) ZoneIDs() []string {
	ids := []string{mouse.ZoneHelpTabShortcuts, mouse.ZoneHelpTabCommands}
	ids = append(ids, m.commands.ZoneIDs()...)
	return ids
}

func (m Model) resolveClickedZone(msg zone.MsgZoneInBounds) string {
	if msg.Zone == nil {
		return ""
	}
	for _, id := range m.ZoneIDs() {
		z := m.zoneManager.Get(id)
		if z != nil && z.InBounds(msg.Event) {
			return id
		}
	}
	return ""
}

// Accessors

// GetHelpTab returns the currently selected help tab
func (m *Model) GetHelpTab() int {
	return m.activeTab
}

// SetHelpTab sets the help tab
func (m *Model) SetHelpTab(tab int) {
	m.activeTab = tab % 2
}

// SetDimensions sets the content area size (used for scroll window height).
func (m *Model) SetDimensions(width, height int) {
	m.width = width
	m.height = height
	m.shortcuts.SetDimensions(width, height)
	m.commands.SetDimensions(width, height)
}

// GetSelectedCommand returns the index of the selected command
func (m *Model) GetSelectedCommand() int {
	return m.commands.GetSelectedCommand()
}

// SetSelectedCommand sets the selected command index
func (m *Model) SetSelectedCommand(idx int) {
	m.commands.SetSelectedCommand(idx)
}

// SetCommandHistoryEntries sets the command history for the Commands sub-tab (called by main model)
func (m *Model) SetCommandHistoryEntries(entries []commandhistory.Entry) {
	m.commands.SetEntries(entries)
}

// GetCommandHistory returns the command history (legacy)
func (m *Model) GetCommandHistory() []CommandInfo {
	return nil
}

// UpdateCommandHistory updates the command history (legacy)
func (m *Model) UpdateCommandHistory(history []CommandInfo) {}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {}
