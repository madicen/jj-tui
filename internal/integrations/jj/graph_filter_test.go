package jj

import (
	"strings"
	"testing"
)

func TestCompileGraphFilterRevset_freeText(t *testing.T) {
	rev, display, err := CompileGraphFilterRevset("golden")
	if err != nil {
		t.Fatal(err)
	}
	if display != "golden" {
		t.Errorf("display = %q, want golden", display)
	}
	want := `description(substring-i:"golden") | author(substring-i:"golden")`
	if rev != want {
		t.Errorf("revset = %q, want %q", rev, want)
	}
}

func TestCompileGraphFilterRevset_rawRevset(t *testing.T) {
	rev, display, err := CompileGraphFilterRevset(":mine() | trunk()")
	if err != nil {
		t.Fatal(err)
	}
	if rev != "mine() | trunk()" || display != "mine() | trunk()" {
		t.Errorf("got revset=%q display=%q", rev, display)
	}
}

func TestCompileGraphFilterRevset_escapesQuotes(t *testing.T) {
	rev, _, err := CompileGraphFilterRevset(`say "hi"`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rev, `\"hi\"`) {
		t.Errorf("revset should escape inner quotes: %q", rev)
	}
}

func TestCompileGraphFilterRevset_empty(t *testing.T) {
	if _, _, err := CompileGraphFilterRevset("  "); err == nil {
		t.Fatal("expected error for empty input")
	}
	if _, _, err := CompileGraphFilterRevset(":"); err == nil {
		t.Fatal("expected error for empty raw revset")
	}
}
