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

// If returns trueVal when condition is true, otherwise falseVal. Generic ternary helper.
func If[T any](condition bool, trueVal, falseVal T) T {
	if condition {
		return trueVal
	}
	return falseVal
}

// PropagateUpdate calls Update(msg) on each updatable (pointer to a model with Update(tea.Msg) (tea.Model, tea.Cmd)),
// updates the value at the pointer, and returns the collected commands.
func PropagateUpdate(msg tea.Msg, updatables ...any) (results []tea.Cmd) {
	for _, updatable := range updatables {
		ptrValue := reflect.ValueOf(updatable)
		if ptrValue.Kind() != reflect.Ptr {
			panic("updatable must be a pointer")
		}
		method := ptrValue.MethodByName("Update")
		if !method.IsValid() {
			panic("updatable must have an Update method")
		}
		callResults := method.Call([]reflect.Value{reflect.ValueOf(msg)})
		if len(callResults) != 2 {
			panic("Update method must return (model, tea.Cmd)")
		}
		updatedValue := callResults[0]
		if updatedValue.Kind() == reflect.Interface && !updatedValue.IsNil() {
			updatedValue = updatedValue.Elem()
		}
		cmd, ok := callResults[1].Interface().(tea.Cmd)
		if !ok {
			panic("second return value from Update must be tea.Cmd")
		}
		if updatedValue.Kind() == reflect.Ptr {
			updatedValue = updatedValue.Elem()
		}
		ptrValue.Elem().Set(updatedValue)
		results = append(results, cmd)
	}
	return results
}
