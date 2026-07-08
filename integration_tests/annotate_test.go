package integration_tests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/madicen/jj-tui/internal/integrations/jj"
)

// TestAnnotateFile_BlamesEachLine builds a two-change history where different
// changes introduced different lines of the same file, then asserts
// AnnotateFile attributes each line to the change that introduced it.
func TestAnnotateFile_BlamesEachLine(t *testing.T) {
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

	// c1 introduces lines 1-2, c2 appends line 3.
	write("f.txt", "line one\nline two\n")
	runJJ("describe", "-m", "c1")
	c1 := changeIDOf(t, runJJ, "@")
	runJJ("new", "-m", "c2")
	write("f.txt", "line one\nline two\nline three\n")
	c2 := changeIDOf(t, runJJ, "@")

	svc, err := jj.NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	lines, err := svc.AnnotateFile(ctx, "", "f.txt")
	if err != nil {
		t.Fatalf("AnnotateFile: %v", err)
	}
	if len(lines) != 3 {
		t.Fatalf("expected 3 annotated lines, got %d: %+v", len(lines), lines)
	}

	// Line numbers must be sequential 1..3 with the original text preserved.
	wantContent := []string{"line one", "line two", "line three"}
	for i, l := range lines {
		if l.LineNumber != i+1 {
			t.Errorf("line %d has LineNumber %d", i, l.LineNumber)
		}
		if l.Content != wantContent[i] {
			t.Errorf("line %d Content = %q, want %q", i+1, l.Content, wantContent[i])
		}
		if l.ChangeID == "" || l.Author == "" || l.Age == "" {
			t.Errorf("line %d missing metadata: %+v", i+1, l)
		}
	}

	// Blame attribution: lines 1-2 come from c1, line 3 from c2. Change-ids are
	// shortened in the template, so compare by prefix in either direction.
	if !changeMatches(lines[0].ChangeID, c1) || !changeMatches(lines[1].ChangeID, c1) {
		t.Errorf("lines 1-2 should be blamed on c1 (%s), got %q and %q", c1, lines[0].ChangeID, lines[1].ChangeID)
	}
	if !changeMatches(lines[2].ChangeID, c2) {
		t.Errorf("line 3 should be blamed on c2 (%s), got %q", c2, lines[2].ChangeID)
	}
}

// changeMatches reports whether two change-ids refer to the same change, given
// one may be a shortened prefix of the other.
func changeMatches(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if len(a) <= len(b) {
		return b[:len(a)] == a
	}
	return a[:len(b)] == b
}
