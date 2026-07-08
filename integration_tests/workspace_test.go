package integration_tests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/madicen/jj-tui/internal/integrations/jj"
)

// TestWorkspaces_ListAddForget exercises the workspace MVP service: list the
// default workspace, add a second one, confirm it shows up (and the current one
// is flagged), then forget it.
func TestWorkspaces_ListAddForget(t *testing.T) {
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj not in PATH")
	}
	ctx := context.Background()
	dir := t.TempDir()

	runJJ := func(args ...string) {
		t.Helper()
		cmd := exec.Command("jj", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("jj %v: %v\n%s", args, err, string(out))
		}
	}

	runJJ("git", "init")
	runJJ("config", "set", "--repo", "user.name", "jj-tui integration")
	runJJ("config", "set", "--repo", "user.email", "integration@example.com")
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runJJ("describe", "-m", "c1")

	svc, err := jj.NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	// Initially just the default workspace, flagged current.
	ws, err := svc.ListWorkspaces(ctx)
	if err != nil {
		t.Fatalf("ListWorkspaces: %v", err)
	}
	if len(ws) != 1 {
		t.Fatalf("expected 1 workspace, got %d: %+v", len(ws), ws)
	}
	if ws[0].Name != "default" {
		t.Fatalf("expected default workspace, got %q", ws[0].Name)
	}
	if !ws[0].Current {
		t.Fatalf("expected default workspace to be flagged current: %+v", ws[0])
	}

	// Add a second workspace in a sibling directory (must be outside this repo's tree).
	secondaryRoot := filepath.Join(t.TempDir(), "secondary")
	if err := svc.AddWorkspace(ctx, secondaryRoot); err != nil {
		t.Fatalf("AddWorkspace: %v", err)
	}

	ws, err = svc.ListWorkspaces(ctx)
	if err != nil {
		t.Fatalf("ListWorkspaces after add: %v", err)
	}
	if len(ws) != 2 {
		t.Fatalf("expected 2 workspaces after add, got %d: %+v", len(ws), ws)
	}
	if !hasWorkspace(ws, "secondary") {
		t.Fatalf("expected a 'secondary' workspace after add, got %+v", ws)
	}
	// Exactly one workspace should be flagged current (the one we operate in).
	if n := countCurrent(ws); n != 1 {
		t.Fatalf("expected exactly one current workspace, got %d: %+v", n, ws)
	}

	// Forget it and confirm it's gone.
	if err := svc.ForgetWorkspace(ctx, "secondary"); err != nil {
		t.Fatalf("ForgetWorkspace: %v", err)
	}
	ws, err = svc.ListWorkspaces(ctx)
	if err != nil {
		t.Fatalf("ListWorkspaces after forget: %v", err)
	}
	if hasWorkspace(ws, "secondary") {
		t.Fatalf("expected 'secondary' workspace to be gone after forget, got %+v", ws)
	}
	if len(ws) != 1 {
		t.Fatalf("expected 1 workspace after forget, got %d: %+v", len(ws), ws)
	}
}

func hasWorkspace(ws []jj.Workspace, name string) bool {
	for _, w := range ws {
		if w.Name == name {
			return true
		}
	}
	return false
}

func countCurrent(ws []jj.Workspace) int {
	n := 0
	for _, w := range ws {
		if w.Current {
			n++
		}
	}
	return n
}
