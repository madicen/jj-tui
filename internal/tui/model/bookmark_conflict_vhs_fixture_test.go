package model

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/state"
	branchestab "github.com/madicen/jj-tui/internal/tui/tabs/branches"
)

// repoRoot finds the jj-tui module root (directory containing go.mod) relative to this test file.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// This file lives in internal/tui/model/
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func runPendingCmdsModel(t *testing.T, m *Model, cmd tea.Cmd, maxRounds int) *Model {
	t.Helper()
	for i := 0; i < maxRounds && cmd != nil; i++ {
		msg := cmd()
		next, c := m.Update(msg)
		m = next.(*Model)
		cmd = c
	}
	return m
}

// TestVHSTapeBookmarkConflictFixture_ModalAppears runs fixtures/setup-bookmark-conflict-vhs-repo.sh
// (same as vhs/bookmark-conflict.tape), wires the repo into the TUI model like the demo, then
// presses `c` on the Branches tab with the diverged bookmark selected. Verifies the bookmark
// conflict modal is shown (ViewMode + rendered title).
//
// Requires jj, git, and bash on PATH. Skips otherwise.
func TestVHSTapeBookmarkConflictFixture_ModalAppears(t *testing.T) {
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj not on PATH")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not on PATH")
	}

	root := repoRoot(t)
	setupScript := filepath.Join(root, "fixtures", "setup-bookmark-conflict-vhs-repo.sh")
	if _, err := os.Stat(setupScript); err != nil {
		t.Fatalf("setup script not found at %s: %v", setupScript, err)
	}

	repoPath := filepath.Join(root, "fixtures", "bookmark-conflict-vhs-repo")
	// Ensure a clean fixture matching the tape (script removes and recreates the repo).
	cmdSetup := exec.Command("bash", setupScript)
	cmdSetup.Dir = root
	cmdSetup.Env = os.Environ()
	if out, err := cmdSetup.CombinedOutput(); err != nil {
		t.Fatalf("setup-bookmark-conflict-vhs-repo.sh failed: %v\n%s", err, out)
	}

	ctx := context.Background()
	svc, err := jj.NewService(repoPath)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	repo, err := svc.GetRepository(ctx, "")
	if err != nil {
		t.Fatalf("GetRepository: %v", err)
	}
	branches, err := svc.ListBranches(ctx, 0)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	branchestab.SortBranchList(branches)

	idx := -1
	for i := range branches {
		b := branches[i]
		if b.Name == "vhs/conflict-feature" && b.IsLocal && b.HasConflict {
			idx = i
			break
		}
	}
	if idx < 0 {
		var names []string
		for _, b := range branches {
			names = append(names, b.Name+"(local="+boolStr(b.IsLocal)+",conflict="+boolStr(b.HasConflict)+")")
		}
		t.Fatalf("no diverged local vhs/conflict-feature in branch list; entries: %s", strings.Join(names, ", "))
	}
	// After LoadBranchesCmd sort, ahead-only diverged locals (e.g. vhs/conflict-feature) are before main;
	// one Down would leave main selected and 'c' would not open the resolver (see vhs/bookmark-conflict.tape).
	if idx != 0 {
		t.Fatalf("fixture expects vhs/conflict-feature at sorted index 0 for tape without Down, got index %d", idx)
	}

	m := New(ctx)
	defer m.Close()
	m.appState.JJService = svc
	m.appState.DemoMode = true
	m.appState.Loading = false

	m.Update(tea.WindowSizeMsg{Width: 130, Height: 40})
	m.SetRepository(repo)
	m.branchesTabModel.UpdateBranches(branches)
	m.branchesTabModel.SetSelectedBranch(idx)
	m.appState.ViewMode = state.ViewBranches

	keyC := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	next, cmd := m.Update(keyC)
	m = next.(*Model)
	if cmd == nil {
		t.Fatal("expected a command after pressing c (LoadBookmarkConflictInfo); check HasConflict / branch selection")
	}
	m = runPendingCmdsModel(t, m, cmd, 8)

	if m.GetViewMode() != state.ViewBookmarkConflict {
		t.Fatalf("expected ViewBookmarkConflict after load, got %v (status=%q)", m.GetViewMode(), m.GetStatusMessage())
	}
	view := m.View()
	if !strings.Contains(view, "Diverged bookmark") {
		t.Fatalf("expected conflict modal in View(); got snippet: %.400q", view)
	}
	if !strings.Contains(view, "vhs/conflict-feature") {
		t.Fatalf("expected bookmark name in modal view; got snippet: %.400q", view)
	}
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// TestVHSTapeBookmarkConflictFixture_KeySequence_bc matches vhs/bookmark-conflict.tape after load:
// Graph → `b` (load branches) → `c` with no Down (diverged bookmark is already selected).
func TestVHSTapeBookmarkConflictFixture_KeySequence_bc(t *testing.T) {
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj not on PATH")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not on PATH")
	}

	root := repoRoot(t)
	cmdSetup := exec.Command("bash", filepath.Join(root, "fixtures", "setup-bookmark-conflict-vhs-repo.sh"))
	cmdSetup.Dir = root
	cmdSetup.Env = os.Environ()
	if out, err := cmdSetup.CombinedOutput(); err != nil {
		t.Fatalf("setup script failed: %v\n%s", err, out)
	}

	repoPath := filepath.Join(root, "fixtures", "bookmark-conflict-vhs-repo")
	ctx := context.Background()
	svc, err := jj.NewService(repoPath)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	repo, err := svc.GetRepository(ctx, "")
	if err != nil {
		t.Fatalf("GetRepository: %v", err)
	}

	m := New(ctx)
	defer m.Close()
	m.appState.JJService = svc
	m.appState.DemoMode = true
	m.appState.Loading = false
	m.Update(tea.WindowSizeMsg{Width: 130, Height: 40})
	m.SetRepository(repo)
	m.appState.ViewMode = state.ViewCommitGraph

	keyB := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}
	next, cmd := m.Update(keyB)
	m = next.(*Model)
	m = runPendingCmdsModel(t, m, cmd, 12)

	if m.GetViewMode() != state.ViewBranches {
		t.Fatalf("after b, want ViewBranches, got %v", m.GetViewMode())
	}
	br := m.GetBranches()
	if len(br) < 2 {
		t.Fatalf("expected branch list, got %d entries", len(br))
	}
	if br[0].Name != "vhs/conflict-feature" || !br[0].HasConflict {
		t.Fatalf("sorted list should put diverged vhs first (tape has no Down); first=%q conflict=%v", br[0].Name, br[0].HasConflict)
	}
	if m.GetSelectedBranch() != 0 {
		t.Fatalf("expected selectedBranch 0 after load, got %d", m.GetSelectedBranch())
	}

	keyC := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	next, cmd = m.Update(keyC)
	m = next.(*Model)
	if cmd == nil {
		t.Fatalf("expected cmd after c (status=%q)", m.GetStatusMessage())
	}
	m = runPendingCmdsModel(t, m, cmd, 8)

	if m.GetViewMode() != state.ViewBookmarkConflict {
		t.Fatalf("expected ViewBookmarkConflict, got %v (status=%q)", m.GetViewMode(), m.GetStatusMessage())
	}
	if !strings.Contains(m.View(), "Diverged bookmark") {
		t.Fatal("expected modal title in view")
	}
}
