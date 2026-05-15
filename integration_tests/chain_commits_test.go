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

// TestListChainCommits_LinearStackFromTrunk builds a small linear chain on top
// of trunk and asserts ListChainCommits returns the chain in oldest → newest
// order with subjects + descriptions intact. This is the core behavior the
// AI bookmark/ticket/PR generation paths rely on for "include the full chain
// of commits up to the selected one" context.
func TestListChainCommits_LinearStackFromTrunk(t *testing.T) {
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
	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	runJJ("git", "init")
	runJJ("config", "set", "--repo", "user.name", "jj-tui integration")
	runJJ("config", "set", "--repo", "user.email", "integration@example.com")

	// Establish a "main" bookmark on the very first real commit so trunk()
	// has something to resolve to (jj's default trunk() looks for main /
	// master / trunk on origin or local).
	write("README.md", "trunk\n")
	runJJ("describe", "-m", "trunk: initial commit")
	runJJ("bookmark", "create", "main", "-r", "@")

	runJJ("new", "-m", "Add login form")
	write("login.go", "package login\n")

	runJJ("new", "-m", "Wire login form to auth API\n\nIncludes retry on 5xx and basic telemetry.")
	write("auth.go", "package auth\n")

	runJJ("new", "-m", "Polish login form styles")
	write("styles.css", ".login{}\n")

	svc, err := jj.NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	commits, err := svc.ListChainCommits(ctx, "main", "@")
	if err != nil {
		t.Fatalf("ListChainCommits: %v", err)
	}
	// Three real commits above main, plus the empty working-copy commit jj
	// auto-creates after each `jj new`. We don't pin the exact count because
	// jj versions differ on whether the trailing empty WC counts; instead we
	// assert the three real subjects appear in order.
	wantSubjects := []string{
		"Add login form",
		"Wire login form to auth API",
		"Polish login form styles",
	}
	var seenIdx int
	for _, c := range commits {
		if seenIdx < len(wantSubjects) && c.Subject == wantSubjects[seenIdx] {
			seenIdx++
		}
	}
	if seenIdx != len(wantSubjects) {
		t.Fatalf("subjects out of order or missing.\nwant in order: %v\ngot commits: %#v", wantSubjects, commits)
	}

	// The middle commit's full description (subject + body) must round-trip;
	// this is what gives the AI the extra context the user asked for.
	var foundBody bool
	for _, c := range commits {
		if c.Subject == "Wire login form to auth API" {
			foundBody = strings.Contains(c.Description, "Includes retry on 5xx and basic telemetry.")
			if !strings.HasPrefix(c.Description, "Wire login form to auth API") {
				t.Fatalf("expected description to start with subject; got: %q", c.Description)
			}
			break
		}
	}
	if !foundBody {
		t.Fatalf("expected to find body 'Includes retry on 5xx and basic telemetry.' on the wire-up commit; got commits: %#v", commits)
	}
}

// TestListChainCommits_EmptyWhenSelectionAtOrBelowBase verifies the helper
// returns an empty slice (not an error) when there is no chain to walk.
// loadChainContext relies on this to fall back to the single-revision diff
// without surfacing a confusing error to the user.
func TestListChainCommits_EmptyWhenSelectionAtOrBelowBase(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runJJ("describe", "-m", "trunk: initial commit")
	runJJ("bookmark", "create", "main", "-r", "@")

	svc, err := jj.NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	commits, err := svc.ListChainCommits(ctx, "main", "main")
	if err != nil {
		t.Fatalf("ListChainCommits returned error for empty chain: %v", err)
	}
	if len(commits) != 0 {
		t.Fatalf("expected empty chain, got %d commits: %#v", len(commits), commits)
	}
}
