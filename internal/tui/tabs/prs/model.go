package prs

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/state"
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
	return m.update(msg, nil)
}

// UpdateWithApp handles messages and when app is non-nil runs requests in place and applies effects to app instead of sending Request/effects to main.
func (m Model) UpdateWithApp(msg tea.Msg, app *state.AppState) (Model, tea.Cmd) {
	return m.update(msg, app)
}

func (m Model) update(msg tea.Msg, app *state.AppState) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case PrsLoadedMsg:
		if msg.Prs == nil {
			if app != nil {
				if app.Repository != nil && app.StatusMessage == "" {
					app.StatusMessage = fmt.Sprintf("PRs: %d", len(app.Repository.PRs))
				}
				m.repository = app.Repository
				return m, nil
			}
			return m, ApplyPrsLoadedEffect{Prs: nil, StatusMessage: ""}.Cmd()
		}
		if app != nil {
			if app.Repository != nil {
				app.Repository.PRs = msg.Prs
			}
			app.StatusMessage = fmt.Sprintf("Loaded %d PRs", len(msg.Prs))
			m.repository = app.Repository
			return m, nil
		}
		return m, ApplyPrsLoadedEffect{
			Prs:           msg.Prs,
			StatusMessage: fmt.Sprintf("Loaded %d PRs", len(msg.Prs)),
		}.Cmd()
	case PrMergedMsg:
		if msg.Err != nil {
			if app != nil {
				app.StatusMessage = fmt.Sprintf("Failed to merge PR #%d: %v", msg.PRNumber, msg.Err)
				return m, nil
			}
			return m, ApplyPrMergeClosedEffect{
				Err:           msg.Err,
				StatusMessage: fmt.Sprintf("Failed to merge PR #%d: %v", msg.PRNumber, msg.Err),
			}.Cmd()
		}
		if app != nil {
			app.StatusMessage = fmt.Sprintf("Merged PR #%d", msg.PRNumber)
			existing := 0
			if app.Repository != nil {
				existing = len(app.Repository.PRs)
			}
			return m, LoadPRsCmd(app.GitHubService, app.GithubInfo, app.DemoMode, existing)
		}
		return m, ApplyPrMergeClosedEffect{StatusMessage: fmt.Sprintf("Merged PR #%d", msg.PRNumber)}.Cmd()
	case PrClosedMsg:
		if msg.Err != nil {
			if app != nil {
				app.StatusMessage = fmt.Sprintf("Failed to close PR #%d: %v", msg.PRNumber, msg.Err)
				return m, nil
			}
			return m, ApplyPrMergeClosedEffect{
				Err:           msg.Err,
				StatusMessage: fmt.Sprintf("Failed to close PR #%d: %v", msg.PRNumber, msg.Err),
			}.Cmd()
		}
		if app != nil {
			app.StatusMessage = fmt.Sprintf("Closed PR #%d", msg.PRNumber)
			existing := 0
			if app.Repository != nil {
				existing = len(app.Repository.PRs)
			}
			return m, LoadPRsCmd(app.GitHubService, app.GithubInfo, app.DemoMode, existing)
		}
		return m, ApplyPrMergeClosedEffect{StatusMessage: fmt.Sprintf("Closed PR #%d", msg.PRNumber)}.Cmd()
	case LoadErrorMsg:
		if app != nil {
			app.StatusMessage = fmt.Sprintf("Error: %v", msg.Err)
			return m, nil
		}
		return m, ApplyPrsLoadErrorEffect{Err: msg.Err}.Cmd()
	case ReauthNeededMsg:
		if app != nil {
			return m, nil
		}
		return m, ApplyReauthNeededEffect{Reason: msg.Reason}.Cmd()
	case PrTickInput:
		if msg.HasError || msg.GitHubService == nil {
			return m, nil
		}
		if !msg.IsPRView || msg.Loading {
			if app != nil {
				return m, PrTickCmd()
			}
			return m, ApplyPrTickEffect{RunCmd: PrTickCmd()}.Cmd()
		}
		if app != nil {
			return m, tea.Batch(
				LoadPRsCmd(msg.GitHubService, msg.GithubInfo, msg.DemoMode, msg.ExistingCount),
				PrTickCmd(),
			)
		}
		return m, ApplyPrTickEffect{
			RunCmd: tea.Batch(
				LoadPRsCmd(msg.GitHubService, msg.GithubInfo, msg.DemoMode, msg.ExistingCount),
				PrTickCmd(),
			),
		}.Cmd()

	case tea.WindowSizeMsg:
		return m, nil
	case tea.KeyMsg:
		updated, req, cmd := m.handleKeyMsg(msg)
		if req != nil && app != nil {
			ctx := BuildRequestContextFromApp(app, &updated)
			statusMsg, runCmd := ExecuteRequest(*req, ctx)
			if statusMsg != "" {
				app.StatusMessage = statusMsg
			}
			return updated, runCmd
		}
		if req != nil {
			return updated, req.Cmd()
		}
		return updated, cmd
	case zone.MsgZoneInBounds:
		updated, req, cmd := m.handleZoneClick(msg.Zone)
		if req != nil && app != nil {
			ctx := BuildRequestContextFromApp(app, &updated)
			statusMsg, runCmd := ExecuteRequest(*req, ctx)
			if statusMsg != "" {
				app.StatusMessage = statusMsg
			}
			return updated, runCmd
		}
		if req != nil {
			return updated, req.Cmd()
		}
		return updated, cmd
	case tea.MouseMsg:
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

// handleKeyMsg handles keyboard input; returns (updated model, optional request, cmd).
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, *Request, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.repository != nil && m.selectedPR < len(m.repository.PRs)-1 {
			m.selectedPR++
			m.scrollToSelectedPR = true
		}
		return m, nil, nil
	case "k", "up":
		if m.selectedPR > 0 {
			m.selectedPR--
			m.scrollToSelectedPR = true
		}
		return m, nil, nil
	case "pgup", "ctrl+u", "ctrl+b":
		m.listYOffset -= 10
		if m.listYOffset < 0 {
			m.listYOffset = 0
		}
		return m, nil, nil
	case "pgdown", "ctrl+d", "ctrl+f":
		m.listYOffset += 10
		return m, nil, nil
	case "home":
		m.listYOffset = 0
		return m, nil, nil
	case "end":
		m.listYOffset = 99999
		return m, nil, nil
	case "o", "enter", "e":
		if m.repository != nil && m.selectedPR >= 0 && m.selectedPR < len(m.repository.PRs) {
			return m, &Request{OpenInBrowser: true}, nil
		}
		return m, nil, nil
	case "M":
		if m.repository != nil && m.selectedPR >= 0 && m.selectedPR < len(m.repository.PRs) {
			return m, &Request{MergePR: true}, nil
		}
		return m, nil, nil
	case "X":
		if m.repository != nil && m.selectedPR >= 0 && m.selectedPR < len(m.repository.PRs) {
			return m, &Request{ClosePR: true}, nil
		}
		return m, nil, nil
	}
	return m, nil, nil
}

// handleZoneClick handles zone clicks; returns (updated model, optional request, cmd).
func (m Model) handleZoneClick(z *zone.ZoneInfo) (Model, *Request, tea.Cmd) {
	if m.zoneManager == nil || z == nil {
		return m, nil, nil
	}
	for i := 0; m.repository != nil && i < len(m.repository.PRs); i++ {
		if m.zoneManager.Get(mouse.ZonePR(i)) == z {
			m.selectedPR = i
			return m, nil, nil
		}
	}
	if m.zoneManager.Get(mouse.ZonePROpenBrowser) == z {
		return m, &Request{OpenInBrowser: true}, nil
	}
	if m.zoneManager.Get(mouse.ZonePRMerge) == z {
		return m, &Request{MergePR: true}, nil
	}
	if m.zoneManager.Get(mouse.ZonePRClose) == z {
		return m, &Request{ClosePR: true}, nil
	}
	return m, nil, nil
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
