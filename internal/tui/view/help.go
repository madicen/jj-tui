package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Tab styles for help view
var (
	helpTabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(lipgloss.Color("#888888"))

	helpTabActiveStyle = lipgloss.NewStyle().
				Padding(0, 2).
				Foreground(ColorPrimary).
				Bold(true).
				Underline(true)
)

// Help renders the help view with sub-tabs
func (r *Renderer) Help(data HelpData) string {
	var lines []string

	lines = append(lines, "")

	// Render sub-tabs
	tabs := r.renderHelpTabs(data.ActiveTab)
	lines = append(lines, tabs)
	lines = append(lines, "")

	// Render content based on active tab
	switch data.ActiveTab {
	case HelpTabShortcuts:
		lines = append(lines, r.renderHelpShortcuts()...)
	case HelpTabCommands:
		lines = append(lines, r.renderHelpCommands(data)...)
	}

	return strings.Join(lines, "\n")
}

// renderHelpTabs renders the tab bar for help view
func (r *Renderer) renderHelpTabs(activeTab HelpTab) string {
	shortcutsStyle := helpTabStyle
	commandsStyle := helpTabStyle

	switch activeTab {
	case HelpTabShortcuts:
		shortcutsStyle = helpTabActiveStyle
	case HelpTabCommands:
		commandsStyle = helpTabActiveStyle
	}

	shortcutsTab := r.Mark(ZoneHelpTabShortcuts, shortcutsStyle.Render("Shortcuts"))
	commandsTab := r.Mark(ZoneHelpTabCommands, commandsStyle.Render("History"))

	return lipgloss.JoinHorizontal(lipgloss.Left, shortcutsTab, " │ ", commandsTab)
}

// renderHelpShortcuts renders the keyboard shortcuts content
func (r *Renderer) renderHelpShortcuts() []string {
	var lines []string

	lines = append(lines, TitleStyle.Render("Commit Graph Shortcuts"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("j/↓"), HelpDescStyle.Render("Move down")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("k/↑"), HelpDescStyle.Render("Move up")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("Tab"), HelpDescStyle.Render("Switch focus: graph ↔ files")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("Enter/e"), HelpDescStyle.Render("Edit selected commit (jj edit)")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("d"), HelpDescStyle.Render("Edit commit description")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("s"), HelpDescStyle.Render("Squash commit into parent")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("r"), HelpDescStyle.Render("Rebase commit (with descendants)")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("a"), HelpDescStyle.Render("Abandon commit (or resolve divergent)")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("n"), HelpDescStyle.Render("Create new commit from selected")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("b"), HelpDescStyle.Render("Create/move bookmark on commit")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("x"), HelpDescStyle.Render("Delete bookmark from commit")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("c"), HelpDescStyle.Render("Create new PR from commit chain")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("u"), HelpDescStyle.Render("Update existing PR with new commits")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("^z"), HelpDescStyle.Render("Undo last jj operation")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("^y"), HelpDescStyle.Render("Redo jj operation")))

	lines = append(lines, "")
	lines = append(lines, TitleStyle.Render("Bookmark Screen"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("j/↓"), HelpDescStyle.Render("Select next existing bookmark")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("k/↑"), HelpDescStyle.Render("Select previous / new input")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("Tab"), HelpDescStyle.Render("Toggle new/existing bookmark")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("Enter"), HelpDescStyle.Render("Create new or move selected")))

	lines = append(lines, "")
	lines = append(lines, TitleStyle.Render("Pull Request Shortcuts"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("j/↓"), HelpDescStyle.Render("Move down")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("k/↑"), HelpDescStyle.Render("Move up")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("Enter/o"), HelpDescStyle.Render("Open PR in browser")))

	lines = append(lines, "")
	lines = append(lines, TitleStyle.Render("Tickets Shortcuts"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("j/↓"), HelpDescStyle.Render("Move down")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("k/↑"), HelpDescStyle.Render("Move up")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("Enter"), HelpDescStyle.Render("Create branch from ticket")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("o"), HelpDescStyle.Render("Open ticket in browser")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("c"), HelpDescStyle.Render("Change ticket status")))

	lines = append(lines, "")
	lines = append(lines, TitleStyle.Render("Branches Shortcuts"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("j/↓"), HelpDescStyle.Render("Move down")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("k/↑"), HelpDescStyle.Render("Move up")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("T"), HelpDescStyle.Render("Track remote branch")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("U"), HelpDescStyle.Render("Untrack remote branch")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("L"), HelpDescStyle.Render("Restore deleted local branch")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("x"), HelpDescStyle.Render("Delete local bookmark")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("P"), HelpDescStyle.Render("Push local branch to remote")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("F"), HelpDescStyle.Render("Fetch from all remotes")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("c"), HelpDescStyle.Render("Resolve conflicted bookmark")))

	lines = append(lines, "")
	lines = append(lines, TitleStyle.Render("Settings Shortcuts"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("^j"), HelpDescStyle.Render("Previous settings tab")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("^k"), HelpDescStyle.Render("Next settings tab")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("Tab"), HelpDescStyle.Render("Next input field")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("^s"), HelpDescStyle.Render("Save settings (global)")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("^l"), HelpDescStyle.Render("Save settings (local to repo)")))

	lines = append(lines, "")
	lines = append(lines, TitleStyle.Render("Navigation"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("g"), HelpDescStyle.Render("Go to commit graph")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("p"), HelpDescStyle.Render("Go to pull requests")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("t"), HelpDescStyle.Render("Go to Tickets")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("R"), HelpDescStyle.Render("Go to Branches")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render(","), HelpDescStyle.Render("Open settings")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("h/?"), HelpDescStyle.Render("Show this help")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("^r"), HelpDescStyle.Render("Refresh")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("Esc"), HelpDescStyle.Render("Back to graph")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("^q"), HelpDescStyle.Render("Quit")))

	lines = append(lines, "")
	lines = append(lines, TitleStyle.Render("Graph Symbols"))
	lines = append(lines, "")
	lines = append(lines, "  @  Working copy (current editing state)")
	lines = append(lines, "  ○  Mutable commit (can be edited)")
	lines = append(lines, "  ◆  Immutable commit (pushed to remote)")
	lines = append(lines, "  ⚠  Commit has conflicts")
	lines = append(lines, "  ⑂  Divergent commit (same change ID in multiple versions)")

	lines = append(lines, "")
	lines = append(lines, TitleStyle.Render("PR Status Symbols"))
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
	lines = append(lines, TitleStyle.Render("Scrolling"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("PgUp/PgDn"), HelpDescStyle.Render("Scroll page up/down")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("^u/^d"), HelpDescStyle.Render("Scroll half page up/down")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("Home/End"), HelpDescStyle.Render("Scroll to top/bottom")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("Mouse"), HelpDescStyle.Render("Use scroll wheel to scroll")))

	lines = append(lines, "")
	lines = append(lines, TitleStyle.Render("Mouse"))
	lines = append(lines, "")
	lines = append(lines, "  Click on tabs, commits, PRs, tickets, or buttons")
	lines = append(lines, "  Click graph/files panes to switch focus")
	lines = append(lines, "  Click footer shortcuts (undo, redo, refresh, etc.)")

	return lines
}

// renderHelpCommands renders the command history content
func (r *Renderer) renderHelpCommands(data HelpData) []string {
	var lines []string

	lines = append(lines, TitleStyle.Render("Command History"))
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("  Commands executed by jj-tui (excluding auto-refresh)"))
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("  Click [copy] or press y to copy command to clipboard"))
	lines = append(lines, "")

	if len(data.CommandHistory) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Italic(true).Render("  No commands executed yet"))
		return lines
	}

	// Style for command entries
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")) // Green
	failStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))    // Red
	timeStyle := lipgloss.NewStyle().Foreground(ColorMuted).Width(8)
	durationStyle := lipgloss.NewStyle().Foreground(ColorMuted).Width(5)
	cmdStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))                // Cyan for command
	copyBtnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C")).Bold(true) // Orange for copy button

	// Show up to 50 most recent commands (filtered list should be smaller)
	maxCommands := min(len(data.CommandHistory), 50)

	for i := range maxCommands {
		entry := data.CommandHistory[i]

		// Build the line
		var statusIcon string
		var entryStyle lipgloss.Style
		if entry.Success {
			statusIcon = successStyle.Render("✓")
			entryStyle = lipgloss.NewStyle()
		} else {
			statusIcon = failStyle.Render("✗")
			entryStyle = failStyle
		}

		// Highlight selected command
		prefix := "  "
		if i == data.SelectedCommand {
			prefix = "> "
			entryStyle = entryStyle.Bold(true).Background(lipgloss.Color("#44475A"))
		}

		// Copy button
		copyBtn := r.Mark(fmt.Sprintf("%s%d", ZoneHelpCommandCopy, i), copyBtnStyle.Render("[copy]"))

		// Format: [time] [duration] [status] command [copy]
		line := fmt.Sprintf("%s%s %s %s %s %s",
			prefix,
			timeStyle.Render(entry.Timestamp),
			durationStyle.Render(entry.Duration),
			statusIcon,
			cmdStyle.Render(entry.Command),
			copyBtn,
		)

		if i == data.SelectedCommand {
			// Wrap the whole line with the highlight style
			line = entryStyle.Render(line)
		}

		lines = append(lines, r.Mark(fmt.Sprintf("%s%d", ZoneHelpCommand, i), line))

		// Show error on next line if command failed and is selected
		if !entry.Success && entry.Error != "" && i == data.SelectedCommand {
			errorLine := fmt.Sprintf("    %s", failStyle.Render("Error: "+entry.Error))
			lines = append(lines, errorLine)
		}
	}

	if len(data.CommandHistory) > maxCommands {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render(
			fmt.Sprintf("  ... and %d more commands", len(data.CommandHistory)-maxCommands)))
	}

	return lines
}
