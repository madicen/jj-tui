package jj

import (
	"testing"
)

func TestParseChangedFilesStatLogOutput(t *testing.T) {
	const sample = "README.md\tM\t4\t0\n src/main.go\tA\t8\t0\n"
	files, err := parseChangedFilesStatLogOutput(sample)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("len %d", len(files))
	}
	if files[0].Path != "README.md" || files[0].Status != "M" || files[0].LinesAdded != 4 || files[0].LinesRemoved != 0 || !files[0].StatsOK {
		t.Errorf("first: %+v", files[0])
	}
	if files[1].Path != "src/main.go" || files[1].Status != "A" || files[1].LinesAdded != 8 || !files[1].StatsOK {
		t.Errorf("second: %+v", files[1])
	}
}
