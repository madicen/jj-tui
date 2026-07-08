package prs

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal/tui/listnav"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/render"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

const longPressThreshold = 500 * time.Millisecond

// ContextMenuState holds the state of the PR-row long-press context menu.
type ContextMenuState struct {
	PRIndex   int
	MouseX    int
	MouseY    int
	PressID   int
	HoverItem int // -1 = none
}

// LongPressTickMsg fires after the long-press threshold to show the PR context menu.
type LongPressTickMsg struct {
	PressID int
}

type prContextMenuItem struct {
	Label   string
	Key     string
	Request Request
	// OpenOnly: only shown when the PR is in "open" state.
	OpenOnly bool
}

func prContextMenuItems() []prContextMenuItem {
	return []prContextMenuItem{
		{Label: "Open in Browser", Key: "o", Request: Request{OpenInBrowser: true}},
		{Label: "Merge", Key: "M", Request: Request{MergePR: true}, OpenOnly: true},
		{Label: "Close", Key: "X", Request: Request{ClosePR: true}, OpenOnly: true},
	}
}

func (m *Model) renderContextMenu(prIsOpen bool) string {
	items := prContextMenuItems()

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
	zoneIdx := 0
	for _, item := range items {
		i := zoneIdx
		zoneIdx++
		if item.OpenOnly && !prIsOpen {
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
		row := render.Mark(m.zoneManager, mouse.ZonePRCtxMenuItem(i), label+key)
		rows = append(rows, row)
	}

	header := ""
	if m.contextMenu != nil && m.repository != nil {
		pi := m.contextMenu.PRIndex
		if pi >= 0 && pi < len(m.repository.PRs) {
			pr := m.repository.PRs[pi]
			title := pr.Title
			if len(title) > 40 {
				title = render.TruncateEllipsis(title, 37)
			}
			header = lipgloss.NewStyle().
				Foreground(styles.ColorSecondary).
				Bold(true).
				Render(fmt.Sprintf("#%d %s", pr.Number, title))
		}
	}

	content := header + "\n" + strings.Join(rows, "\n")
	return menuBorder.Render(content)
}

func (m *Model) handleLongPress(msg tea.MouseMsg) tea.Cmd {
	if m.contextMenu != nil && (msg.Action == tea.MouseActionMotion || msg.Action == tea.MouseActionPress) {
		items := prContextMenuItems()
		hit := -1
		for i := range items {
			z := m.zoneManager.Get(mouse.ZonePRCtxMenuItem(i))
			if z != nil && z.InBounds(msg) {
				hit = i
				break
			}
		}
		m.contextMenu.HoverItem = hit
	}

	itemCount := 0
	if m.repository != nil {
		itemCount = len(m.repository.PRs)
	}
	return m.ArmLongPress(m.zoneManager, msg, listnav.LongPressConfig{
		MenuOpen:  m.contextMenu != nil,
		ItemCount: itemCount,
		RowZoneID: mouse.ZonePR,
		Threshold: longPressThreshold,
		MakeTick:  func(pressID int) tea.Msg { return LongPressTickMsg{PressID: pressID} },
	})
}
