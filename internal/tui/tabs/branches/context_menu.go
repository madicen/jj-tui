package branches

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

const longPressThreshold = 500 * time.Millisecond

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
	keyStyle := lipgloss.NewStyle().
		Foreground(styles.ColorMuted)

	hoverIdx := m.contextMenu.HoverItem

	var rows []string
	for i, item := range items {
		isHovered := i == hoverIdx
		ls := itemStyle
		ks := keyStyle
		if isHovered {
			ls = hoverStyle
			ks = hoverKeyStyle
		}
		label := ls.Render(fmt.Sprintf("  %s", item.Label))
		key := ks.Render(fmt.Sprintf("  %s", item.Key))
		row := mark(m.zoneManager, mouse.ZoneBranchCtxMenuItem(i), label+key)
		rows = append(rows, row)
	}

	branchName := branch.Name
	if len(branchName) > 40 {
		branchName = branchName[:37] + "..."
	}
	header := lipgloss.NewStyle().
		Foreground(styles.ColorSecondary).
		Bold(true).
		Render(branchName)

	content := header + "\n" + strings.Join(rows, "\n")
	return menuBorder.Render(content)
}

func (m *Model) handleLongPress(msg tea.MouseMsg) tea.Cmd {
	if m.contextMenu != nil && (msg.Action == tea.MouseActionMotion || msg.Action == tea.MouseActionPress) {
		if m.contextMenu.BranchIndex >= 0 && m.contextMenu.BranchIndex < len(m.branchList) {
			branch := m.branchList[m.contextMenu.BranchIndex]
			items := branchContextMenuItems(branch)
			hit := -1
			for i := range items {
				z := m.zoneManager.Get(mouse.ZoneBranchCtxMenuItem(i))
				if z != nil && z.InBounds(msg) {
					hit = i
					break
				}
			}
			m.contextMenu.HoverItem = hit
		}
	}

	switch msg.Action {
	case tea.MouseActionMotion:
		if m.contextMenu == nil && m.longPressItemIndex >= 0 {
			m.longPressItemIndex = -1
		}

	case tea.MouseActionPress:
		if msg.Button != tea.MouseButtonLeft {
			return nil
		}
		if m.contextMenu != nil {
			return nil
		}
		for i := range m.branchList {
			z := m.zoneManager.Get(mouse.ZoneBranch(i))
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
		if m.contextMenu != nil {
			bi := m.contextMenu.BranchIndex
			if bi >= 0 && bi < len(m.branchList) {
				items := branchContextMenuItems(m.branchList[bi])
				for i := range items {
					z := m.zoneManager.Get(mouse.ZoneBranchCtxMenuItem(i))
					if z != nil && z.InBounds(msg) {
						return nil
					}
				}
			}
			m.contextMenu = nil
		}
	}
	return nil
}
