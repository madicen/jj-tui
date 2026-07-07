// Package dropdown wraps madicen/bubble-dropdown with the plumbing that every
// settings sub-tab (github, tickets, ai, advanced) previously copy-pasted:
//   - live accent-color syncing to the theme primary on each render,
//   - nil-safe accessors, and
//   - the "wasOpen + ItemChosenMsg index dispatch" selection pattern.
//
// Because the settings sub-models are value types mutated through pointer
// receivers, the selection callback is supplied per Update call (a callback
// stored at construction time could not mutate the live model), so sub-tabs
// keep only a one-line UpdateDropdown that provides their onSelect closure.
package dropdown

import (
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	bubbledropdown "github.com/madicen/bubble-dropdown"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

// Field is a settings dropdown with shared accent-sync and selection dispatch.
type Field struct {
	dd *bubbledropdown.Dropdown
}

// New builds a Field. The theme-primary accent color is always applied; any
// extra bubble-dropdown options (WithOptions, WithMaxVisible, …) are appended.
func New(opts ...bubbledropdown.Option) *Field {
	all := make([]bubbledropdown.Option, 0, len(opts)+1)
	all = append(all, bubbledropdown.WithAccentColor(string(styles.ColorPrimary)))
	all = append(all, opts...)
	return &Field{dd: bubbledropdown.New(all...)}
}

// Dropdown returns the underlying dropdown for rendering/overlay, first syncing
// the accent color to the live theme primary so the panel tracks theme changes.
// Returns nil when unset.
func (f *Field) Dropdown() *bubbledropdown.Dropdown {
	if f == nil || f.dd == nil {
		return nil
	}
	if accent := string(styles.ColorPrimary); f.dd.AccentColor() != accent {
		f.dd.SetAccentColor(accent)
	}
	return f.dd
}

// Open reports whether the dropdown panel is open.
func (f *Field) Open() bool {
	return f != nil && f.dd != nil && f.dd.Open()
}

// SetZoneManager wires the bubblezone manager into the dropdown.
func (f *Field) SetZoneManager(zm *zone.Manager) {
	if f != nil && f.dd != nil {
		f.dd.SetZoneManager(zm)
	}
}

// SetSelectedIndex selects an option by index.
func (f *Field) SetSelectedIndex(i int) {
	if f != nil && f.dd != nil {
		f.dd.SetSelectedIndex(i)
	}
}

// Update advances the dropdown and, when a selection was just committed (the
// dropdown was open and msg is an ItemChosenMsg), invokes onSelect with the
// chosen index. Returns the dropdown's command. Safe on a nil Field.
func (f *Field) Update(msg tea.Msg, onSelect func(index int)) tea.Cmd {
	if f == nil || f.dd == nil {
		return nil
	}
	wasOpen := f.dd.Open()
	dd, cmd := f.dd.Update(msg)
	f.dd = dd
	if chosen, ok := msg.(bubbledropdown.ItemChosenMsg); ok && wasOpen {
		if onSelect != nil {
			onSelect(chosen.Index)
		}
	}
	return cmd
}
