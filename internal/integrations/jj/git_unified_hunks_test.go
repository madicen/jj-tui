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
	hm, bin, err := ParseGitUnifiedHunksPerPath(git)
	if err != nil {
		t.Fatal(err)
	}
	if len(bin) != 0 {
		t.Fatalf("unexpected binary paths: %v", bin)
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

func TestValidateHunkPrefixPlan(t *testing.T) {
	const git = `diff --git a/x.go b/x.go
--- a/x.go
+++ b/x.go
@@ -1,1 +1,2 @@
 a
+b
`
	if err := ValidateHunkPrefixPlan(git, map[string]int{"x.go": 1}); err == nil {
		t.Fatal("expected error when k equals hunk count (all hunks to first commit)")
	}
	if err := ValidateHunkPrefixPlan(git, map[string]int{"x.go": 0}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateHunkPrefixPlan(git, map[string]int{"x.go": 2}); err == nil {
		t.Fatal("expected error when k exceeds hunk count")
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
	hm, bin, err := ParseGitUnifiedHunksPerPath(git)
	if err != nil {
		t.Fatal(err)
	}
	if len(bin) != 0 {
		t.Fatalf("unexpected binary paths: %v", bin)
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

// Regression: the last @@ hunk of a file must not be dropped when the next diff --git starts.
func TestParseGitUnifiedHunksMultiTextFilesKeepsFirstFileHunk(t *testing.T) {
	const git = `diff --git a/one.go b/one.go
--- a/one.go
+++ b/one.go
@@ -1,1 +1,2 @@
 a
+b
diff --git a/two.go b/two.go
--- a/two.go
+++ b/two.go
@@ -1,1 +1,2 @@
 x
+y
`
	hm, bin, err := ParseGitUnifiedHunksPerPath(git)
	if err != nil {
		t.Fatal(err)
	}
	if len(bin) != 0 {
		t.Fatalf("bin=%v", bin)
	}
	if len(hm["one.go"]) != 1 {
		t.Fatalf("one.go hunks: %v", hm["one.go"])
	}
	if len(hm["two.go"]) != 1 {
		t.Fatalf("two.go hunks: %v", hm["two.go"])
	}
}

func TestParseGitUnifiedHunksBinaryMarker(t *testing.T) {
	const git = `diff --git a/logo.png b/logo.png
index 111..222 100644
Binary files a/logo.png and b/logo.png differ
diff --git a/x.go b/x.go
--- a/x.go
+++ b/x.go
@@ -1,1 +1,2 @@
 a
+b
`
	hm, bin, err := ParseGitUnifiedHunksPerPath(git)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := bin["logo.png"]; !ok {
		t.Fatalf("expected logo.png in binaryPaths, got %v", bin)
	}
	if len(hm["logo.png"]) != 0 {
		t.Fatalf("binary path should have no @@ hunks, got %d", len(hm["logo.png"]))
	}
	if len(hm["x.go"]) != 1 {
		t.Fatalf("x.go hunks: %d", len(hm["x.go"]))
	}
}

func TestValidateHunkPrefixPlanWithBinaryInDiff(t *testing.T) {
	const git = `diff --git a/logo.png b/logo.png
Binary files a/logo.png and b/logo.png differ
diff --git a/x.go b/x.go
--- a/x.go
+++ b/x.go
@@ -1,1 +1,2 @@
 a
+b
`
	if err := ValidateHunkPrefixPlan(git, map[string]int{"x.go": 0}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateHunkPrefixPlan(git, map[string]int{"x.go": 0, "logo.png": 0}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateHunkPrefixPlan(git, map[string]int{"logo.png": 1}); err == nil {
		t.Fatal("expected error for binary path with k!=0")
	}
}

func TestSanitizeHunkPrefixMapAgainstDiff_stalePathDropped(t *testing.T) {
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
	got, err := SanitizeHunkPrefixMapAgainstDiff(git, map[string]int{
		"internal/integrations/jj/service.go": 1,
		"a.go":                                  1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got["a.go"] != 1 {
		t.Fatalf("got %#v", got)
	}
	if err := ValidateHunkPrefixPlan(git, got); err != nil {
		t.Fatal(err)
	}
}

func TestSanitizeHunkPrefixMapAgainstDiff_staleOnlyReturnsNil(t *testing.T) {
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
	got, err := SanitizeHunkPrefixMapAgainstDiff(git, map[string]int{"internal/integrations/jj/service.go": 1})
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("want nil map, got %#v", got)
	}
}

func TestSanitizeHunkPrefixMapAgainstDiff_clampsOversizedK(t *testing.T) {
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
	got, err := SanitizeHunkPrefixMapAgainstDiff(git, map[string]int{"a.go": 99})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got["a.go"] != 1 {
		t.Fatalf("want a.go:1 (len-1), got %#v", got)
	}
	if err := ValidateHunkPrefixPlan(git, got); err != nil {
		t.Fatal(err)
	}
}
