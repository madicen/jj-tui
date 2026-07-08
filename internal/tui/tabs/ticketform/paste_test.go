package ticketform

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func pasteMsg(text string) tea.KeyMsg {
	return tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune(text), Paste: true})
}

// TestMultilinePasteIntoDescriptionPreservesLines verifies multi-line paste into the ticket
// description (body textarea) keeps every line (P5.7).
func TestMultilinePasteIntoDescriptionPreservesLines(t *testing.T) {
	m := NewModel(nil)
	m.Show("jira")
	m.SetFocusedField(1) // focus the description textarea

	want := "Steps to reproduce:\n1. open app\n2. observe bug"
	updated, _ := m.Update(pasteMsg(want))

	if !updated.IsShown() {
		t.Fatal("paste closed the ticket form; it must not be treated as a shortcut")
	}
	if got := updated.GetDescription(); got != want {
		t.Errorf("multi-line paste into ticket description not preserved:\n got  %q\n want %q", got, want)
	}
}
