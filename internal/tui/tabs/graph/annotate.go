package graph

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/styles"
	"github.com/mattn/go-runewidth"
)

// annotateView is the scrollable blame overlay opened with `B` on a changed
// file. It shows `jj file annotate` output (change-id / author / age prefix per
// line) and lets the user press Enter on a line to jump the graph selection to
// the change that introduced it. It is a self-contained graph-tab overlay (like
// the context menu / confirm prompt) rather than a root-level modal.
type annotateView struct {
	shown    bool
	loading  bool
	seq      int
	path     string
	shortID  string
	lines    []jj.AnnotationLine
	selected int
	errMsg   string
	vp       viewport.Model
}

// annotateGutterWidth is the fixed width of the change-id/author/age prefix
// column so line content aligns regardless of metadata length.
const annotateGutterWidth = 34

// beginAnnotate opens the overlay in the loading state and returns the sequence
// number the async load command must echo back (stale results are ignored).
func (m *GraphModel) beginAnnotate(shortID, path string) int {
	if m.annotate == nil {
		vp := viewport.New(60, 10)
		vp.MouseWheelEnabled = true
		m.annotate = &annotateView{vp: vp}
	}
	m.annotate.shown = true
	m.annotate.loading = true
	m.annotate.errMsg = ""
	m.annotate.lines = nil
	m.annotate.selected = 0
	m.annotate.path = strings.TrimSpace(path)
	m.annotate.shortID = strings.TrimSpace(shortID)
	m.annotate.seq++
	m.annotate.vp.SetContent("")
	m.annotate.vp.GotoTop()
	return m.annotate.seq
}

// AnnotateShown reports whether the blame overlay is active.
func (m *GraphModel) AnnotateShown() bool {
	return m.annotate != nil && m.annotate.shown
}

// ShowAnnotate opens the blame overlay pre-populated with lines (no async load).
// Exposed for golden/integration tests and preloaded-blame callers.
func (m *GraphModel) ShowAnnotate(shortID, path string, lines []jj.AnnotationLine) {
	seq := m.beginAnnotate(shortID, path)
	m.SetAnnotateResult(seq, lines, nil)
}

// SelectedAnnotateChangeID returns the change-id of the highlighted blame line
// (empty when the overlay is closed or still loading). Exposed for tests.
func (m *GraphModel) SelectedAnnotateChangeID() string {
	return m.selectedAnnotateChangeID()
}

// SetAnnotateResult populates the overlay from an async load (matching seq).
func (m *GraphModel) SetAnnotateResult(seq int, lines []jj.AnnotationLine, err error) {
	if m.annotate == nil || !m.annotate.shown || seq != m.annotate.seq {
		return
	}
	m.annotate.loading = false
	if err != nil {
		m.annotate.errMsg = err.Error()
		m.annotate.lines = nil
		m.annotate.vp.SetContent("")
		return
	}
	m.annotate.errMsg = ""
	m.annotate.lines = lines
	m.annotate.selected = 0
	m.annotate.renderContent()
	m.annotate.vp.GotoTop()
}

// hideAnnotate closes the overlay.
func (m *GraphModel) hideAnnotate() {
	if m.annotate != nil {
		m.annotate.shown = false
		m.annotate.loading = false
		m.annotate.lines = nil
		m.annotate.vp.SetContent("")
	}
}

// selectedAnnotateChangeID returns the change-id of the currently selected blame
// line, or "" when the overlay is not showing parsed lines.
func (m *GraphModel) selectedAnnotateChangeID() string {
	if m.annotate == nil || !m.annotate.shown || len(m.annotate.lines) == 0 {
		return ""
	}
	if m.annotate.selected < 0 || m.annotate.selected >= len(m.annotate.lines) {
		return ""
	}
	return m.annotate.lines[m.annotate.selected].ChangeID
}

// moveSelection moves the highlighted blame line by delta and keeps it in view.
func (a *annotateView) moveSelection(delta int) {
	if len(a.lines) == 0 {
		return
	}
	a.selected += delta
	if a.selected < 0 {
		a.selected = 0
	}
	if a.selected >= len(a.lines) {
		a.selected = len(a.lines) - 1
	}
	a.renderContent()
	a.keepSelectedVisible()
}

func (a *annotateView) keepSelectedVisible() {
	h := a.vp.Height
	if h <= 0 {
		return
	}
	if a.selected < a.vp.YOffset {
		a.vp.SetYOffset(a.selected)
	} else if a.selected >= a.vp.YOffset+h {
		a.vp.SetYOffset(a.selected - h + 1)
	}
}

// handleAnnotateKey owns keyboard input while the blame overlay is shown:
// j/k move the highlighted line, scroll keys page the viewport, Enter jumps the
// graph selection to the change of the highlighted line, and Esc/q closes it.
func (m GraphModel) handleAnnotateKey(msg tea.KeyMsg) (GraphModel, *Request, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.CancelSelection):
		m.hideAnnotate()
		return m, nil, nil
	case key.Matches(msg, m.keys.Checkout):
		cid := m.selectedAnnotateChangeID()
		m.hideAnnotate()
		if cid == "" {
			return m, nil, nil
		}
		idx := m.findCommitIndexByChangeID(cid)
		if idx < 0 {
			return m, nil, nil
		}
		m.selectedCommit = idx
		m.changedFilesCommitID = ""
		m.changedFiles = nil
		m.selectedFile = 0
		m.scrollToSelectedCommit = true
		changeID := m.repository.Graph.Commits[idx].ChangeID
		return m, &Request{LoadChangedFiles: &changeID}, nil
	case key.Matches(msg, m.keys.MoveDown):
		if m.annotate != nil {
			m.annotate.moveSelection(1)
		}
		return m, nil, nil
	case key.Matches(msg, m.keys.MoveUp):
		if m.annotate != nil {
			m.annotate.moveSelection(-1)
		}
		return m, nil, nil
	case key.Matches(msg, m.keys.Scroll):
		if m.annotate != nil {
			var cmd tea.Cmd
			m.annotate.vp, cmd = m.annotate.vp.Update(msg)
			return m, nil, cmd
		}
		return m, nil, nil
	}
	return m, nil, nil
}

// findCommitIndexByChangeID returns the index of the commit whose ChangeID
// matches cid (either is a prefix of the other, since blame uses shortened ids),
// or -1 when the change is not in the currently loaded graph.
func (m *GraphModel) findCommitIndexByChangeID(cid string) int {
	cid = strings.TrimSpace(cid)
	if cid == "" || m.repository == nil {
		return -1
	}
	for i, c := range m.repository.Graph.Commits {
		ch := c.ChangeID
		if ch == cid || strings.HasPrefix(ch, cid) || strings.HasPrefix(cid, ch) {
			return i
		}
	}
	return -1
}

// renderContent rebuilds the viewport body with the current selection highlight.
func (a *annotateView) renderContent() {
	if len(a.lines) == 0 {
		a.vp.SetContent("")
		return
	}
	metaStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	selStyle := lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true)
	var b strings.Builder
	for i, ln := range a.lines {
		meta := fmt.Sprintf("%-8s %-10s %s", trunc(ln.ChangeID, 8), trunc(ln.Author, 10), ln.Age)
		meta = padRight(meta, annotateGutterWidth)
		content := strings.ReplaceAll(ln.Content, "\t", "    ")
		if i == a.selected {
			b.WriteString(selStyle.Render(fmt.Sprintf("▸ %s│ %s", meta, content)))
		} else {
			b.WriteString(metaStyle.Render(fmt.Sprintf("  %s│ ", meta)) + content)
		}
		if i < len(a.lines)-1 {
			b.WriteByte('\n')
		}
	}
	a.vp.SetContent(b.String())
}

// renderAnnotateOverlay renders the centered blame box.
func (m *GraphModel) renderAnnotateOverlay() string {
	a := m.annotate
	outerW := m.width - 4
	if outerW < 20 {
		outerW = max(20, m.width-2)
	}
	if outerW > m.width {
		outerW = m.width
	}
	innerW := max(10, outerW-4)

	maxBodyH := m.height - 8
	if maxBodyH < 3 {
		maxBodyH = 3
	}
	bodyH := maxBodyH
	switch {
	case a.loading:
		bodyH = 1
	case a.errMsg != "":
		bodyH = min(maxBodyH, strings.Count(a.errMsg, "\n")+1)
	default:
		if len(a.lines) < bodyH {
			bodyH = max(1, len(a.lines))
		}
	}
	a.vp.Width = innerW
	a.vp.Height = bodyH
	a.keepSelectedVisible()

	title := fmt.Sprintf("Blame: %s", a.path)
	if a.shortID != "" {
		title += fmt.Sprintf("  @ %s", a.shortID)
	}
	titleLine := lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true).
		Width(innerW).Render(trunc(title, innerW))

	var body string
	switch {
	case a.loading:
		body = lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Loading blame…")
	case a.errMsg != "":
		body = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Width(innerW).Render(a.errMsg)
	default:
		body = a.vp.View()
	}

	footer := lipgloss.NewStyle().Foreground(styles.ColorMuted).
		Render("Enter jump to change · j/k scroll · Esc close")

	inner := lipgloss.JoinVertical(lipgloss.Left, titleLine, "", body, "", footer)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorPrimary).
		Padding(0, 1).
		Width(outerW).
		Render(inner)
}

// trunc shortens s to at most w display columns.
func trunc(s string, w int) string {
	if w <= 0 {
		return ""
	}
	return runewidth.Truncate(s, w, "")
}

// padRight pads s with spaces to at least w display columns.
func padRight(s string, w int) string {
	if lipgloss.Width(s) >= w {
		return s
	}
	return s + runewidth.FillRight("", w-lipgloss.Width(s))
}
