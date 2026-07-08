package testutil

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// updateGolden, when set via `-update`, rewrites golden files instead of
// comparing against them. Run e.g. `go test ./internal/tui/model -run TestGolden -update`.
var updateGolden = flag.Bool("update", false, "update golden files instead of comparing against them")

// ForceDeterministicRendering pins lipgloss/termenv global rendering state so that
// View() output (including ANSI color escapes) is identical across machines,
// CI runners, and TTY/no-TTY environments. Call this from TestMain in any package
// that captures rendered View() output as golden files.
//
// Without this, lipgloss auto-detects the terminal color profile (Ascii when no
// TTY, TrueColor in a rich terminal), which would make goldens flake between
// local runs and CI. main.go pins the same TrueColor profile in demo mode.
func ForceDeterministicRendering() {
	lipgloss.SetColorProfile(termenv.TrueColor)
	lipgloss.SetHasDarkBackground(true)
}

// AssertGolden compares actual against the golden file testdata/<name>.golden,
// failing the test on any difference. With the -update flag it (re)writes the
// golden file instead. name may contain slashes to nest golden files in
// subdirectories of testdata/.
//
// The comparison is byte-exact on the full string (including ANSI escapes), so
// ANY user-visible change — a moved line, a dropped word, a changed color — is a
// test failure. That is the whole point: this is the safety net that must fail
// loudly if a refactor changes rendered output.
func AssertGolden(t *testing.T, name, actual string) {
	t.Helper()
	golden := filepath.Join("testdata", filepath.FromSlash(name)+".golden")

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
			t.Fatalf("golden: mkdir %s: %v", filepath.Dir(golden), err)
		}
		if err := os.WriteFile(golden, []byte(actual), 0o644); err != nil {
			t.Fatalf("golden: write %s: %v", golden, err)
		}
		return
	}

	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("golden: cannot read %s: %v\n(create it with: go test -run '%s' -update)", golden, err, t.Name())
	}
	if string(want) != actual {
		t.Errorf("golden mismatch for %s\n%s\n(if this change is intentional, re-run with -update)",
			golden, firstLineDiff(string(want), actual))
	}
}

// firstLineDiff renders a compact, human-readable line-level diff highlighting
// the first divergence, so failures point at the exact changed line rather than
// dumping two full screens of ANSI.
func firstLineDiff(want, got string) string {
	wl := strings.Split(want, "\n")
	gl := strings.Split(got, "\n")
	lineCount := len(wl)
	if len(gl) > lineCount {
		lineCount = len(gl)
	}
	for i := 0; i < lineCount; i++ {
		var w, g string
		if i < len(wl) {
			w = wl[i]
		}
		if i < len(gl) {
			g = gl[i]
		}
		if w != g {
			return fmt.Sprintf("first difference at line %d:\n  want: %q\n  got:  %q\n(want has %d lines, got has %d lines)",
				i+1, w, g, len(wl), len(gl))
		}
	}
	return fmt.Sprintf("content differs but no line-level difference found (want %d bytes, got %d bytes)", len(want), len(got))
}
