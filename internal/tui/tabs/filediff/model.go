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

// Model is a scrollable full-file diff overlay for one changed file at a commit.
type Model struct {
	shown    bool
	seq      int
	shortID  string
	filePath string
	// When set (ShowPreloadedStyledDiff), View uses these instead of file path + change id.
	overlayTitle string
	overlaySub   string
	loading   bool
	errMsg    string
	body      string
	termW     int
	termH     int
	vp        viewport.Model
	zm        *zone.Manager
	innerW    int
	innerH    int
	headerH   int
	footerH   int
}

// NewModel creates a file diff modal. zoneManager may be nil (no close button zone).
func NewModel(zm *zone.Manager) Model {
	vp := viewport.New(60, 10)
	vp.MouseWheelEnabled = true
	return Model{zm: zm, termW: 100, termH: 24, vp: vp, headerH: 3, footerH: 2}
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
	// Outer box: borders + title lines + footer
	maxOuterW := min(m.termW-2, 110)
	maxOuterH := min(m.termH-2, m.termH-2)
	if maxOuterW < 40 {
		maxOuterW = m.termW - 2
	}
	if maxOuterH < 10 {
		maxOuterH = m.termH - 2
	}
	// inner = outer - 2 border - padding approximation: use outer-4 for text width
	m.innerW = max(36, maxOuterW-4)
	innerBodyH := maxOuterH - m.headerH - m.footerH - 2
	if innerBodyH < 5 {
		innerBodyH = 5
	}
	m.innerH = innerBodyH
	m.vp.Width = m.innerW
	m.vp.Height = m.innerH
}

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
		} else {
			m.errMsg = ""
			m.body = msg.Text
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
	maxOuterW := min(m.termW-2, 110)
	maxOuterH := min(m.termH-2, m.termH-2)
	if maxOuterW < 40 {
		maxOuterW = max(40, m.termW-2)
	}
	if maxOuterH < 10 {
		maxOuterH = max(10, m.termH-2)
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary)
	titleText := m.overlayTitle
	if titleText == "" {
		titleText = "File diff"
	}
	title := titleStyle.Render(titleText)
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
	sub := lipgloss.NewStyle().Foreground(styles.ColorMuted).Width(maxOuterW - 4).Render(subLine)

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

	inner := lipgloss.JoinVertical(lipgloss.Left, title, sub, "", body, "", footer)
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorPrimary).
		Padding(0, 1).
		Width(maxOuterW).
		MaxHeight(maxOuterH).
		Render(inner)

	return box
}
