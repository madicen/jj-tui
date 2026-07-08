package prs

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/github"
	"github.com/madicen/jj-tui/internal/tui/state"
)

// TestMergePRStartsBusySpinner verifies that merging an open PR (a >200ms GitHub API op)
// flips Loading + SpinnerStartPending so the main model can attach the busy spinner. The PR
// merge/close paths previously only set a footer status without an animated loading state
// (P5.6 gap).
func TestMergePRStartsBusySpinner(t *testing.T) {
	m := NewModel(nil)
	repo := &internal.Repository{PRs: []internal.GitHubPR{{Number: 1, State: "open"}}}
	m.OnRepositoryLoaded(repo)

	app := &state.AppState{Repository: repo, GitHubService: &github.Service{}}
	// Shift+M triggers a merge request in handleKeyMsg.
	updated, cmd := m.UpdateWithApp(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("M")}, app)
	_ = updated

	if cmd == nil {
		t.Fatal("expected a merge command, got nil")
	}
	if !app.Loading {
		t.Error("expected app.Loading to be true after starting a PR merge")
	}
	if !app.SpinnerStartPending {
		t.Error("expected app.SpinnerStartPending to be true so main starts the busy spinner")
	}
}
