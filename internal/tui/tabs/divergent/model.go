package divergent

import (
	"fmt"
	"strconv"
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
	commitIDs   []string
	summaries   []string
	selectedIdx int
	zoneManager *zone.Manager
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

// handleKeyMsg handles keyboard input; returns PerformCancelCmd or PerformResolveCmd for main to handle.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.shown = false
		return m, PerformCancelCmd()
	case "enter":
		keepCommitID := m.GetSelectedCommitID()
		if keepCommitID != "" {
			return m, PerformResolveCmd(m.changeID, keepCommitID)
		}
		return m, nil
	case "j", "down":
		n := len(m.commitIDs)
		if n > 0 && m.selectedIdx < n-1 {
			m.selectedIdx++
		}
		return m, nil
	case "k", "up":
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
		return m, nil
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		idx := int(msg.String()[0] - '1')
		if idx >= 0 && idx < len(m.commitIDs) {
			m.selectedIdx = idx
		}
		return m, nil
	}
	return m, nil
}

// ZoneIDs returns the zone IDs this modal uses when rendering. Used to resolve clicks.
func (m Model) ZoneIDs() []string {
	ids := make([]string, 0, len(m.commitIDs)+2)
	for i := range m.commitIDs {
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
		if err == nil && i >= 0 && i < len(m.commitIDs) {
			m.selectedIdx = i
			return m, nil
		}
	}
	if zoneID == mouse.ZoneDivergentConfirm {
		keepCommitID := m.GetSelectedCommitID()
		if keepCommitID != "" {
			return m, PerformResolveCmd(m.changeID, keepCommitID)
		}
		return m, nil
	}
	if zoneID == mouse.ZoneDivergentCancel {
		m.shown = false
		return m, PerformCancelCmd()
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
