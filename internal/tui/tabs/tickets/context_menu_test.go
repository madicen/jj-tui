package tickets

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/tickets"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/state"
)

func newTestModel() Model {
	zm := zone.New()
	m := NewModel(zm)
	m.SetDimensions(80, 40)
	m.UpdateTickets([]tickets.Ticket{
		{Key: "PROJ-1", DisplayKey: "PROJ-1", Summary: "First ticket", Status: "Open"},
		{Key: "PROJ-2", DisplayKey: "PROJ-2", Summary: "Second ticket", Status: "In Progress"},
		{Key: "PROJ-3", DisplayKey: "PROJ-3", Summary: "Third ticket", Status: "Done"},
	})
	m.SetTicketServiceInfo("Jira", true)
	m.canCreateTicket = true
	return m
}

func renderAndScan(m *Model) string {
	v := m.View()
	scanned := m.zoneManager.Scan(v)
	time.Sleep(50 * time.Millisecond)
	return scanned
}

func TestLongPressTickMsg_CreatesContextMenu(t *testing.T) {
	m := newTestModel()
	m.longPressItemIndex = 0
	m.longPressPressID = 1
	m.longPressMouseX = 10
	m.longPressMouseY = 5

	updated, _ := m.Update(LongPressTickMsg{PressID: 1})
	m = updated

	if m.contextMenu == nil {
		t.Fatal("context menu should be set after matching tick")
	}
	if m.contextMenu.TicketIndex != 0 {
		t.Errorf("contextMenu.TicketIndex = %d, want 0", m.contextMenu.TicketIndex)
	}
	if m.contextMenu.MouseX != 10 || m.contextMenu.MouseY != 5 {
		t.Errorf("contextMenu position = (%d,%d), want (10,5)", m.contextMenu.MouseX, m.contextMenu.MouseY)
	}
}

func TestContextMenu_EscDismisses(t *testing.T) {
	m := newTestModel()
	m.contextMenu = &ContextMenuState{TicketIndex: 0, MouseX: 10, MouseY: 5, PressID: 1, HoverItem: -1}

	updated, _, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})

	if updated.contextMenu != nil {
		t.Fatal("Esc should dismiss context menu")
	}
}

func TestStatusSubmenu_EscDismisses(t *testing.T) {
	m := newTestModel()
	m.statusSubmenu = &StatusSubmenuState{MouseX: 10, MouseY: 5, HoverItem: -1}

	updated, _, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})

	if updated.statusSubmenu != nil {
		t.Fatal("Esc should dismiss status submenu")
	}
}

func TestChangeStatus_CascadeFromContextMenu(t *testing.T) {
	m := newTestModel()
	m.selectedTicket = 1
	m.contextMenu = &ContextMenuState{TicketIndex: 1, MouseX: 10, MouseY: 5, PressID: 1, HoverItem: -1}

	renderAndScan(&m)

	itemZone := m.zoneManager.Get(mouse.ZoneTicketCtxMenuItem(2))
	if itemZone == nil {
		t.Skip("menu item zone not registered after scan - zone registration may need full render pipeline")
	}

	event := tea.MouseMsg{
		Action: tea.MouseActionRelease,
		Button: tea.MouseButtonLeft,
		X:      itemZone.StartX + 1,
		Y:      itemZone.StartY,
	}
	updated, req, _ := m.handleZoneClick(itemZone, event)

	if updated.contextMenu != nil {
		t.Error("context menu should be cleared after clicking Change Status")
	}
	if updated.statusSubmenu == nil {
		t.Fatal("statusSubmenu should be created after clicking Change Status")
	}
	if req == nil {
		t.Fatal("should return a request for LoadTransitionsForSelection")
	}
	if !req.LoadTransitionsForSelection {
		t.Errorf("request should have LoadTransitionsForSelection=true, got %+v", req)
	}
}

// TestChangeStatus_SubmenuSurvivesDoubleZoneHit simulates what happens when
// AnyInBoundsAndUpdate fires multiple overlapping zones for the same release.
// The first zone (ticket row) correctly triggers the cascade. The second zone
// (ctx menu item itself) must NOT destroy the just-created submenu.
func TestChangeStatus_SubmenuSurvivesDoubleZoneHit(t *testing.T) {
	m := newTestModel()
	m.selectedTicket = 1
	m.contextMenu = &ContextMenuState{TicketIndex: 1, MouseX: 10, MouseY: 5, PressID: 1, HoverItem: -1}

	renderAndScan(&m)

	itemZone := m.zoneManager.Get(mouse.ZoneTicketCtxMenuItem(2))
	if itemZone == nil {
		t.Skip("menu item zone not registered after scan")
	}

	event := tea.MouseMsg{
		Action: tea.MouseActionRelease,
		Button: tea.MouseButtonLeft,
		X:      itemZone.StartX + 1,
		Y:      itemZone.StartY,
	}

	// Step 1: First zone hit (simulates the ticket row zone being processed first)
	// This correctly triggers the cascade.
	updated, _, _ := m.handleZoneClick(itemZone, event)

	if updated.statusSubmenu == nil {
		t.Fatal("statusSubmenu should be created after first zone hit")
	}
	if updated.contextMenu != nil {
		t.Fatal("contextMenu should be nil after first zone hit")
	}

	// Step 2: Second zone hit (simulates the ctx menu item zone being processed).
	// This should NOT destroy the submenu.
	updated2, _, _ := updated.handleZoneClick(itemZone, event)

	if updated2.statusSubmenu == nil {
		t.Fatal("statusSubmenu must survive the second zone hit (AnyInBoundsAndUpdate overlap)")
	}
}

// TestFullClickFlow_ChangeStatus emulates a complete mouse interaction:
// long-press to open context menu, then press+release on "Change Status >".
func TestFullClickFlow_ChangeStatus(t *testing.T) {
	m := newTestModel()
	renderAndScan(&m)

	ticketZone := m.zoneManager.Get(mouse.ZoneJiraTicket(1))
	if ticketZone == nil {
		t.Skip("ticket zone not registered after scan")
	}

	// 1. Press on ticket row (starts long-press timer)
	pressMsg := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      ticketZone.StartX + 1,
		Y:      ticketZone.StartY,
	}
	cmd := m.handleLongPress(pressMsg)
	if cmd == nil {
		t.Fatal("press on ticket should return a tick command")
	}
	if m.longPressItemIndex != 1 {
		t.Fatalf("longPressItemIndex = %d, want 1", m.longPressItemIndex)
	}

	// 2. Long-press tick fires → creates context menu
	updated, _ := m.Update(LongPressTickMsg{PressID: m.longPressPressID})
	m = updated
	if m.contextMenu == nil {
		t.Fatal("context menu should be created after long-press tick")
	}

	// 3. Re-render to register context menu zones
	renderAndScan(&m)

	changeStatusZone := m.zoneManager.Get(mouse.ZoneTicketCtxMenuItem(2))
	if changeStatusZone == nil {
		t.Skip("Change Status zone not registered after scan")
	}

	// 4. Press on "Change Status >" — should not dismiss menu
	pressOnMenu := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      changeStatusZone.StartX + 1,
		Y:      changeStatusZone.StartY,
	}
	cmd = m.handleLongPress(pressOnMenu)
	if cmd != nil {
		t.Error("press on context menu should not start a new long-press")
	}
	if m.contextMenu == nil {
		t.Fatal("context menu should still be open after press on it")
	}

	// 5. Release on "Change Status >" → zone click fires
	releaseOnMenu := tea.MouseMsg{
		Action: tea.MouseActionRelease,
		Button: tea.MouseButtonLeft,
		X:      changeStatusZone.StartX + 1,
		Y:      changeStatusZone.StartY,
	}
	m.handleLongPress(releaseOnMenu)

	// Simulate zone click (what AnyInBoundsAndUpdate would fire)
	updated2, req, _ := m.handleZoneClick(changeStatusZone, releaseOnMenu)

	if updated2.contextMenu != nil {
		t.Error("contextMenu should be cleared after Change Status click")
	}
	if updated2.statusSubmenu == nil {
		t.Fatal("statusSubmenu should be created after Change Status click")
	}
	if req == nil || !req.LoadTransitionsForSelection {
		t.Fatalf("expected LoadTransitionsForSelection request, got %+v", req)
	}

	// 6. Simulate second zone hit from AnyInBoundsAndUpdate overlap
	updated3, _, _ := updated2.handleZoneClick(changeStatusZone, releaseOnMenu)
	if updated3.statusSubmenu == nil {
		t.Fatal("statusSubmenu must survive overlapping zone hit from AnyInBoundsAndUpdate")
	}
}

func TestHandleLongPress_PressWhileMenuOpen_ReturnsNil(t *testing.T) {
	m := newTestModel()
	m.contextMenu = &ContextMenuState{TicketIndex: 0, MouseX: 10, MouseY: 5, PressID: 1, HoverItem: -1}

	pressMsg := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      50,
		Y:      20,
	}
	cmd := m.handleLongPress(pressMsg)
	if cmd != nil {
		t.Error("press while menu is open should return nil (no new long-press)")
	}
}

func TestHandleLongPress_PressWhileSubmenuOpen_ReturnsNil(t *testing.T) {
	m := newTestModel()
	m.statusSubmenu = &StatusSubmenuState{MouseX: 10, MouseY: 5, HoverItem: -1}

	pressMsg := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      50,
		Y:      20,
	}
	cmd := m.handleLongPress(pressMsg)
	if cmd != nil {
		t.Error("press while submenu is open should return nil")
	}
}

func TestStatusSubmenu_CloseButtonDismisses(t *testing.T) {
	m := newTestModel()
	m.statusSubmenu = &StatusSubmenuState{MouseX: 10, MouseY: 5, HoverItem: -1}
	m.availableTransitions = []tickets.Transition{
		{ID: "11", Name: "In Progress"},
		{ID: "21", Name: "Done"},
	}

	renderAndScan(&m)

	closeZone := m.zoneManager.Get(mouse.ZoneStatusPopoverClose)
	if closeZone == nil {
		t.Skip("close button zone not registered after scan")
	}

	event := tea.MouseMsg{
		Action: tea.MouseActionRelease,
		Button: tea.MouseButtonLeft,
		X:      closeZone.StartX + 1,
		Y:      closeZone.StartY,
	}
	updated, req, _ := m.handleZoneClick(closeZone, event)

	if updated.statusSubmenu != nil {
		t.Fatal("close button should dismiss status submenu")
	}
	if req != nil {
		t.Error("close button should not produce a request")
	}
}

func TestStatusSubmenu_TransitionClick(t *testing.T) {
	m := newTestModel()
	m.selectedTicket = 0
	m.statusSubmenu = &StatusSubmenuState{MouseX: 10, MouseY: 5, HoverItem: -1}
	m.availableTransitions = []tickets.Transition{
		{ID: "11", Name: "In Progress"},
		{ID: "21", Name: "Done"},
	}

	renderAndScan(&m)

	transZone := m.zoneManager.Get(mouse.ZoneJiraTransition + "0")
	if transZone == nil {
		t.Skip("transition zone not registered after scan")
	}

	event := tea.MouseMsg{
		Action: tea.MouseActionRelease,
		Button: tea.MouseButtonLeft,
		X:      transZone.StartX + 1,
		Y:      transZone.StartY,
	}
	updated, req, _ := m.handleZoneClick(transZone, event)

	if updated.statusSubmenu != nil {
		t.Fatal("transition click should dismiss status submenu")
	}
	if req == nil {
		t.Fatal("transition click should produce a request")
	}
	if req.TransitionID != "11" {
		t.Errorf("expected TransitionID '11', got '%s'", req.TransitionID)
	}
}

// TestContextMenu_ClickOutsideDismisses verifies that a zone click outside the
// context menu items dismisses the menu.
func TestContextMenu_ClickOutsideDismisses(t *testing.T) {
	m := newTestModel()
	m.contextMenu = &ContextMenuState{TicketIndex: 0, MouseX: 10, MouseY: 5, PressID: 1, HoverItem: -1}

	renderAndScan(&m)

	ticketZone := m.zoneManager.Get(mouse.ZoneJiraTicket(2))
	if ticketZone == nil {
		t.Skip("ticket zone not registered")
	}

	event := tea.MouseMsg{
		Action: tea.MouseActionRelease,
		Button: tea.MouseButtonLeft,
		X:      ticketZone.StartX + 1,
		Y:      ticketZone.StartY,
	}
	updated, _, _ := m.handleZoneClick(ticketZone, event)

	if updated.contextMenu != nil {
		t.Fatal("clicking outside context menu items should dismiss the menu")
	}
}

// TestFullClickFlow_WithAppState emulates the full flow through UpdateWithApp
// to verify the cascade works with the app-aware code path.
func TestFullClickFlow_WithAppState(t *testing.T) {
	m := newTestModel()
	m.selectedTicket = 1
	m.contextMenu = &ContextMenuState{TicketIndex: 1, MouseX: 10, MouseY: 5, PressID: 1, HoverItem: -1}

	renderAndScan(&m)

	itemZone := m.zoneManager.Get(mouse.ZoneTicketCtxMenuItem(2))
	if itemZone == nil {
		t.Skip("Change Status zone not registered")
	}

	app := &state.AppState{
		TicketService: nil, // no service — simulates cmd being nil
	}

	event := tea.MouseMsg{
		Action: tea.MouseActionRelease,
		Button: tea.MouseButtonLeft,
		X:      itemZone.StartX + 1,
		Y:      itemZone.StartY,
	}

	// Process via UpdateWithApp (as the main model would)
	updated, _ := m.UpdateWithApp(zone.MsgZoneInBounds{Zone: itemZone, Event: event}, app)
	m = updated

	if m.contextMenu != nil {
		t.Error("contextMenu should be cleared")
	}
	if m.statusSubmenu == nil {
		t.Fatal("statusSubmenu should be created via UpdateWithApp")
	}

	// Simulate the second zone hit from AnyInBoundsAndUpdate
	updated2, _ := m.UpdateWithApp(zone.MsgZoneInBounds{Zone: itemZone, Event: event}, app)

	if updated2.statusSubmenu == nil {
		t.Fatal("statusSubmenu must survive the second zone hit through UpdateWithApp")
	}
}
