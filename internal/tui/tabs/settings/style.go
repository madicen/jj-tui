package settings

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

var (
	clearButtonStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#F85149")).Bold(true)
	settingsTabStyle  = lipgloss.NewStyle().Padding(0, 2).Foreground(styles.ColorMuted)
	settingsTabActive = lipgloss.NewStyle().Padding(0, 2).Bold(true).Foreground(styles.ColorPrimary).Underline(true)
	toggleOnStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Bold(true)
	toggleOffStyle    = lipgloss.NewStyle().Foreground(styles.ColorMuted)
)
