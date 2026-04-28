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

// buildColocatedEvologSplitFixture creates a small colocated git+jj repo with a demo/feature bookmark
// whose change has multiple jj evolog revisions (squash + edit), matching fixtures/setup-evolog-split-vhs-repo.sh.
// Used by TestEvologSplitMoveBookmarkDeltaColocated; the shell script is the manual equivalent for VHS / debugging.
func buildColocatedEvologSplitFixture(t *testing.T, repoDir string) {
	t.Helper()
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj not in PATH")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	run := func(name string, args ...string) {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = repoDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s %v: %v\n%s", name, args, err, string(out))
		}
	}
	mkdir := func(rel string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(repoDir, rel), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(repoDir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	run("git", "init", "--initial-branch=main")
	run("jj", "git", "init", "--colocate")
	run("jj", "config", "set", "--repo", "user.name", "jj-tui integration")
	run("jj", "config", "set", "--repo", "user.email", "integration@example.com")

	mkdir("src")
	mkdir("docs")
	write("src/feature-flag.txt", "mode=light\n")
	write("src/ui-settings.toml", "theme = \"light\"\nanimations = true\n")
	write("docs/changelog.md", "# Changelog\n\n## 0.1.0\n- Initial import\n")

	run("jj", "describe", "-m", "Initial import")
	run("jj", "bookmark", "create", "main", "-r", "@")
	run("jj", "new")

	write("src/feature-flag.txt", "mode=dark\n")
	write("src/ui-settings.toml", "theme = \"dark\"\nanimations = true\ndark_mode_preview = true\n")
	appendFile(t, repoDir, "docs/changelog.md", "\n## Unreleased\n- Dark mode and preview toggle\n")
	run("jj", "describe", "-m", "Add feature flag")
	run("jj", "bookmark", "create", "demo/feature", "-r", "@")
	run("jj", "new")

	appendFile(t, repoDir, "src/feature-flag.txt", "rollout=10pct\n")
	appendFile(t, repoDir, "src/ui-settings.toml", "rollout_percent = 10\n")
	run("jj", "squash", "-m", "Add feature flag")
	run("jj", "edit", "demo/feature")
}

func appendFile(t *testing.T, repoDir, rel, extra string) {
	t.Helper()
	p := filepath.Join(repoDir, rel)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, append(b, []byte(extra)...), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestEvologSplitMoveBookmarkDeltaColocated runs the FAQ evolog split (MoveBookmarkDeltaOntoEvologBase) on a
// colocated jj+git repo. This guards against regressions where `jj new` fails with Git checkout errors
// unless jj state is exported to Git before / around the operation.
func TestEvologSplitMoveBookmarkDeltaColocated(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	buildColocatedEvologSplitFixture(t, dir)

	svc, err := jj.NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	entries, err := svc.ListEvolog(ctx, "demo/feature")
	if err != nil {
		t.Fatalf("ListEvolog: %v", err)
	}
	if len(entries) < 2 {
		t.Fatalf("expected at least 2 evolog rows, got %d", len(entries))
	}

	cmd := exec.Command("jj", "log", "-r", "demo/feature", "--no-graph", "-T", "change_id", "--limit", "1")
	cmd.Dir = dir
	tipChangeBytes, err := cmd.Output()
	if err != nil {
		t.Fatalf("jj log tip change_id: %v", err)
	}
	tipChangeID := strings.TrimSpace(string(tipChangeBytes))
	if tipChangeID == "" {
		t.Fatal("empty tip change id")
	}

	// Pick the shallowest evolog row (smallest index > 0) whose tree still differs from the tip for
	// this change — same rule as the user picking a row below @ in the split modal.
	var baseCommitID string
	for i := 1; i < len(entries); i++ {
		cand := strings.TrimSpace(entries[i].CommitID)
		if cand == "" {
			continue
		}
		cmd := exec.Command("jj", "diff", "--from", cand, "--to", tipChangeID, "--summary")
		cmd.Dir = dir
		out, derr := cmd.Output()
		if derr != nil {
			t.Fatalf("jj diff --from %s --to %s: %v", cand, tipChangeID, derr)
		}
		if strings.TrimSpace(string(out)) != "" {
			baseCommitID = cand
			break
		}
	}
	if baseCommitID == "" {
		t.Fatalf("no evolog base with non-empty diff to tip (entries=%d)", len(entries))
	}

	err = svc.MoveBookmarkDeltaOntoEvologBase(ctx, "demo/feature", tipChangeID, "", baseCommitID, nil, nil)
	if err != nil {
		t.Fatalf("MoveBookmarkDeltaOntoEvologBase: %v", err)
	}

	// Bookmark should still exist after the FAQ split (it is moved onto the new @ with --allow-backwards).
	bm := exec.Command("jj", "bookmark", "list")
	bm.Dir = dir
	out, err := bm.CombinedOutput()
	if err != nil {
		t.Fatalf("post jj bookmark list: %v\n%s", err, string(out))
	}
	if !strings.Contains(string(out), "demo/feature") {
		t.Fatalf("expected demo/feature in bookmark list after split, got:\n%s", string(out))
	}
}
