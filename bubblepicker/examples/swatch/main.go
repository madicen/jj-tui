// Swatch example: 2×2 grid of color swatches that open the full modal picker on click.
// Uses SwatchPicker so the client only embeds the component, sets bounds, and
// forwards messages—no overlay math or mouse offset logic.
//
// Run from the bubblepicker directory:
//
//	go run ./examples/swatch
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/bubblepicker"
)

const (
	labelLen = 10 // "Color 1: " etc.
	gap      = 2  // spaces between column 1 and "Color 2: "
)

func main() {
	p := tea.NewProgram(newApp(), tea.WithAltScreen(), tea.WithMouseAllMotion())
	final, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if app, ok := final.(*appModel); ok && app.quitting {
		fmt.Println("Colors:", app.swatches[0].Color(), app.swatches[1].Color(), app.swatches[2].Color(), app.swatches[3].Color())
	}
}

type appModel struct {
	width       int
	height      int
	swatches    [4]*bubblepicker.SwatchPicker
	zm *zone.Manager
	quitting    bool
}

func newApp() *appModel {
	a := &appModel{
		zm: zone.New(),
		swatches: [4]*bubblepicker.SwatchPicker{
			bubblepicker.NewSwatchPicker("#7E00AF", ""),
			bubblepicker.NewSwatchPicker("#00AF7E", ""),
			bubblepicker.NewSwatchPicker("#AF7E00", ""),
			bubblepicker.NewSwatchPicker("#AF007E", ""),
		},
	}
	for _, s := range a.swatches {
		s.SetZoneManager(a.zm)
	}
	return a
}

func (a *appModel) Init() tea.Cmd {
	return nil
}

func (a *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		if a.width <= 0 {
			a.width = 60
		}
		if a.height <= 0 {
			a.height = 20
		}
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			a.quitting = true
			return a, tea.Quit
		}
	}

	var cmd tea.Cmd
	// When a modal is open, only that swatch receives messages so the picker gets correct
	// mouse/key events and other swatches don't open or react.
	openIdx := -1
	for i := range a.swatches {
		if a.swatches[i].Open() {
			openIdx = i
			break
		}
	}
	if openIdx >= 0 {
		if _, isWindowSize := msg.(tea.WindowSizeMsg); !isWindowSize {
			a.swatches[openIdx], cmd = a.swatches[openIdx].Update(msg)
			return a, cmd
		}
	}
	// Use bubblezone to route main-view mouse clicks to the correct swatch (no manual hit test).
	if m, ok := msg.(tea.MouseMsg); ok && m.Action == tea.MouseActionPress && m.Button == tea.MouseButtonLeft {
		for i := range a.swatches {
			z := a.zm.Get(swatchZoneID(i))
			if z != nil && z.InBounds(m) {
				a.swatches[i], cmd = a.swatches[i].Update(msg)
				return a, cmd
			}
		}
		// No zone matched; consume the press so we don't forward to all swatches below
		return a, nil
	}
	for i := range a.swatches {
		var c tea.Cmd
		a.swatches[i], c = a.swatches[i].Update(msg)
		if c != nil {
			cmd = c
		}
	}
	return a, cmd
}

func swatchZoneID(i int) string { return "swatch-" + strconv.Itoa(i) }

func (a *appModel) View() string {
	if a.width <= 0 {
		a.width = 60
	}
	if a.height <= 0 {
		a.height = 20
	}
	mainView := a.buildMainView()
	for _, s := range a.swatches {
		mainView = s.ViewWithOverlay(mainView, a.width, a.height)
	}
	return a.zm.Scan(mainView)
}

func (a *appModel) buildMainView() string {
	title := lipgloss.NewStyle().Bold(true).Render("Click a color to change it")
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("q: quit")

	sw, sh := a.swatches[0].Size()
	// 2x2: Color 1 & 2 on row 2, Color 3 & 4 on row 3
	col1 := labelLen
	col2 := col1 + sw + gap + labelLen

	a.swatches[0].SetBounds(2, col1, sw, sh)
	a.swatches[1].SetBounds(2, col2, sw, sh)
	a.swatches[2].SetBounds(3, col1, sw, sh)
	a.swatches[3].SetBounds(3, col2, sw, sh)

	// Zone covers full "Color N: ■▼" so the whole cell is clickable and zone bounds align correctly
	row1 := a.zm.Mark(swatchZoneID(0), "Color 1: "+a.swatches[0].SwatchView()) + "  " + a.zm.Mark(swatchZoneID(1), "Color 2: "+a.swatches[1].SwatchView())
	row2 := a.zm.Mark(swatchZoneID(2), "Color 3: "+a.swatches[2].SwatchView()) + "  " + a.zm.Mark(swatchZoneID(3), "Color 4: "+a.swatches[3].SwatchView())

	lines := []string{title, "", row1, row2, "", help}
	return strings.Join(lines, "\n")
}
