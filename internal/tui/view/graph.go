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

	// Special styles for rebase mode
	rebaseSourceStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#5555AA")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true)
	rebaseDestStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#55AA55")).
		Foreground(lipgloss.Color("#FFFFFF"))

	// Show rebase mode header if active
	if data.InRebaseMode {
		rebaseHeader := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFAA00")).
			Bold(true).
			Render("🔀 REBASE MODE - Select destination commit (Esc to cancel)")
		lines = append(lines, rebaseHeader)
		lines = append(lines, "")
	}

	for i, commit := range data.Repository.Graph.Commits {
		// Build commit line
		style := CommitStyle

		// In rebase mode, use special styling
		if data.InRebaseMode {
			if i == data.RebaseSourceCommit {
				// Source commit being rebased - highlighted differently
				style = rebaseSourceStyle
			} else if i == data.SelectedCommit {
				// Potential destination - green highlight
				style = rebaseDestStyle
			}
		} else if i == data.SelectedCommit {
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
		if data.InRebaseMode {
			if i == data.RebaseSourceCommit {
				selectionPrefix = "⚡ " // Source being rebased
			} else if i == data.SelectedCommit {
				selectionPrefix = "→ " // Target destination
			}
		} else if i == data.SelectedCommit {
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

	// Don't show action buttons in rebase mode - user is selecting destination
	if data.InRebaseMode {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Press Enter or click to select destination, Esc to cancel"))
		return strings.Join(lines, "\n")
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
				r.Zone.Mark(ZoneActionRebase, ButtonStyle.Render("Rebase (b)")),
				r.Zone.Mark(ZoneActionAbandon, ButtonStyle.Render("Abandon (a)")),
				r.Zone.Mark(ZoneActionBookmark, ButtonStyle.Render("Bookmark (m)")),
			)

			// Check if this commit can push to a PR (either has the bookmark or is a descendant)
			prBranch := ""
			if data.CommitPRBranch != nil {
				prBranch = data.CommitPRBranch[data.SelectedCommit]
			}

			if prBranch != "" {
				// This commit (or an ancestor) has an open PR - show Push button
				buttonLabel := "Push (u)"
				if len(commit.Branches) == 0 {
					// This is a descendant without the bookmark - indicate we'll move the bookmark
					buttonLabel = "Push to PR (u)"
				}
				actionButtons = append(actionButtons,
					r.Zone.Mark(ZoneActionPush, ButtonStyle.Render(buttonLabel)),
				)
			} else if len(commit.Branches) > 0 {
				// Has a bookmark but no open PR - show Create PR button
				actionButtons = append(actionButtons,
					r.Zone.Mark(ZoneActionCreatePR, ButtonStyle.Render("Create PR (c)")),
				)
			}
			lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, actionButtons...))
		}
	} else {
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, actionButtons...))
	}

	return strings.Join(lines, "\n")
}

