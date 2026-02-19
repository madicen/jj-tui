package help

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

var (
	helpTabStyle      = lipgloss.NewStyle().Padding(0, 2).Foreground(lipgloss.Color("#888888"))
	helpTabActiveStyle = lipgloss.NewStyle().Padding(0, 2).Foreground(styles.ColorPrimary).Bold(true).Underline(true)
)
