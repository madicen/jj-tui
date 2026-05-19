package conflict

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/styles"
	"github.com/mattn/go-runewidth"
)

// Model represents the bookmark conflict resolution view
type Model struct {
	shown          bool
	bookmarkName   string
	localCommitID  string
	remoteCommitID string
	localSummary   string
	remoteSummary  string
	localWhen      string
	remoteWhen     string
	selectedOption int // 0=Keep Local, 1=Reset to Remote
	zoneManager    *zone.Manager
	termW          int
	termH          int
}

// NewModel creates a new Conflict model. zoneManager may be nil.
func NewModel(zoneManager *zone.Manager) Model {
	return Model{
		shown:          false,
		selectedOption: 0,
		zoneManager:    zoneManager,
		termW:          100,
		termH:          24,
	}
}

// SetDimensions records terminal size for layout (overlay + compact side-by-side when wide enough).
func (m Model) SetDimensions(w, h int) Model {
	if w < 1 {
		w = 80
	}
	if h < 1 {
		h = 24
	}
	m.termW, m.termH = w, h
	return m
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the Conflict modal
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.shown {
		return m, nil
	}

	switch msg := msg.(type) {
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

// View renders the Conflict modal
func (m Model) View() string {
	if !m.shown {
		return ""
	}
	return m.renderConflict()
}

// mark wraps content in a zone if zoneManager is set
func (m *Model) mark(id, content string) string {
	if m.zoneManager != nil {
		return m.zoneManager.Mark(id, content)
	}
	return content
}

func (m *Model) layoutCols() (sideBySide bool, colW int) {
	const maxModalOuter = 78
	maxOuter := min(m.termW-6, maxModalOuter)
	if maxOuter < 48 {
		maxOuter = min(m.termW-4, 46)
	}
	sideBySide = m.termW >= 70
	if sideBySide {
		colW = max((maxOuter-2)/2, 26)
	} else {
		colW = max(maxOuter, 32)
	}
	return sideBySide, colW
}

// renderConflict draws the side-by-side keep-local / match-origin choice.
// The window title ("Bookmark conflict: <name>") lives in the chrome tab —
// see chromedSlot — so the body no longer prints its own header line.
func (m *Model) renderConflict() string {
	sideBySide, colW := m.layoutCols()
	sumMax := max(colW-4, 18)

	muted := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	hint := muted.Render("Enter applies highlighted side · click a side to apply · Esc cancel · h / ← / l / →")

	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary)
	unselectedStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	idStyle := lipgloss.NewStyle().Foreground(styles.ColorPrimary)

	localTitle := "Keep local"
	remoteTitle := "Match origin"
	if m.selectedOption == 0 {
		localTitle = "► " + localTitle
	} else {
		remoteTitle = "► " + remoteTitle
	}
	localTitleSt := unselectedStyle
	remoteTitleSt := unselectedStyle
	if m.selectedOption == 0 {
		localTitleSt = selectedStyle
	} else {
		remoteTitleSt = selectedStyle
	}

	localWhenLine := ""
	if strings.TrimSpace(m.localWhen) != "" {
		localWhenLine = unselectedStyle.Render(m.localWhen)
	}
	remoteWhenLine := ""
	if strings.TrimSpace(m.remoteWhen) != "" {
		remoteWhenLine = unselectedStyle.Render(m.remoteWhen)
	}

	localParts := []string{
		localTitleSt.Render(localTitle),
		idStyle.Render(m.localCommitID),
		truncateSummary(m.localSummary, sumMax),
	}
	if localWhenLine != "" {
		localParts = append(localParts, localWhenLine)
	}
	localParts = append(localParts, muted.Render("Your tip wins when you push."))
	localBody := lipgloss.JoinVertical(lipgloss.Left, localParts...)

	remoteParts := []string{
		remoteTitleSt.Render(remoteTitle),
		idStyle.Render(m.remoteCommitID),
		truncateSummary(m.remoteSummary, sumMax),
	}
	if remoteWhenLine != "" {
		remoteParts = append(remoteParts, remoteWhenLine)
	}
	remoteParts = append(remoteParts, muted.Render("Bookmark follows origin; local-only tip is dropped."))
	remoteBody := lipgloss.JoinVertical(lipgloss.Left, remoteParts...)

	localBorder := styles.ColorMuted
	remoteBorder := styles.ColorMuted
	if m.selectedOption == 0 {
		localBorder = styles.ColorPrimary
	} else {
		remoteBorder = styles.ColorPrimary
	}

	localBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(localBorder).
		Padding(0, 1).
		Width(colW).
		Render(localBody)

	remoteBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(remoteBorder).
		Padding(0, 1).
		Width(colW).
		Render(remoteBody)

	var choiceRow string
	if sideBySide {
		gap := lipgloss.NewStyle().Width(2).Render("")
		choiceRow = lipgloss.JoinHorizontal(lipgloss.Top,
			m.mark(mouse.ZoneConflictKeepLocal, localBox),
			gap,
			m.mark(mouse.ZoneConflictResetRemote, remoteBox),
		)
	} else {
		choiceRow = lipgloss.JoinVertical(lipgloss.Left,
			m.mark(mouse.ZoneConflictKeepLocal, localBox),
			"",
			m.mark(mouse.ZoneConflictResetRemote, remoteBox),
		)
	}

	cancel := m.mark(mouse.ZoneConflictCancel, muted.Render("Cancel"))
	frame := lipgloss.JoinVertical(
		lipgloss.Left,
		hint,
		"",
		choiceRow,
		"",
		cancel,
	)

	outer := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorMuted).
		Padding(0, 1)
	return outer.Render(frame)
}

func truncateSummary(summary string, maxW int) string {
	return runewidth.Truncate(summary, maxW, "…")
}

// handleKeyMsg handles keyboard input; returns PerformCancelCmd or PerformResolveCmd for main to handle.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.shown = false
		return m, state.NavigateTarget{Kind: state.NavigateCloseBookmarkConflict, StatusMessage: "Conflict resolution cancelled"}.Cmd()
	case "enter":
		return m, state.NavigateTarget{
			Kind:                 state.NavigateResolveConflict,
			ConflictBookmarkName: m.bookmarkName,
			ConflictResolution:   m.GetSelectedOption(),
		}.Cmd()
	case "j", "down":
		if m.selectedOption < 1 {
			m.selectedOption++
		}
		return m, nil
	case "k", "up":
		if m.selectedOption > 0 {
			m.selectedOption--
		}
		return m, nil
	case "l", "right":
		m.selectedOption = 1
		return m, nil
	case "h", "left":
		m.selectedOption = 0
		return m, nil
	case "r", "R":
		m.selectedOption = 1
		return m, nil
	}
	return m, nil
}

// ZoneIDs returns the zone IDs this modal uses when rendering. Used to resolve clicks.
func (m Model) ZoneIDs() []string {
	return []string{mouse.ZoneConflictKeepLocal, mouse.ZoneConflictResetRemote, mouse.ZoneConflictCancel}
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
	switch zoneID {
	case mouse.ZoneConflictKeepLocal:
		return m, state.NavigateTarget{
			Kind:                 state.NavigateResolveConflict,
			ConflictBookmarkName: m.bookmarkName,
			ConflictResolution:   "keep_local",
		}.Cmd()
	case mouse.ZoneConflictResetRemote:
		return m, state.NavigateTarget{
			Kind:                 state.NavigateResolveConflict,
			ConflictBookmarkName: m.bookmarkName,
			ConflictResolution:   "reset_remote",
		}.Cmd()
	case mouse.ZoneConflictCancel:
		m.shown = false
		return m, state.NavigateTarget{Kind: state.NavigateCloseBookmarkConflict, StatusMessage: "Conflict resolution cancelled"}.Cmd()
	}
	return m, nil
}

// Accessors

// IsShown returns whether the modal is displayed
func (m *Model) IsShown() bool {
	return m.shown
}

// Show displays the conflict modal
func (m *Model) Show(bookmarkName, localID, remoteID, localSummary, remoteSummary, localWhen, remoteWhen string) {
	m.shown = true
	m.bookmarkName = bookmarkName
	m.localCommitID = localID
	m.remoteCommitID = remoteID
	m.localSummary = localSummary
	m.remoteSummary = remoteSummary
	m.localWhen = localWhen
	m.remoteWhen = remoteWhen
	m.selectedOption = 0
}

// Hide hides the modal
func (m *Model) Hide() {
	m.shown = false
}

// GetSelectedOption returns "keep_local" or "reset_remote"
func (m *Model) GetSelectedOption() string {
	if m.selectedOption == 0 {
		return "keep_local"
	}
	return "reset_remote"
}

// GetBookmarkName returns the conflicted bookmark name
func (m *Model) GetBookmarkName() string {
	return m.bookmarkName
}

// SetSelectedOption sets the selected option (0=Keep Local, 1=Reset to Remote)
func (m *Model) SetSelectedOption(opt int) {
	if opt >= 0 && opt <= 1 {
		m.selectedOption = opt
	}
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
	_ = repo
}
