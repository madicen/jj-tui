package descedit

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// pasteMsg builds a bracketed-paste key message (bubbletea sets Paste=true and wraps the
// String() in brackets so it can never match a shortcut binding).
func pasteMsg(text string) tea.KeyMsg {
	return tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune(text), Paste: true})
}

// TestMultilinePastePreservesLines verifies that pasting multi-line text into the commit
// description textarea keeps every line (P5.7). A paste must not be swallowed by the
// Ctrl+S / Esc / Ctrl+G shortcut switch.
func TestMultilinePastePreservesLines(t *testing.T) {
	m := NewModel(nil)
	m.Show("change-1", "abc123")

	want := "First line\nSecond line\nThird line"
	updated, _ := m.Update(pasteMsg(want))

	if !updated.IsShown() {
		t.Fatal("paste closed the modal; it must not be treated as a shortcut")
	}
	if got := updated.GetDescriptionValue(); got != want {
		t.Errorf("multi-line paste not preserved:\n got  %q\n want %q", got, want)
	}
}
