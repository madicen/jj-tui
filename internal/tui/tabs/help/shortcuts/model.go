package shortcuts

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

// Model is the Shortcuts sub-tab state. It owns scroll and renders the static shortcuts content.
type Model struct {
	zoneManager *zone.Manager
	width       int
	height      int
	yOffset     int
}

// NewModel creates a new Shortcuts sub-tab model.
func NewModel(zoneManager *zone.Manager) Model {
	return Model{zoneManager: zoneManager}
}

// Update handles messages for the Shortcuts sub-tab (dimensions, mouse wheel).
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.MouseMsg:
		if tea.MouseEvent(msg).IsWheel() {
			delta := 3
			isUp := msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelLeft
			if isUp {
				m.yOffset -= delta
			} else {
				m.yOffset += delta
			}
			if m.yOffset < 0 {
				m.yOffset = 0
			}
		}
		return m, nil
	}
	return m, nil
}

// View returns the full shortcuts content as a string (parent applies scroll).
func (m Model) View() string {
	lines := m.lines()
	return strings.Join(lines, "\n")
}

// Lines returns the full list of shortcut lines (for scroll windowing by parent).
func (m Model) Lines() []string {
	return m.lines()
}

// YOffset returns the current scroll offset.
func (m Model) YOffset() int { return m.yOffset }

// SetYOffset sets the scroll offset.
func (m *Model) SetYOffset(y int) {
	if y < 0 {
		y = 0
	}
	m.yOffset = y
}

// SetDimensions sets width and height.
func (m *Model) SetDimensions(width, height int) {
	m.width = width
	m.height = height
}

func (m Model) lines() []string {
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
