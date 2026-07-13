package integration_tests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/madicen/jj-tui/internal/integrations/jj"
)

// TestMoveBookmarkDeltaOntoOrigin_divergentChangeID covers the amend-after-push + resurrected
// origin tip case: the local bookmark tip shares a change ID with bookmark@origin, so bare change
// IDs fail `jj diff`, but Forgot New Commit? must still restack the intended tree onto origin.
func TestMoveBookmarkDeltaOntoOrigin_divergentChangeID(t *testing.T) {
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj not in PATH")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}

	ctx := context.Background()
	root := t.TempDir()
	origin := filepath.Join(root, "origin.git")
	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}

	run := func(dir string, name string, args ...string) string {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s %v: %v\n%s", name, args, err, string(out))
		}
		return string(out)
	}
	runAllowFail := func(dir string, name string, args ...string) error {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		_, err := cmd.CombinedOutput()
		return err
	}

	run(root, "git", "init", "--bare", origin)
	run(repo, "git", "init", "--initial-branch=main")
	run(repo, "jj", "git", "init", "--colocate")
	run(repo, "git", "remote", "add", "origin", origin)
	run(repo, "jj", "config", "set", "--repo", "user.name", "jj-tui integration")
	run(repo, "jj", "config", "set", "--repo", "user.email", "integration@example.com")

	if err := os.WriteFile(filepath.Join(repo, "f.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(repo, "jj", "describe", "-m", "init")
	run(repo, "jj", "bookmark", "create", "main")
	run(repo, "jj", "new")

	if err := os.WriteFile(filepath.Join(repo, "f.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(repo, "jj", "describe", "-m", "feat")
	run(repo, "jj", "bookmark", "create", "feat")

	push := func(bm string) {
		t.Helper()
		if runAllowFail(repo, "jj", "git", "push", "--bookmark", bm, "--remote", "origin", "--allow-new") != nil {
			run(repo, "jj", "git", "push", "--bookmark", bm, "--remote", "origin")
		}
	}
	push("main")
	push("feat")
	run(repo, "jj", "git", "fetch", "--remote", "origin")

	oldTip := strings.TrimSpace(run(repo, "jj", "log", "-r", "feat", "--no-graph", "-T", "commit_id", "--limit", "1"))
	run(repo, "jj", "edit", "feat")
	run(repo, "jj", "new")
	if err := os.WriteFile(filepath.Join(repo, "extra.txt"), []byte("extra\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(repo, "jj", "squash", "--into", "@-", "-m", "feat + extra")
	// Resurrect the pushed tip so the same change ID has two visible revisions (local tip + @origin).
	run(repo, "jj", "new", oldTip, "--ignore-immutable")
	changeID := strings.TrimSpace(run(repo, "jj", "log", "-r", "@-", "--no-graph", "-T", "change_id", "--limit", "1"))
	run(repo, "jj", "bookmark", "set", "feat", "-r", "change_id("+changeID+") & ~commit_id("+oldTip+")", "--allow-backwards")
	run(repo, "jj", "edit", "feat")

	tipChangeID := strings.TrimSpace(run(repo, "jj", "log", "-r", "feat", "--no-graph", "-T", "change_id", "--limit", "1"))
	tipCommitID := strings.TrimSpace(run(repo, "jj", "log", "-r", "feat", "--no-graph", "-T", "commit_id", "--limit", "1"))
	if tipChangeID == "" || tipCommitID == "" {
		t.Fatal("empty tip ids")
	}
	// Guard: bare change ID must be the divergent failure mode this test targets.
	diffCmd := exec.Command("jj", "diff", "--from", "feat@origin", "--to", tipChangeID, "--summary")
	diffCmd.Dir = repo
	if out, err := diffCmd.CombinedOutput(); err == nil {
		t.Fatalf("expected divergent change ID diff to fail, got ok:\n%s", out)
	}

	svc, err := jj.NewService(repo)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if err := svc.MoveBookmarkDeltaOntoOrigin(ctx, "feat", tipChangeID, tipCommitID); err != nil {
		t.Fatalf("MoveBookmarkDeltaOntoOrigin: %v", err)
	}

	// New tip should sit on feat@origin with the intended follow-up tree (extra.txt).
	parent := strings.TrimSpace(run(repo, "jj", "log", "-r", "feat-", "--no-graph", "-T", "commit_id", "--limit", "1"))
	originTip := strings.TrimSpace(run(repo, "jj", "log", "-r", "feat@origin", "--no-graph", "-T", "commit_id", "--limit", "1"))
	if parent != originTip {
		t.Fatalf("feat parent = %q, want feat@origin %q", parent, originTip)
	}
	summary := strings.TrimSpace(run(repo, "jj", "log", "-r", "feat", "--no-graph", "-T", "description.first_line()", "--limit", "1"))
	if !strings.Contains(strings.ToLower(summary), "follow-up") {
		t.Fatalf("expected follow-up description on new tip, got %q", summary)
	}
	files := run(repo, "jj", "file", "list", "-r", "feat")
	if !strings.Contains(files, "extra.txt") {
		t.Fatalf("expected extra.txt in restacked tip tree, got:\n%s", files)
	}
}
