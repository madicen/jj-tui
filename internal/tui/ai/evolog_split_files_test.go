package ai

import "testing"

func TestPathsFromJJSummaryLines(t *testing.T) {
	m := pathsFromJJSummaryLines([]string{"M foo.go", "A bar/baz.txt", `M ./qux/./x.go`})
	if len(m) != 3 {
		t.Fatalf("got %d", len(m))
	}
	if _, ok := m["foo.go"]; !ok {
		t.Fatal("foo.go")
	}
	if _, ok := m["bar/baz.txt"]; !ok {
		t.Fatal("bar/baz.txt")
	}
	if _, ok := m["qux/x.go"]; !ok {
		t.Fatal("qux/x.go")
	}
}

func TestFilterEvologSplitFilePathsRejectsAll(t *testing.T) {
	allowed := map[string]struct{}{"a.go": {}}
	_, full, err := filterEvologSplitFilePaths(allowed, []string{"other.go"})
	if err == nil {
		t.Fatal("expected error")
	}
	if full {
		t.Fatal("unexpected fullPartition")
	}
}

func TestFilterEvologSplitFilePathsNormalizesSuggested(t *testing.T) {
	allowed := map[string]struct{}{"pkg/a.go": {}, "pkg/b.go": {}}
	kept, full, err := filterEvologSplitFilePaths(allowed, []string{"./pkg/a.go"})
	if err != nil {
		t.Fatal(err)
	}
	if full {
		t.Fatal("unexpected fullPartition")
	}
	if len(kept) != 1 || kept[0] != "pkg/a.go" {
		t.Fatalf("got %v", kept)
	}
}

func TestFilterEvologSplitFilePathsFullPartitionSoftSkips(t *testing.T) {
	allowed := map[string]struct{}{"a.go": {}, "b.go": {}}
	kept, full, err := filterEvologSplitFilePaths(allowed, []string{"a.go", "b.go"})
	if err != nil {
		t.Fatal(err)
	}
	if !full {
		t.Fatal("expected fullPartition when every path listed")
	}
	if kept != nil {
		t.Fatalf("expected nil kept, got %v", kept)
	}
}

func TestNormalizeRepoPathForDiffLeadingSlash(t *testing.T) {
	if got := normalizeRepoPathForDiff("/internal/foo.go"); got != "internal/foo.go" {
		t.Fatalf("got %q", got)
	}
	if got := normalizeRepoPathForDiff("./internal/foo.go"); got != "internal/foo.go" {
		t.Fatalf("got %q", got)
	}
}
