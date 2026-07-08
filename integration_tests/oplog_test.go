package integration_tests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/jj"
)

// TestOperationLog_ListAndRestore performs three mutating operations, verifies
// ListOperations surfaces them (newest first, current flagged), then restores
// to the earliest of the three and confirms the graph reflects the older state.
func TestOperationLog_ListAndRestore(t *testing.T) {
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

	svc, err := jj.NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	// Record how many operations exist before our three mutations, so the test
	// is robust to the number of setup operations (git init / config set).
	baseline, err := svc.ListOperations(ctx, 0)
	if err != nil {
		t.Fatalf("ListOperations baseline: %v", err)
	}

	// Operation 1: describe the working copy.
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("first\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runJJ("describe", "-m", "op-one describe")

	// Capture the operation id after op 1 — this is our restore target.
	afterOp1, err := svc.ListOperations(ctx, 1)
	if err != nil {
		t.Fatalf("ListOperations afterOp1: %v", err)
	}
	if len(afterOp1) == 0 {
		t.Fatal("expected at least one operation after describe")
	}
	restoreTarget := afterOp1[0].ID

	// Operation 2: new commit on top.
	runJJ("new", "-m", "op-two new")

	// Operation 3: describe again.
	runJJ("describe", "-m", "op-three describe")

	// List should include our three operations, newest first, current flagged.
	ops, err := svc.ListOperations(ctx, 0)
	if err != nil {
		t.Fatalf("ListOperations: %v", err)
	}
	if len(ops) < len(baseline)+3 {
		t.Fatalf("expected at least %d operations after 3 mutations, got %d", len(baseline)+3, len(ops))
	}
	if !ops[0].IsCurrent {
		t.Fatalf("newest operation should be flagged current: %+v", ops[0])
	}
	for i, o := range ops {
		if o.ID == "" {
			t.Fatalf("op[%d] has empty id: %+v", i, o)
		}
		if o.Time == "" {
			t.Fatalf("op[%d] has empty time: %+v", i, o)
		}
		if o.Description == "" {
			t.Fatalf("op[%d] has empty description: %+v", i, o)
		}
	}
	// jj auto-generates operation descriptions ("describe commit …", "new empty
	// commit …", etc.) rather than using our -m commit messages, so assert on the
	// operation *verbs* our three mutations produce.
	joined := opDescriptions(ops)
	for _, want := range []string{"describe commit", "new empty commit"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("operation log missing %q; descriptions: %s", want, joined)
		}
	}

	// The graph currently has op-three's description on the working copy.
	repoBefore, err := svc.GetRepository(ctx, "")
	if err != nil {
		t.Fatalf("GetRepository before restore: %v", err)
	}
	if !repoHasSummary(repoBefore.Graph.Commits, "op-three describe") {
		t.Fatalf("expected graph to contain 'op-three describe' before restore")
	}

	// Restore to the state right after op 1: the "op-two"/"op-three" work should
	// no longer be part of the visible repo state.
	if err := svc.RestoreOperation(ctx, restoreTarget); err != nil {
		t.Fatalf("RestoreOperation(%s): %v", restoreTarget, err)
	}

	repoAfter, err := svc.GetRepository(ctx, "")
	if err != nil {
		t.Fatalf("GetRepository after restore: %v", err)
	}
	if !repoHasSummary(repoAfter.Graph.Commits, "op-one describe") {
		t.Fatalf("expected graph to contain 'op-one describe' after restore; got %s",
			commitSummaries(repoAfter.Graph.Commits))
	}
	if repoHasSummary(repoAfter.Graph.Commits, "op-two new") ||
		repoHasSummary(repoAfter.Graph.Commits, "op-three describe") {
		t.Fatalf("restore to op-one should have removed op-two/op-three commits; got %s",
			commitSummaries(repoAfter.Graph.Commits))
	}
}

func opDescriptions(ops []jj.Operation) string {
	var b strings.Builder
	for _, o := range ops {
		b.WriteString(o.Description)
		b.WriteString("\n")
	}
	return b.String()
}

func repoHasSummary(commits []internal.Commit, summary string) bool {
	for _, c := range commits {
		if strings.Contains(c.Summary, summary) || strings.Contains(c.Description, summary) {
			return true
		}
	}
	return false
}

func commitSummaries(commits []internal.Commit) string {
	var b strings.Builder
	for _, c := range commits {
		b.WriteString(c.Summary)
		b.WriteString(" | ")
	}
	return b.String()
}
