package filediff

import (
	"strings"
	"testing"
)

func TestStyleGitUnifiedDiffLineNumberGutter(t *testing.T) {
	raw := `diff --git a/x.txt b/x.txt
index 1111111..2222222 100644
--- a/x.txt
+++ b/x.txt
@@ -1,3 +1,3 @@
 keep
-old
+new
 tail
`
	out := StyleGitUnifiedDiff(raw, 100)
	// Gutter columns: context line 1|1, removal 2|blank, addition blank|2, context 3|3
	if !strings.Contains(out, "   1│   1") {
		t.Fatalf("expected context gutter 1|1 in output:\n%s", out)
	}
	if !strings.Contains(out, "   2│    ") || !strings.Contains(out, "    │   2") {
		t.Fatalf("expected removal/addition gutters in output:\n%s", out)
	}
}

func TestStyleGitUnifiedDiffNonGitPassthrough(t *testing.T) {
	raw := "not a git diff\n"
	if got := StyleGitUnifiedDiff(raw, 40); got != raw {
		t.Fatalf("non-git diff should pass through unchanged, got %q", got)
	}
}
