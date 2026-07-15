package tickets

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/listnav"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/render"
)

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
	Label         string
	Key           string
	Request       Request
	RequireCreate bool // only shown when canCreateTicket
	IsCascade     bool // true for "Change Status >" which opens a submenu instead of firing a request
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
	renderItems := make([]render.ContextMenuItem, len(items))
	for i, item := range items {
		renderItems[i] = render.ContextMenuItem{
			Label:    item.Label,
			Key:      item.Key,
			Disabled: item.RequireCreate && !m.canCreateTicket,
		}
	}

	hoverIdx := -1
	if m.contextMenu != nil {
		hoverIdx = m.contextMenu.HoverItem
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
			header = render.TruncateMenuHeader(displayKey + " " + ticket.Summary)
		}
	}

	return render.ContextMenu(m.zoneManager, renderItems, hoverIdx, header, mouse.ZoneTicketCtxMenuItem)
}

func (m *Model) handleLongPress(msg tea.MouseMsg) tea.Cmd {
	if m.statusSubmenu != nil && (msg.Action == tea.MouseActionMotion || msg.Action == tea.MouseActionPress) {
		m.statusSubmenu.HoverItem = listnav.HoverHitTest(m.zoneManager, msg, func(i int) string {
			return mouse.ZoneJiraTransition + fmt.Sprintf("%d", i)
		}, len(m.availableTransitions))
	}

	if m.contextMenu != nil && (msg.Action == tea.MouseActionMotion || msg.Action == tea.MouseActionPress) {
		m.contextMenu.HoverItem = listnav.HoverHitTest(m.zoneManager, msg, mouse.ZoneTicketCtxMenuItem, len(ticketContextMenuItems()))
	}

	// Stay armed while the cursor remains over the ticket row or within the
	// small slack box around the anchor. MenuOpen suppresses that (and a new
	// press) once the context menu or status submenu is already shown — those
	// have their own hover-tracking branches above.
	return m.ArmLongPress(m.zoneManager, msg, listnav.LongPressConfig{
		MenuOpen:  m.contextMenu != nil || m.statusSubmenu != nil,
		ItemCount: len(m.ticketList),
		RowZoneID: mouse.ZoneJiraTicket,
		Threshold: listnav.LongPressThreshold,
		MakeTick:  func(pressID int) tea.Msg { return LongPressTickMsg{PressID: pressID} },
	})
}
