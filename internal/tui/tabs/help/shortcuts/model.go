package shortcuts

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/tui/keys"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

// Model is the Shortcuts sub-tab state. It owns scroll and renders the shortcuts
// content generated from the KeyMaps (PLAN(P3.3)).
type Model struct {
	zoneManager *zone.Manager
	keys        keys.KeyMaps
	width       int
	height      int
	yOffset     int
}

// NewModel creates a new Shortcuts sub-tab model.
func NewModel(zoneManager *zone.Manager) Model {
	return Model{zoneManager: zoneManager, keys: keys.DefaultKeyMaps(nil)}
}

// SetKeyMaps replaces the KeyMaps used to generate the shortcuts list so config
// rebindings (PLAN(P5.1)) show the effective keys.
func (m *Model) SetKeyMaps(km keys.KeyMaps) {
	m.keys = km
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

// helpKeyColW is the shared key-column width so descriptions align (widest: ctrl+shift+u).
const helpKeyColW = 18

// row renders a "  <key>  <desc>" help line from a bubbles/key Binding, using the
// binding's help Key (display) and Desc. Feeding these from the KeyMaps means the
// help tab stays in sync with the handlers and reflects any config rebinding.
func row(b key.Binding) string {
	h := b.Help()
	return fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render(h.Key), styles.HelpDescStyle.Render(h.Desc))
}

// staticRow renders a documentation-only line whose key isn't a rebindable
// action (mouse gestures, modal-local keys, AI chips).
func staticRow(keyDisp, desc string) string {
	return fmt.Sprintf("  %s  %s", styles.HelpKeyStyle.Width(helpKeyColW).Render(keyDisp), styles.HelpDescStyle.Render(desc))
}

func (m Model) lines() []string {
	g := m.keys.Graph
	pr := m.keys.PRs
	tk := m.keys.Tickets
	br := m.keys.Branches
	hp := m.keys.Help
	gl := m.keys.Global
	lines := make([]string, 0, 150)
	lines = append(lines, styles.TitleStyle.Render("Commit Graph Shortcuts"))
	lines = append(lines, "")
	lines = append(lines, row(g.MoveDown))
	lines = append(lines, row(g.MoveUp))
	lines = append(lines, row(g.ToggleFocus))
	lines = append(lines, row(g.ViewFileDiff))
	lines = append(lines, row(g.OpenExternal))
	lines = append(lines, row(g.Checkout))
	lines = append(lines, row(g.Squash))
	lines = append(lines, row(g.Rebase))
	lines = append(lines, row(g.Merge))
	lines = append(lines, staticRow("mouse", "Drag a commit row onto another to rebase (same as r, then pick destination)"))
	lines = append(lines, staticRow("dbl-click", "Commit row: edit (jj edit); changed-file row: open in external editor"))
	lines = append(lines, row(g.EditDescription))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Commit description editor"))
	lines = append(lines, "")
	lines = append(lines, staticRow("^s", "Save description"))
	lines = append(lines, staticRow("Esc", "Cancel"))
	lines = append(lines, staticRow("ctrl+shift+u", "Clear description text"))
	lines = append(lines, staticRow("✧^g", "Same as the purple ✧ ^g chip beside the title (optional AI; Settings → AI + API key)"))
	lines = append(lines, row(g.Abandon))
	lines = append(lines, row(g.NewCommit))
	lines = append(lines, row(g.CreateBookmark))
	lines = append(lines, row(g.DeleteBookmark))
	lines = append(lines, row(g.CreatePR))
	lines = append(lines, row(g.UpdatePR))
	lines = append(lines, row(g.MoveDelta))
	lines = append(lines, row(g.EvologSplit))
	lines = append(lines, row(g.ResolveConflict))
	lines = append(lines, row(gl.Undo))
	lines = append(lines, row(gl.Redo))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Bookmark Screen"))
	lines = append(lines, "")
	lines = append(lines, staticRow("j/↓", "Select next existing bookmark"))
	lines = append(lines, staticRow("k/↑", "Select previous / new input"))
	lines = append(lines, staticRow("Tab", "Toggle new/existing bookmark"))
	lines = append(lines, staticRow("Enter", "Create new or move selected"))
	lines = append(lines, staticRow("✧^g", "Same as the ✧ ^g chip by the name field (new bookmark only; optional AI)"))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Create PR modal"))
	lines = append(lines, "")
	lines = append(lines, staticRow("^s", "Create pull request"))
	lines = append(lines, staticRow("✧^g", "Same as the ✧ ^g chip beside the modal title"))
	lines = append(lines, staticRow("Esc", "Cancel"))
	lines = append(lines, staticRow("Tab", "Switch title / body"))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Create Ticket modal"))
	lines = append(lines, "")
	lines = append(lines, staticRow("^s", "Create ticket"))
	lines = append(lines, staticRow("✧^g", "Same as the ✧ ^g chip beside the title (uses graph revision or @)"))
	lines = append(lines, staticRow("Esc", "Cancel"))
	lines = append(lines, staticRow("Tab", "Switch title / description"))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Pull Request Shortcuts"))
	lines = append(lines, "")
	lines = append(lines, row(pr.MoveDown))
	lines = append(lines, row(pr.MoveUp))
	lines = append(lines, row(pr.Open))
	lines = append(lines, staticRow("dbl-click", "PR row: open in browser"))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Tickets Shortcuts"))
	lines = append(lines, "")
	lines = append(lines, row(tk.MoveDown))
	lines = append(lines, row(tk.MoveUp))
	lines = append(lines, row(tk.CreateBranch))
	lines = append(lines, row(tk.Open))
	lines = append(lines, staticRow("dbl-click", "Ticket row: open in browser (single click loads transitions)"))
	lines = append(lines, row(tk.ChangeStatus))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Branches Shortcuts"))
	lines = append(lines, "")
	lines = append(lines, row(br.MoveDown))
	lines = append(lines, row(br.MoveUp))
	lines = append(lines, row(br.Track))
	lines = append(lines, row(br.TrackByName))
	lines = append(lines, row(br.Untrack))
	lines = append(lines, row(br.Restore))
	lines = append(lines, row(br.Delete))
	lines = append(lines, row(br.Push))
	lines = append(lines, row(br.Fetch))
	lines = append(lines, row(br.ResolveConflict))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Settings Shortcuts"))
	lines = append(lines, "")
	lines = append(lines, staticRow("^j", "Previous settings tab"))
	lines = append(lines, staticRow("^k", "Next settings tab"))
	lines = append(lines, staticRow("Tab", "Next input field"))
	lines = append(lines, staticRow("^s", "Save settings (global)"))
	lines = append(lines, staticRow("^l", "Save settings (local to repo)"))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Help Tab"))
	lines = append(lines, "")
	lines = append(lines, row(hp.PrevTab))
	lines = append(lines, row(hp.NextTab))
	lines = append(lines, row(hp.SwitchTab))
	lines = append(lines, "")
	lines = append(lines, styles.TitleStyle.Render("Navigation"))
	lines = append(lines, "")
	lines = append(lines, row(gl.NavGraph))
	lines = append(lines, row(gl.NavPRs))
	lines = append(lines, row(gl.NavTickets))
	lines = append(lines, row(gl.NavBranches))
	lines = append(lines, row(gl.NavSettings))
	lines = append(lines, row(gl.NavHelp))
	lines = append(lines, row(gl.Refresh))
	lines = append(lines, row(gl.Back))
	lines = append(lines, row(gl.Quit))
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
	lines = append(lines, "    ● grey    Draft PR")
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
