package warning

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal"
)

// Model represents the warning modal (e.g., for empty commit descriptions)
type Model struct {
	shown         bool
	title         string
	message       string
	commits       []internal.Commit // Commits with issues
	selectedIdx   int               // Selected commit index
	statusMessage string
}

// NewModel creates a new Warning model
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

// Update handles messages for the Warning modal
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

// View renders the Warning modal
func (m Model) View() string {
	if !m.shown {
		return ""
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("3")).
		Padding(1, 2).
		MaxWidth(70)

	content := m.title + "\n\n" + m.message
	if len(m.commits) > 0 {
		content += "\n\nCommits with issues:\n"
		for i, c := range m.commits {
			marker := " "
			if i == m.selectedIdx {
				marker = ">"
			}
			content += marker + " " + c.Summary + "\n"
		}
		content += "\n(Use j/k to select, enter to edit, esc to cancel)"
	}

	return style.Render(content)
}

// handleKeyMsg handles keyboard input
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.shown = false
		m.commits = nil
		return m, nil
	case "enter":
		// Signal to go to selected commit - handled outside
		return m, nil
	case "j", "down":
		if len(m.commits) > 0 && m.selectedIdx < len(m.commits)-1 {
			m.selectedIdx++
		}
		return m, nil
	case "k", "up":
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
		return m, nil
	case "ctrl+q", "ctrl+c":
		return m, nil
	}
	return m, nil
}

// Accessors

// IsShown returns whether the modal is displayed
func (m *Model) IsShown() bool {
	return m.shown
}

// Show displays the warning modal
func (m *Model) Show(title, message string, commits []internal.Commit) {
	m.shown = true
	m.title = title
	m.message = message
	m.commits = commits
	m.selectedIdx = 0
}

// Hide hides the modal
func (m *Model) Hide() {
	m.shown = false
	m.commits = nil
}

// GetSelectedCommit returns the selected commit
func (m *Model) GetSelectedCommit() *internal.Commit {
	if len(m.commits) > 0 && m.selectedIdx < len(m.commits) {
		return &m.commits[m.selectedIdx]
	}
	return nil
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
	// Warning modal doesn't use repository directly
}
