package integration_tests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/data"
)

// TestGraphFilterRevset filters commits by description/author text and surfaces
// revset errors without requiring a loaded graph replacement.
func TestGraphFilterRevset(t *testing.T) {
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj not in PATH")
	}
	ctx := context.Background()
	dir := t.TempDir()

	runJJ := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("jj", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("jj %v: %v\n%s", args, err, string(out))
		}
		return string(out)
	}
	write := func(rel, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	runJJ("git", "init")
	runJJ("config", "set", "--repo", "user.name", "Alice Alpha")
	runJJ("config", "set", "--repo", "user.email", "alice@example.com")
	write("a.txt", "one\n")
	runJJ("describe", "-m", "alpha feature work")
	runJJ("new", "-m", "beta second change")
	runJJ("config", "set", "--repo", "user.name", "Bob Beta")
	runJJ("config", "set", "--repo", "user.email", "bob@example.com")
	write("b.txt", "two\n")
	runJJ("describe", "-m", "beta feature work")

	svc, err := jj.NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	revset, _, err := jj.CompileGraphFilterRevset("alpha")
	if err != nil {
		t.Fatalf("CompileGraphFilterRevset: %v", err)
	}
	repo, err := svc.GetRepository(ctx, revset)
	if err != nil {
		t.Fatalf("GetRepository filter: %v", err)
	}
	if len(repo.Graph.Commits) == 0 {
		t.Fatal("expected at least one commit matching 'alpha'")
	}
	foundAlpha := false
	for _, c := range repo.Graph.Commits {
		if c.Author == "Alice Alpha" || strings.Contains(strings.ToLower(c.Description), "alpha") {
			foundAlpha = true
		}
	}
	if !foundAlpha {
		t.Fatalf("filtered graph should include alpha commit; got %d commits", len(repo.Graph.Commits))
	}

	// Invalid revset returns an error via ApplyGraphFilterCmd without replacing repo.
	allRepo, err := svc.GetRepository(ctx, "")
	if err != nil {
		t.Fatalf("GetRepository all: %v", err)
	}
	cmd := data.ApplyGraphFilterCmd(svc, "!!!invalid!!!", "!!!invalid!!!")
	if cmd == nil {
		t.Fatal("ApplyGraphFilterCmd returned nil")
	}
	msg := cmd()
	filterMsg, ok := msg.(data.GraphFilterLoadedMsg)
	if !ok {
		t.Fatalf("expected GraphFilterLoadedMsg, got %T", msg)
	}
	if filterMsg.Err == nil {
		t.Fatal("expected revset error for invalid expression")
	}
	if filterMsg.Repository != nil {
		t.Fatal("error result should not include a replacement repository")
	}
	_ = allRepo // baseline load succeeded
}
