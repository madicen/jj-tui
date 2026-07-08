// Package form provides a small shared base for the settings sub-tabs, which
// previously each hand-rolled the same text-input form plumbing: a focusedField
// int with clamping, j/k/up/down focus cycling, focus/blur of every
// textinput.Model, width fan-out on resize, and Update routing to the focused
// input.
//
// A sub-tab embeds a form.Model built from its ordered list of inputs and keeps
// its own config getter/setter pairs (which delegate to the fields by index).
// Rendering stays in the parent settings package, which reads Views(),
// Focused(), and SetFocused() exactly as before.
package form

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Model manages an ordered list of text-input fields: focus tracking, vertical
// navigation, focus/blur, width propagation, and view/value access. The zero
// value is an empty form (safe to use, no fields).
type Model struct {
	inputs  []textinput.Model
	focused int
}

// New builds a form from the given inputs and focuses the first one.
func New(inputs ...textinput.Model) Model {
	m := Model{inputs: inputs}
	m.refocus()
	return m
}

// Len returns the number of fields.
func (m *Model) Len() int { return len(m.inputs) }

// Focused returns the index of the focused field (0 when empty).
func (m *Model) Focused() int { return m.focused }

// SetFocused moves focus to index i (clamped to [0, Len-1]) and updates the
// focus/blur state of every field.
func (m *Model) SetFocused(i int) {
	m.focused = clamp(i, 0, len(m.inputs)-1)
	m.refocus()
}

// Focus moves focus to index i (clamped) like SetFocused, but returns the
// focused input's cursor-blink command so callers that need the cursor to show
// (e.g. sub-tabs whose SetFocusedField returns a tea.Cmd) can propagate it.
func (m *Model) Focus(i int) tea.Cmd {
	if len(m.inputs) == 0 {
		return nil
	}
	m.focused = clamp(i, 0, len(m.inputs)-1)
	var cmd tea.Cmd
	for j := range m.inputs {
		if j == m.focused {
			cmd = m.inputs[j].Focus()
		} else {
			m.inputs[j].Blur()
		}
	}
	return cmd
}

// HandleNavKey processes a vertical navigation key, moving focus one field up
// or down within bounds. It reports whether key was a navigation key so callers
// can decide whether to route the message onward to the focused input.
func (m *Model) HandleNavKey(key string) bool {
	switch key {
	case "j", "down":
		if m.focused < len(m.inputs)-1 {
			m.focused++
			m.refocus()
		}
		return true
	case "k", "up":
		if m.focused > 0 {
			m.focused--
			m.refocus()
		}
		return true
	}
	return false
}

// Update forwards msg to the focused input and returns its command. It is a
// no-op returning nil when the form has no fields.
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if len(m.inputs) == 0 {
		return nil
	}
	var cmd tea.Cmd
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return cmd
}

// Value returns the text of field i ("" when out of range).
func (m *Model) Value(i int) string {
	if i < 0 || i >= len(m.inputs) {
		return ""
	}
	return m.inputs[i].Value()
}

// SetValue sets the text of field i (no-op when out of range).
func (m *Model) SetValue(i int, v string) {
	if i < 0 || i >= len(m.inputs) {
		return
	}
	m.inputs[i].SetValue(v)
}

// Input returns a pointer to field i for direct configuration/inspection (nil
// when out of range).
func (m *Model) Input(i int) *textinput.Model {
	if i < 0 || i >= len(m.inputs) {
		return nil
	}
	return &m.inputs[i]
}

// Views returns each field's rendered view in order.
func (m *Model) Views() []string {
	views := make([]string, len(m.inputs))
	for i := range m.inputs {
		views[i] = m.inputs[i].View()
	}
	return views
}

// SetWidth sets every field's display width.
func (m *Model) SetWidth(w int) {
	for i := range m.inputs {
		m.inputs[i].Width = w
	}
}

// refocus focuses the current field and blurs the rest.
func (m *Model) refocus() {
	for i := range m.inputs {
		if i == m.focused {
			m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
