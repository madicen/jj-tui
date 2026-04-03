package model

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

const (
	modalInnerMax = 72
	modalInnerMin = 32
)

// ModalInnerWidth is the content width for form inputs inside centered modals (excluding border).
func ModalInnerWidth(termW int) int {
	if termW < 1 {
		return 56
	}
	// Leave room for rounded border, padding, and overlay margins.
	n := termW - 12
	if n > modalInnerMax {
		n = modalInnerMax
	}
	if n < modalInnerMin {
		n = min(modalInnerMin, max(24, termW-8))
	}
	return n
}

// FrameFormModal draws a rounded border around centered dialog content.
func FrameFormModal(inner string, termW int) string {
	if inner == "" {
		return ""
	}
	innerW := ModalInnerWidth(termW)
	// Outer width: inner + horizontal padding (4) + border (2).
	outerW := min(max(termW-6, 40), innerW+6)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorMuted).
		Padding(1, 2).
		Width(outerW).
		Render(inner)
}
