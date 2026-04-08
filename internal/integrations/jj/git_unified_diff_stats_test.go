package jj

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseGitUnifiedDiffStats(t *testing.T) {
	sample := `diff --git a/README.md b/README.md
index 0506d5b2ed..03abd61c78 100644
--- a/README.md
+++ b/README.md
@@ -3,0 +4,4 @@
+
+## Features
+- Dark mode
+- User settings
diff --git a/src/main.go b/src/main.go
new file mode 100644
index 0000000000..b13e29c032 100644
--- /dev/null
+++ b/src/main.go
@@ -0,0 +1,8 @@
+package main
+
+import "fmt"
+
+func main() {
+    fmt.Println("Hello, World!")
+}
+// TODO: Add more features
diff --git a/src/settings.go b/src/settings.go
new file mode 100644
index 0000000000..f92d5fc293 100644
--- /dev/null
+++ b/src/settings.go
@@ -0,0 +1,7 @@
+package main
+
+// Settings holds user preferences
+type Settings struct {
+    Theme    string
+    Language string
+}
`
	got := parseGitUnifiedDiffStats(sample)
	want := map[string]gitLineCounts{
		"README.md":     {added: 4, removed: 0},
		"src/main.go":   {added: 8, removed: 0},
		"src/settings.go": {added: 7, removed: 0},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseGitUnifiedDiffStats() = %#v, want %#v", got, want)
	}
}

func TestParseDiffGitBPath(t *testing.T) {
	p, ok := parseDiffGitBPath("diff --git a/README.md b/README.md")
	if !ok || p != "README.md" {
		t.Fatalf("got %q ok=%v", p, ok)
	}
	p2, ok2 := parseDiffGitBPath(`diff --git "a/spaced name" "b/spaced name"`)
	if !ok2 || p2 != "spaced name" {
		t.Fatalf("quoted: got %q ok=%v", p2, ok2)
	}
}

func TestMapGitUnifiedDiffByPath(t *testing.T) {
	sample := `diff --git a/README.md b/README.md
--- a/README.md
+++ b/README.md
@@ -1 +1 @@
-a
+b
diff --git a/x.go b/x.go
--- /dev/null
+++ b/x.go
@@ -0,0 +1 @@
+z
`
	m := mapGitUnifiedDiffByPath(sample)
	if len(m) != 2 || !strings.Contains(m["README.md"], "README.md") || !strings.Contains(m["x.go"], "x.go") {
		t.Fatalf("map keys/paths: %#v", m)
	}
}

func TestMaterialGitChunk(t *testing.T) {
	if materialGitChunk("") || materialGitChunk("diff --git a/x b/x\n--- a/x\n+++ b/x\n") {
		t.Fatal("headers-only should not be material")
	}
	if !materialGitChunk("diff --git a/x b/x\n-a\n+b\n") {
		t.Fatal("hunk line should be material")
	}
	if !materialGitChunk("Binary files a/foo and b/foo differ\n") {
		t.Fatal("binary message should be material")
	}
}

// Evolog-style dedupe: same README patch on consecutive steps should be treated as duplicate; new file only on second step stays.
func TestEvologPatchDedupeLogic(t *testing.T) {
	readme := `diff --git a/README.md b/README.md
--- a/README.md
+++ b/README.md
@@ -1 +1 @@
-old
+new
`
	mainGo := `diff --git a/main.go b/main.go
--- /dev/null
+++ b/main.go
@@ -0,0 +1 @@
+hi
`
	gitCur := readme + "\n" + mainGo
	gitPrev := readme
	cur := mapGitUnifiedDiffByPath(gitCur)
	prev := mapGitUnifiedDiffByPath(gitPrev)
	var kept []string
	for _, path := range []string{"README.md", "main.go"} {
		c := cur[path]
		p := prev[path]
		curMat := materialGitChunk(c)
		if !curMat {
			continue
		}
		if materialGitChunk(p) && normalizeGitChunkForCompare(c) == normalizeGitChunkForCompare(p) {
			continue
		}
		kept = append(kept, path)
	}
	if len(kept) != 1 || kept[0] != "main.go" {
		t.Fatalf("expected only main.go kept, got %v", kept)
	}
}

func TestNormalizeGitChunkForCompareIgnoresIndex(t *testing.T) {
	a := "diff --git a/x b/x\nindex aaa..bbb 100644\n--- a/x\n+++ b/x\n@@ -1 +1 @@\n-a\n+b\n"
	b := "diff --git a/x b/x\nindex ccc..ddd 100644\n--- a/x\n+++ b/x\n@@ -1 +1 @@\n-a\n+b\n"
	if normalizeGitChunkForCompare(a) != normalizeGitChunkForCompare(b) {
		t.Fatal("patches should match after stripping index lines")
	}
}
