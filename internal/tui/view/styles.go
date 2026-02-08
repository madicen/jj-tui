package view

import "github.com/charmbracelet/lipgloss"

// Colors
var (
	ColorPrimary   = lipgloss.Color("#BD93F9")
	ColorSecondary = lipgloss.Color("#50FA7B")
	ColorMuted     = lipgloss.Color("#6272A4")
)

// Styles
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	CommitStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8F8F2"))

	CommitSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F8F8F2")).
				Background(lipgloss.Color("#44475A"))

	CommitIDStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	ButtonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8F8F2")).
			Background(lipgloss.Color("#44475A")).
			Padding(0, 1).
			MarginRight(1)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8F8F2"))

	// Style for graph lines (muted color)
	GraphStyle = lipgloss.NewStyle().Foreground(ColorMuted)

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
		return lipgloss.NewStyle().Foreground(ColorMuted), status
	}
}
