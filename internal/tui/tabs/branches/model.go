package branches

import (
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tui/mouse"
)

// Model represents the state of the Branches tab
type Model struct {
	zoneManager    *zone.Manager
	repository     *internal.Repository
	branchList     []internal.Branch
	selectedBranch  int
	listYOffset     int // Scroll offset for list (details stay fixed)
	width           int
	height          int
	loading         bool
	err            error
	statusMessage  string
}

// NewModel creates a new Branches tab model. zoneManager may be nil (e.g. in tests).
// Default dimensions (80x24) ensure wheel scroll works before first View()/SetDimensions, same as Graph viewports.
func NewModel(zoneManager *zone.Manager) Model {
	return Model{
		zoneManager:    zoneManager,
		selectedBranch: -1,
		loading:        false,
		width:          80,
		height:         24,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// SetDimensions sets the content area size (used for list-only scrolling)
func (m *Model) SetDimensions(width, height int) {
	m.width = width
	m.height = height
}

// Update handles messages for the Branches tab
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Do not use full window size: we get content-area dimensions from the main model via SetDimensions()
		// so the list uses the correct height (below header, above status bar), same as the Graph tab.
		return m, nil
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tea.MouseMsg:
		// Wheel: IsWheel() + raw X11 fallback so we accept any terminal encoding; scroll without requiring list to be clicked first
		isWheel := tea.MouseEvent(msg).IsWheel() || msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown
		if isWheel {
			isUp := msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelLeft
			if isUp {
				m.listYOffset -= 3
				if m.listYOffset < 0 {
					m.listYOffset = 0
				}
			} else {
				m.listYOffset += 3
			}
			return m, nil
		}
	}
	return m, nil
}

// View renders the Branches tab (pointer receiver so render can persist listYOffset clamp)
func (m *Model) View() string {
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
	case "T":
		return m, Request{TrackBranch: true}.Cmd()
	case "U":
		return m, Request{UntrackBranch: true}.Cmd()
	case "L":
		return m, Request{RestoreLocalBranch: true}.Cmd()
	case "P":
		return m, Request{PushBranch: true}.Cmd()
	case "F":
		return m, Request{FetchAll: true}.Cmd()
	case "c":
		return m, Request{ResolveBookmarkConflict: true}.Cmd()
	case "x":
		return m, Request{DeleteBranchBookmark: true}.Cmd()
	}
	return m, nil
}

// handleZoneClick handles zone clicks; returns a request cmd for actions.
func (m Model) handleZoneClick(z *zone.ZoneInfo) (Model, tea.Cmd) {
	if m.zoneManager == nil || z == nil {
		return m, nil
	}
	for i := range m.branchList {
		if m.zoneManager.Get(mouse.ZoneBranch(i)) == z {
			m.selectedBranch = i
			return m, nil
		}
	}
	if m.zoneManager.Get(mouse.ZoneBranchTrack) == z {
		return m, Request{TrackBranch: true}.Cmd()
	}
	if m.zoneManager.Get(mouse.ZoneBranchUntrack) == z {
		return m, Request{UntrackBranch: true}.Cmd()
	}
	if m.zoneManager.Get(mouse.ZoneBranchRestore) == z {
		return m, Request{RestoreLocalBranch: true}.Cmd()
	}
	if m.zoneManager.Get(mouse.ZoneBranchDelete) == z {
		return m, Request{DeleteBranchBookmark: true}.Cmd()
	}
	if m.zoneManager.Get(mouse.ZoneBranchPush) == z {
		return m, Request{PushBranch: true}.Cmd()
	}
	if m.zoneManager.Get(mouse.ZoneBranchFetch) == z {
		return m, Request{FetchAll: true}.Cmd()
	}
	if m.zoneManager.Get(mouse.ZoneBranchResolveConflict) == z {
		return m, Request{ResolveBookmarkConflict: true}.Cmd()
	}
	return m, nil
}

// Accessors

// GetSelectedBranch returns the index of the selected branch
func (m *Model) GetSelectedBranch() int {
	return m.selectedBranch
}

// GetListYOffset returns the list scroll offset (for tests and accessors)
func (m *Model) GetListYOffset() int {
	return m.listYOffset
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
