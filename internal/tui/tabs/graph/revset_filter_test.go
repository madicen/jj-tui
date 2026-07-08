package graph

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/mock"
	"github.com/madicen/jj-tui/internal/tui/state"
)

func filterTestModel() GraphModel {
	m := annotateTestModel()
	m.graphFocused = true
	return m
}

func TestFilterInput_OpenClose(t *testing.T) {
	m := filterTestModel()
	app := &state.AppState{}
	if m.FilterInputOpen() {
		t.Fatal("filter input should start closed")
	}
	if cmd := m.beginFilterInput(app); cmd == nil {
		t.Fatal("beginFilterInput should return blink cmd")
	}
	if !m.FilterInputOpen() {
		t.Fatal("filter input should be open")
	}
	m.closeFilterInput()
	if m.FilterInputOpen() {
		t.Fatal("filter input should be closed")
	}
}

func TestFilterInput_EscCloses(t *testing.T) {
	m := filterTestModel()
	app := &state.AppState{}
	_ = m.beginFilterInput(app)
	updated, cmd := m.handleFilterInputKey(tea.KeyMsg{Type: tea.KeyEsc}, app)
	if updated.FilterInputOpen() {
		t.Error("Esc should close filter input")
	}
	if cmd != nil {
		t.Error("Esc should not trigger load cmd")
	}
}

func TestFilterInput_EnterCompilesFreeText(t *testing.T) {
	m := filterTestModel()
	app := &state.AppState{JJService: jj.NewServiceWithRunner("/fake", &mock.FakeRunner{})}
	_ = m.beginFilterInput(app)
	m.filterInput.input.SetValue("golden")
	updated, cmd := m.handleFilterInputKey(tea.KeyMsg{Type: tea.KeyEnter}, app)
	if updated.FilterInputOpen() {
		t.Error("Enter should close filter input")
	}
	if cmd == nil {
		t.Fatal("Enter should return load cmd when jj service is available")
	}
}
