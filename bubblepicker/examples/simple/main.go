// Simple example: the picker is the entire app. No modal, no overlay—just run and pick a color.
// Closes when you confirm (Enter), cancel (Esc), or press q / Ctrl+C.
//
// Run from the bubblepicker directory:
//
//	go run ./examples/simple
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/bubblepicker"
)

func main() {
	zm := zone.New()
	picker := bubblepicker.New("#7E00AF")
	picker.SetZoneManager(zm)
	app := &simpleModel{picker: picker, zm: zm}
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseAllMotion())
	model, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if m, ok := model.(*simpleModel); ok && m.chosenColor != "" {
		fmt.Println("Picked:", m.chosenColor)
	}
}

type simpleModel struct {
	picker       bubblepicker.Model
	zm           *zone.Manager
	chosenColor  string
}

func (m *simpleModel) Init() tea.Cmd { return nil }

func (m *simpleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case bubblepicker.ColorChosenMsg:
		m.chosenColor = msg.Color
		return m, tea.Quit
	case bubblepicker.ColorCanceledMsg:
		return m, tea.Quit
	}
	updated, cmd := m.picker.Update(msg)
	m.picker = updated.(bubblepicker.Model)
	return m, cmd
}

func (m *simpleModel) View() string {
	return m.zm.Scan(m.picker.View())
}
