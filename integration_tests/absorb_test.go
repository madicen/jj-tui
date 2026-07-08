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

// TestAbsorb_DryRunMatchesResult builds a stack where a working-copy edit
// belongs to an ancestor, then asserts the AbsorbDryRun preview names the same
// target revision(s) that a real Absorb reports, and that the change actually
// lands in the ancestor.
func TestAbsorb_DryRunMatchesResult(t *testing.T) {
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
	runJJ("config", "set", "--repo", "user.name", "jj-tui integration")
	runJJ("config", "set", "--repo", "user.email", "integration@example.com")

	// base introduces file.txt with three lines.
	write("file.txt", "line1\nline2\nline3\n")
	runJJ("describe", "-m", "base: add file")
	runJJ("bookmark", "create", "main", "-r", "@")
	baseChange := strings.TrimSpace(runJJ("log", "-r", "@", "--no-graph", "--no-pager", "-T", "change_id.short(8)"))

	// A second commit on top, then an empty working copy where we edit a line
	// that was last modified by base — absorb should route it into base.
	runJJ("new", "-m", "second")
	write("other.txt", "other\n")
	runJJ("new")
	write("file.txt", "line1\nCHANGED\nline3\n")

	svc, err := jj.NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	preview, err := svc.AbsorbDryRun(ctx)
	if err != nil {
		t.Fatalf("AbsorbDryRun: %v", err)
	}
	if preview.Nothing {
		t.Fatalf("expected something to absorb, got Nothing=true (summary=%q)", preview.Summary)
	}
	if len(preview.Targets) == 0 {
		t.Fatalf("expected at least one absorb target, got none (summary=%q)", preview.Summary)
	}
	previewChanges := targetChangeIDs(preview.Targets)
	if !containsChange(previewChanges, baseChange) {
		t.Fatalf("expected base change %q among preview targets %v", baseChange, previewChanges)
	}

	// The dry run must not have mutated the repo: the working copy still carries
	// the edit before we run the real absorb.
	if diff := strings.TrimSpace(runJJ("diff", "-r", "@", "--no-pager")); diff == "" {
		t.Fatalf("expected working-copy diff to remain after dry run, got empty")
	}

	result, err := svc.Absorb(ctx)
	if err != nil {
		t.Fatalf("Absorb: %v", err)
	}
	resultChanges := targetChangeIDs(result.Targets)
	if !sameChangeSet(previewChanges, resultChanges) {
		t.Fatalf("dry-run targets %v != actual targets %v", previewChanges, resultChanges)
	}

	// The edited line must now live in base, and the working copy must be clean.
	baseContent := runJJ("file", "show", "-r", baseChange, "file.txt")
	if !strings.Contains(baseContent, "CHANGED") {
		t.Fatalf("expected absorbed change in base file.txt, got:\n%s", baseContent)
	}
	if diff := strings.TrimSpace(runJJ("diff", "-r", "@", "--no-pager")); diff != "" {
		t.Fatalf("expected clean working copy after absorb, got diff:\n%s", diff)
	}
}

// TestAbsorb_NothingToAbsorb verifies the "nothing absorbed" case is reported
// gracefully rather than as an error.
func TestAbsorb_NothingToAbsorb(t *testing.T) {
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
	runJJ("describe", "-m", "base")
	// Fresh empty working copy: nothing to absorb.
	runJJ("new")

	svc, err := jj.NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	preview, err := svc.AbsorbDryRun(ctx)
	if err != nil {
		t.Fatalf("AbsorbDryRun: %v", err)
	}
	if !preview.Nothing {
		t.Fatalf("expected Nothing=true, got targets=%v summary=%q", preview.Targets, preview.Summary)
	}
}

func targetChangeIDs(targets []jj.AbsorbTarget) []string {
	out := make([]string, 0, len(targets))
	for _, t := range targets {
		out = append(out, t.ChangeID)
	}
	return out
}

func containsChange(ids []string, want string) bool {
	for _, id := range ids {
		if changePrefixMatch(id, want) {
			return true
		}
	}
	return false
}

func sameChangeSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for _, x := range a {
		if !containsChange(b, x) {
			return false
		}
	}
	return true
}

// changePrefixMatch tolerates differing short-id lengths between commands.
func changePrefixMatch(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return strings.HasPrefix(a, b) || strings.HasPrefix(b, a)
}
