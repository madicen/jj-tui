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

// Model represents the divergent commit resolution overlay (same composition pattern as evolog split).
type Model struct {
	shown            bool
	changeID         string
	versions         []jj.DivergentVersion
	selectedIdx      int
	listScrollTop    int
	fileScrollTop    int
	termW, termH     int
	listViewportRows int
	zoneManager      *zone.Manager
}

// NewModel creates a new Divergent model. zoneManager may be nil.
func NewModel(zoneManager *zone.Manager) Model {
	return Model{
		shown:            false,
		selectedIdx:      0,
		listViewportRows: 6,
		termW:            100,
		termH:            24,
		zoneManager:      zoneManager,
	}
}

// SetDimensions records terminal size and a viewport row count for scrolling (like evolog split).
func (m Model) SetDimensions(w, h int) Model {
	if w < 1 {
		w = 80
	}
	if h < 1 {
		h = 24
	}
	m.termW, m.termH = w, h
	// Reserve space for title, explanation, meta line, buttons, hints, border + padding.
	vr := (h - 18) / 2
	if vr < 3 {
		vr = 3
	}
	if vr > 16 {
		vr = 16
	}
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
			vr = 5
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

func formatDivergentFileLine(f jj.ChangedFile, pathMax int) string {
	pm := max(8, pathMax)
	p := runewidth.Truncate(f.Path, pm, "…")
	st, ch := styles.GetStatusStyle(f.Status)
	return fmt.Sprintf("%s %s", st.Render(ch), p)
}

// buildDivergentRightColumn returns exactly vr+1 lines (title + vr body) to align with the left column.
func buildDivergentRightColumn(v jj.DivergentVersion, vr, rightW, fileScrollTop int, muted lipgloss.Style) []string {
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F85149"))
	title := muted.Render("Files vs parent · [ ] scroll")
	if vr < 1 {
		vr = 1
	}
	body := make([]string, vr)

	if v.ChangedFiles == nil {
		body[0] = errStyle.Render(runewidth.Truncate(strings.TrimSpace(v.FilesLine), max(4, rightW-2), "…"))
		for i := 1; i < vr; i++ {
			body[i] = ""
		}
		if vr > 1 {
			body[vr-1] = muted.Render("—")
		}
		return append([]string{title}, body...)
	}

	files := v.ChangedFiles
	if len(files) == 0 {
		body[0] = muted.Render("(no changes vs parent)")
		for i := 1; i < vr-1; i++ {
			body[i] = ""
		}
		if vr > 1 {
			body[vr-1] = muted.Render("—")
		}
		return append([]string{title}, body...)
	}

	fileSlots := vr - 1 // last row: scroll / overflow hint
	if fileSlots < 1 {
		if len(files) == 1 {
			body[0] = formatDivergentFileLine(files[0], rightW-4)
		} else {
			body[0] = muted.Render(fmt.Sprintf("… +%d files", len(files)))
		}
		return append([]string{title}, body...)
	}

	maxTop := max(0, len(files)-fileSlots)
	ft := fileScrollTop
	if ft > maxTop {
		ft = maxTop
	}
	for i := 0; i < fileSlots; i++ {
		idx := ft + i
		if idx < len(files) {
			body[i] = formatDivergentFileLine(files[idx], rightW-4)
		} else {
			body[i] = ""
		}
	}
	last := vr - 1
	var hint []string
	if ft > 0 {
		hint = append(hint, fmt.Sprintf("↑%d", ft))
	}
	if nBelow := len(files) - ft - fileSlots; nBelow > 0 {
		hint = append(hint, fmt.Sprintf("↓%d", nBelow))
	}
	if len(hint) > 0 {
		body[last] = muted.Render(strings.Join(hint, " · "))
	} else {
		body[last] = muted.Render("—")
	}
	return append([]string{title}, body...)
}

func (m *Model) renderDivergent() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8BE9FD"))
	muted := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	var lines []string
	lines = append(lines, titleStyle.Render(styles.DivergentMark+" Divergent"))
	lines = append(lines, "")
	lines = append(lines, muted.Render("Same change ID on multiple commits. Keep one row; others are abandoned."))
	lines = append(lines, "")

	// Use nearly the full terminal width; overlay centers with a small margin.
	modalW := max(64, m.termW-4)
	// Even split: versions | files (gap matches JoinHorizontal "  ").
	const colGap = 2
	inner := modalW - colGap
	leftW := inner / 2
	rightW := inner - leftW // if inner is odd, right column gets the extra cell

	vr := m.listViewportRows
	if vr < 1 {
		vr = 5
	}

	if m.selectedIdx >= 0 && m.selectedIdx < len(m.versions) {
		v := m.versions[m.selectedIdx]
		metaParts := make([]string, 0, 4)
		if s := strings.TrimSpace(v.Author); s != "" {
			metaParts = append(metaParts, s)
		}
		if s := strings.TrimSpace(v.WhenDisplay); s != "" {
			metaParts = append(metaParts, s)
		}
		if s := strings.TrimSpace(v.ParentsShort); s != "" {
			metaParts = append(metaParts, "parents "+s)
		}
		if s := strings.TrimSpace(v.Bookmarks); s != "" {
			metaParts = append(metaParts, "bookmarks "+s)
		}
		meta := strings.Join(metaParts, " · ")
		lines = append(lines, muted.Render(runewidth.Truncate(meta, max(20, modalW-6), "…")))
		lines = append(lines, "")
	}

	selStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary)
	normal := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	cidStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))

	var leftLines []string
	leftLines = append(leftLines, muted.Render("Versions (j/k · PgUp/PgDn · wheel)"))
	end := min(m.listScrollTop+vr, len(m.versions))
	for i := m.listScrollTop; i < end; i++ {
		v := m.versions[i]
		prefix := "  "
		style := normal
		if i == m.selectedIdx {
			prefix = "► "
			style = selStyle
		}
		sum := v.Summary
		if sum == "" {
			sum = "(no description)"
		}
		maxSum := leftW - 22
		if maxSum < 8 {
			maxSum = 8
		}
		sum = runewidth.Truncate(sum, maxSum, "…")
		shortID := v.CommitIDShort
		if shortID == "" && len(v.CommitID) >= 12 {
			shortID = v.CommitID[:12]
		}
		line := fmt.Sprintf("%s%s  %s", prefix, cidStyle.Render(shortID), style.Render(sum))
		leftLines = append(leftLines, m.mark(mouse.ZoneDivergentCommit(i), line))
	}
	for len(leftLines) < vr+1 {
		leftLines = append(leftLines, "")
	}
	leftBlock := lipgloss.NewStyle().Width(leftW).Render(strings.Join(leftLines, "\n"))

	var rightLines []string
	if len(m.versions) > 0 && m.selectedIdx >= 0 && m.selectedIdx < len(m.versions) {
		rightLines = buildDivergentRightColumn(m.versions[m.selectedIdx], vr, rightW, m.fileScrollTop, muted)
	} else {
		rightLines = append(rightLines, muted.Render("Files vs parent"))
		for i := 0; i < vr; i++ {
			rightLines = append(rightLines, "")
		}
	}
	rightBlock := lipgloss.NewStyle().Width(rightW).Render(strings.Join(rightLines, "\n"))

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, "  ", rightBlock)
	lines = append(lines, body)
	lines = append(lines, "")

	confirmBtn := m.mark(mouse.ZoneDivergentConfirm, styles.ButtonStyle.Render("Keep Selected (Enter)"))
	cancelBtn := m.mark(mouse.ZoneDivergentCancel, styles.ButtonSecondaryStyle.Render("Cancel (Esc)"))
	lines = append(lines, confirmBtn+"  "+cancelBtn)
	lines = append(lines, "")
	lines = append(lines, muted.Render("Pick a version on the left; inspect files on the right."))

	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(styles.ColorMuted).Padding(1, 2).Width(modalW)
	return box.Render(strings.Join(lines, "\n"))
}

func (m Model) syncListScroll() Model {
	vr := m.listViewportRows
	if vr < 1 {
		vr = 5
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

func (m Model) syncFileScroll() Model {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.versions) {
		return m
	}
	v := m.versions[m.selectedIdx]
	if len(v.ChangedFiles) == 0 {
		m.fileScrollTop = 0
		return m
	}
	vr := m.listViewportRows
	if vr < 1 {
		vr = 5
	}
	fileSlots := max(1, vr-1)
	maxTop := max(0, len(v.ChangedFiles)-fileSlots)
	if m.fileScrollTop > maxTop {
		m.fileScrollTop = maxTop
	}
	if m.fileScrollTop < 0 {
		m.fileScrollTop = 0
	}
	return m
}

// handleKeyMsg handles keyboard input; returns PerformCancelCmd or PerformResolveCmd for main to handle.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	vr := m.listViewportRows
	if vr < 1 {
		vr = 5
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
			m.fileScrollTop = 0
			m = m.syncListScroll()
			m = m.syncFileScroll()
		}
		return m, nil
	case "k", "up":
		if m.selectedIdx > 0 {
			m.selectedIdx--
			m.fileScrollTop = 0
			m = m.syncListScroll()
			m = m.syncFileScroll()
		}
		return m, nil
	case "pgdown", "ctrl+f":
		m.listScrollTop = min(maxScroll, m.listScrollTop+vr)
		return m, nil
	case "pgup", "ctrl+b":
		m.listScrollTop = max(0, m.listScrollTop-vr)
		return m, nil
	case "[":
		if m.fileScrollTop > 0 {
			m.fileScrollTop--
		}
		return m, nil
	case "]":
		if m.selectedIdx < 0 || m.selectedIdx >= len(m.versions) {
			return m, nil
		}
		v := m.versions[m.selectedIdx]
		if len(v.ChangedFiles) > 0 {
			fileSlots := max(1, vr-1)
			maxTop := max(0, len(v.ChangedFiles)-fileSlots)
			if m.fileScrollTop < maxTop {
				m.fileScrollTop++
			}
		}
		return m, nil
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		idx := int(msg.String()[0] - '1')
		if idx >= 0 && idx < len(m.versions) {
			m.selectedIdx = idx
			m.fileScrollTop = 0
			m = m.syncListScroll()
			m = m.syncFileScroll()
		}
		return m, nil
	}
	return m, nil
}

// ZoneIDs returns the zone IDs this modal uses when rendering. Used to resolve clicks.
func (m Model) ZoneIDs() []string {
	ids := make([]string, 0, len(m.versions)+2)
	for i := range m.versions {
		ids = append(ids, mouse.ZoneDivergentCommit(i))
	}
	ids = append(ids, mouse.ZoneDivergentConfirm, mouse.ZoneDivergentCancel)
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

// handleZoneClick handles a zone click by zone ID (called from Update after resolve). Returns (updated model, cmd).
func (m Model) handleZoneClick(zoneID string) (Model, tea.Cmd) {
	const prefix = "zone:divergent:commit:"
	if strings.HasPrefix(zoneID, prefix) {
		s := strings.TrimPrefix(zoneID, prefix)
		i, err := strconv.Atoi(s)
		if err == nil && i >= 0 && i < len(m.versions) {
			m.selectedIdx = i
			m.fileScrollTop = 0
			m = m.syncListScroll()
			m = m.syncFileScroll()
			return m, nil
		}
	}
	if zoneID == mouse.ZoneDivergentConfirm {
		keepCommitID := m.GetSelectedCommitID()
		if keepCommitID != "" {
			return m, state.NavigateTarget{
				Kind:                  state.NavigateResolveDivergent,
				DivergentChangeID:     m.changeID,
				DivergentKeepCommitID: keepCommitID,
			}.Cmd()
		}
		return m, nil
	}
	if zoneID == mouse.ZoneDivergentCancel {
		m.shown = false
		return m, state.NavigateTarget{Kind: state.NavigateBackToGraph, StatusMessage: "Divergent commit resolution cancelled"}.Cmd()
	}
	return m, nil
}

// Accessors

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
	x.fileScrollTop = 0
	x = x.syncListScroll()
	x = x.syncFileScroll()
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
		x.fileScrollTop = 0
		x = x.syncListScroll()
		x = x.syncFileScroll()
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
	// Divergent modal doesn't use repository directly
}
