package tickets

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	overlay "github.com/madicen/bubble-overlay"
	"github.com/madicen/jj-tui/internal/tickets"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

// transitionShortcutAndStyle maps a transition name to keyboard hint and button styling (same rules as former inline bar).
func transitionShortcutAndStyle(t tickets.Transition) (shortcut string, btnStyle lipgloss.Style) {
	lowerName := strings.ToLower(t.Name)
	btnStyle = styles.ButtonStyle
	isNotStarted := strings.Contains(lowerName, "not") && strings.Contains(lowerName, "start")
	isInProgress := strings.Contains(lowerName, "progress") ||
		(strings.Contains(lowerName, "start") && !strings.Contains(lowerName, "not start") && !strings.Contains(lowerName, "not_start"))

	switch {
	case isNotStarted:
		shortcut = " (N)"
		btnStyle = lipgloss.NewStyle().Background(lipgloss.Color("#6272A4")).Foreground(lipgloss.Color("#FFFFFF")).Padding(0, 1).Bold(true)
	case isInProgress:
		shortcut = " (i)"
		btnStyle = lipgloss.NewStyle().Background(lipgloss.Color("#FFB86C")).Foreground(lipgloss.Color("#000000")).Padding(0, 1).Bold(true)
	case strings.Contains(lowerName, "done") || strings.Contains(lowerName, "complete") || strings.Contains(lowerName, "resolve"):
		shortcut = " (D)"
		btnStyle = lipgloss.NewStyle().Background(lipgloss.Color("#50FA7B")).Foreground(lipgloss.Color("#000000")).Padding(0, 1).Bold(true)
	case strings.Contains(lowerName, "block"):
		shortcut = " (B)"
		btnStyle = lipgloss.NewStyle().Background(lipgloss.Color("#FF5555")).Foreground(lipgloss.Color("#FFFFFF")).Padding(0, 1).Bold(true)
	}
	return shortcut, btnStyle
}

// renderStatusPopoverPanel returns a bordered panel listing transitions with zone marks.
// hoverIdx enables hover highlighting (-1 means no hover). Used by both the actions-bar
// popover and the cascading submenu from the long-press context menu.
func (m *Model) renderStatusPopoverPanel(hoverIdx int) string {
	hoverStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2")).Background(styles.ColorPrimary)

	var lines []string
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8BE9FD")).Render("Change status")
	lines = append(lines, title)
	lines = append(lines, "")

	if m.loadingTransitions {
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Italic(true).Render("  Loading..."))
	} else if len(m.availableTransitions) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Italic(true).Render("  No transitions"))
	} else {
		for i, t := range m.availableTransitions {
			shortcut, btnStyle := transitionShortcutAndStyle(t)
			var label string
			if i == hoverIdx {
				label = hoverStyle.Render(fmt.Sprintf("  %s%s  ", t.Name, shortcut))
			} else {
				label = " " + btnStyle.Render(t.Name+shortcut)
			}
			zoneID := mouse.ZoneJiraTransition + fmt.Sprintf("%d", i)
			lines = append(lines, mark(m.zoneManager, zoneID, label))
			if i < len(m.availableTransitions)-1 {
				lines = append(lines, "")
			}
		}
	}

	lines = append(lines, "")
	closeLabel := lipgloss.NewStyle().Foreground(styles.ColorMuted).Italic(true).Render(" ✕ Close ")
	lines = append(lines, mark(m.zoneManager, mouse.ZoneStatusPopoverClose, closeLabel))
	inner := strings.Join(lines, "\n")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorMuted).
		Padding(0, 1).
		Render(inner)
}

// overlayStatusPopover composites the popover to the right of the Change Status button on the actions row.
// anchorLeft is the terminal column for the popover's left edge (already includes gap after the button).
func overlayStatusPopover(baseView string, popover string, termW, termH, actionsRowIndex, anchorLeft int) string {
	if popover == "" || termW <= 0 || termH <= 0 {
		return baseView
	}
	return overlay.OverlayViewAtPoint(baseView, popover, termW, termH, actionsRowIndex, anchorLeft)
}
