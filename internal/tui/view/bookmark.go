package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Bookmark renders the bookmark creation view
func (r *Renderer) Bookmark(data BookmarkData) string {
	var lines []string
	lines = append(lines, TitleStyle.Render("Create Bookmark"))
	lines = append(lines, "")

	// Show commit info
	if data.Repository != nil && data.CommitIndex >= 0 && data.CommitIndex < len(data.Repository.Graph.Commits) {
		commit := data.Repository.Graph.Commits[data.CommitIndex]
		lines = append(lines, fmt.Sprintf("Commit: %s",
			lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(commit.ShortID)))
		lines = append(lines, fmt.Sprintf("Summary: %s", commit.Summary))
		lines = append(lines, "")
	}

	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Enter a name for the bookmark. Use letters, numbers, -, _, or /"))
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Press Enter to create, Esc to cancel"))
	lines = append(lines, "")

	// Bookmark name field
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("Bookmark Name:"))
	lines = append(lines, r.Zone.Mark(ZoneBookmarkName, "  "+data.NameInput))
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Tip: Use a descriptive name like 'feature/my-feature' or 'JIRA-123'"))
	lines = append(lines, "")
	lines = append(lines, "")

	// Action buttons
	submitButton := r.Zone.Mark(ZoneBookmarkSubmit, ButtonStyle.Render("Create (Enter)"))
	cancelButton := r.Zone.Mark(ZoneBookmarkCancel, ButtonStyle.Render("Cancel (Esc)"))
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, submitButton, " ", cancelButton))

	return strings.Join(lines, "\n")
}

