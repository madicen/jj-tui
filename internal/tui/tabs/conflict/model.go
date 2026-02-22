package conflict

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tui/mouse"
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
	statusMessage  string
	zoneManager    *zone.Manager
}

// NewModel creates a new Conflict model. zoneManager may be nil.
func NewModel(zoneManager *zone.Manager) Model {
	return Model{
		shown:          false,
		selectedOption: 0,
		zoneManager:    zoneManager,
	}
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

func (m *Model) renderConflict() string {
	var lines []string

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF5555"))
	lines = append(lines, titleStyle.Render("⚠ Bookmark Conflict: "+m.bookmarkName))
	lines = append(lines, "")

	explanationStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	lines = append(lines, explanationStyle.Render("This bookmark has diverged - local and remote point to different commits."))
	lines = append(lines, "")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorPrimary).
		Padding(0, 1).
		Width(60)

	localHeader := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#50FA7B")).Render("Local Version")
	localCommitID := lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render(m.localCommitID)
	localContent := fmt.Sprintf("%s\n%s\n%s", localHeader, localCommitID, truncateSummary(m.localSummary, 55))
	lines = append(lines, boxStyle.Render(localContent))
	lines = append(lines, "")

	remoteHeader := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8BE9FD")).Render("Remote Version (origin)")
	remoteCommitID := lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render(m.remoteCommitID)
	remoteContent := fmt.Sprintf("%s\n%s\n%s", remoteHeader, remoteCommitID, truncateSummary(m.remoteSummary, 55))
	lines = append(lines, boxStyle.Render(remoteContent))
	lines = append(lines, "")

	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(strings.Repeat("─", 60)))
	lines = append(lines, "")

	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Choose Resolution:"))
	lines = append(lines, "")

	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary)
	unselectedStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)

	keepLocalPrefix := "  "
	keepLocalStyle := unselectedStyle
	if m.selectedOption == 0 {
		keepLocalPrefix = "► "
		keepLocalStyle = selectedStyle
	}
	keepLocalLine := fmt.Sprintf("%s%s", keepLocalPrefix, "Keep Local (force push to remote)")
	lines = append(lines, m.mark(mouse.ZoneConflictKeepLocal, keepLocalStyle.Render(keepLocalLine)))
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Overwrites remote with your local version"))
	lines = append(lines, "")

	resetRemotePrefix := "  "
	resetRemoteStyle := unselectedStyle
	if m.selectedOption == 1 {
		resetRemotePrefix = "► "
		resetRemoteStyle = selectedStyle
	}
	resetRemoteLine := fmt.Sprintf("%s%s", resetRemotePrefix, "Reset to Remote (discard local)")
	lines = append(lines, m.mark(mouse.ZoneConflictResetRemote, resetRemoteStyle.Render(resetRemoteLine)))
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Updates local bookmark to match remote"))
	lines = append(lines, "")

	lines = append(lines, "")
	confirmBtn := m.mark(mouse.ZoneConflictConfirm, styles.ButtonStyle.Render("Confirm (Enter)"))
	cancelBtn := m.mark(mouse.ZoneConflictCancel, styles.ButtonSecondaryStyle.Render("Cancel (Esc)"))
	lines = append(lines, confirmBtn+"  "+cancelBtn)

	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Use j/k or click to select, Enter to confirm"))

	return strings.Join(lines, "\n")
}

func truncateSummary(summary string, maxLen int) string {
	if len(summary) <= maxLen {
		return summary
	}
	return summary[:maxLen-3] + "..."
}

// handleKeyMsg handles keyboard input
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.shown = false
		return m, nil
	case "enter":
		// Signal to resolve - handled outside
		return m, nil
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
