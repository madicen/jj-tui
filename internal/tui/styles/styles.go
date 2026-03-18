package styles

import "github.com/charmbracelet/lipgloss"

// Colors (updated by SetTheme when config or theme picker changes)
var (
	ColorPrimary   = lipgloss.Color("#7E00AF")
	ColorSecondary = lipgloss.Color("#50FA7B")
	ColorMuted     = lipgloss.Color("#6272A4")
)

// SetTheme updates the global theme colors and rebuilds styles that use them.
// Pass hex strings (e.g. "#7E00AF"). Empty strings are ignored (keep current).
func SetTheme(primary, secondary, muted string) {
	if primary != "" {
		ColorPrimary = lipgloss.Color(primary)
	}
	if secondary != "" {
		ColorSecondary = lipgloss.Color(secondary)
	}
	if muted != "" {
		ColorMuted = lipgloss.Color(muted)
	}
	rebuildThemeStyles()
}

func rebuildThemeStyles() {
	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary)
	CommitIDStyle = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true)
	HelpKeyStyle = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true)
	GraphStyle = lipgloss.NewStyle().Foreground(ColorMuted)
	TabActiveStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F9FAFB")).
		Background(ColorPrimary).
		Padding(0, 2)
	StatusBarStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("#1F2937")).
		Foreground(ColorMuted).
		Padding(0, 1)
}

// Styles (TitleStyle, CommitIDStyle, HelpKeyStyle, GraphStyle, TabActiveStyle, StatusBarStyle rebuilt in rebuildThemeStyles)
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

	ButtonSecondaryStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F8F8F2")).
				Background(lipgloss.Color("#6272A4")).
				Padding(0, 1).
				MarginRight(1)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8F8F2"))

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

	// Header and layout (main model view)
	HeaderStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1F2937")).
			Foreground(lipgloss.Color("#F9FAFB")).
			Padding(0, 1)

	TabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Padding(0, 2)

	TabActiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F9FAFB")).
			Background(ColorPrimary).
			Padding(0, 2)

	ContentStyle = lipgloss.NewStyle().
			Padding(1, 2)

	StatusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1F2937")).
			Foreground(ColorMuted).
			Padding(0, 1)
)

func init() {
	rebuildThemeStyles()
}

// GetStatusStyle returns the style and character for a file status
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
