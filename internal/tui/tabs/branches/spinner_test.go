package branches

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/state"
)

// TestPushBranchStartsBusySpinner verifies that pushing a local branch (a >200ms network op)
// flips Loading + SpinnerStartPending so the main model can attach the busy spinner. Fetch-all
// already did this; push previously only set a footer status, freezing without an animated
// loading state (P5.6 gap).
func TestPushBranchStartsBusySpinner(t *testing.T) {
	m := NewModel(nil)
	m.UpdateBranches([]internal.Branch{{Name: "feature-x", IsLocal: true}})
	m.SetSelectedBranch(0)

	app := &state.AppState{JJService: &jj.Service{}}
	// Shift+P triggers a push request in handleKeyMsg.
	updated, cmd := m.UpdateWithApp(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("P")}, app)
	_ = updated

	if cmd == nil {
		t.Fatal("expected a push command, got nil")
	}
	if !app.Loading {
		t.Error("expected app.Loading to be true after starting a branch push")
	}
	if !app.SpinnerStartPending {
		t.Error("expected app.SpinnerStartPending to be true so main starts the busy spinner")
	}
}
