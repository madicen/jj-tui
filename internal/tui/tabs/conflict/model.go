package conflict

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

// Model represents the bookmark conflict resolution view
type Model struct {
	shown          bool
	bookmarkName   string
	localCommitID  string
	remoteCommitID string
	localSummary   string
	remoteSummary  string
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
	// Cap modal width on large terminals so columns do not stretch across empty space.
	const maxModalOuter = 86
	maxOuter := min(m.termW-6, maxModalOuter)
	if maxOuter < 52 {
		maxOuter = min(m.termW-4, 50)
	}
	sideBySide = m.termW >= 72
	if sideBySide {
		colW = max((maxOuter-2)/2, 24)
	} else {
		colW = max(maxOuter, 32)
	}
	return sideBySide, colW
}

func (m *Model) renderConflict() string {
	sideBySide, colW := m.layoutCols()
	sumMax := max(colW-4, 20)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorPrimary).
		Padding(0, 1)

	localHeader := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#50FA7B")).Render("Local")
	localID := lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render(m.localCommitID)
	localBody := fmt.Sprintf("%s\n%s\n%s", localHeader, localID, truncateSummary(m.localSummary, sumMax))
	localBox := boxStyle.Width(colW).Render(localBody)

	remoteHeader := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8BE9FD")).Render("Remote (origin)")
	remoteID := lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render(m.remoteCommitID)
	remoteBody := fmt.Sprintf("%s\n%s\n%s", remoteHeader, remoteID, truncateSummary(m.remoteSummary, sumMax))
	remoteBox := boxStyle.Width(colW).Render(remoteBody)

	var versionRow string
	if sideBySide {
		gap := lipgloss.NewStyle().Width(2).Render("")
		versionRow = lipgloss.JoinHorizontal(lipgloss.Top, localBox, gap, remoteBox)
	} else {
		versionRow = lipgloss.JoinVertical(lipgloss.Left, localBox, "", remoteBox)
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF5555"))
	title := titleStyle.Render("⚠ Diverged bookmark: " + m.bookmarkName)

	explanationStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	explanation := explanationStyle.Render("Local and origin disagree. Pick a side, then confirm.")

	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary)
	unselectedStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	optBorder := lipgloss.RoundedBorder()

	keepStyle := lipgloss.NewStyle().Border(optBorder).Padding(0, 1).Width(colW)
	if m.selectedOption == 0 {
		keepStyle = keepStyle.BorderForeground(styles.ColorPrimary)
	} else {
		keepStyle = keepStyle.BorderForeground(styles.ColorMuted)
	}
	resetStyle := lipgloss.NewStyle().Border(optBorder).Padding(0, 1).Width(colW)
	if m.selectedOption == 1 {
		resetStyle = resetStyle.BorderForeground(styles.ColorPrimary)
	} else {
		resetStyle = resetStyle.BorderForeground(styles.ColorMuted)
	}

	keepTitle := unselectedStyle
	resetTitle := unselectedStyle
	if m.selectedOption == 0 {
		keepTitle = selectedStyle
	}
	if m.selectedOption == 1 {
		resetTitle = selectedStyle
	}

	keepBlock := keepStyle.Render(fmt.Sprintf(
		"%s\n%s\n%s",
		keepTitle.Render("► Keep local"),
		unselectedStyle.Render("jj bookmark set … then jj git push"),
		unselectedStyle.Render("(overwrites remote if checks pass)"),
	))
	resetBlock := resetStyle.Render(fmt.Sprintf(
		"%s\n%s\n%s",
		resetTitle.Render("► Reset to origin"),
		unselectedStyle.Render("jj bookmark set … @origin"),
		unselectedStyle.Render("discard local tip"),
	))

	var choiceRow string
	if sideBySide {
		gap := lipgloss.NewStyle().Width(2).Render("")
		choiceRow = lipgloss.JoinHorizontal(lipgloss.Top,
			m.mark(mouse.ZoneConflictKeepLocal, keepBlock),
			gap,
			m.mark(mouse.ZoneConflictResetRemote, resetBlock),
		)
	} else {
		choiceRow = lipgloss.JoinVertical(lipgloss.Left,
			m.mark(mouse.ZoneConflictKeepLocal, keepBlock),
			"",
			m.mark(mouse.ZoneConflictResetRemote, resetBlock),
		)
	}

	sepW := lipgloss.Width(versionRow)
	if sepW < 32 {
		sepW = 32
	}
	sep := lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(strings.Repeat("─", min(sepW, m.termW-4)))

	btnRow := lipgloss.JoinHorizontal(lipgloss.Left,
		m.mark(mouse.ZoneConflictConfirm, styles.ButtonStyle.Render("Confirm (Enter)")),
		"  ",
		m.mark(mouse.ZoneConflictCancel, styles.ButtonSecondaryStyle.Render("Cancel (Esc)")),
	)

	hintStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	hint := lipgloss.JoinVertical(
		lipgloss.Left,
		hintStyle.Render("j/k · Enter confirm · Esc cancel"),
		hintStyle.Render("h / ← keep local · l / → / r reset to origin"),
	)

	frame := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		explanation,
		"",
		versionRow,
		"",
		sep,
		"",
		lipgloss.NewStyle().Bold(true).Render("Resolution"),
		"",
		choiceRow,
		"",
		btnRow,
		"",
		hint,
	)

	outer := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorMuted).
		Padding(1, 2)
	return outer.Render(frame)
}

func truncateSummary(summary string, maxLen int) string {
	if len(summary) <= maxLen {
		return summary
	}
	return summary[:maxLen-3] + "..."
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
	return []string{mouse.ZoneConflictKeepLocal, mouse.ZoneConflictResetRemote, mouse.ZoneConflictConfirm, mouse.ZoneConflictCancel}
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
		m.selectedOption = 0
		return m, nil
	case mouse.ZoneConflictResetRemote:
		m.selectedOption = 1
		return m, nil
	case mouse.ZoneConflictConfirm:
		return m, state.NavigateTarget{
			Kind:                 state.NavigateResolveConflict,
			ConflictBookmarkName: m.bookmarkName,
			ConflictResolution:   m.GetSelectedOption(),
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
func (m *Model) Show(bookmarkName, localID, remoteID, localSummary, remoteSummary string) {
	m.shown = true
	m.bookmarkName = bookmarkName
	m.localCommitID = localID
	m.remoteCommitID = remoteID
	m.localSummary = localSummary
	m.remoteSummary = remoteSummary
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
	// Conflict modal doesn't use repository directly
}
