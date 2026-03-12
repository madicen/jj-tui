package bubblepicker

import (
	"fmt"
	"math"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

// Focus indicates which part of the picker is focused for keyboard input.
type Focus int

const (
	FocusHueBar Focus = iota
	FocusGrid
)

// Lightness range for the S/L grid (top to bottom). Kept below 100 so the top row isn't pure white.
const gridLightTop, gridLightBottom = 98.0, 2.0

// ColorChosenMsg is sent when the user confirms the color (Enter).
type ColorChosenMsg struct {
	Color string // Hex, e.g. "#rrggbb"
}

// ColorCanceledMsg is sent when the user cancels (Esc).
type ColorCanceledMsg struct{}

// Zone IDs used when zone manager is set (for zone-based mouse interaction).
const (
	ZoneHueBar = "picker-hue"
	ZoneGrid   = "picker-grid"
)

// Model is the color picker state. It implements tea.Model.
type Model struct {
	HSL   HSL
	Focus Focus

	// Layout (from WindowSizeMsg)
	width  int
	height int

	// Grid size for S/L (saturation = x, lightness = y)
	gridCols int
	gridRows int

	// Optional: when set, mouse is handled via zones (hue bar → H, grid → S/L; release on grid accepts).
	zm *zone.Manager

	// Styles
	titleStyle   lipgloss.Style
	valueStyle   lipgloss.Style
	helpStyle    lipgloss.Style
	outlineStyle lipgloss.Style // outline for current selection
}

// New creates a color picker with an optional initial color (hex, e.g. "#ff0000").
// If initial is empty or invalid, starts at red (H=0, S=100, L=50).
func New(initial string) Model {
	m := Model{
		HSL:          HSL{H: 0, S: 100, L: 50},
		Focus:        FocusHueBar,
		gridCols:     24,
		gridRows:     12,
		titleStyle:   lipgloss.NewStyle().Bold(true),
		valueStyle:   lipgloss.NewStyle().Padding(0, 1),
		helpStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		outlineStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Bold(true),
	}
	if initial != "" {
		if hsl, err := HexToHSL(initial); err == nil {
			m.HSL = hsl.Clamp()
		}
	}
	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Value returns the current color as hex (e.g. "#rrggbb").
func (m Model) Value() string {
	return m.HSL.Clamp().ToHex()
}

// SetZoneManager sets the zone manager for zone-based mouse interaction. When set,
// moving the mouse over the hue bar sets H and over the grid sets S/L; left-button release on the grid accepts the color.
// The host must run zone.Scan() on the view that contains the picker so zones are registered.
func (m *Model) SetZoneManager(zm *zone.Manager) {
	m.zm = zm
}

// ViewSize returns the display size of the picker view (width, height in cells),
// including the built-in double border and padding.
func (m Model) ViewSize() (width, height int) {
	cols := m.gridCols
	if cols <= 0 {
		cols = 24
	}
	rows := m.gridRows
	if rows <= 0 {
		rows = 12
	}
	innerW, innerH := cols+2, 10+rows
	// Frame: DoubleBorder (1 each side) + Padding(0,1) (1 each side) = +2 width, +2 height
	return innerW + 2 + 2, innerH + 2
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		// Keep the picker small; bar and grid share the same width and are centered.
		const maxGridCols = 24
		const maxGridRows = 10
		contentW := max(msg.Width-2, 16)
		m.width = contentW
		m.gridCols = contentW
		if m.gridCols > maxGridCols {
			m.gridCols = maxGridCols
		}
		availH := msg.Height - 8
		if availH > maxGridRows {
			m.gridRows = maxGridRows
		} else if availH > 4 {
			m.gridRows = availH
		} else {
			m.gridRows = 4
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			return m, func() tea.Msg { return ColorChosenMsg{Color: m.Value()} }
		case "esc":
			return m, func() tea.Msg { return ColorCanceledMsg{} }
		case "tab":
			if m.Focus == FocusHueBar {
				m.Focus = FocusGrid
			} else {
				m.Focus = FocusHueBar
			}
			return m, nil
		case "left", "h":
			if m.Focus == FocusHueBar {
				m.HSL.H = math.Mod(m.HSL.H-8+360, 360)
			} else {
				m.HSL.S = math.Max(0, m.HSL.S-4)
			}
			m.HSL = m.HSL.Clamp()
			return m, nil
		case "right", "l":
			if m.Focus == FocusHueBar {
				m.HSL.H = math.Mod(m.HSL.H+8, 360)
			} else {
				m.HSL.S = math.Min(100, m.HSL.S+4)
			}
			m.HSL = m.HSL.Clamp()
			return m, nil
		case "up", "k":
			if m.Focus == FocusGrid {
				m.HSL.L = math.Min(100, m.HSL.L+4)
			}
			m.HSL = m.HSL.Clamp()
			return m, nil
		case "down", "j":
			if m.Focus == FocusGrid {
				m.HSL.L = math.Max(0, m.HSL.L-4)
			}
			m.HSL = m.HSL.Clamp()
			return m, nil
		}
		return m, nil

	case tea.MouseMsg:
		action := msg.Action
		// Zone-based: motion (hover) updates selection; release on grid accepts.
		if m.zm == nil {
			return m, nil
		}
		isClick := action == tea.MouseActionPress || action == tea.MouseActionRelease
		isMotion := action == tea.MouseActionMotion
		if !isClick && !isMotion {
			return m, nil
		}
		// Hue bar: set H from relative X (hover or click); zone has border so content X is 1..cols
		if z := m.zm.Get(ZoneHueBar); z != nil && z.InBounds(msg) {
			relX, _ := z.Pos(msg)
			w := m.gridCols
			if w <= 0 {
				w = 24
			}
			contentCol := relX - 1
			if contentCol >= 0 && contentCol < w {
				m.HSL.H = math.Mod((float64(contentCol)+0.5)/float64(w)*360, 360)
				if m.HSL.H < 0 {
					m.HSL.H += 360
				}
				m.HSL = m.HSL.Clamp()
			}
			return m, nil
		}
		// Grid: set S/L from relative position (hover or click); left-button release on grid accepts and closes
		if z := m.zm.Get(ZoneGrid); z != nil && z.InBounds(msg) {
			relX, relY := z.Pos(msg)
			w := m.gridCols
			if w <= 0 {
				w = 24
			}
			rows := m.gridRows
			if rows <= 0 {
				rows = 12
			}
			contentCol := relX - 1
			contentRow := relY - 1
			if contentCol >= 0 && contentCol < w && contentRow >= 0 && contentRow < rows {
				sx := (float64(contentCol) + 0.5) / float64(w) * 100
				m.HSL.S = math.Max(0, math.Min(100, sx))
				sy := (float64(contentRow) + 0.5) / float64(rows)
				m.HSL.L = gridLightTop - sy*(gridLightTop-gridLightBottom)
				m.HSL = m.HSL.Clamp()
			}
			// Left-button release on grid = accept color and close
			if isClick && action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
				return m, func() tea.Msg { return ColorChosenMsg{Color: m.Value()} }
			}
			return m, nil
		}
		return m, nil
	}

	return m, nil
}

// View renders the picker.
func (m Model) View() string {
	cols := m.gridCols
	if cols <= 0 {
		cols = 24
	}
	rows := m.gridRows
	if rows <= 0 {
		rows = 12
	}
	// Focus outline: zero padding/margin. Visible when focused, dim when not.
	focusBorderFocused := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderLeft(true).BorderRight(true).BorderTop(true).BorderBottom(true).
		BorderForeground(lipgloss.Color("250")).
		PaddingTop(0).PaddingBottom(0).PaddingLeft(0).PaddingRight(0).
		MarginTop(0).MarginBottom(0)
	focusBorderUnfocused := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderLeft(true).BorderRight(true).BorderTop(true).BorderBottom(true).
		BorderForeground(lipgloss.Color("241")).
		PaddingTop(0).PaddingBottom(0).PaddingLeft(0).PaddingRight(0).
		MarginTop(0).MarginBottom(0)

	// Hue bar: one row of colored blocks (same width as grid)
	hueBar := ""
	for i := 0; i < cols; i++ {
		h := math.Mod(float64(i)/float64(cols)*360, 360)
		r, g, b := HSLToRGB(h, 100, 50)
		hexStr := rgbToHexString(r, g, b)
		style := lipgloss.NewStyle().Background(lipgloss.Color(hexStr))
		hueBar += style.Render(" ")
	}
	hueBar = lipgloss.NewStyle().Width(cols).Render(hueBar)

	// Hue bar: arrow marker directly under bar (inside border so less gap); ▲ for visibility
	hueIdx := int(m.HSL.H / 360 * float64(cols))
	if hueIdx >= cols {
		hueIdx = cols - 1
	}
	hueMarkerLine := ""
	for i := 0; i < cols; i++ {
		if i == hueIdx {
			hueMarkerLine += m.outlineStyle.Render("▲")
		} else {
			hueMarkerLine += " "
		}
	}
	hueMarkerLine = lipgloss.NewStyle().Width(cols).Render(hueMarkerLine)

	// S/L grid: current cell (sCol, sRow) shows circle
	sCol := int(m.HSL.S / 100 * float64(cols))
	if sCol >= cols {
		sCol = cols - 1
	}
	sRow := int((gridLightTop - m.HSL.L) / (gridLightTop - gridLightBottom) * float64(rows))
	if sRow >= rows {
		sRow = rows - 1
	}
	if sRow < 0 {
		sRow = 0
	}
	grid := ""
	for row := 0; row < rows; row++ {
		ly := gridLightTop - float64(row)/float64(rows)*(gridLightTop-gridLightBottom)
		var line strings.Builder
		for col := 0; col < cols; col++ {
			sx := float64(col) / float64(cols) * 100
			r, g, b := HSLToRGB(m.HSL.H, sx, ly)
			hexStr := rgbToHexString(r, g, b)
			style := lipgloss.NewStyle().Background(lipgloss.Color(hexStr)).Width(1)
			isSelected := col == sCol && row == sRow
			if isSelected {
				line.WriteString(m.outlineStyle.Background(lipgloss.Color(hexStr)).Width(1).Render("●"))
			} else {
				line.WriteString(style.Render(" "))
			}
		}
		grid += line.String() + "\n"
	}

	// One column before and after content (cols) so each line is cols+2 wide; avoids wrapping.
	trunc := lipgloss.NewStyle().MaxWidth(cols)
	toCols := func(content string) string {
		content = trunc.Render(content)
		w := min(lipgloss.Width(content), cols)
		return content + strings.Repeat(" ", cols-w)
	}
	wrap := func(s string) string { return " " + toCols(s) + " " } // column before + content + column after
	title := wrap(m.titleStyle.Render("Pick a color"))
	value := wrap(m.valueStyle.Render(m.Value()))
	help1 := wrap(m.helpStyle.Render("↵ pick  ⎋ close"))
	help2 := wrap(m.helpStyle.Render("⇥ switch  ←↑↓→ move"))

	// Hue block: bar + marker inside full border (top, bar, marker, bottom)
	hueBorder := focusBorderUnfocused
	if m.Focus == FocusHueBar {
		hueBorder = focusBorderFocused
	}
	hueBlock := hueBorder.Width(cols).Render(hueBar + "\n" + hueMarkerLine)

	// Grid block: always same layout (border + rows + border); border visible when focused, dim when not
	gridTrimmed := strings.TrimSuffix(grid, "\n")
	gridBorder := focusBorderUnfocused
	if m.Focus == FocusGrid {
		gridBorder = focusBorderFocused
	}
	gridBlock := gridBorder.Width(cols).Render(gridTrimmed)

	// When zone manager is set, wrap clickable areas for zone-based mouse handling
	if m.zm != nil {
		hueBlock = m.zm.Mark(ZoneHueBar, hueBlock)
		gridBlock = m.zm.Mark(ZoneGrid, gridBlock)
	}

	inner := lipgloss.JoinVertical(lipgloss.Left,
		title,
		hueBlock,
		gridBlock,
		value,
		help1,
		help2,
	)
	// Outer frame: double border colored by current pick (innate to the picker)
	frame := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color(m.Value())).
		Padding(0, 1)
	return frame.Render(inner)
}

func rgbToHexString(r, g, b float64) string {
	rr := byte(math.Round(math.Max(0, math.Min(1, r)) * 255))
	gg := byte(math.Round(math.Max(0, math.Min(1, g)) * 255))
	bb := byte(math.Round(math.Max(0, math.Min(1, b)) * 255))
	return fmt.Sprintf("#%02x%02x%02x", rr, gg, bb)
}
