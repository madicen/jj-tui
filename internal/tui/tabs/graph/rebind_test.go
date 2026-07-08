package graph

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/keys"
)

func runeKeyMsg(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

// TestRebindAbandonKey verifies P5.1 end-to-end at the tab level: rebinding
// graph.abandon to "x" makes "x" trigger the abandon request and "a" a no-op.
func TestRebindAbandonKey(t *testing.T) {
	m := newTestGraphModel()
	m.selectedCommit = 0

	// Default: "a" abandons.
	if _, req, _ := m.handleKeyMsg(runeKeyMsg('a')); req == nil || !req.Abandon {
		t.Fatal("default: 'a' should request abandon")
	}

	// Rebind graph.abandon -> "x".
	m.SetKeyMap(keys.DefaultGraphKeyMap(map[string]string{"graph.abandon": "x"}))

	if _, req, _ := m.handleKeyMsg(runeKeyMsg('x')); req == nil || !req.Abandon {
		t.Fatal("after rebind: 'x' should request abandon")
	}
	if _, req, _ := m.handleKeyMsg(runeKeyMsg('a')); req != nil && req.Abandon {
		t.Error("after rebind: 'a' should no longer request abandon")
	}
}
