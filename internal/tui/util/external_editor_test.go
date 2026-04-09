package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRepoAbsPath(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o700); err != nil {
		t.Fatal(err)
	}
	got, err := RepoAbsPath(root, "src/foo.go")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "src", "foo.go")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	_, err = RepoAbsPath(root, "../escape")
	if err == nil {
		t.Fatal("expected error for path escape")
	}
}
