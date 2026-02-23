package prs

import (
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tui/mouse"
)

// Model represents the state of the PRs tab
type Model struct {
	zoneManager   *zone.Manager
	repository    *internal.Repository
	selectedPR    int // Index of selected PR in the PRs list
	listYOffset   int // Scroll offset for list (details stay fixed)
	width           int
	height          int
	githubService   bool // whether GitHub is connected (for rendering)
	// scrollToSelectedPR: when true, next render will adjust listYOffset to keep selection in view (key/click only; mouse scroll can move selection off screen)
	scrollToSelectedPR bool
}

// NewModel creates a new PRs tab model. zoneManager may be nil (e.g. in tests).
// Default dimensions (80x24) ensure wheel scroll works before first View()/SetDimensions, same as Graph viewports.
func NewModel(zoneManager *zone.Manager) Model {
	return Model{
		zoneManager: zoneManager,
		selectedPR:  -1,
		width:       80,
		height:      24,
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

// Update handles messages for the PRs tab
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Do not use full window size: we get content-area dimensions from the main model via SetDimensions()
		// so the list uses the correct height (below header, above status bar), same as the Graph tab.
		return m, nil
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case zone.MsgZoneInBounds:
		return m.handleZoneClick(msg.Zone)
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

// View renders the PRs tab (pointer receiver so render can persist listYOffset clamp)
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	if m.repository == nil {
		return "Loading pull requests..."
	}
	return m.renderPRs()
}

// SetGithubService sets whether GitHub is connected (used by main model when rendering)
func (m *Model) SetGithubService(connected bool) {
	m.githubService = connected
}

// handleKeyMsg handles keyboard input specific to the PRs tab
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.repository != nil && m.selectedPR < len(m.repository.PRs)-1 {
			m.selectedPR++
			m.scrollToSelectedPR = true
		}
		return m, nil
	case "k", "up":
		if m.selectedPR > 0 {
			m.selectedPR--
			m.scrollToSelectedPR = true
		}
		return m, nil
	case "pgup", "ctrl+u", "ctrl+b":
		m.listYOffset -= 10
		if m.listYOffset < 0 {
			m.listYOffset = 0
		}
		return m, nil
	case "pgdown", "ctrl+d", "ctrl+f":
		m.listYOffset += 10
		return m, nil
	case "home":
		m.listYOffset = 0
		return m, nil
	case "end":
		m.listYOffset = 99999
		return m, nil
	case "o", "enter", "e":
		if m.repository != nil && m.selectedPR >= 0 && m.selectedPR < len(m.repository.PRs) {
			return m, Request{OpenInBrowser: true}.Cmd()
		}
		return m, nil
	case "M":
		if m.repository != nil && m.selectedPR >= 0 && m.selectedPR < len(m.repository.PRs) {
			return m, Request{MergePR: true}.Cmd()
		}
		return m, nil
	case "X":
		if m.repository != nil && m.selectedPR >= 0 && m.selectedPR < len(m.repository.PRs) {
			return m, Request{ClosePR: true}.Cmd()
		}
		return m, nil
	}
	return m, nil
}

// handleZoneClick handles zone clicks; returns a request cmd for actions.
func (m Model) handleZoneClick(z *zone.ZoneInfo) (Model, tea.Cmd) {
	if m.zoneManager == nil || z == nil {
		return m, nil
	}
	for i := 0; m.repository != nil && i < len(m.repository.PRs); i++ {
		if m.zoneManager.Get(mouse.ZonePR(i)) == z {
			m.selectedPR = i
			return m, nil
		}
	}
	if m.zoneManager.Get(mouse.ZonePROpenBrowser) == z {
		return m, Request{OpenInBrowser: true}.Cmd()
	}
	if m.zoneManager.Get(mouse.ZonePRMerge) == z {
		return m, Request{MergePR: true}.Cmd()
	}
	if m.zoneManager.Get(mouse.ZonePRClose) == z {
		return m, Request{ClosePR: true}.Cmd()
	}
	return m, nil
}

// Accessors

// GetSelectedPR returns the index of the selected PR
func (m *Model) GetSelectedPR() int {
	return m.selectedPR
}

// GetListYOffset returns the list scroll offset (for tests and accessors)
func (m *Model) GetListYOffset() int {
	return m.listYOffset
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

// UpdateRepository updates the repository and auto-selects the first PR when the list loads or changes.
func (m *Model) UpdateRepository(repo *internal.Repository) {
	m.repository = repo
	if m.repository == nil {
		return
	}
	n := len(m.repository.PRs)
	if n == 0 {
		m.selectedPR = -1
		return
	}
	// Keep selection in range; if none or invalid, select first
	if m.selectedPR < 0 || m.selectedPR >= n {
		m.selectedPR = 0
		return
	}
	m.selectedPR = min(m.selectedPR, n-1)
}
