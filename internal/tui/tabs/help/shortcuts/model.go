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
	// Single key column for every shortcut row so descriptions align (widest: ctrl+shift+u).
	const helpKeyColW = 18
	var lines []string
	lines = append(lines, styles.TitleStyle.Render("Commit Graph Shortcuts"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("j/↓"), styles.HelpDescStyle.Render("Move down")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("k/↑"), styles.HelpDescStyle.Render("Move up")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("Tab"), styles.HelpDescStyle.Render("Switch focus: graph ↔ files")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("o"), styles.HelpDescStyle.Render("View full jj diff for selected changed file (files pane)")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("O"), styles.HelpDescStyle.Render("Open selected file in external editor (files pane; set editor in Settings → Advanced)")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("Enter/e"), styles.HelpDescStyle.Render("Edit selected commit (jj edit)")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("s"), styles.HelpDescStyle.Render("Squash commit into parent")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("r"), styles.HelpDescStyle.Render("Rebase commit (with descendants)")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("mouse"), styles.HelpDescStyle.Render("Drag a commit row onto another to rebase (same as r, then pick destination)")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("dbl-click"), styles.HelpDescStyle.Render("Commit row: edit (jj edit); changed-file row: open in external editor")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("d"), styles.HelpDescStyle.Render("Edit description; or resolve divergent when commit is divergent")))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Commit description editor"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("^s"), styles.HelpDescStyle.Render("Save description")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("Esc"), styles.HelpDescStyle.Render("Cancel")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("ctrl+shift+u"), styles.HelpDescStyle.Render("Clear description text")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("✧^g"), styles.HelpDescStyle.Render("Same as the purple ✧ ^g chip beside the title (optional AI; Settings → Advanced + API key)")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("a"), styles.HelpDescStyle.Render("Abandon commit")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("n"), styles.HelpDescStyle.Render("Create new commit from selected")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("m"), styles.HelpDescStyle.Render("Create/move bookmark on commit")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("x"), styles.HelpDescStyle.Render("Delete bookmark from commit")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("c"), styles.HelpDescStyle.Render("Create new PR from commit chain")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("u"), styles.HelpDescStyle.Render("Update existing PR with new commits")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("f"), styles.HelpDescStyle.Render("Forgot new commit? Stack on bookmark@origin (avoid force-push)")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("z"), styles.HelpDescStyle.Render("split (experimental, when shown): jj evolog parent + step file list; o patch; p plan overlay (Enter runs split from overlay); s / ✧^g AI suggest; Graph (g) vs preview after split; FAQ bases on evolog row you pick, not main unless you choose that row; if AI says no split, Enter twice (or j/k); d optional AI describe; moves change (and feature bookmark if present)")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("C"), styles.HelpDescStyle.Render("Resolve diverged bookmark (when shown): graph pane focused; same flow as Branches (c)")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("^z"), styles.HelpDescStyle.Render("Undo last jj operation")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("^y"), styles.HelpDescStyle.Render("Redo jj operation")))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Bookmark Screen"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("j/↓"), styles.HelpDescStyle.Render("Select next existing bookmark")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("k/↑"), styles.HelpDescStyle.Render("Select previous / new input")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("Tab"), styles.HelpDescStyle.Render("Toggle new/existing bookmark")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("Enter"), styles.HelpDescStyle.Render("Create new or move selected")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("✧^g"), styles.HelpDescStyle.Render("Same as the ✧ ^g chip by the name field (new bookmark only; optional AI)")))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Create PR modal"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("^s"), styles.HelpDescStyle.Render("Create pull request")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("✧^g"), styles.HelpDescStyle.Render("Same as the ✧ ^g chip beside the modal title")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("Esc"), styles.HelpDescStyle.Render("Cancel")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("Tab"), styles.HelpDescStyle.Render("Switch title / body")))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Create Ticket modal"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("^s"), styles.HelpDescStyle.Render("Create ticket")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("✧^g"), styles.HelpDescStyle.Render("Same as the ✧ ^g chip beside the title (uses graph revision or @)")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("Esc"), styles.HelpDescStyle.Render("Cancel")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("Tab"), styles.HelpDescStyle.Render("Switch title / description")))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Pull Request Shortcuts"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("j/↓"), styles.HelpDescStyle.Render("Move down")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("k/↑"), styles.HelpDescStyle.Render("Move up")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("Enter/o"), styles.HelpDescStyle.Render("Open PR in browser")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("dbl-click"), styles.HelpDescStyle.Render("PR row: open in browser")))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Tickets Shortcuts"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("j/↓"), styles.HelpDescStyle.Render("Move down")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("k/↑"), styles.HelpDescStyle.Render("Move up")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("Enter"), styles.HelpDescStyle.Render("Create branch from ticket")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("o"), styles.HelpDescStyle.Render("Open ticket in browser")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("dbl-click"), styles.HelpDescStyle.Render("Ticket row: open in browser (single click loads transitions)")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("c"), styles.HelpDescStyle.Render("Change ticket status")))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Branches Shortcuts"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("j/↓"), styles.HelpDescStyle.Render("Move down")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("k/↑"), styles.HelpDescStyle.Render("Move up")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("T"), styles.HelpDescStyle.Render("Track remote branch")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("U"), styles.HelpDescStyle.Render("Untrack remote branch")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("L"), styles.HelpDescStyle.Render("Restore deleted local branch")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("x"), styles.HelpDescStyle.Render("Delete local bookmark")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("P"), styles.HelpDescStyle.Render("Push local branch to remote")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("F"), styles.HelpDescStyle.Render("Fetch from all remotes")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("c"), styles.HelpDescStyle.Render("Resolve conflicted bookmark")))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Settings Shortcuts"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("^j"), styles.HelpDescStyle.Render("Previous settings tab")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("^k"), styles.HelpDescStyle.Render("Next settings tab")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("Tab"), styles.HelpDescStyle.Render("Next input field")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("^s"), styles.HelpDescStyle.Render("Save settings (global)")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("^l"), styles.HelpDescStyle.Render("Save settings (local to repo)")))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Help Tab"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("^j"), styles.HelpDescStyle.Render("Previous sub-tab (Shortcuts ↔ History)")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("^k"), styles.HelpDescStyle.Render("Next sub-tab")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("Tab"), styles.HelpDescStyle.Render("Next sub-tab")))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Navigation"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("g"), styles.HelpDescStyle.Render("Go to commit graph")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("p"), styles.HelpDescStyle.Render("Go to pull requests")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("t"), styles.HelpDescStyle.Render("Go to Tickets")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("b"), styles.HelpDescStyle.Render("Go to Branches")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render(","), styles.HelpDescStyle.Render("Open settings")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("h/?"), styles.HelpDescStyle.Render("Show this help")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("^r"), styles.HelpDescStyle.Render("Refresh")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("Esc"), styles.HelpDescStyle.Render("Back to graph")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("^q"), styles.HelpDescStyle.Render("Quit")))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Graph Symbols"))
	lines = append(lines, "")
	lines = append(lines, "  @  Working copy (current editing state)")
	lines = append(lines, "  ○  Mutable commit (can be edited)")
	lines = append(lines, "  ◆  Immutable commit (pushed to remote)")
	lines = append(lines, "  ⚠  Commit has conflicts")
	lines = append(lines, fmt.Sprintf("  %s  Divergent commit (same change ID in multiple versions)", styles.DivergentMark))
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
	lines = append(lines, fmt.Sprintf("    %s  Approved", styles.ReviewApprovedMark))
	lines = append(lines, fmt.Sprintf("    %s  Changes requested", styles.ReviewChangesRequestedMark))
	lines = append(lines, fmt.Sprintf("    %s  Review pending", styles.ReviewPendingMark))
	lines = append(lines, "    ·   No reviews yet")
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Scrolling"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("PgUp/PgDn"), styles.HelpDescStyle.Render("Scroll page up/down")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("^u/^d"), styles.HelpDescStyle.Render("Scroll half page up/down")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("Home/End"), styles.HelpDescStyle.Render("Scroll to top/bottom")))
	lines = append(lines, fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render("Mouse"), styles.HelpDescStyle.Render("Use scroll wheel to scroll")))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Mouse"))
	lines = append(lines, "")
	lines = append(lines, "  Click on tabs, commits, PRs, tickets, or buttons")
	lines = append(lines, "  Click graph/files panes to switch focus")
	lines = append(lines, "  Click footer shortcuts (undo, redo, refresh, etc.)")
	return lines
}
