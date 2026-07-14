package render

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

// ContextMenuItem is one row in a shared list-tab context menu.
type ContextMenuItem struct {
	Label    string
	Key      string
	Disabled bool
}

// TruncateMenuHeader shortens long context-menu titles the same way every list
// tab does: TruncateEllipsis at 37 when the string exceeds 40 bytes.
func TruncateMenuHeader(s string) string {
	if len(s) > 40 {
		return TruncateEllipsis(s, 37)
	}
	return s
}

// ContextMenu renders a bordered popup with header + labeled shortcut rows.
// Disabled rows are shown muted and unmarked; enabled rows use zoneFn(i).
func ContextMenu(zm *zone.Manager, items []ContextMenuItem, hoverIdx int, header string, zoneFn func(i int) string) string {
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

	var rows []string
	for i, item := range items {
		if item.Disabled {
			rows = append(rows, disabledStyle.Render(fmt.Sprintf("  %s  %s", item.Label, item.Key)))
			continue
		}
		ls, ks := itemStyle, keyStyle
		if i == hoverIdx {
			ls, ks = hoverStyle, hoverKeyStyle
		}
		label := ls.Render(fmt.Sprintf("  %s", item.Label))
		key := ks.Render(fmt.Sprintf("  %s", item.Key))
		row := label + key
		if zoneFn != nil {
			row = Mark(zm, zoneFn(i), row)
		}
		rows = append(rows, row)
	}

	if header != "" {
		header = lipgloss.NewStyle().
			Foreground(styles.ColorSecondary).
			Bold(true).
			Render(header)
	}

	content := header + "\n" + strings.Join(rows, "\n")
	return menuBorder.Render(content)
}
