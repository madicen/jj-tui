package integration_tests

import (
	"context"
	"os/exec"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui"
	"github.com/madicen/jj-tui/internal/tui/data"
)

// TestAutoRefresh_ExternalChangeAppearsViaSilentReload proves the P5.2 silent-reload path picks
// up repository activity performed OUTSIDE the TUI (e.g. a `jj new` run in another terminal): a
// silent reload replaces the graph in place without a manual refresh. This exercises the same
// data.LoadRepositorySilent -> SilentRepositoryLoadedMsg path that the auto-refresh tick drives;
// the tick's gating (off-by-default, skip-while-modal-open / in-flight) is covered by unit tests
// in the model package (tickMsg is unexported so it cannot be constructed from this package).
func TestAutoRefresh_ExternalChangeAppearsViaSilentReload(t *testing.T) {
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

	if loadCmd := data.LoadRepository(jjSvc); loadCmd != nil {
		m = updateModel(m, loadCmd())
	}
	before := m.GetRepository()
	if before == nil || len(before.Graph.Commits) == 0 {
		t.Fatal("expected a loaded repository with commits")
	}
	initialCount := len(before.Graph.Commits)

	// Simulate external activity: a `jj new` in another terminal adds a commit.
	if err := repo.runCommand("jj", "new", "-m", "external change"); err != nil {
		t.Fatalf("jj new: %v", err)
	}

	// Drive the silent reload exactly as the auto-refresh tick would.
	silentCmd := data.LoadRepositorySilent(jjSvc, "")
	if silentCmd == nil {
		t.Fatal("LoadRepositorySilent returned nil command")
	}
	m = updateModel(m, silentCmd())

	after := m.GetRepository()
	if after == nil {
		t.Fatal("repository became nil after silent reload")
	}
	if len(after.Graph.Commits) <= initialCount {
		t.Fatalf("external jj new should appear after a silent reload: before=%d after=%d",
			initialCount, len(after.Graph.Commits))
	}
}
