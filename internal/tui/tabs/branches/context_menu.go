package branches

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tui/listnav"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/render"
)

// ContextMenuState holds the state of the branch-row long-press context menu.
type ContextMenuState struct {
	BranchIndex int
	MouseX      int
	MouseY      int
	PressID     int
	HoverItem   int // -1 = none
}

// LongPressTickMsg fires after the long-press threshold to show the branch context menu.
type LongPressTickMsg struct {
	PressID int
}

type branchContextMenuItem struct {
	Label   string
	Key     string
	Request Request
}

// branchContextMenuItems returns the applicable menu items for the given branch.
func branchContextMenuItems(branch internal.Branch) []branchContextMenuItem {
	var items []branchContextMenuItem

	if branch.IsLocal {
		items = append(items,
			branchContextMenuItem{Label: "Push", Key: "P", Request: Request{PushBranch: true}},
			branchContextMenuItem{Label: "Delete", Key: "x", Request: Request{DeleteBranchBookmark: true}},
		)
		if branch.HasConflict {
			items = append(items,
				branchContextMenuItem{Label: "Resolve Conflict", Key: "c", Request: Request{ResolveBookmarkConflict: true}},
			)
		}
	} else if branch.IsTracked {
		items = append(items,
			branchContextMenuItem{Label: "Untrack", Key: "U", Request: Request{UntrackBranch: true}},
		)
		if branch.LocalDeleted {
			items = append(items,
				branchContextMenuItem{Label: "Restore Local", Key: "L", Request: Request{RestoreLocalBranch: true}},
			)
		}
	} else {
		items = append(items,
			branchContextMenuItem{Label: "Track", Key: "T", Request: Request{TrackBranch: true}},
		)
	}

	items = append(items,
		branchContextMenuItem{Label: "Fetch All", Key: "F", Request: Request{FetchAll: true}},
	)

	return items
}

func (m *Model) renderContextMenu() string {
	if m.contextMenu == nil {
		return ""
	}

	bi := m.contextMenu.BranchIndex
	if bi < 0 || bi >= len(m.branchList) {
		return ""
	}
	branch := m.branchList[bi]
	items := branchContextMenuItems(branch)

	renderItems := make([]render.ContextMenuItem, len(items))
	for i, item := range items {
		renderItems[i] = render.ContextMenuItem{Label: item.Label, Key: item.Key}
	}
	return render.ContextMenu(m.zoneManager, renderItems, m.contextMenu.HoverItem,
		render.TruncateMenuHeader(branch.Name), mouse.ZoneBranchCtxMenuItem)
}

func (m *Model) handleLongPress(msg tea.MouseMsg) tea.Cmd {
	if m.contextMenu != nil && (msg.Action == tea.MouseActionMotion || msg.Action == tea.MouseActionPress) {
		if m.contextMenu.BranchIndex >= 0 && m.contextMenu.BranchIndex < len(m.branchList) {
			n := len(branchContextMenuItems(m.branchList[m.contextMenu.BranchIndex]))
			m.contextMenu.HoverItem = listnav.HoverHitTest(m.zoneManager, msg, mouse.ZoneBranchCtxMenuItem, n)
		}
	}

	return m.ArmLongPress(m.zoneManager, msg, listnav.LongPressConfig{
		MenuOpen:  m.contextMenu != nil,
		ItemCount: len(m.branchList),
		RowZoneID: mouse.ZoneBranch,
		Threshold: listnav.LongPressThreshold,
		MakeTick:  func(pressID int) tea.Msg { return LongPressTickMsg{PressID: pressID} },
	})
}
