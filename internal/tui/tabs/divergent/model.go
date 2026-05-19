package divergent

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/styles"
	"github.com/mattn/go-runewidth"
)

// Model represents the divergent commit resolution overlay.
type Model struct {
	shown            bool
	changeID         string
	versions         []jj.DivergentVersion
	selectedIdx      int
	listScrollTop    int
	termW, termH     int
	listViewportRows int // max visible version rows (scroll when more)
	zoneManager      *zone.Manager
}

// NewModel creates a new Divergent model. zoneManager may be nil.
func NewModel(zoneManager *zone.Manager) Model {
	return Model{
		shown:            false,
		selectedIdx:      0,
		listViewportRows: 5,
		termW:            100,
		termH:            24,
		zoneManager:      zoneManager,
	}
}

// SetDimensions records terminal size and how many version rows fit before scrolling.
func (m Model) SetDimensions(w, h int) Model {
	if w < 1 {
		w = 80
	}
	if h < 1 {
		h = 24
	}
	m.termW, m.termH = w, h
	vr := min(6, max(2, (h-10)/3))
	m.listViewportRows = vr
	return m
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the Divergent modal
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.shown {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.MouseMsg:
		isWheel := tea.MouseEvent(msg).IsWheel() || msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown
		if !isWheel || len(m.versions) == 0 {
			break
		}
		vr := m.listViewportRows
		if vr < 1 {
			vr = 4
		}
		maxScroll := max(0, len(m.versions)-vr)
		if msg.Button == tea.MouseButtonWheelUp {
			m.listScrollTop = max(0, m.listScrollTop-1)
		} else {
			m.listScrollTop = min(maxScroll, m.listScrollTop+1)
		}
		return m, nil
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case zone.MsgZoneInBounds:
		if m.zoneManager != nil {
			if zoneID := m.resolveClickedZone(msg); zoneID != "" {
				return m.handleZoneClick(zoneID)
			}
		}
		return m, nil
	}
	return m, nil
}

// View renders the Divergent modal
func (m Model) View() string {
	if !m.shown {
		return ""
	}
	return m.renderDivergent()
}

// mark wraps content in a zone if zoneManager is set
func (m *Model) mark(id, content string) string {
	if m.zoneManager != nil {
		return m.zoneManager.Mark(id, content)
	}
	return content
}

// renderDivergent draws the divergent-commit picker. The window title
// ("Divergent commit") lives in the chrome tab — see chromedSlot — so the
// body opens directly with the action hint.
func (m *Model) renderDivergent() string {
	muted := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	cidStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))
	normalBorder := styles.ColorMuted

	modalW := min(max(48, m.termW-8), 78)

	var lines []string
	lines = append(lines, muted.Render("Pick one revision to keep · click a row to apply, or j/k + Enter · Esc cancel"))
	lines = append(lines, "")

	vr := m.listViewportRows
	if vr < 1 {
		vr = 4
	}
	end := min(m.listScrollTop+vr, len(m.versions))
	for i := m.listScrollTop; i < end; i++ {
		v := m.versions[i]
		summary := strings.TrimSpace(v.Summary)
		if summary == "" {
			summary = "(no description)"
		}
		shortID := v.CommitIDShort
		if shortID == "" && len(v.CommitID) >= 12 {
			shortID = v.CommitID[:12]
		}

		fileHint := strings.TrimSpace(v.FilesLine)
		if fileHint == "" {
			fileHint = "(file list unavailable)"
		}
		fileHint = runewidth.Truncate(fileHint, modalW-8, "…")

		metaParts := make([]string, 0, 5)
		if s := strings.TrimSpace(v.Author); s != "" {
			metaParts = append(metaParts, s)
		}
		if s := strings.TrimSpace(v.WhenDisplay); s != "" {
			metaParts = append(metaParts, s)
		}
		if s := strings.TrimSpace(v.Bookmarks); s != "" {
			metaParts = append(metaParts, "bookmarks "+s)
		}
		if v.Immutable {
			metaParts = append(metaParts, "immutable")
		}
		meta := muted.Render(runewidth.Truncate(strings.Join(metaParts, " · "), modalW-8, "…"))

		borderCol := normalBorder
		if i == m.selectedIdx {
			borderCol = styles.ColorPrimary
		}
		prefix := "  "
		if i == m.selectedIdx {
			prefix = "► "
		}
		sumMax := max(8, modalW-16-runewidth.StringWidth(shortID))
		summaryShown := runewidth.Truncate(summary, sumMax, "…")
		line1 := lipgloss.JoinHorizontal(lipgloss.Left,
			lipgloss.NewStyle().Render(prefix),
			cidStyle.Render(shortID),
			lipgloss.NewStyle().Render("  "+summaryShown),
		)
		inner := lipgloss.JoinVertical(lipgloss.Left,
			line1,
			"  "+fileHint,
			"  "+meta,
		)
		// Width(W) in lipgloss covers content+padding but not borders, so a
		// child box with Width(modalW-2) actually renders modalW cells wide
		// and overflows the outer modal's content area (modalW-2). Use
		// modalW-4 so the rendered child (incl. border) is modalW-2 wide.
		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderCol).
			Padding(0, 1).
			Width(modalW - 4).
			Render(inner)
		lines = append(lines, m.mark(mouse.ZoneDivergentCommit(i), box))
	}

	if len(m.versions) > vr {
		lines = append(lines, muted.Render(fmtScrollHint(m.listScrollTop, len(m.versions), vr)))
	}

	lines = append(lines, "")
	lines = append(lines, m.mark(mouse.ZoneDivergentCancel, muted.Render("Cancel")))

	outer := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorMuted).
		Padding(0, 1).
		Width(modalW)
	return outer.Render(strings.Join(lines, "\n"))
}

func fmtScrollHint(top, total, vr int) string {
	if total <= vr {
		return ""
	}
	return fmt.Sprintf("Scroll %d–%d of %d (wheel / PgUp / PgDn)", top+1, min(top+vr, total), total)
}

func (m Model) syncListScroll() Model {
	vr := m.listViewportRows
	if vr < 1 {
		vr = 4
	}
	maxScroll := max(0, len(m.versions)-vr)
	if m.listScrollTop > maxScroll {
		m.listScrollTop = maxScroll
	}
	if m.selectedIdx < m.listScrollTop {
		m.listScrollTop = m.selectedIdx
	}
	if m.selectedIdx >= m.listScrollTop+vr {
		m.listScrollTop = m.selectedIdx - vr + 1
	}
	if m.listScrollTop < 0 {
		m.listScrollTop = 0
	}
	return m
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	vr := m.listViewportRows
	if vr < 1 {
		vr = 4
	}
	maxScroll := max(0, len(m.versions)-vr)

	switch msg.String() {
	case "esc":
		m.shown = false
		return m, state.NavigateTarget{Kind: state.NavigateBackToGraph, StatusMessage: "Divergent commit resolution cancelled"}.Cmd()
	case "enter":
		keepCommitID := m.GetSelectedCommitID()
		if keepCommitID != "" {
			return m, state.NavigateTarget{
				Kind:                  state.NavigateResolveDivergent,
				DivergentChangeID:     m.changeID,
				DivergentKeepCommitID: keepCommitID,
			}.Cmd()
		}
		return m, nil
	case "j", "down":
		n := len(m.versions)
		if n > 0 && m.selectedIdx < n-1 {
			m.selectedIdx++
			m = m.syncListScroll()
		}
		return m, nil
	case "k", "up":
		if m.selectedIdx > 0 {
			m.selectedIdx--
			m = m.syncListScroll()
		}
		return m, nil
	case "pgdown", "ctrl+f":
		m.listScrollTop = min(maxScroll, m.listScrollTop+vr)
		return m, nil
	case "pgup", "ctrl+b":
		m.listScrollTop = max(0, m.listScrollTop-vr)
		return m, nil
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		idx := int(msg.String()[0] - '1')
		if idx >= 0 && idx < len(m.versions) {
			m.selectedIdx = idx
			m = m.syncListScroll()
		}
		return m, nil
	}
	return m, nil
}

// ZoneIDs returns the zone IDs this modal uses when rendering. Used to resolve clicks.
func (m Model) ZoneIDs() []string {
	ids := make([]string, 0, len(m.versions)+1)
	for i := range m.versions {
		ids = append(ids, mouse.ZoneDivergentCommit(i))
	}
	ids = append(ids, mouse.ZoneDivergentCancel)
	return ids
}

func (m Model) resolveClickedZone(msg zone.MsgZoneInBounds) string {
	if msg.Zone == nil {
		return ""
	}
	for _, id := range m.ZoneIDs() {
		z := m.zoneManager.Get(id)
		if z != nil && z.InBounds(msg.Event) {
			return id
		}
	}
	return ""
}

func (m Model) handleZoneClick(zoneID string) (Model, tea.Cmd) {
	const prefix = "zone:divergent:commit:"
	if strings.HasPrefix(zoneID, prefix) {
		s := strings.TrimPrefix(zoneID, prefix)
		i, err := strconv.Atoi(s)
		if err == nil && i >= 0 && i < len(m.versions) {
			keep := m.versions[i].CommitID
			if keep != "" {
				return m, state.NavigateTarget{
					Kind:                  state.NavigateResolveDivergent,
					DivergentChangeID:     m.changeID,
					DivergentKeepCommitID: keep,
				}.Cmd()
			}
		}
	}
	if zoneID == mouse.ZoneDivergentCancel {
		m.shown = false
		return m, state.NavigateTarget{Kind: state.NavigateBackToGraph, StatusMessage: "Divergent commit resolution cancelled"}.Cmd()
	}
	return m, nil
}

// IsShown returns whether the modal is displayed
func (m *Model) IsShown() bool {
	return m.shown
}

// Show displays the divergent modal
func (m *Model) Show(changeID string, versions []jj.DivergentVersion) {
	x := *m
	x.shown = true
	x.changeID = changeID
	x.versions = append([]jj.DivergentVersion(nil), versions...)
	x.selectedIdx = 0
	x.listScrollTop = 0
	x = x.syncListScroll()
	*m = x
}

// Hide hides the modal
func (m *Model) Hide() {
	m.shown = false
}

// GetSelectedCommitID returns the selected commit ID
func (m *Model) GetSelectedCommitID() string {
	if m.selectedIdx < len(m.versions) {
		return m.versions[m.selectedIdx].CommitID
	}
	return ""
}

// GetChangeID returns the change ID
func (m *Model) GetChangeID() string {
	return m.changeID
}

// SetSelectedIdx sets the selected commit index
func (m *Model) SetSelectedIdx(idx int) {
	if idx >= 0 && idx < len(m.versions) {
		x := *m
		x.selectedIdx = idx
		x = x.syncListScroll()
		*m = x
	}
}

// GetSelectedIdx returns the selected commit index
func (m *Model) GetSelectedIdx() int {
	return m.selectedIdx
}

// GetCommitCount returns the number of divergent commits
func (m *Model) GetCommitCount() int {
	return len(m.versions)
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
	_ = repo
}
