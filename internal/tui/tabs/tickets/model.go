package tickets

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	overlay "github.com/madicen/bubble-overlay"
	"github.com/madicen/jj-tui/internal/tickets"
	"github.com/madicen/jj-tui/internal/tui/keys"
	"github.com/madicen/jj-tui/internal/tui/listnav"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/mousedouble"
	"github.com/madicen/jj-tui/internal/tui/state"
)

// Model represents the state of the Tickets tab
type Model struct {
	listnav.Model // shared list scroll + long-press state

	keys keys.TicketsKeyMap

	zoneManager          *zone.Manager
	ticketList           []tickets.Ticket
	selectedTicket       int
	availableTransitions []tickets.Transition
	transitionInProgress bool
	statusChangeMode     bool
	width                int
	height               int
	providerName         string // e.g. "Jira", "Codecks"
	jiraService          bool   // whether a ticket service is connected
	canCreateTicket      bool   // true when provider supports creating tickets (from TicketsLoadedInput)
	// scrollToSelectedTicket: when true, next render will adjust listYOffset to keep selection in view (key/click only; mouse scroll can move selection off screen)
	scrollToSelectedTicket bool
	loadingTransitions     bool // true while loading available transitions for selected ticket

	contextMenu   *ContextMenuState
	statusSubmenu *StatusSubmenuState

	rowDoubleClick mousedouble.DoubleClick
}

// NewModel creates a new Tickets tab model. zoneManager may be nil (e.g. in tests).
// Default dimensions (80x24) ensure wheel scroll works before first View()/SetDimensions, same as Graph viewports.
func NewModel(zoneManager *zone.Manager) Model {
	return Model{
		Model:          listnav.New(),
		keys:           keys.DefaultTicketsKeyMap(nil),
		zoneManager:    zoneManager,
		selectedTicket: -1,
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

// Update handles messages for the Tickets tab
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m.update(msg, nil)
}

// UpdateWithApp handles messages and when app is non-nil runs requests in place and applies effects to app instead of sending Request/effects to main.
func (m Model) UpdateWithApp(msg tea.Msg, app *state.AppState) (Model, tea.Cmd) {
	return m.update(msg, app)
}

func (m Model) update(msg tea.Msg, app *state.AppState) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case LongPressTickMsg:
		if msg.PressID == m.LongPressPressID && m.LongPressItemIndex >= 0 {
			m.contextMenu = &ContextMenuState{
				TicketIndex: m.LongPressItemIndex,
				MouseX:      m.LongPressMouseX,
				MouseY:      m.LongPressMouseY,
				PressID:     msg.PressID,
				HoverItem:   -1,
			}
			m.selectedTicket = m.LongPressItemIndex
			m.scrollToSelectedTicket = true
		}
		return m, nil

	case TicketsLoadedInput:
		m.UpdateTickets(msg.Tickets)
		m.SetTicketServiceInfo(msg.ProviderName, msg.HasService)
		m.canCreateTicket = msg.CanCreate
		pName := "tickets"
		if msg.HasService && msg.ProviderName != "" {
			pName = msg.ProviderName + " tickets"
		}
		if app != nil {
			app.StatusMessage = fmt.Sprintf("Loaded %d %s", len(msg.Tickets), pName)
			m.SetAvailableTransitions(nil)
			m.SetLoadingTransitions(true)
			return m, LoadTransitionsCmd(app.TicketService, m.GetTickets(), m.GetSelectedTicket())
		}
		return m, ApplyTicketsLoadedEffect{
			StatusMessage: fmt.Sprintf("Loaded %d %s", len(msg.Tickets), pName),
		}.Cmd()
	case TransitionsLoadedMsg:
		m.SetLoadingTransitions(false)
		m.SetAvailableTransitions(msg.Transitions)
		return m, nil
	case TransitionCompletedMsg:
		m.SetTransitionInProgress(false)
		m.SetStatusChangeMode(false)
		m.statusSubmenu = nil
		if msg.Err != nil {
			if app != nil {
				app.StatusMessage = fmt.Sprintf("Failed to transition %s: %v", msg.TicketKey, msg.Err)
				return m, nil
			}
			return m, ApplyTransitionCompletedEffect{
				Err:           msg.Err,
				StatusMessage: fmt.Sprintf("Failed to transition %s: %v", msg.TicketKey, msg.Err),
			}.Cmd()
		}
		reload := msg.NewStatus != ""
		statusMsg := ""
		if msg.NewStatus != "" {
			statusMsg = fmt.Sprintf("Ticket %s transitioned to %s", msg.TicketKey, msg.NewStatus)
		}
		if app != nil {
			app.StatusMessage = statusMsg
			if reload {
				return m, LoadTicketsCmd(app.TicketService, app.DemoMode)
			}
			return m, nil
		}
		return m, ApplyTransitionCompletedEffect{
			StatusMessage: statusMsg,
			ReloadTickets: reload,
		}.Cmd()
	case LoadErrorMsg:
		if app != nil {
			app.StatusMessage = fmt.Sprintf("Error: %v", msg.Err)
			return m, nil
		}
		return m, ApplyTicketsLoadErrorEffect(msg).Cmd()

	case tea.WindowSizeMsg:
		return m, nil
	case tea.KeyMsg:
		updated, req, cmd := m.handleKeyMsg(msg)
		if req != nil && app != nil {
			if req.ToggleStatusChangeMode {
				newMode := !updated.IsStatusChangeMode()
				updated.SetStatusChangeMode(newMode)
				if newMode {
					app.StatusMessage = "Change status (i/D/B/N)"
				} else {
					app.StatusMessage = "Ready"
				}
				return updated, nil
			}
			ctx := BuildRequestContextFromApp(app, &updated)
			statusMsg, runCmd := ExecuteRequest(*req, ctx)
			if statusMsg != "" {
				app.StatusMessage = statusMsg
			}
			if runCmd != nil && req.TransitionID != "" {
				updated.SetTransitionInProgress(true)
			}
			return updated, runCmd
		}
		if req != nil {
			ctx := BuildRequestContextFromApp(app, &updated)
			statusMsg, runCmd := ExecuteRequest(*req, ctx)
			_ = statusMsg
			return updated, runCmd
		}
		return updated, cmd
	case zone.MsgZoneInBounds:
		updated, req, cmd := m.handleZoneClick(msg.Zone, msg.Event)
		if req != nil && app != nil {
			if req.ToggleStatusChangeMode {
				newMode := !updated.IsStatusChangeMode()
				updated.SetStatusChangeMode(newMode)
				if newMode {
					app.StatusMessage = "Change status (i/D/B/N)"
				} else {
					app.StatusMessage = "Ready"
				}
				return updated, nil
			}
			ctx := BuildRequestContextFromApp(app, &updated)
			statusMsg, runCmd := ExecuteRequest(*req, ctx)
			if statusMsg != "" {
				app.StatusMessage = statusMsg
			}
			if runCmd != nil && req.TransitionID != "" {
				updated.SetTransitionInProgress(true)
			}
			return updated, runCmd
		}
		if req != nil {
			ctx := BuildRequestContextFromApp(app, &updated)
			statusMsg, runCmd := ExecuteRequest(*req, ctx)
			_ = statusMsg
			return updated, runCmd
		}
		return updated, cmd
	case tea.MouseMsg:
		if m.WheelScroll(msg) {
			return m, nil
		}
		if cmd := m.handleLongPress(msg); cmd != nil {
			return m, cmd
		}
	}
	return m, nil
}

// View renders the Tickets tab (pointer receiver so render can persist listYOffset clamp)
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	v := m.renderTickets()

	if m.contextMenu != nil {
		menuView := m.renderContextMenu()
		v = overlay.OverlayViewAtPoint(v, menuView, m.width, m.height, m.contextMenu.MouseY, m.contextMenu.MouseX)
	}

	if m.statusSubmenu != nil {
		subView := m.renderStatusPopoverPanel(m.statusSubmenu.HoverItem)
		v = overlay.OverlayViewAtPoint(v, subView, m.width, m.height, m.statusSubmenu.MouseY, m.statusSubmenu.MouseX)
	}

	return v
}

// SetTicketServiceInfo sets provider name and whether a ticket service is connected (used by main model)
func (m *Model) SetTicketServiceInfo(providerName string, connected bool) {
	m.providerName = providerName
	m.jiraService = connected
}

// SetAvailableTransitions sets the available status transitions (called by main model when loaded)
func (m *Model) SetAvailableTransitions(t []tickets.Transition) {
	m.availableTransitions = t
}

// SetTransitionInProgress sets whether a transition is in progress (called by main model)
func (m *Model) SetTransitionInProgress(inProgress bool) {
	m.transitionInProgress = inProgress
}

// SetStatusChangeMode sets whether status change buttons are expanded (called by main model)
func (m *Model) SetStatusChangeMode(mode bool) {
	m.statusChangeMode = mode
}

// GetTransitionInProgress returns whether a transition is in progress (for main model request context)
func (m *Model) GetTransitionInProgress() bool {
	return m.transitionInProgress
}

// SetLoadingTransitions sets whether transitions are being loaded (called by main model)
func (m *Model) SetLoadingTransitions(loading bool) {
	m.loadingTransitions = loading
}

// GetLoadingTransitions returns whether transitions are being loaded
func (m *Model) GetLoadingTransitions() bool {
	return m.loadingTransitions
}

// handleKeyMsg handles keyboard input; returns (updated model, optional request, cmd).
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, *Request, tea.Cmd) {
	if m.statusSubmenu != nil && msg.String() == "esc" {
		m.statusSubmenu = nil
		return m, nil, nil
	}
	if m.contextMenu != nil && msg.String() == "esc" {
		m.contextMenu = nil
		return m, nil, nil
	}
	if msg.String() == "esc" {
		if m.statusChangeMode {
			m.statusChangeMode = false
		}
		return m, nil, nil
	}
	switch {
	case key.Matches(msg, m.keys.MoveDown):
		if m.selectedTicket < len(m.ticketList)-1 {
			m.selectedTicket++
			m.scrollToSelectedTicket = true
			return m, &Request{LoadTransitionsForSelection: true}, nil
		}
		return m, nil, nil
	case key.Matches(msg, m.keys.MoveUp):
		if m.selectedTicket > 0 {
			m.selectedTicket--
			m.scrollToSelectedTicket = true
			return m, &Request{LoadTransitionsForSelection: true}, nil
		}
		return m, nil, nil
	case key.Matches(msg, m.keys.ChangeStatus):
		return m, &Request{ToggleStatusChangeMode: true}, nil
	case key.Matches(msg, m.keys.StatusInProgress):
		return m.tryStatusTransition("i")
	case key.Matches(msg, m.keys.StatusDone):
		return m.tryStatusTransition("D")
	case key.Matches(msg, m.keys.StatusBlocked):
		return m.tryStatusTransition("B")
	case key.Matches(msg, m.keys.StatusNotStarted):
		return m.tryStatusTransition("N")
	case key.Matches(msg, m.keys.Open):
		return m, &Request{OpenInBrowser: true}, nil
	case key.Matches(msg, m.keys.NewTicket):
		if m.canCreateTicket {
			return m, &Request{StartCreateTicket: true}, nil
		}
		return m, nil, nil
	case key.Matches(msg, m.keys.CreateBranch):
		if m.selectedTicket >= 0 && m.selectedTicket < len(m.ticketList) {
			return m, &Request{StartBookmarkFromTicket: true}, nil
		}
		return m, nil, nil
	}
	return m, nil, nil
}

// tryStatusTransition issues a status-change request for the given semantic key
// (see transitionIDByKey) when in status-change mode with a valid selection.
func (m Model) tryStatusTransition(semantic string) (Model, *Request, tea.Cmd) {
	if m.statusChangeMode && !m.transitionInProgress && m.selectedTicket >= 0 && m.selectedTicket < len(m.ticketList) {
		if id := m.transitionIDByKey(semantic); id != "" {
			return m, &Request{TransitionID: id}, nil
		}
	}
	return m, nil, nil
}

func (m *Model) transitionIDByKey(key string) string {
	for _, t := range m.availableTransitions {
		lower := strings.ToLower(t.Name)
		switch key {
		case "i":
			if strings.Contains(lower, "progress") {
				return t.ID
			}
			if strings.Contains(lower, "start") && !strings.Contains(lower, "not start") && !strings.Contains(lower, "not_start") {
				return t.ID
			}
		case "D":
			if strings.Contains(lower, "done") || strings.Contains(lower, "complete") || strings.Contains(lower, "resolve") {
				return t.ID
			}
		case "B":
			if strings.Contains(lower, "block") {
				return t.ID
			}
		case "N":
			if strings.Contains(lower, "not") && strings.Contains(lower, "start") {
				return t.ID
			}
		}
	}
	return ""
}

// handleZoneClick handles zone clicks; returns (updated model, optional request, cmd).
func (m Model) handleZoneClick(z *zone.ZoneInfo, event tea.MouseMsg) (Model, *Request, tea.Cmd) {
	inBounds := func(id string) bool {
		zm := m.zoneManager.Get(id)
		return zm != nil && zm.InBounds(event)
	}

	// Status submenu takes priority over everything else.
	if m.statusSubmenu != nil {
		if inBounds(mouse.ZoneStatusPopoverClose) {
			m.statusSubmenu = nil
			return m, nil, nil
		}
		for i, t := range m.availableTransitions {
			zoneID := mouse.ZoneJiraTransition + fmt.Sprintf("%d", i)
			if inBounds(zoneID) {
				m.statusSubmenu = nil
				req := Request{TransitionID: t.ID}
				return m, &req, nil
			}
		}
		// Don't dismiss unconditionally — AnyInBoundsAndUpdate fires multiple
		// overlapping zones per release (e.g. ticket row + ctx menu item).
		// The second zone fires after the submenu was just created but before
		// it's rendered with transitions. Dismiss only via close button, a
		// transition click, or Esc.
		return m, nil, nil
	}

	if m.contextMenu != nil {
		items := ticketContextMenuItems()
		zoneIdx := 0
		for _, item := range items {
			i := zoneIdx
			zoneIdx++
			if item.RequireCreate && !m.canCreateTicket {
				continue
			}
			if inBounds(mouse.ZoneTicketCtxMenuItem(i)) {
				ti := m.contextMenu.TicketIndex
				mouseX := m.contextMenu.MouseX
				mouseY := m.contextMenu.MouseY
				m.contextMenu = nil
				m.selectedTicket = ti
				m.scrollToSelectedTicket = true

				if item.IsCascade {
					m.statusSubmenu = &StatusSubmenuState{
						MouseX:    mouseX,
						MouseY:    mouseY,
						HoverItem: -1,
					}
					// Always reload transitions to ensure freshness and to return
					// a non-nil cmd that prevents the main model from falling
					// through to handleZoneClick (which would double-process the
					// event and immediately dismiss the just-created submenu).
					return m, &Request{LoadTransitionsForSelection: true}, nil
				}

				req := item.Request
				return m, &req, nil
			}
		}
		m.contextMenu = nil
		return m, nil, nil
	}

	if m.zoneManager == nil || z == nil {
		return m, nil, nil
	}
	for i := range m.ticketList {
		if m.zoneManager.Get(mouse.ZoneJiraTicket(i)) == z {
			m.selectedTicket = i
			m.scrollToSelectedTicket = true
			key := fmt.Sprintf("tickets:%d", i)
			if m.rowDoubleClick.ObserveLeftRelease(key, event, time.Now(), mousedouble.DefaultDoubleClickWindow) {
				return m, &Request{OpenInBrowser: true}, nil
			}
			return m, &Request{LoadTransitionsForSelection: true}, nil
		}
	}
	if m.zoneManager.Get(mouse.ZoneJiraCreateBranch) == z {
		return m, &Request{StartBookmarkFromTicket: true}, nil
	}
	if m.zoneManager.Get(mouse.ZoneTicketNew) == z {
		return m, &Request{StartCreateTicket: true}, nil
	}
	if m.zoneManager.Get(mouse.ZoneJiraChangeStatus) == z {
		return m, &Request{ToggleStatusChangeMode: true}, nil
	}
	if m.zoneManager.Get(mouse.ZoneTicketOpenBrowser) == z {
		return m, &Request{OpenInBrowser: true}, nil
	}
	if m.statusChangeMode && inBounds(mouse.ZoneStatusPopoverClose) {
		return m, &Request{ToggleStatusChangeMode: true}, nil
	}
	if m.statusChangeMode && !m.transitionInProgress && m.selectedTicket >= 0 && m.selectedTicket < len(m.ticketList) {
		for i, t := range m.availableTransitions {
			zoneID := mouse.ZoneJiraTransition + fmt.Sprintf("%d", i)
			if m.zoneManager.Get(zoneID) == z {
				return m, &Request{TransitionID: t.ID}, nil
			}
		}
	}
	return m, nil, nil
}

// Accessors

// GetSelectedTicket returns the index of the selected ticket
func (m *Model) GetSelectedTicket() int {
	return m.selectedTicket
}

// GetListYOffset returns the list scroll offset (for tests and accessors)
func (m *Model) GetListYOffset() int {
	return m.YOffset
}

// SetSelectedTicket sets the selected ticket index
func (m *Model) SetSelectedTicket(idx int) {
	if idx >= 0 && idx < len(m.ticketList) {
		m.selectedTicket = idx
	}
}

// GetTickets returns the ticket list
func (m *Model) GetTickets() []tickets.Ticket {
	return m.ticketList
}

// UpdateTickets updates the ticket list
func (m *Model) UpdateTickets(ticketList []tickets.Ticket) {
	m.ticketList = ticketList
	if len(ticketList) == 0 {
		m.selectedTicket = -1
		return
	}
	if m.selectedTicket < 0 {
		m.selectedTicket = 0
		return
	}
	// After a reload (e.g. ticket transitioned to done and filtered out), selection may be out of bounds
	if m.selectedTicket >= len(ticketList) {
		m.selectedTicket = len(ticketList) - 1
		m.scrollToSelectedTicket = true
	}
}

// P2.8: tickets loads its own data and never used the repository, so the no-op
// UpdateRepository hook was removed rather than renamed — the root no longer fans
// the repository out to tabs that don't consume it.

// GetAvailableTransitions returns available transitions
func (m *Model) GetAvailableTransitions() []tickets.Transition {
	return m.availableTransitions
}

// IsStatusChangeMode returns whether we're in status change mode
func (m *Model) IsStatusChangeMode() bool {
	return m.statusChangeMode
}
