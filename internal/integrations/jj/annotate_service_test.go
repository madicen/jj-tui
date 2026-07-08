package jj_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/mock"
)

// TestAnnotateFileArgs verifies AnnotateFile builds the expected argv (file
// annotate + template, optional -r, trailing path) and parses the canned
// template output through the fake runner.
func TestAnnotateFileArgs(t *testing.T) {
	const sample = "abcd1234\talice\t1 hour ago\t1\thello\n"
	run := func(rev string) ([]string, []jj.AnnotationLine) {
		var gotArgs []string
		fake := &mock.FakeRunner{
			RunOutputFn: func(_ context.Context, _ jj.RunOpts, args ...string) (string, error) {
				gotArgs = append([]string(nil), args...)
				return sample, nil
			},
		}
		svc := jj.NewServiceWithRunner("/fake/repo", fake)
		lines, err := svc.AnnotateFile(context.Background(), rev, "f.txt")
		if err != nil {
			t.Fatalf("AnnotateFile(rev=%q): %v", rev, err)
		}
		return gotArgs, lines
	}

	argsNoRev, lines := run("")
	if len(argsNoRev) < 3 || argsNoRev[0] != "file" || argsNoRev[1] != "annotate" {
		t.Fatalf("expected `file annotate ...`, got %v", argsNoRev)
	}
	if argsNoRev[len(argsNoRev)-1] != "f.txt" {
		t.Fatalf("expected path as last arg, got %v", argsNoRev)
	}
	for _, a := range argsNoRev {
		if a == "-r" {
			t.Fatalf("did not expect -r when revision empty, got %v", argsNoRev)
		}
	}
	if len(lines) != 1 || lines[0].ChangeID != "abcd1234" || lines[0].Content != "hello" || lines[0].LineNumber != 1 {
		t.Fatalf("parsed lines wrong: %+v", lines)
	}

	argsWithRev, _ := run("main")
	// -r main must appear immediately before the trailing path.
	tail := argsWithRev[len(argsWithRev)-3:]
	if !reflect.DeepEqual(tail, []string{"-r", "main", "f.txt"}) {
		t.Fatalf("with-rev tail = %v, want [-r main f.txt]", tail)
	}
}

func TestAnnotateFile_EmptyPath(t *testing.T) {
	svc := jj.NewServiceWithRunner("/fake/repo", &mock.FakeRunner{})
	if _, err := svc.AnnotateFile(context.Background(), "@", "  "); err == nil {
		t.Fatal("expected error for empty path")
	}
}
