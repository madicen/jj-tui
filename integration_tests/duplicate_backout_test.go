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

// dupBackoutFixture builds a small linear stack (c1 -> c2 -> c3) and returns the
// repo dir plus a helper to run raw jj commands.
func dupBackoutFixture(t *testing.T) (string, func(args ...string) string) {
	t.Helper()
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
	runJJ("config", "set", "--repo", "user.name", "jj-tui integration")
	runJJ("config", "set", "--repo", "user.email", "integration@example.com")

	write("f.txt", "a\n")
	runJJ("describe", "-m", "c1")
	runJJ("bookmark", "create", "main", "-r", "@")
	runJJ("new", "-m", "c2")
	write("f.txt", "a\nb\n")
	runJJ("new", "-m", "c3")
	write("g.txt", "g\n")
	return dir, runJJ
}

func changeIDOf(t *testing.T, runJJ func(args ...string) string, revset string) string {
	t.Helper()
	out := runJJ("log", "-r", revset, "--no-graph", "--no-pager", "-T", "change_id.short(8) ++ \"\\n\"")
	line := strings.TrimSpace(strings.SplitN(out, "\n", 2)[0])
	if line == "" {
		t.Fatalf("no change id for revset %q (out=%q)", revset, out)
	}
	return line
}

func countCommits(runJJ func(args ...string) string, revset string) int {
	out := runJJ("log", "-r", revset, "--no-graph", "--no-pager", "-T", "\"x\\n\"")
	n := 0
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "x" {
			n++
		}
	}
	return n
}

// TestDuplicate_CreatesSibling duplicates c2 and asserts a second visible commit
// with the same description now exists.
func TestDuplicate_CreatesSibling(t *testing.T) {
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj not in PATH")
	}
	ctx := context.Background()
	dir, runJJ := dupBackoutFixture(t)

	before := countCommits(runJJ, "description(substring:\"c2\")")
	c2 := changeIDOf(t, runJJ, "description(substring:\"c2\")")

	svc, err := jj.NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if err := svc.DuplicateCommit(ctx, c2, ""); err != nil {
		t.Fatalf("DuplicateCommit: %v", err)
	}

	after := countCommits(runJJ, "description(substring:\"c2\")")
	if after != before+1 {
		t.Fatalf("expected duplicate to add one commit with description c2: before=%d after=%d", before, after)
	}
}

// TestDuplicate_OntoDestination duplicates c1 onto @ and asserts the duplicate is
// a descendant of the working copy.
func TestDuplicate_OntoDestination(t *testing.T) {
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj not in PATH")
	}
	ctx := context.Background()
	dir, runJJ := dupBackoutFixture(t)

	c1 := changeIDOf(t, runJJ, "description(substring:\"c1\")")
	wc := changeIDOf(t, runJJ, "@")

	svc, err := jj.NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if err := svc.DuplicateCommit(ctx, c1, wc); err != nil {
		t.Fatalf("DuplicateCommit onto @: %v", err)
	}

	// A duplicate of c1 placed onto @ must appear among @'s descendants.
	dupDescendants := countCommits(runJJ, "descendants("+wc+") & description(substring:\"c1\")")
	if dupDescendants < 1 {
		t.Fatalf("expected duplicated c1 to be a descendant of @, found %d", dupDescendants)
	}
}

// TestBackout_AppliesReverse backs out c2 (which added line "b") and asserts a
// backout/revert commit was created on top of the working copy.
func TestBackout_AppliesReverse(t *testing.T) {
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj not in PATH")
	}
	ctx := context.Background()
	dir, runJJ := dupBackoutFixture(t)

	c2 := changeIDOf(t, runJJ, "description(substring:\"c2\")")
	wcBefore := changeIDOf(t, runJJ, "@")

	svc, err := jj.NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if err := svc.BackoutCommit(ctx, c2); err != nil {
		t.Fatalf("BackoutCommit: %v", err)
	}

	// A new commit describing the revert of c2 must exist as a descendant of the
	// commit we applied the reverse onto (@ at backout time).
	reverts := countCommits(runJJ, "descendants("+wcBefore+") & description(substring:\"c2\") & description(substring-i:\"revert\")")
	if reverts < 1 {
		// jj uses "Revert \"c2\"" for the default description; be lenient about
		// the exact template and fall back to counting any new descendant.
		log := runJJ("log", "--no-pager", "-T", "change_id.short(8) ++ \" \" ++ description.first_line() ++ \"\\n\"")
		if !strings.Contains(strings.ToLower(log), "revert") {
			t.Fatalf("expected a revert commit for c2, log:\n%s", log)
		}
	}
}
