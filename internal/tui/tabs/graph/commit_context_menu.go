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

// CommitContextMenuState holds the state of the commit-row long-press context menu.
type CommitContextMenuState struct {
	CommitIndex int
	MouseX      int
	MouseY      int
	PressID     int
	HoverItem   int // -1 = none
}

// CommitLongPressTickMsg fires after the long-press threshold to show the commit context menu.
type CommitLongPressTickMsg struct {
	PressID int
}

type commitContextMenuItem struct {
	Label   string
	Key     string
	Request Request
	Mutable bool
	// HideWhenFirstParentImmutable hides this item when the first parent commit is immutable.
	HideWhenFirstParentImmutable bool
}

func commitContextMenuItems() []commitContextMenuItem {
	return []commitContextMenuItem{
		{Label: "New", Key: "n", Request: Request{NewCommit: true}},
		{Label: "Edit", Key: "e", Request: Request{Checkout: true}, Mutable: true},
		{Label: "Describe", Key: "d", Request: Request{StartEditDescription: true}, Mutable: true},
		{Label: "Squash", Key: "s", Request: Request{Squash: true}, Mutable: true, HideWhenFirstParentImmutable: true},
		{Label: "Rebase", Key: "r", Request: Request{StartRebaseMode: true}, Mutable: true},
		{Label: "Merge from", Key: "M", Request: Request{StartMergeMode: true}, Mutable: true},
		{Label: "Abandon", Key: "a", Request: Request{Abandon: true}, Mutable: true},
		{Label: "Bookmark", Key: "m", Request: Request{CreateBookmark: true}, Mutable: true},
	}
}

// commitContextMenuRows is the full menu for a commit row: base actions plus Update PR / Create PR when
// the same GraphData rules as the actions bar apply (see view_helpers Graph).
func (m *GraphModel) commitContextMenuRows(ci int, firstParentImmutable bool) []commitContextMenuItem {
	var out []commitContextMenuItem
	for _, item := range commitContextMenuItems() {
		if item.HideWhenFirstParentImmutable && firstParentImmutable {
			continue
		}
		out = append(out, item)
	}
	if m.repository == nil || ci < 0 || ci >= len(m.repository.Graph.Commits) {
		return out
	}
	if m.repository.Graph.Commits[ci].Immutable {
		return out
	}
	data := m.buildGraphData()
	prBranch := ""
	if data.CommitPRBranch != nil {
		prBranch = data.CommitPRBranch[ci]
	}
	if prBranch != "" {
		label := "Update PR"
		if len(m.repository.Graph.Commits[ci].Branches) == 0 {
			label = fmt.Sprintf("Update PR [%s]", prBranch)
		}
		out = append(out, commitContextMenuItem{Label: label, Key: "u", Request: Request{UpdatePR: true}})
	}
	createPRBranch := ""
	if data.CommitBookmark != nil {
		createPRBranch = data.CommitBookmark[ci]
	}
	c := m.repository.Graph.Commits[ci]
	if createPRBranch != "" && !isDefaultBranch(createPRBranch) && len(c.ConflictedBranches) == 0 {
		label := "Create PR"
		if len(c.Branches) == 0 || prBranch != "" {
			label = fmt.Sprintf("Create PR [%s]", createPRBranch)
		}
		out = append(out, commitContextMenuItem{Label: label, Key: "c", Request: Request{CreatePR: true}})
	}
	return out
}

func (m *GraphModel) renderCommitContextMenu(isMutable bool, firstParentImmutable bool) string {
	if m.commitContextMenu == nil {
		return ""
	}
	items := m.commitContextMenuRows(m.commitContextMenu.CommitIndex, firstParentImmutable)

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
	if m.commitContextMenu != nil {
		hoverIdx = m.commitContextMenu.HoverItem
	}

	var rows []string
	zoneIdx := 0
	for _, item := range items {
		if item.HideWhenFirstParentImmutable && firstParentImmutable {
			continue
		}
		i := zoneIdx
		zoneIdx++
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
		row := m.zoneManager.Mark(mouse.ZoneCommitCtxMenuItem(i), label+key)
		rows = append(rows, row)
	}

	header := ""
	if m.commitContextMenu != nil && m.repository != nil {
		ci := m.commitContextMenu.CommitIndex
		if ci >= 0 && ci < len(m.repository.Graph.Commits) {
			desc := m.repository.Graph.Commits[ci].Description
			if len(desc) > 40 {
				desc = desc[:37] + "..."
			}
			header = lipgloss.NewStyle().
				Foreground(styles.ColorSecondary).
				Bold(true).
				Render(desc)
		}
	}

	content := header + "\n" + strings.Join(rows, "\n")
	return menuBorder.Render(content)
}

// handleCommitLongPress detects press/release on commit zones for the long-press
// context menu. Returns a tea.Cmd (the tick) on press, nil otherwise.
func (m *GraphModel) handleCommitLongPress(msg tea.MouseMsg) tea.Cmd {
	if m.commitContextMenu != nil && (msg.Action == tea.MouseActionMotion || msg.Action == tea.MouseActionPress) {
		items := m.commitContextMenuRows(m.commitContextMenu.CommitIndex, m.commitMenuFirstParentImmutable())
		hit := -1
		for i := range items {
			z := m.zoneManager.Get(mouse.ZoneCommitCtxMenuItem(i))
			if z != nil && z.InBounds(msg) {
				hit = i
				break
			}
		}
		m.commitContextMenu.HoverItem = hit
	}

	switch msg.Action {
	case tea.MouseActionMotion:
		// Tolerate small drift while the press is armed; the zone match is
		// authoritative when bubblezone has laid out, slack box is the fallback.
		if m.commitContextMenu == nil && m.longPressCommitIndex >= 0 {
			origin := mouse.ZoneCommit(m.longPressCommitIndex)
			if !longpress.StillArmed(m.zoneManager, origin, m.longPressCommitMouseX, m.longPressCommitMouseY, msg) {
				m.longPressCommitIndex = -1
			}
		}

	case tea.MouseActionPress:
		if msg.Button != tea.MouseButtonLeft {
			return nil
		}
		if m.commitContextMenu != nil {
			return nil
		}
		if m.repository == nil {
			return nil
		}
		for i := range m.repository.Graph.Commits {
			z := m.zoneManager.Get(mouse.ZoneCommit(i))
			if z != nil && z.InBounds(msg) {
				m.longPressCommitPressID++
				m.longPressCommitIndex = i
				m.longPressCommitMouseX = msg.X
				m.longPressCommitMouseY = msg.Y
				pressID := m.longPressCommitPressID
				return tea.Tick(longPressThreshold, func(time.Time) tea.Msg {
					return CommitLongPressTickMsg{PressID: pressID}
				})
			}
		}

	case tea.MouseActionRelease:
		m.longPressCommitIndex = -1
	}
	return nil
}

func (m *GraphModel) commitMenuIsMutable() bool {
	if m.commitContextMenu == nil || m.repository == nil {
		return false
	}
	ci := m.commitContextMenu.CommitIndex
	if ci < 0 || ci >= len(m.repository.Graph.Commits) {
		return false
	}
	return !m.repository.Graph.Commits[ci].Immutable
}

func (m *GraphModel) commitMenuFirstParentImmutable() bool {
	if m.commitContextMenu == nil || m.repository == nil {
		return true
	}
	return isFirstParentImmutable(m.repository.Graph.Commits, m.commitContextMenu.CommitIndex)
}
