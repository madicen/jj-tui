package jj

import "testing"

func TestParseResolveListOutput(t *testing.T) {
	out := "Warning: There are unresolved conflicts at these paths:\nf.txt    2-sided conflict\nother.go\t3-sided conflict\n"
	files := parseResolveListOutput(out)
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2: %+v", len(files), files)
	}
	if files[0].Path != "f.txt" || files[0].Kind != "2-sided conflict" {
		t.Errorf("files[0] = %+v", files[0])
	}
	if files[1].Path != "other.go" {
		t.Errorf("files[1] = %+v", files[1])
	}
}

func TestParseResolveListOutput_empty(t *testing.T) {
	if files := parseResolveListOutput("Error: No conflicts found at this revision\n"); len(files) != 0 {
		t.Fatalf("expected empty, got %+v", files)
	}
}
