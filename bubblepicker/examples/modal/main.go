// Modal example: picker appears as an overlay on top of the main view, centered on the
// color box you clicked or selected. Shows the full pattern (viewWithModal, mouse offset, etc.).
//
// Run from the bubblepicker directory:
//
//	go run ./examples/modal
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

func main() {
	p := tea.NewProgram(newApp(), tea.WithAltScreen(), tea.WithMouseAllMotion())
	final, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if app, ok := final.(*appModel); ok && app.quitting {
		fmt.Println("Colors:", app.colors["Primary"], app.colors["Secondary"])
	}
}

const (
	zoneColorPrefix = "color-"
)

type appModel struct {
	width  int
	height int

	zm *zone.Manager

	// Color slots: click or select + Enter to open picker
	colors        map[string]string
	labels        []string
	selected      int
	modalOpen     bool
	editingKey    string
	picker        bubblepicker.Model
	quitting      bool
	boxStyle      lipgloss.Style
	selectedStyle lipgloss.Style

	// Last modal dimensions and position when rendering; used for mouse offset so it matches draw.
	lastModalW         int
	lastOverlayHeight int
	lastOverlayLeft   int
	lastOverlayTop    int
}

func newApp() *appModel {
	return &appModel{
		zm: zone.New(),
		colors: map[string]string{
			"Primary":   "#7E00AF",
			"Secondary": "#50FA7B",
		},
		labels:   []string{"Primary", "Secondary"},
		selected: 0,
		boxStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("241")).
			Padding(0, 2).
			Height(3).
			Width(14),
		selectedStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 2).
			Height(3).
			Width(14),
	}
}

func (a *appModel) Init() tea.Cmd {
	return nil
}

func (a *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		if a.modalOpen {
			// Pass modal box size to picker so it stays small
			picker, cmd := a.picker.Update(tea.WindowSizeMsg{Width: 42, Height: 22})
			a.picker = picker.(bubblepicker.Model)
			return a, cmd
		}
		return a, nil

	case tea.KeyMsg:
		if a.modalOpen {
			updated, cmd := a.picker.Update(msg)
			a.picker = updated.(bubblepicker.Model)
			return a, cmd
		}
		switch msg.String() {
		case "q", "ctrl+c":
			a.quitting = true
			return a, tea.Quit
		case "tab":
			a.selected = (a.selected + 1) % len(a.labels)
			return a, nil
		case "shift+tab":
			a.selected--
			if a.selected < 0 {
				a.selected = len(a.labels) - 1
			}
			return a, nil
		case "enter":
			a.editingKey = a.labels[a.selected]
			a.picker = bubblepicker.New(a.colors[a.editingKey])
			a.picker.SetZoneManager(a.zm)
			a.modalOpen = true
			picker, cmd := a.picker.Update(tea.WindowSizeMsg{Width: 42, Height: 22})
			a.picker = picker.(bubblepicker.Model)
			return a, cmd
		}
		return a, nil

	case tea.MouseMsg:
		if a.modalOpen {
			// Picker uses zones; forward raw screen coords so zone InBounds/Pos work
			updated, cmd := a.picker.Update(msg)
			a.picker = updated.(bubblepicker.Model)
			return a, cmd
		}
		// Use bubblezone to detect which color box was clicked (no manual bounds).
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			for i := range a.labels {
				z := a.zm.Get(colorBoxZoneID(i))
				if z != nil && z.InBounds(msg) {
					a.selected = i
					a.editingKey = a.labels[i]
					a.picker = bubblepicker.New(a.colors[a.editingKey])
					a.picker.SetZoneManager(a.zm)
					a.modalOpen = true
					picker, cmd := a.picker.Update(tea.WindowSizeMsg{Width: 42, Height: 22})
					a.picker = picker.(bubblepicker.Model)
					return a, cmd
				}
			}
		}
		return a, nil

	case bubblepicker.ColorChosenMsg:
		if a.modalOpen && a.editingKey != "" {
			a.colors[a.editingKey] = msg.Color
		}
		a.modalOpen = false
		a.editingKey = ""
		return a, nil

	case bubblepicker.ColorCanceledMsg:
		a.modalOpen = false
		a.editingKey = ""
		return a, nil
	}

	if a.modalOpen {
		updated, cmd := a.picker.Update(msg)
		a.picker = updated.(bubblepicker.Model)
		return a, cmd
	}
	return a, nil
}

func colorBoxZoneID(i int) string { return zoneColorPrefix + strconv.Itoa(i) }

func (a *appModel) View() string {
	if a.width <= 0 {
		a.width = 60
	}
	if a.height <= 0 {
		a.height = 20
	}

	// Scan so all zones (main view + picker when modal open) are registered
	if a.modalOpen {
		return a.zm.Scan(a.viewWithModal())
	}
	return a.zm.Scan(a.viewMain())
}

func (a *appModel) viewMain() string {
	title := lipgloss.NewStyle().Bold(true).Render("Theme colors — click a box or Tab + Enter to change")
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Tab: select  Enter: open picker  q: quit")

	box1 := a.colorBox(a.labels[0], a.colors[a.labels[0]], a.selected == 0)
	box2 := a.colorBox(a.labels[1], a.colors[a.labels[1]], a.selected == 1)
	row := lipgloss.JoinHorizontal(lipgloss.Top, a.zm.Mark(colorBoxZoneID(0), box1), "  ", a.zm.Mark(colorBoxZoneID(1), box2))

	lines := []string{title, "", "", row, "", help}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (a *appModel) colorBox(label, hex string, selected bool) string {
	style := a.boxStyle
	if selected {
		style = a.selectedStyle.BorderForeground(lipgloss.Color(hex))
	}
	inner := lipgloss.NewStyle().
		Background(lipgloss.Color(hex)).
		Width(10).
		Height(1).
		Render(" ")
	content := lipgloss.JoinVertical(lipgloss.Left, inner, label, hex)
	return style.Render(content)
}

// colorBoxBounds returns (x0, x1, y0, y1) for box index for mouse hit-test.
// Bounds are 1-based inclusive to match xterm-style mouse coordinates (column/row 1 = top-left).
// Uses the same style as colorBox to get exact frame size.
func (a *appModel) colorBoxBounds(boxIndex int) (x0, x1, y0, y1 int) {
	st := a.boxStyle
	wFrame, hFrame := st.GetFrameSize() // horizontal and vertical frame (border + padding)
	// Content is 14 wide, 3 tall; total box size includes frame
	boxW := 14 + wFrame
	boxH := 3 + hFrame
	const gap = 2
	// View: line 1=title, 2=blank, 3=blank, 4..4+boxH-1=box row (1-based)
	y0 = 4
	y1 = 4 + boxH - 1
	x0 = 1 + boxIndex*(boxW+gap)
	x1 = x0 + boxW - 1
	return x0, x1, y0, y1
}

func (a *appModel) viewWithModal() string {
	mainView := a.viewMain()
	// Picker draws its own double border (color = current value); no extra wrapper needed.
	modalContent := a.picker.View()
	modalLines := strings.Split(modalContent, "\n")
	overlayHeight := len(modalLines)
	a.lastOverlayHeight = overlayHeight

	// Use actual rendered width so positioning and mouse offset match; keeps symmetric gaps.
	modalW := 0
	for _, l := range modalLines {
		if w := lipgloss.Width(l); w > modalW {
			modalW = w
		}
	}
	a.lastModalW = modalW

	// Position modal centered on the color box we're editing (pops right where the box was)
	x0, x1, y0, y1 := a.colorBoxBounds(a.selected)
	centerX := (x0 + x1) / 2
	centerY := (y0 + y1) / 2
	// Bounds are 1-based; convert to 0-based for row/col indexing
	leftPad := centerX - 1 - modalW/2
	topPad := centerY - 1 - overlayHeight/2
	leftPad = max(leftPad, 0)
	if leftPad+modalW > a.width {
		leftPad = max(a.width-modalW, 0)
	}
	topPad = max(topPad, 0)
	if topPad+overlayHeight > a.height {
		topPad = max(a.height-overlayHeight, 0)
	}
	a.lastOverlayLeft = leftPad
	a.lastOverlayTop = topPad

	// Overlay: replace only the modal rectangle; main view stays visible everywhere else.
	return bubblepicker.OverlayView(mainView, modalContent, a.width, a.height, topPad, leftPad)
}
