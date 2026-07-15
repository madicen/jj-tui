package model

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/genmenu"
	"github.com/madicen/jj-tui/internal/tui/state"
	bookmarktab "github.com/madicen/jj-tui/internal/tui/tabs/bookmark"
	descedittab "github.com/madicen/jj-tui/internal/tui/tabs/descedit"
	prformtab "github.com/madicen/jj-tui/internal/tui/tabs/prform"
	ticketformtab "github.com/madicen/jj-tui/internal/tui/tabs/ticketform"
)

// formModal is the genmenu surface shared by the four form modals.
type formModal interface {
	Update(tea.Msg) tea.Cmd
	MenuState() *genmenu.State
	MenuOverlay() (string, int, int)
}

// formModalHandle adapts a modal field (value-returning Update) to formModal.
type formModalHandle struct {
	update  func(tea.Msg) tea.Cmd
	menu    func() *genmenu.State
	overlay func() (string, int, int)
}

func (h formModalHandle) Update(msg tea.Msg) tea.Cmd  { return h.update(msg) }
func (h formModalHandle) MenuState() *genmenu.State   { return h.menu() }
func (h formModalHandle) MenuOverlay() (string, int, int) {
	return h.overlay()
}

func bindForm[T any](
	ptr *T,
	update func(T, tea.Msg) (T, tea.Cmd),
	menu func(*T) *genmenu.State,
	overlay func(*T) (string, int, int),
) formModal {
	return formModalHandle{
		update: func(msg tea.Msg) tea.Cmd {
			u, cmd := update(*ptr, msg)
			*ptr = u
			return cmd
		},
		menu:    func() *genmenu.State { return menu(ptr) },
		overlay: func() (string, int, int) { return overlay(ptr) },
	}
}

// activeFormModal returns the form modal for the current ViewMode, or nil.
func (m *Model) activeFormModal() formModal {
	switch m.appState.ViewMode {
	case state.ViewEditDescription:
		return bindForm(&m.desceditModal, descedittab.Model.Update, (*descedittab.Model).MenuState, (*descedittab.Model).MenuOverlay)
	case state.ViewCreatePR:
		return bindForm(&m.prFormModal, prformtab.Model.Update, (*prformtab.Model).MenuState, (*prformtab.Model).MenuOverlay)
	case state.ViewCreateBookmark:
		return bindForm(&m.bookmarkModal, bookmarktab.Model.Update, (*bookmarktab.Model).MenuState, (*bookmarktab.Model).MenuOverlay)
	case state.ViewCreateTicket:
		return bindForm(&m.ticketFormModal, ticketformtab.Model.Update, (*ticketformtab.Model).MenuState, (*ticketformtab.Model).MenuOverlay)
	}
	return nil
}

func (m *Model) activeFormModalGenMenu() *genmenu.State {
	if f := m.activeFormModal(); f != nil {
		return f.MenuState()
	}
	return nil
}

func (m *Model) forwardMouseToActiveFormModal(msg tea.MouseMsg) tea.Cmd {
	if f := m.activeFormModal(); f != nil {
		return f.Update(msg)
	}
	return nil
}

func (m *Model) forwardGenMenuTickToActiveFormModal(msg genmenu.TickMsg) tea.Cmd {
	if f := m.activeFormModal(); f != nil {
		return f.Update(msg)
	}
	return nil
}

func (m *Model) activeFormModalGenMenuOverlay() (string, int, int) {
	if f := m.activeFormModal(); f != nil {
		return f.MenuOverlay()
	}
	return "", 0, 0
}
