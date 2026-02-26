package commandhistory

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/styles"
	"github.com/madicen/jj-tui/internal/tui/util"
)

// Entry is one entry for the command history list (display format from jj service).
type Entry struct {
	Command   string
	Timestamp string
	Duration  string
	Success   bool
	Error     string
}

// Request is sent to the main model for Help tab actions (e.g. copy to clipboard).
type Request struct {
	CopyCommand string // When set, main copies this command string to clipboard
}

// Cmd returns a tea.Cmd that sends this request.
func (r Request) Cmd() tea.Cmd {
	return func() tea.Msg { return r }
}

// ExecuteRequest runs the help request (e.g. copy command to clipboard).
// Returns (statusMsg, cmd). Main sets statusMsg and returns the cmd.
func ExecuteRequest(r Request) (statusMsg string, cmd tea.Cmd) {
	if r.CopyCommand == "" {
		return "", nil
	}
	return "Copied: " + r.CopyCommand, util.CopyToClipboard(r.CopyCommand)
}

// Model is the Command History sub-tab state. It owns entries, selection, scroll, and copy requests.
type Model struct {
	zoneManager *zone.Manager
	width       int
	height      int
	entries     []Entry
	selectedIdx int
	yOffset     int
}

// NewModel creates a new Command History sub-tab model.
func NewModel(zoneManager *zone.Manager) Model {
	return Model{zoneManager: zoneManager, selectedIdx: -1}
}

// Update handles messages for the Command History sub-tab (dimensions, keys, zone clicks, mouse wheel).
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			maxCmd := len(m.entries) - 1
			if maxCmd >= 0 && m.selectedIdx < maxCmd {
				m.selectedIdx++
			}
			return m, nil
		case "k", "up":
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
			return m, nil
		case "y":
			if len(m.entries) > 0 && m.selectedIdx >= 0 && m.selectedIdx < len(m.entries) {
				return m, Request{CopyCommand: m.entries[m.selectedIdx].Command}.Cmd()
			}
			return m, nil
		}
		return m, nil
	case tea.MouseMsg:
		if tea.MouseEvent(msg).IsWheel() {
			delta := 3
			isUp := msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelLeft
			if isUp {
				m.yOffset -= delta
			} else {
				m.yOffset += delta
			}
			if m.yOffset < 0 {
				m.yOffset = 0
			}
		}
		return m, nil
	case zone.MsgZoneInBounds:
		if m.zoneManager != nil && msg.Zone != nil {
			zoneID := m.resolveZone(msg)
			if zoneID != "" {
				return m.handleZoneClick(zoneID)
			}
		}
		return m, nil
	}
	return m, nil
}

func (m Model) resolveZone(msg zone.MsgZoneInBounds) string {
	if msg.Zone == nil {
		return ""
	}
	n := len(m.entries)
	if n > 50 {
		n = 50
	}
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("%s%d", mouse.ZoneHelpCommandCopy, i)
		z := m.zoneManager.Get(id)
		if z != nil && z.InBounds(msg.Event) {
			return id
		}
	}
	return ""
}

func (m Model) handleZoneClick(zoneID string) (Model, tea.Cmd) {
	if strings.HasPrefix(zoneID, mouse.ZoneHelpCommandCopy) {
		s := strings.TrimPrefix(zoneID, mouse.ZoneHelpCommandCopy)
		i, err := strconv.Atoi(s)
		if err == nil && i >= 0 && i < len(m.entries) {
			return m, Request{CopyCommand: m.entries[i].Command}.Cmd()
		}
	}
	return m, nil
}

// View returns the full command history content as a string (parent applies scroll).
func (m Model) View() string {
	lines := m.lines()
	return strings.Join(lines, "\n")
}

// Lines returns the full list of command history lines (for scroll windowing by parent).
func (m Model) Lines() []string {
	return m.lines()
}

// YOffset returns the current scroll offset.
func (m Model) YOffset() int { return m.yOffset }

// SetYOffset sets the scroll offset.
func (m *Model) SetYOffset(y int) {
	if y < 0 {
		y = 0
	}
	m.yOffset = y
}

// SetDimensions sets width and height.
func (m *Model) SetDimensions(width, height int) {
	m.width = width
	m.height = height
}

// SetEntries sets the command history entries (called by parent when entering Help or refreshing).
func (m *Model) SetEntries(entries []Entry) {
	m.entries = entries
	if m.selectedIdx >= len(entries) {
		m.selectedIdx = max(-1, len(entries)-1)
	}
}

// GetSelectedCommand returns the selected command index.
func (m Model) GetSelectedCommand() int {
	return m.selectedIdx
}

// SetSelectedCommand sets the selected command index.
func (m *Model) SetSelectedCommand(idx int) {
	if idx >= -1 && idx < len(m.entries) {
		m.selectedIdx = idx
	}
}

// ZoneIDs returns zone IDs used by this sub-tab (for parent to resolve clicks).
func (m Model) ZoneIDs() []string {
	var ids []string
	n := len(m.entries)
	if n > 50 {
		n = 50
	}
	for i := 0; i < n; i++ {
		ids = append(ids, fmt.Sprintf("%s%d", mouse.ZoneHelpCommandCopy, i))
	}
	return ids
}

func (m Model) lines() []string {
	mark := func(id, content string) string {
		if m.zoneManager == nil {
			return content
		}
		return m.zoneManager.Mark(id, content)
	}
	var lines []string
	lines = append(lines, styles.TitleStyle.Render("Command History"))
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("  Commands executed by jj-tui (excluding auto-refresh)"))
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("  Click [copy] or press y to copy command to clipboard"))
	lines = append(lines, "")

	if len(m.entries) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Italic(true).Render("  No commands executed yet"))
		return lines
	}

	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B"))
	failStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))
	timeStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted).Width(8)
	durationStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted).Width(5)
	cmdStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))
	copyBtnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C")).Bold(true)

	maxCommands := len(m.entries)
	if maxCommands > 50 {
		maxCommands = 50
	}

	for i := 0; i < maxCommands; i++ {
		entry := m.entries[i]
		var statusIcon string
		var entryStyle lipgloss.Style
		if entry.Success {
			statusIcon = successStyle.Render("✓")
			entryStyle = lipgloss.NewStyle()
		} else {
			statusIcon = failStyle.Render("✗")
			entryStyle = failStyle
		}
		prefix := "  "
		if i == m.selectedIdx {
			prefix = "> "
			entryStyle = entryStyle.Bold(true).Background(lipgloss.Color("#44475A"))
		}
		copyBtn := mark(fmt.Sprintf("%s%d", mouse.ZoneHelpCommandCopy, i), copyBtnStyle.Render("[copy]"))
		line := fmt.Sprintf("%s%s %s %s %s %s",
			prefix,
			timeStyle.Render(entry.Timestamp),
			durationStyle.Render(entry.Duration),
			statusIcon,
			cmdStyle.Render(entry.Command),
			copyBtn,
		)
		if i == m.selectedIdx {
			line = entryStyle.Render(line)
		}
		lines = append(lines, mark(fmt.Sprintf("%s%d", mouse.ZoneHelpCommand, i), line))
		if !entry.Success && entry.Error != "" && i == m.selectedIdx {
			lines = append(lines, fmt.Sprintf("    %s", failStyle.Render("Error: "+entry.Error)))
		}
	}

	if len(m.entries) > maxCommands {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(
			fmt.Sprintf("  ... and %d more commands", len(m.entries)-maxCommands)))
	}
	return lines
}
