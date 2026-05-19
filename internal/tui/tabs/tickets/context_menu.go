package tickets

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

// ContextMenuState holds the state of the ticket-row long-press context menu.
type ContextMenuState struct {
	TicketIndex int
	MouseX      int
	MouseY      int
	PressID     int
	HoverItem   int // -1 = none
}

// StatusSubmenuState holds the state of the cascading status-transition submenu.
type StatusSubmenuState struct {
	MouseX    int
	MouseY    int
	HoverItem int // -1 = none
}

// LongPressTickMsg fires after the long-press threshold to show the ticket context menu.
type LongPressTickMsg struct {
	PressID int
}

type ticketContextMenuItem struct {
	Label          string
	Key            string
	Request        Request
	RequireCreate  bool // only shown when canCreateTicket
	IsCascade      bool // true for "Change Status >" which opens a submenu instead of firing a request
}

func ticketContextMenuItems() []ticketContextMenuItem {
	return []ticketContextMenuItem{
		{Label: "Create Branch", Key: "Enter", Request: Request{StartBookmarkFromTicket: true}},
		{Label: "Open in Browser", Key: "o", Request: Request{OpenInBrowser: true}},
		{Label: "Change Status >", Key: "c", IsCascade: true},
		{Label: "New Ticket", Key: "n", Request: Request{StartCreateTicket: true}, RequireCreate: true},
	}
}

func (m *Model) renderContextMenu() string {
	items := ticketContextMenuItems()

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
		if item.RequireCreate && !m.canCreateTicket {
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
		row := mark(m.zoneManager, mouse.ZoneTicketCtxMenuItem(i), label+key)
		rows = append(rows, row)
	}

	header := ""
	if m.contextMenu != nil {
		ti := m.contextMenu.TicketIndex
		if ti >= 0 && ti < len(m.ticketList) {
			ticket := m.ticketList[ti]
			displayKey := ticket.DisplayKey
			if displayKey == "" {
				displayKey = ticket.Key
			}
			title := displayKey + " " + ticket.Summary
			if len(title) > 40 {
				title = title[:37] + "..."
			}
			header = lipgloss.NewStyle().
				Foreground(styles.ColorSecondary).
				Bold(true).
				Render(title)
		}
	}

	content := header + "\n" + strings.Join(rows, "\n")
	return menuBorder.Render(content)
}

func (m *Model) handleLongPress(msg tea.MouseMsg) tea.Cmd {
	// Track hover over status submenu zones.
	if m.statusSubmenu != nil && (msg.Action == tea.MouseActionMotion || msg.Action == tea.MouseActionPress) {
		hit := -1
		for i := range m.availableTransitions {
			zoneID := mouse.ZoneJiraTransition + fmt.Sprintf("%d", i)
			z := m.zoneManager.Get(zoneID)
			if z != nil && z.InBounds(msg) {
				hit = i
				break
			}
		}
		m.statusSubmenu.HoverItem = hit
	}

	// Track hover over context menu zones.
	if m.contextMenu != nil && (msg.Action == tea.MouseActionMotion || msg.Action == tea.MouseActionPress) {
		items := ticketContextMenuItems()
		hit := -1
		for i := range items {
			z := m.zoneManager.Get(mouse.ZoneTicketCtxMenuItem(i))
			if z != nil && z.InBounds(msg) {
				hit = i
				break
			}
		}
		m.contextMenu.HoverItem = hit
	}

	switch msg.Action {
	case tea.MouseActionMotion:
		// Stay armed while the cursor remains over the ticket row or within
		// the small slack box around the anchor. Doesn't apply once the
		// context menu or status submenu is already shown — those have
		// their own hover-tracking branches above.
		if m.contextMenu == nil && m.statusSubmenu == nil && m.longPressItemIndex >= 0 {
			origin := mouse.ZoneJiraTicket(m.longPressItemIndex)
			if !longpress.StillArmed(m.zoneManager, origin, m.longPressMouseX, m.longPressMouseY, msg) {
				m.longPressItemIndex = -1
			}
		}

	case tea.MouseActionPress:
		if msg.Button != tea.MouseButtonLeft {
			return nil
		}
		if m.contextMenu != nil || m.statusSubmenu != nil {
			return nil
		}
		for i := range m.ticketList {
			z := m.zoneManager.Get(mouse.ZoneJiraTicket(i))
			if z != nil && z.InBounds(msg) {
				m.longPressPressID++
				m.longPressItemIndex = i
				m.longPressMouseX = msg.X
				m.longPressMouseY = msg.Y
				pressID := m.longPressPressID
				return tea.Tick(longPressThreshold, func(time.Time) tea.Msg {
					return LongPressTickMsg{PressID: pressID}
				})
			}
		}

	case tea.MouseActionRelease:
		m.longPressItemIndex = -1
	}
	return nil
}
