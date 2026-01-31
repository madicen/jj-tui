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
)

