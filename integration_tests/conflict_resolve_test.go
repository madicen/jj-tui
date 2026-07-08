package integration_tests

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/madicen/jj-tui/internal/integrations/jj"
)

// TestResolveFileWithTool_clearsConflict creates a 2-parent merge conflict, resolves
// one file via jj resolve --tool :ours, and verifies jj resolve --list is empty.
func TestResolveFileWithTool_clearsConflict(t *testing.T) {
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj not in PATH")
	}
	ctx := context.Background()
	dir, mergeRev := setupMergeConflictRepo(t)
	svc, err := jj.NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	conflicts, err := svc.ListUnresolvedConflicts(ctx, mergeRev)
	if err != nil {
		t.Fatalf("ListUnresolvedConflicts: %v", err)
	}
	if len(conflicts) == 0 {
		t.Fatal("expected unresolved conflicts before resolve")
	}
	found := false
	for _, c := range conflicts {
		if c.Path == "f.txt" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected f.txt in conflicts, got %+v", conflicts)
	}

	files, err := svc.GetChangedFiles(ctx, mergeRev)
	if err != nil {
		t.Fatalf("GetChangedFiles: %v", err)
	}
	flagged := false
	for _, f := range files {
		if f.Path == "f.txt" && f.Conflicted {
			flagged = true
		}
	}
	if !flagged {
		t.Fatalf("GetChangedFiles should flag f.txt as conflicted: %+v", files)
	}

	if err := svc.ResolveFileWithTool(ctx, mergeRev, "f.txt", ":ours"); err != nil {
		t.Fatalf("ResolveFileWithTool: %v", err)
	}
	after, err := svc.ListUnresolvedConflicts(ctx, mergeRev)
	if err != nil {
		t.Fatalf("ListUnresolvedConflicts after: %v", err)
	}
	if len(after) != 0 {
		t.Fatalf("expected no conflicts after resolve, got %+v", after)
	}
}

func setupMergeConflictRepo(t *testing.T) (dir, mergeChangeID string) {
	t.Helper()
	repo := NewTestRepository(t)
	t.Cleanup(repo.Cleanup)

	if err := repo.writeFile("f.txt", "base\n"); err != nil {
		t.Fatal(err)
	}
	if err := repo.runCommand("jj", "describe", "-m", "base"); err != nil {
		t.Fatal(err)
	}
	baseOut, err := repo.runCommandOutput("jj", "log", "-r", "@", "--no-graph", "-T", "commit_id")
	if err != nil {
		t.Fatal(err)
	}
	baseID := strings.Fields(strings.TrimSpace(baseOut))[0]

	if err := repo.runCommand("jj", "new", "-m", "a"); err != nil {
		t.Fatal(err)
	}
	if err := repo.writeFile("f.txt", "aaa\n"); err != nil {
		t.Fatal(err)
	}
	aOut, err := repo.runCommandOutput("jj", "log", "-r", "@", "--no-graph", "-T", "commit_id")
	if err != nil {
		t.Fatal(err)
	}
	aID := strings.Fields(strings.TrimSpace(aOut))[0]

	if err := repo.runCommand("jj", "edit", baseID); err != nil {
		t.Fatal(err)
	}
	if err := repo.runCommand("jj", "new", "-m", "b"); err != nil {
		t.Fatal(err)
	}
	if err := repo.writeFile("f.txt", "bbb\n"); err != nil {
		t.Fatal(err)
	}
	bOut, err := repo.runCommandOutput("jj", "log", "-r", "@", "--no-graph", "-T", "commit_id")
	if err != nil {
		t.Fatal(err)
	}
	bID := strings.Fields(strings.TrimSpace(bOut))[0]

	if err := repo.runCommand("jj", "new", aID, bID, "-m", "merge"); err != nil {
		t.Fatal(err)
	}
	mergeOut, err := repo.runCommandOutput("jj", "log", "-r", "@", "--no-graph", "-T", "change_id")
	if err != nil {
		t.Fatal(err)
	}
	return repo.Path, strings.Fields(strings.TrimSpace(mergeOut))[0]
}
