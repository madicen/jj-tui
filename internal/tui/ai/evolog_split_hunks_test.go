package ai

import (
	"strings"
	"testing"

	"github.com/madicen/jj-tui/internal/integrations/jj"
)

func TestResolveEvologHunkPathKeyBasename(t *testing.T) {
	const git = `diff --git a/pkg/deep/foo.go b/pkg/deep/foo.go
--- a/pkg/deep/foo.go
+++ b/pkg/deep/foo.go
@@ -1,1 +1,2 @@
 a
+b
`
	hm, bin, err := jj.ParseGitUnifiedHunksPerPath(git)
	if err != nil {
		t.Fatal(err)
	}
	if len(bin) != 0 {
		t.Fatalf("bin=%v", bin)
	}
	key, ok := resolveEvologHunkPathKey("foo.go", hm, bin)
	if !ok || key != "pkg/deep/foo.go" {
		t.Fatalf("want pkg/deep/foo.go ok=%v got %q", ok, key)
	}
}

func TestValidateEvologHunkPrefixAgainstGitDiffOmitsWhenNoMatch(t *testing.T) {
	const git = `diff --git a/only.go b/only.go
--- a/only.go
+++ b/only.go
@@ -1,1 +1,2 @@
 x
+y
`
	clean, auto, note, err := validateEvologHunkPrefixAgainstGitDiff(git, map[string]int{"other.go": 1, "nope.go": 0})
	if err != nil {
		t.Fatal(err)
	}
	if len(auto) != 0 {
		t.Fatalf("unexpected auto files: %v", auto)
	}
	if len(clean) != 0 {
		t.Fatalf("expected empty clean, got %v", clean)
	}
	if !strings.Contains(note, "omitting hunk peel") {
		t.Fatalf("expected omit note, got %q", note)
	}
}

func TestValidateEvologHunkPrefixAgainstGitDiffBasenameResolve(t *testing.T) {
	const git = `diff --git a/pkg/x.go b/pkg/x.go
--- a/pkg/x.go
+++ b/pkg/x.go
@@ -1,1 +1,2 @@
 1
+2
`
	clean, auto, _, err := validateEvologHunkPrefixAgainstGitDiff(git, map[string]int{"x.go": 0})
	if err != nil {
		t.Fatal(err)
	}
	if len(auto) != 0 {
		t.Fatalf("unexpected auto: %v", auto)
	}
	if clean["pkg/x.go"] != 0 {
		t.Fatalf("got %v", clean)
	}
}

func TestValidateEvologHunkPrefixAgainstGitDiffSingleHunkPromotesToFiles(t *testing.T) {
	const git = `diff --git a/m.go b/m.go
--- a/m.go
+++ b/m.go
@@ -1,1 +1,2 @@
 x
+y
`
	clean, auto, note, err := validateEvologHunkPrefixAgainstGitDiff(git, map[string]int{"m.go": 4})
	if err != nil {
		t.Fatal(err)
	}
	if len(clean) != 0 {
		t.Fatalf("expected empty hunk map, got %v", clean)
	}
	if len(auto) != 1 || auto[0] != "m.go" {
		t.Fatalf("auto files: %#v", auto)
	}
	if !strings.Contains(note, "files_first_commit") {
		t.Fatalf("note: %q", note)
	}
}

func TestValidateEvologHunkPrefixAgainstGitDiffMixedSingleAndMultiHunkPaths(t *testing.T) {
	const git = `diff --git a/one.go b/one.go
--- a/one.go
+++ b/one.go
@@ -1,1 +1,2 @@
 a
+b
diff --git a/two.go b/two.go
--- a/two.go
+++ b/two.go
@@ -1,1 +1,3 @@
 x
+y
@@ -5,1 +6,2 @@
 z
+w
`
	clean, auto, _, err := validateEvologHunkPrefixAgainstGitDiff(git, map[string]int{"one.go": 1, "two.go": 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(auto) != 1 || auto[0] != "one.go" {
		t.Fatalf("auto=%v", auto)
	}
	if len(clean) != 1 || clean["two.go"] != 1 {
		t.Fatalf("clean=%v", clean)
	}
}
