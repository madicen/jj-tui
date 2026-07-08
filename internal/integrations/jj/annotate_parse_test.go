package jj

import (
	"reflect"
	"testing"
)

func TestParseAnnotateOutput(t *testing.T) {
	// Second line is blank (content empty); third line has a leading tab that
	// must be preserved in Content (indented source code).
	const sample = "snnsylnk\tmichael.madicen\t2 minutes ago\t1\tline one\n" +
		"abcd1234\talice\t3 hours ago\t2\t\n" +
		"snnsylnk\tmichael.madicen\t2 minutes ago\t3\t\tindented\n"
	lines, err := parseAnnotateOutput(sample)
	if err != nil {
		t.Fatalf("parseAnnotateOutput: %v", err)
	}
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %+v", len(lines), lines)
	}
	want0 := AnnotationLine{ChangeID: "snnsylnk", Author: "michael.madicen", Age: "2 minutes ago", LineNumber: 1, Content: "line one"}
	if !reflect.DeepEqual(lines[0], want0) {
		t.Errorf("lines[0] = %+v, want %+v", lines[0], want0)
	}
	if lines[1].Content != "" || lines[1].LineNumber != 2 || lines[1].Author != "alice" {
		t.Errorf("lines[1] = %+v, want blank content on line 2 by alice", lines[1])
	}
	if lines[2].Content != "\tindented" {
		t.Errorf("lines[2].Content = %q, want leading tab preserved", lines[2].Content)
	}
}

func TestParseAnnotateOutput_BadLineNumber(t *testing.T) {
	if _, err := parseAnnotateOutput("abc\talice\t1 hour ago\tNaN\ttext\n"); err == nil {
		t.Fatal("expected error for non-numeric line number")
	}
}
