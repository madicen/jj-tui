package evologsplit

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

// Model is an experimental evolog-driven split picker (FAQ: move recent work into a new change).
type Model struct {
	shown         bool
	loading       bool
	loadErr       string
	bookmarkName  string
	tipChangeID   string
	tipCommitHint string
	entries       []jj.EvologEntry
	selectedIdx   int
	listScrollTop int
	// term size (SetDimensions from main)
	termW, termH     int
	listViewportRows int
	// diff vs tip for selected base
	diffSeq    int
	diffLoading bool
	diffErr    string
	diffFiles  []jj.ChangedFile
	zoneManager *zone.Manager
}

// NewModel creates the evolog split modal. zoneManager may be nil.
func NewModel(zm *zone.Manager) Model {
	return Model{zoneManager: zm, listViewportRows: 6, termW: 100, termH: 24}
}

func (m *Model) SetZoneManager(zm *zone.Manager) { m.zoneManager = zm }

// SetDimensions records terminal size and viewport row count for scrolling.
func (m Model) SetDimensions(w, h int) Model {
	if w < 1 {
		w = 80
	}
	if h < 1 {
		h = 24
	}
	m.termW, m.termH = w, h
	// Reserve space for header, two column titles, buttons, hints, border.
	vr := (h - 16) / 2
	if vr < 3 {
		vr = 3
	}
	if vr > 16 {
		vr = 16
	}
	m.listViewportRows = vr
	return m
}

// Show resets state for a new session (caller should batch LoadEvologCmd).
func (m *Model) Show(commit internal.Commit, bookmarkName string) {
	m.shown = true
	m.loading = true
	m.loadErr = ""
	m.bookmarkName = bookmarkName
	m.tipChangeID = commit.ChangeID
	m.tipCommitHint = commit.ID
	m.entries = nil
	m.selectedIdx = 0
	m.listScrollTop = 0
	m.diffSeq = 0
	m.diffLoading = false
	m.diffErr = ""
	m.diffFiles = nil
}

// Hide clears the modal.
func (m *Model) Hide() {
	m.shown = false
	m.loading = false
	m.loadErr = ""
	m.entries = nil
	m.diffFiles = nil
	m.diffErr = ""
	m.diffLoading = false
}

// IsShown reports whether the modal is active.
func (m *Model) IsShown() bool { return m.shown }

func (m Model) mark(id, s string) string {
	if m.zoneManager == nil {
		return s
	}
	return m.zoneManager.Mark(id, s)
}

func (m Model) syncListScroll() Model {
	vr := m.listViewportRows
	if vr < 1 {
		vr = 5
	}
	maxScroll := max(0, len(m.entries)-vr)
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

func (m Model) bumpDiffRequest() (Model, tea.Cmd) {
	m.diffSeq++
	m.diffLoading = true
	m.diffErr = ""
	return m, EvologDiffLoadRequestedCmd()
}

// DiffSnapshotForLoad returns seq and revisions for the in-flight diff (after bumpDiffRequest).
func (m Model) DiffSnapshotForLoad() (seq int, fromCommitID, tipChangeID string, ok bool) {
	if len(m.entries) == 0 || m.selectedIdx < 0 || m.selectedIdx >= len(m.entries) {
		return 0, "", "", false
	}
	return m.diffSeq, m.entries[m.selectedIdx].CommitID, m.tipChangeID, true
}

// Update handles keys, zone clicks, mouse wheel, and async messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.shown {
		return m, nil
	}
	switch msg := msg.(type) {
	case EvologLoadedMsg:
		m.loading = false
		if msg.Err != nil {
			m.loadErr = msg.Err.Error()
			return m, nil
		}
		m.entries = msg.Entries
		if len(m.entries) > 0 {
			if len(m.entries) > 1 {
				m.selectedIdx = 1
			} else {
				m.selectedIdx = 0
			}
		}
		m.listScrollTop = 0
		m = m.syncListScroll()
		return m.bumpDiffRequest()

	case EvologSplitDiffLoadedMsg:
		if msg.Seq != m.diffSeq {
			return m, nil
		}
		m.diffLoading = false
		if msg.Err != nil {
			m.diffErr = msg.Err.Error()
			m.diffFiles = nil
		} else {
			m.diffErr = ""
			m.diffFiles = msg.Files
		}
		return m, nil

	case tea.MouseMsg:
		isWheel := tea.MouseEvent(msg).IsWheel() || msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown
		if !isWheel || m.loading || m.loadErr != "" || len(m.entries) == 0 {
			break
		}
		vr := m.listViewportRows
		if vr < 1 {
			vr = 5
		}
		maxScroll := max(0, len(m.entries)-vr)
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
			if zid := m.resolveClickedZone(msg); zid != "" {
				return m.handleZoneClick(zid)
			}
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.loading {
		if msg.String() == "esc" {
			m.shown = false
			m.loading = false
			return m, state.NavigateTarget{Kind: state.NavigateBackToGraph}.Cmd()
		}
		return m, nil
	}
	if m.loadErr != "" {
		if msg.String() == "esc" || msg.String() == "enter" {
			m.shown = false
			return m, state.NavigateTarget{Kind: state.NavigateBackToGraph}.Cmd()
		}
		return m, nil
	}
	vr := m.listViewportRows
	if vr < 1 {
		vr = 5
	}
	maxScroll := max(0, len(m.entries)-vr)

	switch msg.String() {
	case "esc":
		m.shown = false
		return m, state.NavigateTarget{Kind: state.NavigateBackToGraph, StatusMessage: "Split cancelled"}.Cmd()
	case "enter":
		if len(m.entries) == 0 || m.selectedIdx < 0 || m.selectedIdx >= len(m.entries) {
			return m, nil
		}
		baseID := m.entries[m.selectedIdx].CommitID
		return m, state.NavigateTarget{
			Kind:                 state.NavigatePerformEvologSplit,
			EvologBookmarkName:   m.bookmarkName,
			EvologTipChangeID:    m.tipChangeID,
			EvologTipCommitHint:  m.tipCommitHint,
			EvologBaseCommitID:   baseID,
		}.Cmd()
	case "j", "down":
		if m.selectedIdx < len(m.entries)-1 {
			m.selectedIdx++
			m = m.syncListScroll()
			return m.bumpDiffRequest()
		}
		return m, nil
	case "k", "up":
		if m.selectedIdx > 0 {
			m.selectedIdx--
			m = m.syncListScroll()
			return m.bumpDiffRequest()
		}
		return m, nil
	case "pgdown", "ctrl+f":
		m.listScrollTop = min(maxScroll, m.listScrollTop+vr)
		return m, nil
	case "pgup", "ctrl+b":
		m.listScrollTop = max(0, m.listScrollTop-vr)
		return m, nil
	}
	return m, nil
}

func (m Model) ZoneIDs() []string {
	ids := make([]string, 0, len(m.entries)+2)
	for i := range m.entries {
		ids = append(ids, mouse.ZoneEvologSplitEntry(i))
	}
	ids = append(ids, mouse.ZoneEvologSplitConfirm, mouse.ZoneEvologSplitCancel)
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
	const prefix = "zone:evologsplit:entry:"
	if strings.HasPrefix(zoneID, prefix) {
		idx, err := strconv.Atoi(strings.TrimPrefix(zoneID, prefix))
		if err == nil && idx >= 0 && idx < len(m.entries) {
			m.selectedIdx = idx
			m = m.syncListScroll()
			return m.bumpDiffRequest()
		}
	}
	if zoneID == mouse.ZoneEvologSplitConfirm {
		if len(m.entries) == 0 || m.selectedIdx < 0 || m.selectedIdx >= len(m.entries) {
			return m, nil
		}
		baseID := m.entries[m.selectedIdx].CommitID
		return m, state.NavigateTarget{
			Kind:                 state.NavigatePerformEvologSplit,
			EvologBookmarkName:   m.bookmarkName,
			EvologTipChangeID:    m.tipChangeID,
			EvologTipCommitHint:  m.tipCommitHint,
			EvologBaseCommitID:   baseID,
		}.Cmd()
	}
	if zoneID == mouse.ZoneEvologSplitCancel {
		m.shown = false
		return m, state.NavigateTarget{Kind: state.NavigateBackToGraph, StatusMessage: "Split cancelled"}.Cmd()
	}
	return m, nil
}

// buildEvologSplitRightColumn returns exactly vr+1 lines (title + vr body rows) so layout
// matches the history column; the last body row is always used (overflow count or "—").
func buildEvologSplitRightColumn(m Model, vr, rightW int, muted lipgloss.Style) []string {
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F85149"))
	title := muted.Render("Files vs tip (selected →)")
	if vr < 1 {
		vr = 1
	}
	body := make([]string, vr)

	if m.diffLoading {
		body[0] = muted.Render("Loading diff…")
		return append([]string{title}, body...)
	}
	if m.diffErr != "" {
		body[0] = errStyle.Render(runewidth.Truncate(m.diffErr, max(4, rightW-2), "…"))
		return append([]string{title}, body...)
	}
	if len(m.diffFiles) == 0 {
		body[0] = muted.Render("(no file changes vs tip)")
		return append([]string{title}, body...)
	}

	if vr == 1 {
		if len(m.diffFiles) == 1 {
			f := m.diffFiles[0]
			p := runewidth.Truncate(f.Path, max(8, rightW-6), "…")
			body[0] = fmt.Sprintf("%s %s", lipgloss.NewStyle().Foreground(styles.ColorSecondary).Render(f.Status), p)
		} else {
			body[0] = muted.Render(fmt.Sprintf("… +%d files", len(m.diffFiles)))
		}
		return append([]string{title}, body...)
	}

	fileSlots := vr - 1
	sec := lipgloss.NewStyle().Foreground(styles.ColorSecondary)
	for i := 0; i < fileSlots && i < len(m.diffFiles); i++ {
		f := m.diffFiles[i]
		p := runewidth.Truncate(f.Path, max(8, rightW-6), "…")
		body[i] = fmt.Sprintf("%s %s", sec.Render(f.Status), p)
	}
	last := vr - 1
	extra := len(m.diffFiles) - fileSlots
	if extra > 0 {
		body[last] = muted.Render(fmt.Sprintf("… +%d more", extra))
	} else {
		body[last] = muted.Render("—")
	}
	return append([]string{title}, body...)
}

// View renders the modal.
func (m Model) View() string {
	if !m.shown {
		return ""
	}
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8BE9FD"))
	muted := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	var lines []string
	lines = append(lines, titleStyle.Render("Split (experimental)"))
	lines = append(lines, "")
	if strings.TrimSpace(m.bookmarkName) != "" {
		lines = append(lines, muted.Render(fmt.Sprintf("Bookmark: %s  ·  change: %s", m.bookmarkName, m.tipChangeID)))
	} else {
		lines = append(lines, muted.Render(fmt.Sprintf("Change: %s (no bookmark — only this revision is moved)", m.tipChangeID)))
	}
	lines = append(lines, muted.Render("Pick a parent revision; the new commit gets the current tip’s tree."))
	lines = append(lines, "")

	if m.loading {
		lines = append(lines, "Loading jj evolog…")
		box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(styles.ColorMuted).Padding(1, 2).Width(min(m.termW-4, 120))
		return box.Render(strings.Join(lines, "\n"))
	}
	if m.loadErr != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#F85149")).Render("Error: "+m.loadErr))
		lines = append(lines, "")
		lines = append(lines, muted.Render("Esc or Enter to close"))
		box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(styles.ColorMuted).Padding(1, 2).Width(min(m.termW-4, 120))
		return box.Render(strings.Join(lines, "\n"))
	}

	modalW := min(m.termW-4, 120)
	if modalW < 72 {
		modalW = 72
	}
	leftW := max(38, modalW*42/100)
	rightW := modalW - leftW - 3
	if rightW < 28 {
		rightW = 28
		leftW = modalW - rightW - 3
	}

	vr := m.listViewportRows
	if vr < 1 {
		vr = 5
	}

	selStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary)
	normal := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	cidStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))

	var leftLines []string
	leftLines = append(leftLines, muted.Render("History (j/k · PgUp/PgDn scroll)"))
	end := min(m.listScrollTop+vr, len(m.entries))
	for i := m.listScrollTop; i < end; i++ {
		e := m.entries[i]
		prefix := "  "
		style := normal
		if i == m.selectedIdx {
			prefix = "► "
			style = selStyle
		}
		sum := e.Summary
		maxSum := leftW - 22
		if maxSum < 8 {
			maxSum = 8
		}
		sum = runewidth.Truncate(sum, maxSum, "…")
		line := fmt.Sprintf("%s%s  %s", prefix, cidStyle.Render(e.CommitIDShort), style.Render(sum))
		leftLines = append(leftLines, m.mark(mouse.ZoneEvologSplitEntry(i), line))
	}
	for len(leftLines) < vr+1 {
		leftLines = append(leftLines, "")
	}
	leftBlock := lipgloss.NewStyle().Width(leftW).Render(strings.Join(leftLines, "\n"))

	rightLines := buildEvologSplitRightColumn(m, vr, rightW, muted)
	rightBlock := lipgloss.NewStyle().Width(rightW).Render(strings.Join(rightLines, "\n"))

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, "  ", rightBlock)
	lines = append(lines, body)
	lines = append(lines, "")
	confirm := m.mark(mouse.ZoneEvologSplitConfirm, styles.ButtonStyle.Render("Split here (Enter)"))
	cancel := m.mark(mouse.ZoneEvologSplitCancel, styles.ButtonSecondaryStyle.Render("Cancel (Esc)"))
	lines = append(lines, confirm+"  "+cancel)
	lines = append(lines, "")
	lines = append(lines, muted.Render("Wheel scrolls history · Do not pick the current tip row as base"))

	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(styles.ColorMuted).Padding(1, 2).Width(modalW)
	return box.Render(strings.Join(lines, "\n"))
}
