// Package termmouse writes CSI sequences to turn off xterm mouse reporting.
// Must be callable from tea.Model.Update (synchronously) before tea.Quit so the terminal
// stops emitting SGR mouse payloads while the user moves the pointer off the window.
package termmouse

import (
	"io"
	"os"
	"runtime"

	"github.com/charmbracelet/x/ansi"
)

var offSeq = ansi.ResetModeMouseX10 +
	ansi.ResetModeMouseNormal +
	ansi.ResetModeMouseHighlight +
	ansi.ResetModeMouseButtonEvent +
	ansi.ResetModeMouseAnyEvent +
	ansi.ResetModeMouseExtUtf8 +
	ansi.ResetModeMouseExtSgr +
	ansi.ResetModeMouseExtUrxvt +
	ansi.ResetModeMouseExtSgrPixel +
	"\x1b[0m"

func writeAndSync(w io.Writer) {
	if w == nil {
		return
	}
	_, _ = io.WriteString(w, offSeq)
	if f, ok := w.(*os.File); ok {
		_ = f.Sync()
	}
}

// Flush disables mouse tracking on stdout, stderr, and (Unix) /dev/tty.
func Flush() {
	writeAndSync(os.Stdout)
	writeAndSync(os.Stderr)
	if runtime.GOOS == "windows" {
		return
	}
	f, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if err != nil {
		return
	}
	_, _ = io.WriteString(f, offSeq)
	_ = f.Sync()
	_ = f.Close()
}
