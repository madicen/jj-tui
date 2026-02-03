package view

import (
	"fmt"
	"strings"
)

// Help renders the help view
func (r *Renderer) Help() string {
	var lines []string

	lines = append(lines, TitleStyle.Render("Commit Graph Shortcuts"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("j/↓"), HelpDescStyle.Render("Move down")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("k/↑"), HelpDescStyle.Render("Move up")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("Enter/e"), HelpDescStyle.Render("Edit selected commit (jj edit)")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("d"), HelpDescStyle.Render("Edit commit description")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("s"), HelpDescStyle.Render("Squash commit into parent")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("r"), HelpDescStyle.Render("Rebase commit (select destination)")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("a"), HelpDescStyle.Render("Abandon commit")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("n"), HelpDescStyle.Render("Create new commit (jj new)")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("b"), HelpDescStyle.Render("Create/move bookmark on commit")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("x"), HelpDescStyle.Render("Delete bookmark from commit")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("c"), HelpDescStyle.Render("Create new PR from commit chain")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("u"), HelpDescStyle.Render("Update existing PR with new commits")))

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
	lines = append(lines, TitleStyle.Render("Navigation"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("g"), HelpDescStyle.Render("Go to commit graph")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("p"), HelpDescStyle.Render("Go to pull requests")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("t"), HelpDescStyle.Render("Go to Tickets")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render(","), HelpDescStyle.Render("Open settings")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("h/?"), HelpDescStyle.Render("Show this help")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("Ctrl+r"), HelpDescStyle.Render("Refresh")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("Esc"), HelpDescStyle.Render("Back to graph")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("Ctrl+q"), HelpDescStyle.Render("Quit")))

	lines = append(lines, "")
	lines = append(lines, TitleStyle.Render("Tickets Shortcuts"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("j/↓"), HelpDescStyle.Render("Move down")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("k/↑"), HelpDescStyle.Render("Move up")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("Enter"), HelpDescStyle.Render("Create branch from ticket")))

	lines = append(lines, "")
	lines = append(lines, TitleStyle.Render("Symbols"))
	lines = append(lines, "")
	lines = append(lines, "  @  Working copy (current editing state)")
	lines = append(lines, "  ○  Mutable commit (can be edited)")
	lines = append(lines, "  ◆  Immutable commit (pushed to remote)")
	lines = append(lines, "  ⚠  Commit has conflicts")
	lines = append(lines, "")
	lines = append(lines, "  ● green   Open PR")
	lines = append(lines, "  ● red     Closed PR")
	lines = append(lines, "  ● purple  Merged PR")

	lines = append(lines, "")
	lines = append(lines, TitleStyle.Render("Scrolling"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("PgUp/PgDn"), HelpDescStyle.Render("Scroll page up/down")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("Ctrl+U/D"), HelpDescStyle.Render("Scroll half page up/down")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("Home/End"), HelpDescStyle.Render("Scroll to top/bottom")))
	lines = append(lines, fmt.Sprintf("  %s  %s", HelpKeyStyle.Width(10).Render("Mouse"), HelpDescStyle.Render("Use scroll wheel to scroll")))

	lines = append(lines, "")
	lines = append(lines, TitleStyle.Render("Mouse"))
	lines = append(lines, "")
	lines = append(lines, "  Click on tabs, commits, PRs, or buttons to interact")

	return strings.Join(lines, "\n")
}

