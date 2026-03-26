package branches

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/util"
	"github.com/madicen/jj-tui/internal/tui/state"
)

// Model represents the state of the Branches tab
type Model struct {
	zoneManager    *zone.Manager
	repository     *internal.Repository
	branchList     []internal.Branch
	selectedBranch  int
	listYOffset     int // Scroll offset for list (details stay fixed)
	width  int
	height int
}

// NewModel creates a new Branches tab model. zoneManager may be nil (e.g. in tests).
// Default dimensions (80x24) ensure wheel scroll works before first View()/SetDimensions, same as Graph viewports.
func NewModel(zoneManager *zone.Manager) Model {
	return Model{
		zoneManager:    zoneManager,
		selectedBranch: -1,
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
	return m.update(msg, nil)
}

// UpdateWithApp handles messages and when app is non-nil runs requests in place (sets status, runs cmds) instead of sending Request/effects to main.
func (m Model) UpdateWithApp(msg tea.Msg, app *state.AppState) (Model, tea.Cmd) {
	return m.update(msg, app)
}

func (m Model) update(msg tea.Msg, app *state.AppState) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case BranchesLoadedInput:
		if msg.Err != nil {
			if app != nil {
				app.StatusMessage = fmt.Sprintf("Failed to load branches: %v", msg.Err)
				return m, nil
			}
			return m, ApplyBranchesLoadedEffect{
				Err:           msg.Err,
				StatusMessage: fmt.Sprintf("Failed to load branches: %v", msg.Err),
			}.Cmd()
		}
		m.UpdateBranches(msg.Branches)
		statusMsg := ""
		if !msg.HasError && !msg.InCreateBookmarkView {
			statusMsg = fmt.Sprintf("Loaded %d branches", len(msg.Branches))
		}
		if app != nil {
			app.StatusMessage = statusMsg
			// When InCreateBookmarkView, caller (main) sets bookmark conflict sources after UpdateWithApp.
			return m, nil
		}
		return m, ApplyBranchesLoadedEffect{
			StatusMessage:        statusMsg,
			InCreateBookmarkView: msg.InCreateBookmarkView,
		}.Cmd()
	case BranchActionMsg:
		if msg.Err != nil {
			if app != nil {
				app.StatusMessage = fmt.Sprintf("Failed to %s branch: %v", msg.Action, msg.Err)
				return m, nil
			}
			return m, ApplyBranchActionEffect{
				Err:           msg.Err,
				StatusMessage: fmt.Sprintf("Failed to %s branch: %v", msg.Action, msg.Err),
			}.Cmd()
		}
		var statusMsg string
		switch msg.Action {
		case "track":
			statusMsg = fmt.Sprintf("Now tracking branch %s", msg.Branch)
		case "untrack":
			statusMsg = fmt.Sprintf("Stopped tracking branch %s", msg.Branch)
		case "restore":
			statusMsg = fmt.Sprintf("Restored local branch %s", msg.Branch)
		case "delete":
			statusMsg = fmt.Sprintf("Deleted bookmark %s", msg.Branch)
		case "push":
			statusMsg = fmt.Sprintf("Pushed branch %s to remote", msg.Branch)
		case "fetch":
			statusMsg = "Fetched from all remotes"
		default:
			statusMsg = ""
		}
		if app != nil {
			app.StatusMessage = statusMsg
			// Main adds reload cmd (LoadBranches + LoadRepository) when handling BranchActionMsg.
			return m, nil
		}
		return m, ApplyBranchActionEffect{StatusMessage: statusMsg}.Cmd()

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
			if req.FetchAll && runCmd != nil {
				app.BranchRemoteFetchPending = true
				app.Loading = true
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
			if req.FetchAll && runCmd != nil {
				app.BranchRemoteFetchPending = true
				app.Loading = true
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

// View renders the Branches tab (pointer receiver so render can persist listYOffset clamp)
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	return m.renderBranches()
}

// handleKeyMsg handles keyboard input; returns (updated model, optional request, cmd).
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, *Request, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.selectedBranch < len(m.branchList)-1 {
			m.selectedBranch++
		}
		return m, nil, nil
	case "k", "up":
		if m.selectedBranch > 0 {
			m.selectedBranch--
		}
		return m, nil, nil
	case "T":
		return m, &Request{TrackBranch: true}, nil
	case "U":
		return m, &Request{UntrackBranch: true}, nil
	case "L":
		return m, &Request{RestoreLocalBranch: true}, nil
	case "P":
		return m, &Request{PushBranch: true}, nil
	case "F":
		return m, &Request{FetchAll: true}, nil
	case "c":
		return m, &Request{ResolveBookmarkConflict: true}, nil
	case "x":
		return m, &Request{DeleteBranchBookmark: true}, nil
	}
	return m, nil, nil
}

// handleZoneClick handles zone clicks; returns (updated model, optional request, cmd).
func (m Model) handleZoneClick(z *zone.ZoneInfo) (Model, *Request, tea.Cmd) {
	if m.zoneManager == nil || z == nil {
		return m, nil, nil
	}
	for i := range m.branchList {
		if m.zoneManager.Get(mouse.ZoneBranch(i)) == z {
			m.selectedBranch = i
			return m, nil, nil
		}
	}
	if m.zoneManager.Get(mouse.ZoneBranchTrack) == z {
		return m, &Request{TrackBranch: true}, nil
	}
	if m.zoneManager.Get(mouse.ZoneBranchUntrack) == z {
		return m, &Request{UntrackBranch: true}, nil
	}
	if m.zoneManager.Get(mouse.ZoneBranchRestore) == z {
		return m, &Request{RestoreLocalBranch: true}, nil
	}
	if m.zoneManager.Get(mouse.ZoneBranchDelete) == z {
		return m, &Request{DeleteBranchBookmark: true}, nil
	}
	if m.zoneManager.Get(mouse.ZoneBranchPush) == z {
		return m, &Request{PushBranch: true}, nil
	}
	if m.zoneManager.Get(mouse.ZoneBranchFetch) == z {
		return m, &Request{FetchAll: true}, nil
	}
	if m.zoneManager.Get(mouse.ZoneBranchResolveConflict) == z {
		return m, &Request{ResolveBookmarkConflict: true}, nil
	}
	return m, nil, nil
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

// BuildBookmarkNameConflictSources returns branch names and all commit branch names, for the bookmark modal's "name exists" check. Uses the tab's own repository and branch list (same data as appState, kept in sync by main).
func (m *Model) BuildBookmarkNameConflictSources() []string {
	var names []string
	for _, b := range m.branchList {
		names = append(names, b.Name)
	}
	if m.repository != nil {
		for _, commit := range m.repository.Graph.Commits {
			for _, br := range commit.Branches {
				names = append(names, br)
				if loc := util.LocalBookmarkName(br); loc != "" && loc != br {
					names = append(names, loc)
				}
			}
		}
	}
	return names
}
