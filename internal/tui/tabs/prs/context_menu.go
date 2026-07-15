package prs

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/listnav"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/render"
)

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
	renderItems := make([]render.ContextMenuItem, len(items))
	for i, item := range items {
		renderItems[i] = render.ContextMenuItem{
			Label:    item.Label,
			Key:      item.Key,
			Disabled: item.OpenOnly && !prIsOpen,
		}
	}

	hoverIdx := -1
	if m.contextMenu != nil {
		hoverIdx = m.contextMenu.HoverItem
	}

	header := ""
	if m.contextMenu != nil && m.repository != nil {
		pi := m.contextMenu.PRIndex
		if pi >= 0 && pi < len(m.repository.PRs) {
			pr := m.repository.PRs[pi]
			header = fmt.Sprintf("#%d %s", pr.Number, render.TruncateMenuHeader(pr.Title))
		}
	}

	return render.ContextMenu(m.zoneManager, renderItems, hoverIdx, header, mouse.ZonePRCtxMenuItem)
}

func (m *Model) handleLongPress(msg tea.MouseMsg) tea.Cmd {
	if m.contextMenu != nil && (msg.Action == tea.MouseActionMotion || msg.Action == tea.MouseActionPress) {
		m.contextMenu.HoverItem = listnav.HoverHitTest(m.zoneManager, msg, mouse.ZonePRCtxMenuItem, len(prContextMenuItems()))
	}

	itemCount := 0
	if m.repository != nil {
		itemCount = len(m.repository.PRs)
	}
	return m.ArmLongPress(m.zoneManager, msg, listnav.LongPressConfig{
		MenuOpen:  m.contextMenu != nil,
		ItemCount: itemCount,
		RowZoneID: mouse.ZonePR,
		Threshold: listnav.LongPressThreshold,
		MakeTick:  func(pressID int) tea.Msg { return LongPressTickMsg{PressID: pressID} },
	})
}
