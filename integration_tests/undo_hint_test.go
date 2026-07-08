package integration_tests

import (
	"context"
	"os/exec"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui"
	"github.com/madicen/jj-tui/internal/tui/data"
	"github.com/madicen/jj-tui/internal/tui/state"
)

// TestUndoHint_VisibleAfterDescribe drives a real describe through the model and
// asserts the "Ctrl+z undoes: …" hint (P5.4) appears in the footer afterward.
//
// The drain stops as soon as the hint is visible so the 6s expiry tea.Tick never
// fires (which would clear it); it also skips executing background tea.Tick cmds
// (auto-refresh / spinner) which would otherwise block the test for seconds.
func TestUndoHint_VisibleAfterDescribe(t *testing.T) {
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj not in PATH")
	}
	ctx := context.Background()

	repo := NewTestRepository(t)
	defer repo.Cleanup()
	if err := repo.writeFile("a.txt", "hello\n"); err != nil {
		t.Fatal(err)
	}
	if err := repo.runCommand("jj", "describe", "-m", "seed"); err != nil {
		t.Fatalf("jj describe: %v", err)
	}

	jjSvc, err := jj.NewService(repo.Path)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	m := tui.NewWithServices(ctx, jjSvc, nil)
	defer m.Close()
	m.SetDimensions(120, 40)
	m.SetLoading(false)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	if loadCmd := data.LoadRepository(jjSvc, ""); loadCmd != nil {
		m = updateModel(m, loadCmd())
	}

	repoState := m.GetRepository()
	if repoState == nil || len(repoState.Graph.Commits) == 0 {
		t.Fatal("expected a loaded repository with commits")
	}
	changeID := repoState.Graph.Commits[0].ChangeID
	for _, c := range repoState.Graph.Commits {
		if c.IsWorking {
			changeID = c.ChangeID
			break
		}
	}

	start := state.NavigateMsg{Target: state.NavigateTarget{
		Kind:            state.NavigateSaveDescription,
		SaveCommitID:    changeID,
		SaveDescription: "hint me please",
	}}

	m, ok := drainUntilFooter(m, start, "Ctrl+z undoes:", 40)
	if !ok {
		t.Errorf("footer never showed the undo hint after describe; view snippet: %s",
			truncateView(m.View(), 600))
	}
}

// drainUntilFooter runs msg through the model, then processes returned commands
// breadth-first, checking View() after each Update. It stops (returning true) as
// soon as the view contains want — so the 6s expiry tea.Tick that clears the
// hint is never executed.
func drainUntilFooter(m *tui.Model, msg tea.Msg, want string, maxSteps int) (*tui.Model, bool) {
	if containsString(m.View(), want) {
		return m, true
	}
	newModel, cmd := m.Update(msg)
	m = newModel.(*tui.Model)
	if containsString(m.View(), want) {
		return m, true
	}
	var queue []tea.Cmd
	if cmd != nil {
		queue = append(queue, cmd)
	}
	for steps := 0; steps < maxSteps; steps++ {
		if len(queue) == 0 {
			return m, false
		}
		next := queue[0]
		queue = queue[1:]
		out := next()
		if out == nil {
			continue
		}
		if batch, ok := out.(tea.BatchMsg); ok {
			for _, c := range batch {
				if c != nil {
					queue = append(queue, c)
				}
			}
			continue
		}
		newModel, cmd := m.Update(out)
		m = newModel.(*tui.Model)
		if containsString(m.View(), want) {
			return m, true
		}
		if cmd != nil {
			queue = append(queue, cmd)
		}
	}
	return m, false
}
