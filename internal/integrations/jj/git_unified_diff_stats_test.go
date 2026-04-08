package jj

import (
	"reflect"
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
