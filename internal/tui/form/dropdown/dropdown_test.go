package dropdown

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	bubbledropdown "github.com/madicen/bubble-dropdown"
)

func TestNewAndAccessors(t *testing.T) {
	f := New(bubbledropdown.WithOptions([]string{"a", "b", "c"}))
	if f.Dropdown() == nil {
		t.Fatal("Dropdown() should be non-nil after New")
	}
	if f.Open() {
		t.Fatal("dropdown should start closed")
	}
	f.SetSelectedIndex(2) // must not panic
	f.SetZoneManager(nil) // must not panic
}

func TestNilFieldSafe(t *testing.T) {
	var f *Field
	if f.Dropdown() != nil {
		t.Fatal("nil Field Dropdown() should be nil")
	}
	if f.Open() {
		t.Fatal("nil Field Open() should be false")
	}
	f.SetSelectedIndex(1) // must not panic
	f.SetZoneManager(nil) // must not panic
	if f.Update(nil, nil) != nil {
		t.Fatal("nil Field Update() should return nil")
	}
}

func TestUpdateDispatchesOnSelectOnlyWhenWasOpen(t *testing.T) {
	f := New(bubbledropdown.WithOptions([]string{"x", "y", "z"}))

	called := -1
	onSelect := func(i int) { called = i }

	// ItemChosenMsg while the dropdown is CLOSED must NOT dispatch (guards against
	// spurious selection on open), matching the sub-tabs' wasOpen guard.
	f.Update(bubbledropdown.ItemChosenMsg{Index: 1}, onSelect)
	if called != -1 {
		t.Fatalf("onSelect should not fire while closed, got index %d", called)
	}
}

func TestUpdateDispatchesWhenOpen(t *testing.T) {
	f := New(bubbledropdown.WithOptions([]string{"x", "y", "z"}))
	// Open the panel: focus, then Enter opens it (see bubble-dropdown Update).
	f.dd.SetFocused(true)
	f.dd, _ = f.dd.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !f.Open() {
		t.Fatal("precondition: dropdown should be open")
	}

	called := -1
	f.Update(bubbledropdown.ItemChosenMsg{Index: 2}, func(i int) { called = i })
	if called != 2 {
		t.Fatalf("onSelect should fire with index 2 when open, got %d", called)
	}
}
