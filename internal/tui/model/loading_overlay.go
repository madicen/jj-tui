package model

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	overlay "github.com/madicen/bubble-overlay"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

func newBusySpinner() spinner.Model {
	return spinner.New(
		spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(styles.ColorSecondary)),
	)
}

func (m *Model) startBusySpinnerCmd() tea.Cmd {
	return func() tea.Msg {
		return m.busySpinner.Tick()
	}
}

// applyLoadingOverlay composites a centered busy box over the full main layout when Loading.
func (m *Model) applyLoadingOverlay(fullView string) string {
	if !m.appState.Loading || m.width <= 0 || m.height <= 0 {
		return fullView
	}
	msg := m.appState.StatusMessage
	if msg == "" {
		msg = "Loading…"
	}
	msg = strings.ReplaceAll(msg, "\n", " ")
	line := lipgloss.JoinHorizontal(lipgloss.Center, m.busySpinner.View(), " ", msg)
	maxOuter := m.width - 4
	if maxOuter < 1 {
		maxOuter = 1
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(styles.ColorMuted).
		Padding(0, 0).
		MaxWidth(maxOuter).
		Background(lipgloss.Color("#1F2937")).
		Render(line)
	boxLines := strings.Split(box, "\n")
	modalH := len(boxLines)
	modalW := 0
	for _, l := range boxLines {
		if w := lipgloss.Width(l); w > modalW {
			modalW = w
		}
	}
	top := max((m.height-modalH)/2, 0)
	left := max((m.width-modalW)/2, 0)
	return overlay.OverlayView(fullView, box, m.width, m.height, top, left)
}
