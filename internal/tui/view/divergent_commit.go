package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// DivergentCommit renders the divergent commit resolution view
func (r *Renderer) DivergentCommit(data DivergentCommitData) string {
	var lines []string

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF79C6"))
	lines = append(lines, titleStyle.Render("⑂ Divergent Commit: "+data.ChangeID))
	lines = append(lines, "")

	// Explanation
	explanationStyle := lipgloss.NewStyle().Foreground(ColorMuted)
	lines = append(lines, explanationStyle.Render("This change ID has multiple versions. Select which one to keep."))
	lines = append(lines, explanationStyle.Render("The other version(s) will be abandoned."))
	lines = append(lines, "")

	// Separator
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render(strings.Repeat("─", 60)))
	lines = append(lines, "")

	// List all versions
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Versions:"))
	lines = append(lines, "")

	// Option styles
	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary)
	unselectedStyle := lipgloss.NewStyle().Foreground(ColorMuted)
	commitIDStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))

	for i, commitID := range data.CommitIDs {
		prefix := "  "
		style := unselectedStyle
		if i == data.SelectedIdx {
			prefix = "► "
			style = selectedStyle
		}

		summary := "(no description)"
		if i < len(data.Summaries) {
			summary = data.Summaries[i]
		}

		// Truncate summary if too long
		if len(summary) > 50 {
			summary = summary[:47] + "..."
		}

		line := fmt.Sprintf("%s%s  %s",
			prefix,
			commitIDStyle.Render(commitID),
			style.Render(summary),
		)
		lines = append(lines, r.Mark(ZoneDivergentCommit(i), line))
	}

	lines = append(lines, "")

	// Separator
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render(strings.Repeat("─", 60)))
	lines = append(lines, "")

	// Action buttons
	confirmBtn := r.Mark(ZoneDivergentConfirm, ButtonStyle.Render("Keep Selected (Enter)"))
	cancelBtn := r.Mark(ZoneDivergentCancel, ButtonSecondaryStyle.Render("Cancel (Esc)"))
	lines = append(lines, confirmBtn+"  "+cancelBtn)

	// Help text
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Use j/k or click to select, Enter to confirm"))

	return strings.Join(lines, "\n")
}

