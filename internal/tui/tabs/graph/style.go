package graph

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal/tui/styles"
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
			Foreground(styles.ColorPrimary).
			Bold(true)

	// Style for graph lines (muted color)
	GraphStyle = lipgloss.NewStyle().Foreground(styles.ColorMuted)
)
