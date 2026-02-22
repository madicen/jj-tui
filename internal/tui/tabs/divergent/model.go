package divergent

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

// Model represents the divergent commit resolution view
type Model struct {
	shown         bool
	changeID      string
	commitIDs     []string
	summaries     []string
	selectedIdx   int
	statusMessage string
	zoneManager   *zone.Manager
}

// NewModel creates a new Divergent model. zoneManager may be nil.
func NewModel(zoneManager *zone.Manager) Model {
	return Model{
		shown:       false,
		selectedIdx: 0,
		zoneManager: zoneManager,
	}
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
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
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

func (m *Model) renderDivergent() string {
	var lines []string

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF79C6"))
	lines = append(lines, titleStyle.Render("⑂ Divergent Commit: "+m.changeID))
	lines = append(lines, "")

	explanationStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	lines = append(lines, explanationStyle.Render("This change ID has multiple versions. Select which one to keep."))
	lines = append(lines, explanationStyle.Render("The other version(s) will be abandoned."))
	lines = append(lines, "")

	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(strings.Repeat("─", 60)))
	lines = append(lines, "")

	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Versions:"))
	lines = append(lines, "")

	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary)
	unselectedStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	commitIDStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))

	for i, commitID := range m.commitIDs {
		prefix := "  "
		style := unselectedStyle
		if i == m.selectedIdx {
			prefix = "► "
			style = selectedStyle
		}

		summary := "(no description)"
		if i < len(m.summaries) {
			summary = m.summaries[i]
		}
		if len(summary) > 50 {
			summary = summary[:47] + "..."
		}

		line := fmt.Sprintf("%s%s  %s",
			prefix,
			commitIDStyle.Render(commitID),
			style.Render(summary),
		)
		lines = append(lines, m.mark(mouse.ZoneDivergentCommit(i), line))
	}

	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(strings.Repeat("─", 60)))
	lines = append(lines, "")

	confirmBtn := m.mark(mouse.ZoneDivergentConfirm, styles.ButtonStyle.Render("Keep Selected (Enter)"))
	cancelBtn := m.mark(mouse.ZoneDivergentCancel, styles.ButtonSecondaryStyle.Render("Cancel (Esc)"))
	lines = append(lines, confirmBtn+"  "+cancelBtn)

	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Use j/k or click to select, Enter to confirm"))

	return strings.Join(lines, "\n")
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
		if m.selectedIdx < len(m.summaries)-1 {
			m.selectedIdx++
		}
		return m, nil
	case "k", "up":
		if m.selectedIdx > 0 {
			m.selectedIdx--
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

// Show displays the divergent modal
func (m *Model) Show(changeID string, commitIDs, summaries []string) {
	m.shown = true
	m.changeID = changeID
	m.commitIDs = commitIDs
	m.summaries = summaries
	m.selectedIdx = 0
}

// Hide hides the modal
func (m *Model) Hide() {
	m.shown = false
}

// GetSelectedCommitID returns the selected commit ID
func (m *Model) GetSelectedCommitID() string {
	if m.selectedIdx < len(m.commitIDs) {
		return m.commitIDs[m.selectedIdx]
	}
	return ""
}

// GetChangeID returns the change ID
func (m *Model) GetChangeID() string {
	return m.changeID
}

// SetSelectedIdx sets the selected commit index
func (m *Model) SetSelectedIdx(idx int) {
	if idx >= 0 && idx < len(m.commitIDs) {
		m.selectedIdx = idx
	}
}

// GetSelectedIdx returns the selected commit index
func (m *Model) GetSelectedIdx() int {
	return m.selectedIdx
}

// GetCommitCount returns the number of divergent commits
func (m *Model) GetCommitCount() int {
	return len(m.commitIDs)
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
	// Divergent modal doesn't use repository directly
}
