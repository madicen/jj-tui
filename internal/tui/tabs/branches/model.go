package branches

import (
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
)

// Model represents the state of the Branches tab
type Model struct {
	zoneManager    *zone.Manager
	repository     *internal.Repository
	branchList     []internal.Branch
	selectedBranch  int
	width          int
	height         int
	loading        bool
	err            error
	statusMessage  string
}

// NewModel creates a new Branches tab model. zoneManager may be nil (e.g. in tests).
func NewModel(zoneManager *zone.Manager) Model {
	return Model{
		zoneManager:   zoneManager,
		selectedBranch: -1,
		loading:        false,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the Branches tab
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}
	return m, nil
}

// View renders the Branches tab
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	return m.renderBranches()
}

// handleKeyMsg handles keyboard input specific to the Branches tab
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.selectedBranch < len(m.branchList)-1 {
			m.selectedBranch++
		}
		return m, nil
	case "k", "up":
		if m.selectedBranch > 0 {
			m.selectedBranch--
		}
		return m, nil
	}
	return m, nil
}

// Accessors

// GetSelectedBranch returns the index of the selected branch
func (m *Model) GetSelectedBranch() int {
	return m.selectedBranch
}

// SetSelectedBranch sets the selected branch index
func (m *Model) SetSelectedBranch(idx int) {
	if idx >= 0 && idx < len(m.branchList) {
		m.selectedBranch = idx
	}
}

// GetBranches returns the branch list
func (m *Model) GetBranches() []internal.Branch {
	return m.branchList
}

// UpdateBranches updates the branch list
func (m *Model) UpdateBranches(branches []internal.Branch) {
	m.branchList = branches
	if m.selectedBranch < 0 && len(branches) > 0 {
		m.selectedBranch = 0
	}
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
	m.repository = repo
	// Branches are loaded via separate loadBranches() call, not from repository directly
}
