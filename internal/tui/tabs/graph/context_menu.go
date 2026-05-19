package graph

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal/tui/longpress"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

const longPressThreshold = 500 * time.Millisecond

// ContextMenuState holds the state of the long-press context menu overlay.
type ContextMenuState struct {
	FileIndex int // which changed file this menu is for
	MouseX    int // terminal-relative press position for overlay placement
	MouseY    int
	PressID   int // matches longPressPressID to guard against stale ticks
	HoverItem int // index of menu item under the mouse (-1 = none)
}

// LongPressTickMsg fires after the long-press threshold to show the context menu.
type LongPressTickMsg struct {
	PressID int
}

// contextMenuItem describes one row in the context menu.
type contextMenuItem struct {
	Label   string
	Key     string
	Request Request
	Mutable bool // true = only shown for mutable commits
}

func contextMenuItems() []contextMenuItem {
	return []contextMenuItem{
		{Label: "View diff", Key: "o", Request: Request{ViewFileDiff: true}},
		{Label: "Open in editor", Key: "O", Request: Request{OpenInExternalEditor: true}},
		{Label: "Move to Parent", Key: "[", Request: Request{MoveFileUp: true}, Mutable: true},
		{Label: "Move to Child", Key: "]", Request: Request{MoveFileDown: true}, Mutable: true},
		{Label: "Revert Changes", Key: "v", Request: Request{RevertFile: true}, Mutable: true},
	}
}

// renderContextMenu returns the styled, zone-marked context menu string.
func (m *GraphModel) renderContextMenu(isMutable bool) string {
	items := contextMenuItems()

	menuBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorPrimary).
		Padding(0, 1)

	itemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F8F8F2"))
	hoverStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F8F8F2")).
		Background(styles.ColorPrimary)
	hoverKeyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CCCCCC")).
		Background(styles.ColorPrimary)
	disabledStyle := lipgloss.NewStyle().
		Foreground(styles.ColorMuted)
	keyStyle := lipgloss.NewStyle().
		Foreground(styles.ColorMuted)

	hoverIdx := -1
	if m.contextMenu != nil {
		hoverIdx = m.contextMenu.HoverItem
	}

	var rows []string
	for i, item := range items {
		if item.Mutable && !isMutable {
			row := disabledStyle.Render(fmt.Sprintf("  %s  %s", item.Label, item.Key))
			rows = append(rows, row)
			continue
		}
		isHovered := i == hoverIdx
		ls := itemStyle
		ks := keyStyle
		if isHovered {
			ls = hoverStyle
			ks = hoverKeyStyle
		}
		label := ls.Render(fmt.Sprintf("  %s", item.Label))
		key := ks.Render(fmt.Sprintf("  %s", item.Key))
		row := m.zoneManager.Mark(mouse.ZoneCtxMenuItem(i), label+key)
		rows = append(rows, row)
	}

	fileName := ""
	if m.contextMenu != nil && m.contextMenu.FileIndex >= 0 && m.contextMenu.FileIndex < len(m.changedFiles) {
		fileName = m.changedFiles[m.contextMenu.FileIndex].Path
		parts := strings.Split(fileName, "/")
		fileName = parts[len(parts)-1]
	}
	header := lipgloss.NewStyle().
		Foreground(styles.ColorSecondary).
		Bold(true).
		Render(fileName)

	content := header + "\n" + strings.Join(rows, "\n")
	return menuBorder.Render(content)
}

// handleFileLongPress detects press/release on changed-file zones for the long-press
// context menu. Returns a tea.Cmd (the tick) on press, nil otherwise.
func (m *GraphModel) handleFileLongPress(msg tea.MouseMsg) tea.Cmd {
	// Track hover over menu items while the context menu is visible.
	if m.contextMenu != nil && (msg.Action == tea.MouseActionMotion || msg.Action == tea.MouseActionPress) {
		items := contextMenuItems()
		hit := -1
		for i := range items {
			z := m.zoneManager.Get(mouse.ZoneCtxMenuItem(i))
			if z != nil && z.InBounds(msg) {
				hit = i
				break
			}
		}
		m.contextMenu.HoverItem = hit
	}

	switch msg.Action {
	case tea.MouseActionMotion:
		// Tolerate small drift so trackpad jitter / cell-boundary grazes
		// don't kill the press before the threshold fires. See longpress.
		if m.contextMenu == nil && m.longPressFileIndex >= 0 {
			origin := mouse.ZoneChangedFile(m.longPressFileIndex)
			if !longpress.StillArmed(m.zoneManager, origin, m.longPressMouseX, m.longPressMouseY, msg) {
				m.longPressFileIndex = -1
			}
		}

	case tea.MouseActionPress:
		if msg.Button != tea.MouseButtonLeft {
			return nil
		}
		if m.contextMenu != nil {
			return nil
		}
		for i := range m.changedFiles {
			z := m.zoneManager.Get(mouse.ZoneChangedFile(i))
			if z != nil && z.InBounds(msg) {
				m.longPressPressID++
				m.longPressFileIndex = i
				m.longPressMouseX = msg.X
				m.longPressMouseY = msg.Y
				pressID := m.longPressPressID
				return tea.Tick(longPressThreshold, func(time.Time) tea.Msg {
					return LongPressTickMsg{PressID: pressID}
				})
			}
		}

	case tea.MouseActionRelease:
		m.longPressFileIndex = -1
	}
	return nil
}
