package commandhistory

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/styles"
	"github.com/madicen/jj-tui/internal/tui/util"
	"github.com/mattn/go-runewidth"
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
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			maxCmd := len(m.entries) - 1
			if maxCmd >= 0 && m.selectedIdx < maxCmd {
				m.selectedIdx++
				m.ensureVisible()
			}
			return m, nil
		case "k", "up":
			if m.selectedIdx > 0 {
				m.selectedIdx--
				m.ensureVisible()
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
	n := min(len(m.entries), 50)
	for i := range n {
		id := fmt.Sprintf("%s%d", mouse.ZoneHelpCommandCopy, i)
		z := m.zoneManager.Get(id)
		if z != nil && z.InBounds(msg.Event) {
			return id
		}
	}
	return ""
}

func (m Model) handleZoneClick(zoneID string) (Model, tea.Cmd) {
	if after, ok := strings.CutPrefix(zoneID, mouse.ZoneHelpCommandCopy); ok {
		s := after
		i, err := strconv.Atoi(s)
		if err == nil && i >= 0 && i < len(m.entries) {
			return m, Request{CopyCommand: m.entries[i].Command}.Cmd()
		}
	}
	return m, nil
}

func (m *Model) ensureVisible() {
	if m.selectedIdx < 0 || m.height <= 0 {
		return
	}
	const headerHeight = 5
	visualIdx := headerHeight + m.selectedIdx

	// Check if the selected item has an error displayed (adds 1 extra line)
	itemHeight := 1
	if m.selectedIdx < len(m.entries) {
		entry := m.entries[m.selectedIdx]
		if !entry.Success && entry.Error != "" {
			itemHeight = 2
		}
	}

	// Scroll up if top of item is above viewport
	if visualIdx < m.yOffset {
		m.yOffset = visualIdx
	} else if visualIdx+itemHeight > m.yOffset+m.height {
		// Scroll down if bottom of item is below viewport
		m.yOffset = visualIdx + itemHeight - m.height
	}
}

// View returns the full command history content as a string (parent applies scroll).
func (m Model) View() string {
	lines := m.lines()
	if m.height > 0 {
		totalLines := len(lines)
		maxOffset := max(0, totalLines-m.height)
		offset := min(m.yOffset, maxOffset)
		end := min(offset+m.height, totalLines)
		lines = lines[offset:end]
	}
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
	m.yOffset = max(y, 0)
}

// SetDimensions sets width and height.
func (m *Model) SetDimensions(width, height int) {
	m.width = width
	// Account for parent Help tab header (sub-tabs) which consumes vertical space
	// but isn't subtracted from the height passed down.
	m.height = max(1, height-3)
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
	n := min(len(m.entries), 50)
	for i := range n {
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

	maxCommands := min(len(m.entries), 50)

	for i := range maxCommands {
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
			entryStyle = entryStyle.Bold(true)
		}
		copyBtn := mark(fmt.Sprintf("%s%d", mouse.ZoneHelpCommandCopy, i), copyBtnStyle.Render("[copy]"))

		// Clean up command text (remove tabs, newlines, excess spaces)
		cmdText := strings.Join(strings.Fields(entry.Command), " ")
		// Truncate to fit: width - fixed parts (approx 27 chars) - safety margin
		availableWidth := max(10, m.width-32)
		cmdText = runewidth.Truncate(cmdText, availableWidth, "...")

		line := fmt.Sprintf("%s%s %s %s %s %s",
			prefix,
			timeStyle.Render(entry.Timestamp),
			durationStyle.Render(entry.Duration),
			statusIcon,
			cmdStyle.Render(cmdText),
			copyBtn,
		)
		if i == m.selectedIdx {
			line = entryStyle.Render(line)
		}
		lines = append(lines, mark(fmt.Sprintf("%s%d", mouse.ZoneHelpCommand, i), line))
		if !entry.Success && entry.Error != "" && i == m.selectedIdx {
			errText := strings.Join(strings.Fields(entry.Error), " ")
			errText = runewidth.Truncate(errText, max(10, m.width-15), "...")
			lines = append(lines, fmt.Sprintf("    %s", failStyle.Render("Error: "+errText)))
		}
	}

	if len(m.entries) > maxCommands {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(
			fmt.Sprintf("  ... and %d more commands", len(m.entries)-maxCommands)))
	}
	return lines
}
