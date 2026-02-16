package graph

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal/tui/view"
)

var (
	// Special styles for rebase mode
	RebaseSourceStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#5555AA")).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)
	RebaseDestStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#55AA55")).
			Foreground(lipgloss.Color("#FFFFFF"))
	RebaseHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFAA00")).
				Bold(true)

	CommitStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8F8F2"))

	CommitSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F8F8F2")).
				Background(lipgloss.Color("#44475A"))

	CommitIDStyle = lipgloss.NewStyle().
			Foreground(view.ColorPrimary).
			Bold(true)

	// Style for graph lines (muted color)
	GraphStyle = lipgloss.NewStyle().Foreground(view.ColorMuted)
)

// getStatusStyle returns the style and character for a file status
func GetStatusStyle(status string) (lipgloss.Style, string) {
	switch status {
	case "M":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C")), "M" // Orange for modified
	case "A":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")), "A" // Green for added
	case "D":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")), "D" // Red for deleted
	case "R":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD")), "R" // Cyan for renamed
	default:
		return lipgloss.NewStyle().Foreground(view.ColorMuted), status
	}
}
