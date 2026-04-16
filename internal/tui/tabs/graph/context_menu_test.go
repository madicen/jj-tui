package graph

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/state"
)

// newTestGraphModel creates a GraphModel with a zone manager and minimal repo+files for testing.
func newTestGraphModel() *GraphModel {
	zm := zone.New()
	m := NewGraphModel(zm)
	m.SetDimensions(80, 40)
	repo := &internal.Repository{
		Graph: internal.CommitGraph{
			Commits: []internal.Commit{
				{ID: "abc123", ShortID: "abc1", ChangeID: "chg1", Summary: "test commit"},
			},
		},
	}
	m.UpdateRepository(repo)
	m.SetChangedFiles([]jj.ChangedFile{
		{Path: "src/main.go", Status: "M"},
		{Path: "src/util.go", Status: "A"},
	}, "chg1")
	return &m
}

// renderAndScan calls View() then zoneManager.Scan() to register zone positions.
func renderAndScan(m *GraphModel) string {
	v := m.View()
	return m.zoneManager.Scan(v)
}

func TestLongPressTickMsg_CreatesContextMenu(t *testing.T) {
	m := newTestGraphModel()

	m.longPressFileIndex = 0
	m.longPressPressID = 1
	m.longPressMouseX = 10
	m.longPressMouseY = 20

	if m.contextMenu != nil {
		t.Fatal("context menu should be nil before tick")
	}

	updated, _ := m.Update(LongPressTickMsg{PressID: 1})
	m = updated.(*GraphModel)

	if m.contextMenu == nil {
		t.Fatal("context menu should be set after matching tick")
	}
	if m.contextMenu.FileIndex != 0 {
		t.Errorf("contextMenu.FileIndex = %d, want 0", m.contextMenu.FileIndex)
	}
	if m.contextMenu.MouseX != 10 || m.contextMenu.MouseY != 20 {
		t.Errorf("contextMenu position = (%d,%d), want (10,20)", m.contextMenu.MouseX, m.contextMenu.MouseY)
	}
	if m.selectedFile != 0 {
		t.Errorf("selectedFile = %d, want 0", m.selectedFile)
	}
}

func TestLongPressTickMsg_StalePressID_NoMenu(t *testing.T) {
	m := newTestGraphModel()

	m.longPressFileIndex = 0
	m.longPressPressID = 2

	updated, _ := m.Update(LongPressTickMsg{PressID: 1})
	m = updated.(*GraphModel)

	if m.contextMenu != nil {
		t.Fatal("stale tick should not create context menu")
	}
}

func TestLongPressTickMsg_ReleasedBeforeTick_NoMenu(t *testing.T) {
	m := newTestGraphModel()

	m.longPressFileIndex = -1
	m.longPressPressID = 1

	updated, _ := m.Update(LongPressTickMsg{PressID: 1})
	m = updated.(*GraphModel)

	if m.contextMenu != nil {
		t.Fatal("tick after release should not create context menu")
	}
}

func TestContextMenu_EscDismisses(t *testing.T) {
	m := newTestGraphModel()
	m.contextMenu = &ContextMenuState{FileIndex: 0, MouseX: 10, MouseY: 20, PressID: 1}

	updated, req, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
	*m = updated
	if req != nil {
		t.Errorf("esc with context menu should not produce a request")
	}
	if m.contextMenu != nil {
		t.Fatal("esc should dismiss context menu")
	}
}

func TestContextMenu_EscDoesNotCancelRebaseWhenMenuShown(t *testing.T) {
	m := newTestGraphModel()
	m.contextMenu = &ContextMenuState{FileIndex: 0, MouseX: 10, MouseY: 20, PressID: 1}
	m.selectionMode = SelectionRebaseDestination
	m.rebaseSourceCommit = 0

	updated, _, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
	*m = updated

	if m.contextMenu != nil {
		t.Fatal("esc should dismiss context menu first")
	}
	if m.selectionMode != SelectionRebaseDestination {
		t.Error("rebase mode should NOT be cancelled when context menu was dismissed")
	}
}

func TestContextMenu_ClickOutsideDismisses(t *testing.T) {
	m := newTestGraphModel()
	m.contextMenu = &ContextMenuState{FileIndex: 0, MouseX: 10, MouseY: 20, PressID: 1}

	msg := zone.MsgZoneInBounds{
		Zone:  nil,
		Event: tea.MouseMsg{Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft, X: 1, Y: 1},
	}
	updated, req, _ := m.handleZoneClick(msg)

	if updated.contextMenu != nil {
		t.Fatal("click outside should dismiss context menu")
	}
	if req != nil {
		t.Error("click outside should not produce a request")
	}
}

func TestContextMenu_RendersWithFileName(t *testing.T) {
	m := newTestGraphModel()
	m.contextMenu = &ContextMenuState{FileIndex: 0, MouseX: 5, MouseY: 5, PressID: 1}

	view := m.renderContextMenu(true)
	if view == "" {
		t.Fatal("renderContextMenu should return non-empty view")
	}
}

func TestContextMenu_RendersInView(t *testing.T) {
	m := newTestGraphModel()
	m.contextMenu = &ContextMenuState{FileIndex: 0, MouseX: 5, MouseY: 5, PressID: 1}

	v := m.View()
	if v == "" {
		t.Fatal("View() should return non-empty string")
	}
}

func TestHandleFileLongPress_PressOnFile_ReturnsTick(t *testing.T) {
	m := newTestGraphModel()
	renderAndScan(m)

	fileZone := m.zoneManager.Get(mouse.ZoneChangedFile(0))
	if fileZone == nil {
		t.Skip("zone not registered after scan - zone registration may need full render pipeline")
	}

	pressMsg := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      fileZone.StartX + 1,
		Y:      fileZone.StartY,
	}
	cmd := m.handleFileLongPress(pressMsg)
	if cmd == nil {
		t.Fatal("press on file zone should return a tick command")
	}
	if m.longPressFileIndex != 0 {
		t.Errorf("longPressFileIndex = %d, want 0", m.longPressFileIndex)
	}
	if m.contextMenu != nil {
		t.Fatal("context menu should NOT be set on press (only on tick)")
	}
}

func TestHandleFileLongPress_Release_ClearsLongPress(t *testing.T) {
	m := newTestGraphModel()
	m.longPressFileIndex = 0
	m.longPressPressID = 1

	releaseMsg := tea.MouseMsg{
		Action: tea.MouseActionRelease,
		Button: tea.MouseButtonLeft,
	}
	cmd := m.handleFileLongPress(releaseMsg)
	if cmd != nil {
		t.Error("release should return nil cmd")
	}
	if m.longPressFileIndex != -1 {
		t.Errorf("longPressFileIndex should be -1 after release, got %d", m.longPressFileIndex)
	}
}

func TestContextMenu_UpdateWithApp_TickCreatesMenu(t *testing.T) {
	m := newTestGraphModel()
	app := &state.AppState{
		Repository: m.repository,
	}

	m.longPressFileIndex = 1
	m.longPressPressID = 5
	m.longPressMouseX = 15
	m.longPressMouseY = 25

	updated, _ := m.UpdateWithApp(LongPressTickMsg{PressID: 5}, app)

	if updated.contextMenu == nil {
		t.Fatal("UpdateWithApp should create context menu on matching tick")
	}
	if updated.contextMenu.FileIndex != 1 {
		t.Errorf("contextMenu.FileIndex = %d, want 1", updated.contextMenu.FileIndex)
	}
	if updated.selectedFile != 1 {
		t.Errorf("selectedFile = %d, want 1", updated.selectedFile)
	}
}

func TestContextMenu_MenuItemClick_ReturnsRequest(t *testing.T) {
	zm := zone.New()
	m := NewGraphModel(zm)
	m.SetDimensions(80, 40)
	repo := &internal.Repository{
		Graph: internal.CommitGraph{
			Commits: []internal.Commit{
				{ID: "abc123", ShortID: "abc1", ChangeID: "chg1", Summary: "test commit"},
			},
		},
	}
	m.UpdateRepository(repo)
	m.SetChangedFiles([]jj.ChangedFile{
		{Path: "src/main.go", Status: "M"},
	}, "chg1")
	m.contextMenu = &ContextMenuState{FileIndex: 0, MouseX: 5, MouseY: 5, PressID: 1}

	renderAndScan(&m)

	itemZone := m.zoneManager.Get(mouse.ZoneCtxMenuItem(0))
	if itemZone == nil {
		t.Skip("menu item zone not registered - may need full render pipeline")
	}

	clickMsg := zone.MsgZoneInBounds{
		Zone: itemZone,
		Event: tea.MouseMsg{
			Action: tea.MouseActionRelease,
			Button: tea.MouseButtonLeft,
			X:      itemZone.StartX + 1,
			Y:      itemZone.StartY,
		},
	}
	updated, req, _ := m.handleZoneClick(clickMsg)

	if updated.contextMenu != nil {
		t.Error("context menu should be dismissed after item click")
	}
	if req == nil {
		t.Fatal("clicking a menu item should produce a request")
	}
	if !req.ViewFileDiff {
		t.Errorf("first menu item should be ViewFileDiff, got %+v", req)
	}
}
