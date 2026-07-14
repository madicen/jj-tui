package util

import (
	"os/exec"
	"reflect"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
)

// IsNilInterface reports whether an interface holds a nil concrete value.
// In Go, an interface is only nil if both type and value are nil; an interface
// holding a nil pointer (e.g. (*Service)(nil)) is not nil.
func IsNilInterface(i any) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	return v.Kind() == reflect.Pointer && v.IsNil()
}

// OpenURL opens a URL in the default browser. Returns a tea.Cmd that starts the browser.
func OpenURL(url string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "linux":
			cmd = exec.Command("xdg-open", url)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		default:
			return nil
		}
		_ = cmd.Start()
		return nil
	}
}
