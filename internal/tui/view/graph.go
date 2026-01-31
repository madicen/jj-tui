package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Graph renders the commit graph view
func (r *Renderer) Graph(data GraphData) string {
	if data.Repository == nil || len(data.Repository.Graph.Commits) == 0 {
		return "No commits found. Press 'r' to refresh."
	}

	var lines []string

	// Style for graph lines (muted color)
	graphStyle := lipgloss.NewStyle().Foreground(ColorMuted)

	for i, commit := range data.Repository.Graph.Commits {
		// Build commit line
		style := CommitStyle
		if i == data.SelectedCommit {
			style = CommitSelectedStyle
		}

		// Use jj's graph prefix if available, otherwise fall back to simple display
		var graphPrefix string
		if commit.GraphPrefix != "" {
			// Use jj's native graph prefix
			graphPrefix = graphStyle.Render(commit.GraphPrefix)
		} else {
			// Fall back to simple prefix
			if commit.IsWorking {
				graphPrefix = graphStyle.Render("@  ")
			} else if commit.Immutable {
				graphPrefix = graphStyle.Render("◆  ")
			} else {
				graphPrefix = graphStyle.Render("○  ")
			}
		}

		// Selection indicator (prepended before graph)
		selectionPrefix := "  "
		if i == data.SelectedCommit {
			selectionPrefix = "► "
		}

		// Show conflict indicator
		conflictIndicator := ""
		if commit.Conflicts {
			conflictIndicator = " ⚠"
		}

		// Show branches/bookmarks
		branchStr := ""
		if len(commit.Branches) > 0 {
			branchStr = " " + lipgloss.NewStyle().Foreground(ColorSecondary).Render("["+strings.Join(commit.Branches, ", ")+"]")
		}

		// Format the commit line: selection + graph + commit_id + summary + branches + conflict
		commitLine := fmt.Sprintf("%s%s%s %s%s%s",
			selectionPrefix,
			graphPrefix,
			CommitIDStyle.Render(commit.ShortID),
			commit.Summary,
			branchStr,
			conflictIndicator,
		)

		// Wrap in zone for click detection
		lines = append(lines, r.Zone.Mark(ZoneCommit(i), style.Render(commitLine)))

		// Render graph connector lines after this commit (if any)
		for _, graphLine := range commit.GraphLines {
			// These are the lines between commits (like │, ├─╯, etc.)
			// Add spacing to align with commit lines
			paddedLine := "  " + graphStyle.Render(graphLine)
			lines = append(lines, paddedLine)
		}
	}

	// Add action buttons
	lines = append(lines, "")
	lines = append(lines, "Actions:")

	// Always show "New" action
	actionButtons := []string{
		r.Zone.Mark(ZoneActionNewCommit, ButtonStyle.Render("New (n)")),
	}

	// Add commit-specific actions if a commit is selected
	if data.SelectedCommit >= 0 && data.SelectedCommit < len(data.Repository.Graph.Commits) {
		commit := data.Repository.Graph.Commits[data.SelectedCommit]

		if commit.Immutable {
			lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, actionButtons...))
			lines = append(lines, "")
			lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("◆ Selected commit is immutable (pushed to remote)"))
		} else {
			actionButtons = append(actionButtons,
				r.Zone.Mark(ZoneActionCheckout, ButtonStyle.Render("Edit (e)")),
				r.Zone.Mark(ZoneActionDescribe, ButtonStyle.Render("Describe (d)")),
				r.Zone.Mark(ZoneActionSquash, ButtonStyle.Render("Squash (s)")),
				r.Zone.Mark(ZoneActionAbandon, ButtonStyle.Render("Abandon (a)")),
			)
			lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, actionButtons...))
		}
	} else {
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, actionButtons...))
	}

	return strings.Join(lines, "\n")
}

