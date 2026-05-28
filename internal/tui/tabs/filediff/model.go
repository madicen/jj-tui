package filediff

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

// Horizontal layout: leave fileDiffTermSideColumns on each side of the bordered modal,
// and reserve fileDiffOuterInnerDelta between outer box width and viewport inner width.
// The modal sizes to whatever the diff content needs (longest line + gutter) so the
// window hugs the patch rather than always claiming a fixed fraction of the terminal;
// fileDiffInnerMinComfort is the floor used while content is empty/loading so the box
// doesn't pop in at a tiny size and then expand.
const (
	fileDiffTermSideColumns = 2
	fileDiffOuterInnerDelta = 4
	fileDiffInnerMinComfort = 80
)

// Vertical layout: the bubble-overlay chrome adds 4 rows of overhead around our box —
// 1 mask spacer + 1 tab cap + 1 tab-on-border row (the box's top border merged with
// the title) + 1 bottom border. fileDiffChromeRowOverhead reserves them so the box +
// chrome together fit inside the terminal instead of scrolling off the bottom.
// fileDiffBottomSafety leaves one additional row clear so the modal doesn't kiss
// the very last terminal row (which some terminals scroll/clip).
const (
	fileDiffChromeRowOverhead = 4
	fileDiffBottomSafety      = 1
	// fileDiffMinBodyRows keeps the viewport from collapsing to 0 rows for
	// degenerate cases (empty diff, single-line errors) so the box always has
	// a visible body region under the subtitle.
	fileDiffMinBodyRows = 3
)

// Model is a scrollable full-file diff overlay for one changed file at a commit.
type Model struct {
	shown    bool
	seq      int
	shortID  string
	filePath string
	// When set (ShowPreloadedStyledDiff), View uses these instead of file path + change id.
	overlayTitle string
	overlaySub   string
	loading      bool
	errMsg       string
	body         string
	termW        int
	termH        int
	vp           viewport.Model
	zm           *zone.Manager
	innerW       int
	innerH       int
	outerW       int
	headerH      int
	footerH      int
	// naturalOuterW/H record the "content + chrome wants this much room" dimensions
	// computed by the most recent layoutViewport. naturalDimsSeq increments whenever
	// they change so the parent can ask "did the modal's natural size shift since
	// last frame?" without diffing the numbers itself — see DimensionsSeq().
	naturalOuterW   int
	naturalOuterH   int
	naturalDimsSeq  int
}

// NewModel creates a file diff modal. zoneManager may be nil (no close button zone).
// headerH covers the in-body header (subtitle + blank separator); the modal title
// itself lives in the bubble-overlay chrome tab (see chromedSlot). footerH covers
// the blank separator + the Esc/scroll hint line at the bottom of the box.
func NewModel(zm *zone.Manager) Model {
	vp := viewport.New(60, 10)
	vp.MouseWheelEnabled = true
	m := Model{zm: zm, termW: 100, termH: 24, vp: vp, headerH: 2, footerH: 2}
	m.layoutViewport()
	return m
}

// SetDimensions records full terminal size for centered modal layout.
func (m Model) SetDimensions(w, h int) Model {
	if w < 1 {
		w = 80
	}
	if h < 1 {
		h = 24
	}
	m.termW, m.termH = w, h
	m.layoutViewport()
	if m.shown && !m.loading && m.errMsg == "" && m.body != "" {
		m.vp.SetContent(StyleGitUnifiedDiff(m.body, m.innerW))
	}
	return m
}

// ShowPreloadedStyledDiff shows a git unified diff that is already loaded (no async jj call).
// title/subtitle appear in the header; empty title defaults to "Patch" in View.
func (m Model) ShowPreloadedStyledDiff(title, subtitle, rawGit string) Model {
	m.shown = true
	m.loading = false
	m.errMsg = ""
	m.seq = -1
	m.body = rawGit
	m.overlayTitle = strings.TrimSpace(title)
	m.overlaySub = strings.TrimSpace(subtitle)
	m.filePath = ""
	m.shortID = ""
	m.vp.GotoTop()
	m.layoutViewport()
	m.vp.SetContent(StyleGitUnifiedDiff(rawGit, m.innerW))
	return m
}

func (m *Model) layoutViewport() {
	maxOuterFromTerm := m.termW - 2*fileDiffTermSideColumns
	if maxOuterFromTerm < 1 {
		maxOuterFromTerm = 1
	}
	innerCap := maxOuterFromTerm - fileDiffOuterInnerDelta
	if innerCap < 1 {
		innerCap = 1
	}

	// Default to "as wide as the diff needs" — long lines push the modal wider,
	// short diffs leave the box hugging the content instead of artificially
	// stretching to a fixed fraction of the terminal. The comfort floor only
	// kicks in for the empty/loading state so the box doesn't pop in tiny and
	// then jump wider once the patch arrives.
	contentWant := fileDiffInnerMinComfort
	if strings.TrimSpace(m.body) != "" {
		if cw := MinInnerWidthForDiffText(m.body); cw > contentWant {
			contentWant = cw
		}
	}
	m.innerW = min(innerCap, contentWant)
	m.outerW = m.innerW + fileDiffOuterInnerDelta
	if m.outerW > maxOuterFromTerm {
		m.outerW = maxOuterFromTerm
		m.innerW = max(1, m.outerW-fileDiffOuterInnerDelta)
	}

	// Reserve room for the chrome rows the bubble-overlay window draws around our
	// box so the modal+chrome together stay inside the terminal; without this
	// reservation the bottom border (and the "Close" footer) scroll off-screen.
	maxOuterH := m.termH - fileDiffChromeRowOverhead - fileDiffBottomSafety
	if maxOuterH < 10 {
		maxOuterH = 10
	}
	// Cap the viewport at whatever rows remain after our own header/footer/border;
	// then shrink further to the actual content line count so a 14-line diff
	// doesn't sit inside a half-empty viewport that stretches to the terminal
	// floor. The View() box drops Height(...) to mirror this — short content
	// produces a short window instead of a fixed-height shell.
	maxBodyH := maxOuterH - m.headerH - m.footerH - 2
	if maxBodyH < fileDiffMinBodyRows {
		maxBodyH = fileDiffMinBodyRows
	}

	desiredBodyH := maxBodyH
	switch {
	case m.body != "":
		desiredBodyH = strings.Count(m.body, "\n") + 1
	case m.errMsg != "":
		desiredBodyH = strings.Count(m.errMsg, "\n") + 1
	case m.loading:
		desiredBodyH = 1
	}
	if desiredBodyH > maxBodyH {
		desiredBodyH = maxBodyH
	}
	if desiredBodyH < fileDiffMinBodyRows {
		desiredBodyH = fileDiffMinBodyRows
	}
	m.innerH = desiredBodyH
	m.vp.Width = m.innerW
	m.vp.Height = m.innerH

	// The lipgloss box renders sub (1) + blank (1) + body (innerH) + blank (1) +
	// footer (1) inside a 2-row border — that's the height we want bubble-overlay
	// to treat as our "natural" content height. Bumping naturalDimsSeq whenever
	// outerW or this derived outerH actually changes lets the parent re-seed the
	// chrome's cached ContentWidth/ContentHeight on real size shifts (loaded
	// patch, terminal resize) without trampling the in-progress drag/resize
	// state on quiet frames.
	naturalH := m.headerH + m.footerH + m.innerH + 2
	if m.outerW != m.naturalOuterW || naturalH != m.naturalOuterH {
		m.naturalOuterW = m.outerW
		m.naturalOuterH = naturalH
		m.naturalDimsSeq++
	}
}

// DimensionsSeq returns a counter that ticks whenever the modal's natural outer
// width or height changes (load complete, terminal resize, etc.). The parent
// model compares it across frames to decide when bubble-overlay's chrome needs
// to re-measure us — see view_helpers.go.
func (m *Model) DimensionsSeq() int { return m.naturalDimsSeq }

// BeginLoad prepares state for an async diff load; returns seq for the command/message.
func (m *Model) BeginLoad(commit internal.Commit, path string) int {
	m.shown = true
	m.loading = true
	m.errMsg = ""
	m.body = ""
	m.shortID = strings.TrimSpace(commit.ShortID)
	m.filePath = strings.TrimSpace(path)
	m.overlayTitle = ""
	m.overlaySub = ""
	m.seq++
	m.vp.SetContent("")
	m.vp.GotoTop()
	m.layoutViewport()
	return m.seq
}

// Hide clears the modal.
func (m *Model) Hide() {
	m.shown = false
	m.loading = false
	m.errMsg = ""
	m.body = ""
	m.seq = 0
	m.overlayTitle = ""
	m.overlaySub = ""
	m.vp.SetContent("")
}

// IsShown reports whether the modal is active.
func (m *Model) IsShown() bool { return m.shown }

// OverlayTitle returns the title set by the caller (e.g. "Evolog step"),
// or "" when no custom title was supplied. Main uses this to drive the
// chromed window's titlebar, falling back to "File diff" when empty.
func (m *Model) OverlayTitle() string { return m.overlayTitle }

// Update handles load result, keys, mouse, and viewport scroll.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.shown {
		return m, nil
	}
	switch msg := msg.(type) {
	case FileDiffLoadedMsg:
		if msg.Seq != m.seq {
			return m, nil
		}
		m.loading = false
		if msg.Err != nil {
			m.errMsg = msg.Err.Error()
			m.body = ""
			m.vp.SetContent("")
			m.layoutViewport()
		} else {
			m.errMsg = ""
			m.body = msg.Text
			m.layoutViewport()
			m.vp.SetContent(StyleGitUnifiedDiff(msg.Text, m.innerW))
			m.vp.GotoTop()
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			m.shown = false
			m.loading = false
			return m, state.NavigateTarget{Kind: state.NavigateCloseFileDiff}.Cmd()
		}
		if m.loading || m.errMsg != "" {
			return m, nil
		}
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd

	case tea.MouseMsg:
		if m.loading || m.errMsg != "" {
			return m, nil
		}
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd

	case zone.MsgZoneInBounds:
		if m.zm == nil || msg.Zone == nil {
			return m, nil
		}
		z := m.zm.Get(mouse.ZoneFileDiffClose)
		if z != nil && z.InBounds(msg.Event) {
			m.shown = false
			m.loading = false
			return m, state.NavigateTarget{Kind: state.NavigateCloseFileDiff}.Cmd()
		}
		return m, nil
	}
	return m, nil
}

// View renders the centered modal.
func (m Model) View() string {
	if !m.shown {
		return ""
	}
	maxOuterW := m.outerW
	if maxOuterW < 1 {
		maxOuterW = 1
	}
	maxOuterH := m.termH - 2
	if maxOuterH < 10 {
		maxOuterH = max(10, m.termH-2)
	}

	var subLine string
	if m.overlaySub != "" {
		subLine = m.overlaySub
	} else {
		pathDisp := m.filePath
		if pathDisp == "" {
			pathDisp = "(unknown path)"
		}
		subLine = fmt.Sprintf("%s  @ %s", pathDisp, m.shortID)
	}
	sub := lipgloss.NewStyle().Foreground(styles.ColorMuted).Width(maxOuterW - fileDiffOuterInnerDelta).Render(subLine)

	var body string
	if m.loading {
		body = lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Loading diff…")
	} else if m.errMsg != "" {
		body = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Width(m.innerW).Render(m.errMsg)
	} else {
		body = m.vp.View()
	}

	closeLabel := "Close"
	if m.zm != nil {
		closeLabel = m.zm.Mark(mouse.ZoneFileDiffClose, styles.ButtonStyle.Render("Close"))
	}
	footer := lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Esc · j/k · PgUp/PgDn scroll  ") + closeLabel

	inner := lipgloss.JoinVertical(lipgloss.Left, sub, "", body, "", footer)
	// Width is fixed (we picked m.outerW to fit the diff); height is left to
	// the content. MaxHeight caps it so an enormous patch still fits the
	// terminal, but a short diff produces a short window — together with the
	// content-driven m.innerH in layoutViewport this means the box hugs the
	// actual line count instead of always claiming the full available height.
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorPrimary).
		Padding(0, 1).
		Width(maxOuterW).
		MaxHeight(maxOuterH).
		Render(inner)

	return box
}
