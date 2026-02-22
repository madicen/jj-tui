package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal/tui/mouse"
)

// BookmarkConflict renders the bookmark conflict resolution view
func (r *Renderer) BookmarkConflict(data BookmarkConflictData) string {
	var lines []string

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF5555"))
	lines = append(lines, titleStyle.Render("⚠ Bookmark Conflict: "+data.BookmarkName))
	lines = append(lines, "")

	// Explanation
	explanationStyle := lipgloss.NewStyle().Foreground(ColorMuted)
	lines = append(lines, explanationStyle.Render("This bookmark has diverged - local and remote point to different commits."))
	lines = append(lines, "")

	// Box style for commit info
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1).
		Width(60)

	// Local commit info
	localHeader := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#50FA7B")).Render("Local Version")
	localCommitID := lipgloss.NewStyle().Foreground(ColorPrimary).Render(data.LocalCommitID)
	localContent := fmt.Sprintf("%s\n%s\n%s", localHeader, localCommitID, truncateSummary(data.LocalSummary, 55))
	lines = append(lines, boxStyle.Render(localContent))
	lines = append(lines, "")

	// Remote commit info
	remoteHeader := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8BE9FD")).Render("Remote Version (origin)")
	remoteCommitID := lipgloss.NewStyle().Foreground(ColorPrimary).Render(data.RemoteCommitID)
	remoteContent := fmt.Sprintf("%s\n%s\n%s", remoteHeader, remoteCommitID, truncateSummary(data.RemoteSummary, 55))
	lines = append(lines, boxStyle.Render(remoteContent))
	lines = append(lines, "")

	// Separator
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render(strings.Repeat("─", 60)))
	lines = append(lines, "")

	// Resolution options
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Choose Resolution:"))
	lines = append(lines, "")

	// Option styles
	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary)
	unselectedStyle := lipgloss.NewStyle().Foreground(ColorMuted)

	// Option 0: Keep Local (force push)
	keepLocalPrefix := "  "
	keepLocalStyle := unselectedStyle
	if data.SelectedOption == 0 {
		keepLocalPrefix = "► "
		keepLocalStyle = selectedStyle
	}
	keepLocalLine := fmt.Sprintf("%s%s", keepLocalPrefix, "Keep Local (force push to remote)")
	lines = append(lines, r.Mark(mouse.ZoneConflictKeepLocal, keepLocalStyle.Render(keepLocalLine)))
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Overwrites remote with your local version"))
	lines = append(lines, "")

	// Option 1: Reset to Remote
	resetRemotePrefix := "  "
	resetRemoteStyle := unselectedStyle
	if data.SelectedOption == 1 {
		resetRemotePrefix = "► "
		resetRemoteStyle = selectedStyle
	}
	resetRemoteLine := fmt.Sprintf("%s%s", resetRemotePrefix, "Reset to Remote (discard local)")
	lines = append(lines, r.Mark(mouse.ZoneConflictResetRemote, resetRemoteStyle.Render(resetRemoteLine)))
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Updates local bookmark to match remote"))
	lines = append(lines, "")

	// Action buttons
	lines = append(lines, "")
	confirmBtn := r.Mark(mouse.ZoneConflictConfirm, ButtonStyle.Render("Confirm (Enter)"))
	cancelBtn := r.Mark(mouse.ZoneConflictCancel, ButtonSecondaryStyle.Render("Cancel (Esc)"))
	lines = append(lines, confirmBtn+"  "+cancelBtn)

	// Help text
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Use j/k or click to select, Enter to confirm"))

	return strings.Join(lines, "\n")
}

// truncateSummary truncates a summary to fit within maxLen characters
func truncateSummary(summary string, maxLen int) string {
	if len(summary) <= maxLen {
		return summary
	}
	return summary[:maxLen-3] + "..."
}
