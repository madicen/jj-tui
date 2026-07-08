package integration_tests

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/madicen/jj-tui/internal/integrations/jj"
)

// TestAbandonCommitsBatch_SingleUndo verifies jj batches multiple -r revisions into one
// abandon operation so a single undo restores all abandoned commits.
func TestAbandonCommitsBatch_SingleUndo(t *testing.T) {
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj not in PATH")
	}
	ctx := context.Background()
	dir := t.TempDir()
	repo := &TestRepository{Path: dir, t: t}
	if err := repo.runCommand("jj", "git", "init"); err != nil {
		t.Fatal(err)
	}
	_ = repo.runCommand("jj", "config", "set", "--repo", "user.name", "Test User")
	_ = repo.runCommand("jj", "config", "set", "--repo", "user.email", "test@example.com")

	if err := repo.writeFile("a.txt", "one\n"); err != nil {
		t.Fatal(err)
	}
	if err := repo.runCommand("jj", "describe", "-m", "c1"); err != nil {
		t.Fatal(err)
	}
	out1, _ := repo.runCommandOutput("jj", "log", "-r", "@", "--no-graph", "-T", "change_id")
	id1 := strings.Fields(strings.TrimSpace(out1))[0]

	if err := repo.runCommand("jj", "new", "-m", "c2"); err != nil {
		t.Fatal(err)
	}
	if err := repo.writeFile("b.txt", "two\n"); err != nil {
		t.Fatal(err)
	}
	out2, _ := repo.runCommandOutput("jj", "log", "-r", "@", "--no-graph", "-T", "change_id")
	id2 := strings.Fields(strings.TrimSpace(out2))[0]

	if err := repo.runCommand("jj", "new", "-m", "c3"); err != nil {
		t.Fatal(err)
	}
	if err := repo.writeFile("c.txt", "three\n"); err != nil {
		t.Fatal(err)
	}
	out3, _ := repo.runCommandOutput("jj", "log", "-r", "@", "--no-graph", "-T", "change_id")
	id3 := strings.Fields(strings.TrimSpace(out3))[0]

	svc, err := jj.NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	before, err := svc.GetRepository(ctx, "all()")
	if err != nil {
		t.Fatalf("GetRepository before: %v", err)
	}
	beforeCount := len(before.Graph.Commits)

	if err := svc.AbandonCommitsBatch(ctx, []string{id1, id2, id3}); err != nil {
		t.Fatalf("AbandonCommitsBatch: %v", err)
	}

	after, err := svc.GetRepository(ctx, "all()")
	if err != nil {
		t.Fatalf("GetRepository after abandon: %v", err)
	}
	if len(after.Graph.Commits) >= beforeCount {
		t.Fatalf("expected fewer commits after batch abandon: before=%d after=%d", beforeCount, len(after.Graph.Commits))
	}

	if _, err := svc.Undo(ctx); err != nil {
		t.Fatalf("Undo after batch abandon: %v", err)
	}
	restored, err := svc.GetRepository(ctx, "all()")
	if err != nil {
		t.Fatalf("GetRepository after undo: %v", err)
	}
	if len(restored.Graph.Commits) < beforeCount {
		t.Fatalf("single undo should restore batch-abandoned commits: before=%d restored=%d", beforeCount, len(restored.Graph.Commits))
	}
}

// TestRebaseCommitsBatch_rebasesSelection rebases two sibling commits onto a destination.
func TestRebaseCommitsBatch_rebasesSelection(t *testing.T) {
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj not in PATH")
	}
	ctx := context.Background()
	dir := t.TempDir()
	repo := &TestRepository{Path: dir, t: t}
	if err := repo.runCommand("jj", "git", "init"); err != nil {
		t.Fatal(err)
	}
	_ = repo.runCommand("jj", "config", "set", "--repo", "user.name", "Test User")
	_ = repo.runCommand("jj", "config", "set", "--repo", "user.email", "test@example.com")

	if err := repo.writeFile("base.txt", "base\n"); err != nil {
		t.Fatal(err)
	}
	if err := repo.runCommand("jj", "describe", "-m", "base"); err != nil {
		t.Fatal(err)
	}
	baseOut, _ := repo.runCommandOutput("jj", "log", "-r", "@", "--no-graph", "-T", "change_id")
	baseID := strings.Fields(strings.TrimSpace(baseOut))[0]

	if err := repo.runCommand("jj", "new", "-m", "side-a"); err != nil {
		t.Fatal(err)
	}
	if err := repo.writeFile("a.txt", "a\n"); err != nil {
		t.Fatal(err)
	}
	aOut, _ := repo.runCommandOutput("jj", "log", "-r", "@", "--no-graph", "-T", "change_id")
	aID := strings.Fields(strings.TrimSpace(aOut))[0]

	if err := repo.runCommand("jj", "new", baseID, "-m", "side-b"); err != nil {
		t.Fatal(err)
	}
	if err := repo.writeFile("b.txt", "b\n"); err != nil {
		t.Fatal(err)
	}
	bOut, _ := repo.runCommandOutput("jj", "log", "-r", "@", "--no-graph", "-T", "change_id")
	bID := strings.Fields(strings.TrimSpace(bOut))[0]

	if err := repo.runCommand("jj", "new", "-m", "dest"); err != nil {
		t.Fatal(err)
	}
	if err := repo.writeFile("dest.txt", "dest\n"); err != nil {
		t.Fatal(err)
	}
	destOut, _ := repo.runCommandOutput("jj", "log", "-r", "@", "--no-graph", "-T", "change_id")
	destID := strings.Fields(strings.TrimSpace(destOut))[0]

	svc, err := jj.NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	if err := svc.RebaseCommitsBatch(ctx, []string{aID, bID}, destID); err != nil {
		t.Fatalf("RebaseCommitsBatch: %v", err)
	}
}
