package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Bookmark renders the bookmark creation view
func (r *Renderer) Bookmark(data BookmarkData) string {
	var lines []string
	lines = append(lines, TitleStyle.Render("Create or Move Bookmark"))
	lines = append(lines, "")

	// Show commit info
	if data.Repository != nil && data.CommitIndex >= 0 && data.CommitIndex < len(data.Repository.Graph.Commits) {
		commit := data.Repository.Graph.Commits[data.CommitIndex]
		commitBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1).
			Render(fmt.Sprintf("Target: %s\n%s",
				lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(commit.ShortID),
				commit.Summary,
			))
		lines = append(lines, commitBox)
		lines = append(lines, "")
	}

	// Show existing bookmarks section if there are any
	if len(data.ExistingBookmarks) > 0 {
		lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Move Existing Bookmark:"))
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Click or use j/k to select, Enter to move"))
		lines = append(lines, "")

		for i, bookmark := range data.ExistingBookmarks {
			prefix := "  "
			style := CommitStyle
			if i == data.SelectedBookmark {
				prefix = "► "
				style = CommitSelectedStyle
			}
			bookmarkLine := fmt.Sprintf("%s%s", prefix, bookmark)
			lines = append(lines, r.Zone.Mark(ZoneExistingBookmark(i), style.Render(bookmarkLine)))
		}
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("─────────────────────────────────"))
		lines = append(lines, "")
	}

	// New bookmark section
	newStyle := lipgloss.NewStyle().Bold(true)
	if data.SelectedBookmark == -1 {
		newStyle = newStyle.Foreground(ColorPrimary)
	}
	lines = append(lines, newStyle.Render("Create New Bookmark:"))
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Type a name and press Enter"))
	lines = append(lines, "")

	// Bookmark name field
	inputStyle := lipgloss.NewStyle()
	if data.SelectedBookmark == -1 {
		inputStyle = inputStyle.Foreground(ColorPrimary)
	}
	lines = append(lines, inputStyle.Render("Name:"))
	lines = append(lines, r.Zone.Mark(ZoneBookmarkName, "  "+data.NameInput))
	lines = append(lines, "")

	// Action buttons
	var submitLabel string
	if data.SelectedBookmark >= 0 && data.SelectedBookmark < len(data.ExistingBookmarks) {
		submitLabel = fmt.Sprintf("Move '%s' (Enter)", data.ExistingBookmarks[data.SelectedBookmark])
	} else {
		submitLabel = "Create (Enter)"
	}
	submitButton := r.Zone.Mark(ZoneBookmarkSubmit, ButtonStyle.Render(submitLabel))
	cancelButton := r.Zone.Mark(ZoneBookmarkCancel, ButtonStyle.Render("Cancel (Esc)"))
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, submitButton, " ", cancelButton))

	return strings.Join(lines, "\n")
}

