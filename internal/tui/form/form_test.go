package form

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func newTestForm(n int) Model {
	inputs := make([]textinput.Model, n)
	for i := range inputs {
		inputs[i] = textinput.New()
	}
	return New(inputs...)
}

func TestNewFocusesFirst(t *testing.T) {
	m := newTestForm(3)
	if m.Focused() != 0 {
		t.Fatalf("Focused() = %d, want 0", m.Focused())
	}
	if !m.Input(0).Focused() {
		t.Fatal("field 0 should be focused")
	}
	if m.Input(1).Focused() || m.Input(2).Focused() {
		t.Fatal("only field 0 should be focused")
	}
}

func TestEmptyFormSafe(t *testing.T) {
	var m Model
	if m.Len() != 0 {
		t.Fatalf("Len() = %d, want 0", m.Len())
	}
	if got := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}); got != nil {
		t.Fatal("Update on empty form should return nil")
	}
	m.SetFocused(5) // must not panic
	if m.Focused() != 0 {
		t.Fatalf("Focused() = %d, want 0", m.Focused())
	}
	if m.Value(0) != "" {
		t.Fatal("Value on empty form should be empty")
	}
	if m.Input(0) != nil {
		t.Fatal("Input on empty form should be nil")
	}
}

func TestSetFocusedClamps(t *testing.T) {
	m := newTestForm(3)
	m.SetFocused(10)
	if m.Focused() != 2 {
		t.Fatalf("SetFocused(10) -> %d, want 2 (clamped)", m.Focused())
	}
	if !m.Input(2).Focused() {
		t.Fatal("field 2 should be focused after clamp")
	}
	m.SetFocused(-5)
	if m.Focused() != 0 {
		t.Fatalf("SetFocused(-5) -> %d, want 0 (clamped)", m.Focused())
	}
}

func TestHandleNavKey(t *testing.T) {
	m := newTestForm(3)

	if !m.HandleNavKey("j") {
		t.Fatal(`"j" should be handled as nav`)
	}
	if m.Focused() != 1 {
		t.Fatalf("after j, Focused() = %d, want 1", m.Focused())
	}
	if !m.Input(1).Focused() {
		t.Fatal("field 1 should be focused after j")
	}

	m.HandleNavKey("down")
	if m.Focused() != 2 {
		t.Fatalf("after down, Focused() = %d, want 2", m.Focused())
	}
	// At the bottom edge, j does not wrap.
	m.HandleNavKey("j")
	if m.Focused() != 2 {
		t.Fatalf("j at bottom should stay at 2, got %d", m.Focused())
	}

	m.HandleNavKey("k")
	if m.Focused() != 1 {
		t.Fatalf("after k, Focused() = %d, want 1", m.Focused())
	}
	m.HandleNavKey("up")
	m.HandleNavKey("up") // at top edge, no wrap
	if m.Focused() != 0 {
		t.Fatalf("up at top should stay at 0, got %d", m.Focused())
	}

	if m.HandleNavKey("x") {
		t.Fatal(`"x" should not be a nav key`)
	}
}

func TestUpdateRoutesToFocused(t *testing.T) {
	m := newTestForm(2)
	m.SetFocused(1)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("abc")})
	if m.Value(0) != "" {
		t.Fatalf("field 0 should be untouched, got %q", m.Value(0))
	}
	if m.Value(1) != "abc" {
		t.Fatalf("field 1 should receive input, got %q", m.Value(1))
	}
}

func TestSetValueAndViews(t *testing.T) {
	m := newTestForm(2)
	m.SetValue(0, "hello")
	m.SetValue(1, "world")
	if m.Value(0) != "hello" || m.Value(1) != "world" {
		t.Fatalf("values = %q, %q", m.Value(0), m.Value(1))
	}
	views := m.Views()
	if len(views) != 2 {
		t.Fatalf("Views() len = %d, want 2", len(views))
	}
	// Out-of-range SetValue is a no-op (must not panic).
	m.SetValue(5, "ignored")
}

func TestSetWidth(t *testing.T) {
	m := newTestForm(3)
	m.SetWidth(42)
	for i := 0; i < m.Len(); i++ {
		if m.Input(i).Width != 42 {
			t.Fatalf("field %d width = %d, want 42", i, m.Input(i).Width)
		}
	}
}
