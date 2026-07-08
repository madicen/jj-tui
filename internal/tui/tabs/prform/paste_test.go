package prform

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func pasteMsg(text string) tea.KeyMsg {
	return tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune(text), Paste: true})
}

// TestMultilinePasteIntoBodyPreservesLines is the core P5.7 acceptance: multi-line paste into
// a PR description (body textarea) preserves every line.
func TestMultilinePasteIntoBodyPreservesLines(t *testing.T) {
	m := NewModel(nil)
	m.Show(0, "main", "feature-x")
	m.SetFocusedField(1) // focus the body textarea

	want := "## Summary\n\n- bullet one\n- bullet two"
	updated, _ := m.Update(pasteMsg(want))

	if !updated.IsShown() {
		t.Fatal("paste closed the PR form; it must not be treated as a shortcut")
	}
	if got := updated.GetBody(); got != want {
		t.Errorf("multi-line paste into PR body not preserved:\n got  %q\n want %q", got, want)
	}
}

// TestPasteIntoTitleStaysSingleLine verifies that pasting text containing newlines into the
// single-line title input does not introduce line breaks (textinput sanitizes newlines).
func TestPasteIntoTitleStaysSingleLine(t *testing.T) {
	m := NewModel(nil)
	m.Show(0, "main", "feature-x") // focus is on the title input

	updated, _ := m.Update(pasteMsg("title part one\ntitle part two"))

	if got := updated.GetTitle(); strings.Contains(got, "\n") {
		t.Errorf("paste into single-line title kept a newline: %q", got)
	}
}
