package help

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

var (
	helpTabStyle     = lipgloss.NewStyle().Padding(0, 2).Foreground(lipgloss.Color("#888888"))
	helpTabActiveStyle = lipgloss.NewStyle().Padding(0, 2).Foreground(styles.ColorPrimary).Bold(true).Underline(true)
)

func mark(z *zone.Manager, id, content string) string {
	if z == nil {
		return content
	}
	return z.Mark(id, content)
}

func (m Model) renderHelp() string {
	var fullLines []string
	fullLines = append(fullLines, "")
	fullLines = append(fullLines, m.renderHelpTabs())
	fullLines = append(fullLines, "")
	if m.helpTab == 0 {
		fullLines = append(fullLines, m.renderHelpShortcuts()...)
	} else {
		fullLines = append(fullLines, m.renderHelpCommands()...)
	}
	totalLines := len(fullLines)
	visibleHeight := max(1, m.height-3)
	var start int
	if m.helpTab == 0 {
		start = m.shortcutsYOffset
	} else {
		start = m.historyYOffset
	}
	if start > totalLines-visibleHeight {
		start = max(0, totalLines-visibleHeight)
	}
	if start < 0 {
		start = 0
	}
	end := min(start+visibleHeight, totalLines)
	return strings.Join(fullLines[start:end], "\n")
}

func (m Model) renderHelpTabs() string {
	shortcutsStyle := helpTabStyle
	commandsStyle := helpTabStyle
	if m.helpTab == 0 {
		shortcutsStyle = helpTabActiveStyle
	} else {
		commandsStyle = helpTabActiveStyle
	}
	shortcutsTab := mark(m.zoneManager, mouse.ZoneHelpTabShortcuts, shortcutsStyle.Render("Shortcuts"))
	commandsTab := mark(m.zoneManager, mouse.ZoneHelpTabCommands, commandsStyle.Render("History"))
	return lipgloss.JoinHorizontal(lipgloss.Left, shortcutsTab, " │ ", commandsTab)
}

func (m Model) renderHelpShortcuts() []string {
	var lines []string
	lines = append(lines, styles.TitleStyle.Render("Commit Graph Shortcuts"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("j/↓"), styles.HelpDescStyle.Render("Move down")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("k/↑"), styles.HelpDescStyle.Render("Move up")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("Tab"), styles.HelpDescStyle.Render("Switch focus: graph ↔ files")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("Enter/e"), styles.HelpDescStyle.Render("Edit selected commit (jj edit)")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("d"), styles.HelpDescStyle.Render("Edit commit description")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("s"), styles.HelpDescStyle.Render("Squash commit into parent")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("r"), styles.HelpDescStyle.Render("Rebase commit (with descendants)")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("a"), styles.HelpDescStyle.Render("Abandon commit (or resolve divergent)")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("n"), styles.HelpDescStyle.Render("Create new commit from selected")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("b"), styles.HelpDescStyle.Render("Create/move bookmark on commit")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("x"), styles.HelpDescStyle.Render("Delete bookmark from commit")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("c"), styles.HelpDescStyle.Render("Create new PR from commit chain")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("u"), styles.HelpDescStyle.Render("Update existing PR with new commits")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("^z"), styles.HelpDescStyle.Render("Undo last jj operation")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("^y"), styles.HelpDescStyle.Render("Redo jj operation")))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Bookmark Screen"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("j/↓"), styles.HelpDescStyle.Render("Select next existing bookmark")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("k/↑"), styles.HelpDescStyle.Render("Select previous / new input")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("Tab"), styles.HelpDescStyle.Render("Toggle new/existing bookmark")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("Enter"), styles.HelpDescStyle.Render("Create new or move selected")))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Pull Request Shortcuts"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("j/↓"), styles.HelpDescStyle.Render("Move down")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("k/↑"), styles.HelpDescStyle.Render("Move up")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("Enter/o"), styles.HelpDescStyle.Render("Open PR in browser")))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Tickets Shortcuts"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("j/↓"), styles.HelpDescStyle.Render("Move down")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("k/↑"), styles.HelpDescStyle.Render("Move up")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("Enter"), styles.HelpDescStyle.Render("Create branch from ticket")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("o"), styles.HelpDescStyle.Render("Open ticket in browser")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("c"), styles.HelpDescStyle.Render("Change ticket status")))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Branches Shortcuts"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("j/↓"), styles.HelpDescStyle.Render("Move down")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("k/↑"), styles.HelpDescStyle.Render("Move up")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("T"), styles.HelpDescStyle.Render("Track remote branch")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("U"), styles.HelpDescStyle.Render("Untrack remote branch")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("L"), styles.HelpDescStyle.Render("Restore deleted local branch")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("x"), styles.HelpDescStyle.Render("Delete local bookmark")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("P"), styles.HelpDescStyle.Render("Push local branch to remote")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("F"), styles.HelpDescStyle.Render("Fetch from all remotes")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("c"), styles.HelpDescStyle.Render("Resolve conflicted bookmark")))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Settings Shortcuts"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("^j"), styles.HelpDescStyle.Render("Previous settings tab")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("^k"), styles.HelpDescStyle.Render("Next settings tab")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("Tab"), styles.HelpDescStyle.Render("Next input field")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("^s"), styles.HelpDescStyle.Render("Save settings (global)")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("^l"), styles.HelpDescStyle.Render("Save settings (local to repo)")))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Navigation"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("g"), styles.HelpDescStyle.Render("Go to commit graph")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("p"), styles.HelpDescStyle.Render("Go to pull requests")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("t"), styles.HelpDescStyle.Render("Go to Tickets")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("R"), styles.HelpDescStyle.Render("Go to Branches")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render(","), styles.HelpDescStyle.Render("Open settings")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("h/?"), styles.HelpDescStyle.Render("Show this help")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("^r"), styles.HelpDescStyle.Render("Refresh")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("Esc"), styles.HelpDescStyle.Render("Back to graph")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("^q"), styles.HelpDescStyle.Render("Quit")))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Graph Symbols"))
	lines = append(lines, "")
	lines = append(lines, "  @  Working copy (current editing state)")
	lines = append(lines, "  ○  Mutable commit (can be edited)")
	lines = append(lines, "  ◆  Immutable commit (pushed to remote)")
	lines = append(lines, "  ⚠  Commit has conflicts")
	lines = append(lines, "  ⑂  Divergent commit (same change ID in multiple versions)")
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("PR Status Symbols"))
	lines = append(lines, "")
	lines = append(lines, "  State:")
	lines = append(lines, "    ● green   Open PR")
	lines = append(lines, "    ● red     Closed PR")
	lines = append(lines, "    ● purple  Merged PR")
	lines = append(lines, "")
	lines = append(lines, "  CI Checks:")
	lines = append(lines, "    ✓  All checks passed")
	lines = append(lines, "    ✗  Checks failed")
	lines = append(lines, "    ○  Checks pending/running")
	lines = append(lines, "    ·  No checks configured")
	lines = append(lines, "")
	lines = append(lines, "  Reviews:")
	lines = append(lines, "    👍  Approved")
	lines = append(lines, "    📝  Changes requested")
	lines = append(lines, "    ⏳  Review pending")
	lines = append(lines, "    ·   No reviews yet")
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Scrolling"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("PgUp/PgDn"), styles.HelpDescStyle.Render("Scroll page up/down")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("^u/^d"), styles.HelpDescStyle.Render("Scroll half page up/down")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("Home/End"), styles.HelpDescStyle.Render("Scroll to top/bottom")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(10).Render("Mouse"), styles.HelpDescStyle.Render("Use scroll wheel to scroll")))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Mouse"))
	lines = append(lines, "")
	lines = append(lines, "  Click on tabs, commits, PRs, tickets, or buttons")
	lines = append(lines, "  Click graph/files panes to switch focus")
	lines = append(lines, "  Click footer shortcuts (undo, redo, refresh, etc.)")
	return lines
}

func (m Model) renderHelpCommands() []string {
	var lines []string
	lines = append(lines, styles.TitleStyle.Render("Command History"))
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("  Commands executed by jj-tui (excluding auto-refresh)"))
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("  Click [copy] or press y to copy command to clipboard"))
	lines = append(lines, "")

	if len(m.commandHistoryEntries) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Italic(true).Render("  No commands executed yet"))
		return lines
	}

	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B"))
	failStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))
	timeStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted).Width(8)
	durationStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted).Width(5)
	cmdStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))
	copyBtnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C")).Bold(true)

	maxCommands := len(m.commandHistoryEntries)
	if maxCommands > 50 {
		maxCommands = 50
	}

	for i := 0; i < maxCommands; i++ {
		entry := m.commandHistoryEntries[i]
		var statusIcon string
		var entryStyle lipgloss.Style
		if entry.Success {
			statusIcon = successStyle.Render("✓")
			entryStyle = lipgloss.NewStyle()
		} else {
			statusIcon = failStyle.Render("✗")
			entryStyle = failStyle
		}
		prefix := "  "
		if i == m.helpSelectedCommand {
			prefix = "> "
			entryStyle = entryStyle.Bold(true).Background(lipgloss.Color("#44475A"))
		}
		copyBtn := mark(m.zoneManager, fmt.Sprintf("%s%d", mouse.ZoneHelpCommandCopy, i), copyBtnStyle.Render("[copy]"))
		line := fmt.Sprintf("%s%s %s %s %s %s",
			prefix,
			timeStyle.Render(entry.Timestamp),
			durationStyle.Render(entry.Duration),
			statusIcon,
			cmdStyle.Render(entry.Command),
			copyBtn,
		)
		if i == m.helpSelectedCommand {
			line = entryStyle.Render(line)
		}
		lines = append(lines, mark(m.zoneManager, fmt.Sprintf("%s%d", mouse.ZoneHelpCommand, i), line))
		if !entry.Success && entry.Error != "" && i == m.helpSelectedCommand {
			lines = append(lines, fmt.Sprintf("    %s", failStyle.Render("Error: "+entry.Error)))
		}
	}

	if len(m.commandHistoryEntries) > maxCommands {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(
			fmt.Sprintf("  ... and %d more commands", len(m.commandHistoryEntries)-maxCommands)))
	}
	return lines
}
