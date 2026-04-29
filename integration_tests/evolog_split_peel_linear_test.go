package integration_tests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/madicen/jj-tui/internal/integrations/jj"
)

// assertJJAncestorsOfAtHaveNoMerges fails if any commit on the first-parent chain from @ through
// roots has more than one parent (a merge), which would indicate sibling peels instead of a chain.
func assertJJAncestorsOfAtHaveNoMerges(t *testing.T, repoDir string) {
	t.Helper()
	cmd := exec.Command("jj", "log", "-r", "::@", "--no-graph", "-T", `parents.len() ++ "\n"`)
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("jj log ancestors of @: %v", err)
	}
	for i, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		n, perr := strconv.Atoi(line)
		if perr != nil {
			t.Fatalf("line %d: parse parents.len %q: %v", i, line, perr)
		}
		if n > 1 {
			t.Fatalf("merge or multi-parent commit on ::@ at line %d (parents.len=%d) — expected a linear first-parent chain after sequential peels.\njj log -r ::@ --no-graph:\n%s",
				i, n, jjLogNoGraph(t, repoDir, "::@"))
		}
	}
}

// assertJJCommandHistoryContainsInsertBefore fails unless some recorded jj split included
// --insert-before (used when @ has exactly one child so peels stack linearly).
func assertJJCommandHistoryContainsInsertBefore(t *testing.T, svc *jj.Service) {
	t.Helper()
	for _, e := range svc.GetCommandHistory() {
		if strings.Contains(e.Command, " split ") && strings.Contains(e.Command, "--insert-before") {
			return
		}
	}
	var hist strings.Builder
	for _, e := range svc.GetCommandHistory() {
		hist.WriteString(e.Command)
		hist.WriteByte('\n')
	}
	t.Fatalf("expected at least one jj split with --insert-before in command history; got:\n%s", hist.String())
}

func mustJJCwd(t *testing.T, repoDir string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command("jj", args...)
	cmd.Dir = repoDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("jj %v in %s: %v\n%s", args, repoDir, err, string(out))
	}
	return out
}

func jjLogNoGraph(t *testing.T, repoDir, revset string) string {
	t.Helper()
	cmd := exec.Command("jj", "log", "-r", revset, "--no-graph", "-T", `commit_id.shortest(8) ++ "  " ++ description.first_line() ++ "\n"`)
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return "(jj log failed: " + err.Error() + ")"
	}
	return string(out)
}

// TestSplitRevisionByFilesetsSequentialPeelsLinearWhenChildBehindWC builds a revision @ with
// exactly one direct child (stack continues above the peeled layer), runs two sequential
// SplitRevisionByFilesets peels, and asserts the result stays a linear first-parent chain and
// that jj split was invoked with --insert-before (appendSplitInsertBeforeArgs).
func TestSplitRevisionByFilesetsSequentialPeelsLinearWhenChildBehindWC(t *testing.T) {
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj not in PATH")
	}
	ctx := context.Background()
	dir := t.TempDir()

	run := func(name string, args ...string) {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s %v: %v\n%s", name, args, err, string(out))
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

	run("jj", "git", "init")
	run("jj", "config", "set", "--repo", "user.name", "jj-tui integration")
	run("jj", "config", "set", "--repo", "user.email", "integration@example.com")

	write("f1.txt", "a")
	write("f2.txt", "b")
	run("jj", "describe", "-m", "both files")
	// Child on top so @ can be edited back to "both files" with exactly one direct child.
	run("jj", "new")
	write("f3.txt", "c")
	run("jj", "describe", "-m", "stack tip child")
	run("jj", "edit", "@-")

	// Sanity: exactly one direct child of the working-copy revision (insert-before triggers here).
	atID := strings.TrimSpace(string(mustJJCwd(t, dir, "log", "-r", "@", "--no-graph", "-T", "commit_id", "--limit", "1")))
	if atID == "" {
		t.Fatal("empty @ commit id")
	}
	revset := "children(" + atID + ")"
	chOut := mustJJCwd(t, dir, "log", "-r", revset, "--no-graph", "-T", "commit_id")
	chOutStr := strings.TrimSpace(string(chOut))
	if chOutStr == "" {
		t.Fatalf("expected exactly one child of @ before peels; revset=%q out empty.\njj log -r 'all()':\n%s",
			revset, strings.TrimSpace(string(mustJJCwd(t, dir, "log", "-r", "all()", "--no-graph", "-T", `commit_id.shortest(8) ++ " " ++ description.first_line() ++ "\n"`))))
	}
	if strings.Count(chOutStr, "\n") > 0 {
		t.Fatalf("expected a single child of @, got multiple lines: %q", chOutStr)
	}

	svc, err := jj.NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	if err := svc.SplitRevisionByFilesets(ctx, "@", jj.EvologSplitFilePeelMessage, []string{"f1.txt"}); err != nil {
		t.Fatalf("first SplitRevisionByFilesets: %v", err)
	}
	if err := svc.SplitRevisionByFilesets(ctx, "@", jj.EvologSplitFilePeelMessage, []string{"f2.txt"}); err != nil {
		t.Fatalf("second SplitRevisionByFilesets: %v", err)
	}

	assertJJAncestorsOfAtHaveNoMerges(t, dir)
	assertJJCommandHistoryContainsInsertBefore(t, svc)
}

// TestDiffSummaryLinesFromTo_WorkingCopySmoke exercises DiffSummaryLinesFromTo("@-","@") used by
// the evolog split outcome preview overlay (jj diff --summary parent WC → working copy).
func TestDiffSummaryLinesFromTo_WorkingCopySmoke(t *testing.T) {
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
	runJJ("describe", "-m", "root")
	p := filepath.Join(dir, "tracked.txt")
	if err := os.WriteFile(p, []byte("v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runJJ("describe", "-m", "add file")

	svc, err := jj.NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	lines, err := svc.DiffSummaryLinesFromTo(ctx, "@-", "@")
	if err != nil {
		t.Fatalf("DiffSummaryLinesFromTo: %v", err)
	}
	if len(lines) == 0 {
		t.Fatal("expected non-empty diff summary lines for @- vs @")
	}
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "tracked.txt") {
		t.Fatalf("expected summary to mention tracked.txt; got:\n%s", joined)
	}
}
