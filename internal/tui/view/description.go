package view

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal/tui/mouse"
)

// Description renders the description editing view
func (r *Renderer) Description(data DescriptionData) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#8BE9FD"))

	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6272A4"))

	// Find the commit being edited
	var commitInfo string
	if data.Repository != nil {
		for _, commit := range data.Repository.Graph.Commits {
			if commit.ChangeID == data.EditingCommitID {
				changeIDShort := commit.ChangeID
				if len(changeIDShort) > 8 {
					changeIDShort = changeIDShort[:8]
				}
				commitInfo = fmt.Sprintf("%s (%s)", commit.ShortID, changeIDShort)
				break
			}
		}
	}
	if commitInfo == "" {
		commitInfo = data.EditingCommitID
	}

	header := titleStyle.Render("Edit Commit Description")
	commitLine := subtitleStyle.Render(fmt.Sprintf("Commit: %s", commitInfo))

	// Clickable action buttons
	actionButtons := lipgloss.JoinHorizontal(
		lipgloss.Left,
		r.Mark(mouse.ZoneDescSave, ButtonStyle.Render("Save (Ctrl+S)")),
		r.Mark(mouse.ZoneDescCancel, ButtonStyle.Render("Cancel (Esc)")),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		commitLine,
		"",
		data.InputView,
		"",
		actionButtons,
	)
}
