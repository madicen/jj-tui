package divergent

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal"
)

// Model represents the divergent commit resolution view
type Model struct {
	shown         bool
	changeID      string
	commitIDs     []string
	summaries     []string
	selectedIdx   int
	statusMessage string
}

// NewModel creates a new Divergent model
func NewModel() Model {
	return Model{
		shown:       false,
		selectedIdx: 0,
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

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("3")).
		Padding(1, 2).
		MaxWidth(70)

	content := "Divergent Commits (Change: " + m.changeID + ")\n\n"
	content += "Multiple commits with the same change ID exist.\nSelect which one to keep:\n\n"

	for i, summary := range m.summaries {
		marker := " "
		if i == m.selectedIdx {
			marker = ">"
		}
		content += marker + " " + summary + "\n"
	}

	content += "\n(Use j/k to select, enter to keep, esc to cancel)"

	return style.Render(content)
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

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
	// Divergent modal doesn't use repository directly
}
