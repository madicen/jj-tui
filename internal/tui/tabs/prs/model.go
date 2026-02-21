package prs

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
)

// Model represents the state of the PRs tab
type Model struct {
	repository    *internal.Repository
	selectedPR    int // Index of selected PR in the PRs list
	loading       bool
	err           error
	statusMessage string
}

// NewModel creates a new PRs tab model
func NewModel() Model {
	return Model{
		selectedPR: -1,
		loading:    false,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the PRs tab
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}
	return m, nil
}

// View renders the PRs tab
func (m Model) View() string {
	if m.repository == nil {
		return "Loading pull requests..."
	}
	if len(m.repository.PRs) == 0 {
		return "No pull requests found"
	}
	return ""
}

// handleKeyMsg handles keyboard input specific to the PRs tab
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.repository != nil && m.selectedPR < len(m.repository.PRs)-1 {
			m.selectedPR++
		}
		return m, nil
	case "k", "up":
		if m.selectedPR > 0 {
			m.selectedPR--
		}
		return m, nil
	}
	return m, nil
}

// Accessors

// GetSelectedPR returns the index of the selected PR
func (m *Model) GetSelectedPR() int {
	return m.selectedPR
}

// SetSelectedPR sets the selected PR index
func (m *Model) SetSelectedPR(idx int) {
	if m.repository != nil && idx >= 0 && idx < len(m.repository.PRs) {
		m.selectedPR = idx
	}
}

// GetRepository returns the repository
func (m *Model) GetRepository() *internal.Repository {
	return m.repository
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
	m.repository = repo
	if m.repository != nil && m.selectedPR >= len(m.repository.PRs) {
		m.selectedPR = len(m.repository.PRs) - 1
	}
}
