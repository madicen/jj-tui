package model

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	colorPrimary   = lipgloss.Color("#7C3AED") // Purple
	colorSecondary = lipgloss.Color("#10B981") // Green
	colorMuted     = lipgloss.Color("#6B7280") // Gray
	colorBg        = lipgloss.Color("#1F2937") // Dark gray
	colorBgLight   = lipgloss.Color("#374151") // Lighter gray
	colorText      = lipgloss.Color("#F9FAFB") // White
	colorTextMuted = lipgloss.Color("#9CA3AF") // Light gray
)

// Styles
var (
	// Header styles
	HeaderStyle = lipgloss.NewStyle().
			Background(colorBg).
			Foreground(colorText).
			Padding(0, 1)

	TitleStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	// Tab styles
	TabStyle = lipgloss.NewStyle().
			Foreground(colorTextMuted).
			Padding(0, 2)

	TabActiveStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorPrimary).
			Padding(0, 2)

	// Button styles
	ButtonStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorBgLight).
			Padding(0, 1).
			MarginRight(1)

	ButtonActiveStyle = lipgloss.NewStyle().
				Foreground(colorText).
				Background(colorPrimary).
				Padding(0, 1).
				MarginRight(1)

	// Status bar styles
	StatusBarStyle = lipgloss.NewStyle().
			Background(colorBg).
			Foreground(colorTextMuted).
			Padding(0, 1)

	// Content styles
	ContentStyle = lipgloss.NewStyle().
			Padding(1, 2)

	// Commit styles
	CommitStyle = lipgloss.NewStyle().
			Foreground(colorText)

	CommitSelectedStyle = lipgloss.NewStyle().
				Foreground(colorText).
				Background(colorBgLight)

	CommitIDStyle = lipgloss.NewStyle().
			Foreground(colorSecondary)

	// Help styles
	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(colorTextMuted)
)

