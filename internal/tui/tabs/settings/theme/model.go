package theme

import (
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/bubble-color-picker"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

// Model holds three SwatchPickers for Primary, Secondary, and Muted theme colors.
type Model struct {
	swatches [3]*bubblepicker.SwatchPicker
}

const (
	idxPrimary = iota
	idxSecondary
	idxMuted
)

// Default theme colors (hex).
const (
	DefaultPrimary   = "#7E00AF"
	DefaultSecondary = "#50FA7B"
	DefaultMuted     = "#6272A4"
)

// NewModel creates a theme model with default colors.
func NewModel() Model {
	return Model{
		swatches: [3]*bubblepicker.SwatchPicker{
			bubblepicker.NewSwatchPicker("#7E00AF", "Primary"),
			bubblepicker.NewSwatchPicker("#50FA7B", "Secondary"),
			bubblepicker.NewSwatchPicker("#6272A4", "Muted"),
		},
	}
}

// NewModelFromConfig creates a theme model from config (uses defaults for empty).
func NewModelFromConfig(cfg *config.Config) Model {
	m := NewModel()
	if cfg != nil {
		m.swatches[idxPrimary].SetColor(cfg.GetThemePrimary())
		m.swatches[idxSecondary].SetColor(cfg.GetThemeSecondary())
		m.swatches[idxMuted].SetColor(cfg.GetThemeMuted())
	}
	return m
}

// SetZoneManager sets the zone manager on all swatches (call from settings when zone manager is set).
func (m *Model) SetZoneManager(zm *zone.Manager) {
	for _, s := range m.swatches {
		s.SetZoneManager(zm)
	}
}

// Primary returns the current primary color (hex).
func (m *Model) Primary() string   { return m.swatches[idxPrimary].Color() }
func (m *Model) Secondary() string { return m.swatches[idxSecondary].Color() }
func (m *Model) Muted() string     { return m.swatches[idxMuted].Color() }

// Swatch returns the SwatchPicker at index (0=Primary, 1=Secondary, 2=Muted).
func (m *Model) Swatch(i int) *bubblepicker.SwatchPicker {
	if i < 0 || i > 2 {
		return nil
	}
	return m.swatches[i]
}

// SetSwatchToDefault resets the swatch at index to its default color and updates live styles.
func (m *Model) SetSwatchToDefault(index int) {
	switch index {
	case idxPrimary:
		m.swatches[idxPrimary].SetColor(DefaultPrimary)
	case idxSecondary:
		m.swatches[idxSecondary].SetColor(DefaultSecondary)
	case idxMuted:
		m.swatches[idxMuted].SetColor(DefaultMuted)
	default:
		return
	}
	styles.SetTheme(m.Primary(), m.Secondary(), m.Muted())
}

// AnyOpen returns true if any swatch's picker modal is open.
func (m *Model) AnyOpen() bool {
	for _, s := range m.swatches {
		if s.Open() {
			return true
		}
	}
	return false
}

// UpdateSwatch forwards a message to the swatch at index and applies theme on ColorChosenMsg.
// Call this when the user clicked the swatch zone or when a picker is open and events go to this tab.
func (m *Model) UpdateSwatch(index int, msg tea.Msg) (Model, tea.Cmd) {
	if index < 0 || index > 2 {
		return *m, nil
	}
	updated, cmd := m.swatches[index].Update(msg)
	m.swatches[index] = updated
	switch msg.(type) {
	case bubblepicker.ColorChosenMsg:
		// Picker closed with a new color; update live styles for preview
		styles.SetTheme(m.Primary(), m.Secondary(), m.Muted())
	case bubblepicker.ColorCanceledMsg:
		// No change
	}
	return *m, cmd
}

// Update forwards msg to the open swatch if any; otherwise no-op (zone click is handled by settings).
func (m *Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	for i := range m.swatches {
		if m.swatches[i].Open() {
			return m.UpdateSwatch(i, msg)
		}
	}
	return *m, nil
}

// ViewWithOverlay applies each swatch's overlay to mainView (call when Theme tab is active and AnyOpen()).
func (m *Model) ViewWithOverlay(mainView string, viewWidth, viewHeight int) string {
	for _, s := range m.swatches {
		mainView = s.ViewWithOverlay(mainView, viewWidth, viewHeight)
	}
	return mainView
}

// SetBounds sets the position of the swatch at index (0-based row, col, width, height in cells).
func (m *Model) SetBounds(index int, row, col, w, h int) {
	if index >= 0 && index <= 2 {
		m.swatches[index].SetBounds(row, col, w, h)
	}
}
