package jj

import (
	"strings"
	"testing"
)

func TestParseAndApplyUnifiedHunkPrefix(t *testing.T) {
	const git = `diff --git a/foo.txt b/foo.txt
index 111..222 100644
--- a/foo.txt
+++ b/foo.txt
@@ -1,2 +1,3 @@
 line1
-old2
+new2
+line3
`
	hm, err := ParseGitUnifiedHunksPerPath(git)
	if err != nil {
		t.Fatal(err)
	}
	hunks := hm["foo.txt"]
	if len(hunks) != 1 {
		t.Fatalf("hunks: %d", len(hunks))
	}
	left := "line1\nold2\n"
	right := "line1\nnew2\nline3\n"
	if err := VerifyUnifiedHunksReconstructRight(left, right, hunks); err != nil {
		t.Fatal(err)
	}
	got, err := ApplyUnifiedHunkPrefix(left, hunks, 0)
	if err != nil || got != left {
		t.Fatalf("prefix 0: %v %q", err, got)
	}
	got1, err := ApplyUnifiedHunkPrefix(left, hunks, 1)
	if err != nil {
		t.Fatal(err)
	}
	if normalizePatchText(got1) != normalizePatchText(right) {
		t.Fatalf("prefix 1 got %q want %q", got1, right)
	}
}

func TestParseTwoHunksPrefix(t *testing.T) {
	const git = `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,3 +1,4 @@
 package p
+import "fmt"
 const x = 1
@@ -8,2 +9,3 @@
 func f() {
+	fmt.Println()
 }
`
	hm, err := ParseGitUnifiedHunksPerPath(git)
	if err != nil {
		t.Fatal(err)
	}
	hunks := hm["a.go"]
	if len(hunks) != 2 {
		t.Fatalf("want 2 hunks got %d", len(hunks))
	}
	left := "package p\nconst x = 1\n\n\n\n\n\nfunc f() {\n}\n"
	right := "package p\nimport \"fmt\"\nconst x = 1\n\n\n\n\n\nfunc f() {\n\tfmt.Println()\n}\n"
	if err := VerifyUnifiedHunksReconstructRight(left, right, hunks); err != nil {
		t.Fatal(err)
	}
	mid, err := ApplyUnifiedHunkPrefix(left, hunks, 1)
	if err != nil {
		t.Fatal(err)
	}
	// After first hunk only: import added
	if !strings.Contains(mid, `import "fmt"`) || strings.Contains(mid, "fmt.Println") {
		t.Fatalf("unexpected mid %q", mid)
	}
}
