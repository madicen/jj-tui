package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// PullRequests renders the PR list view
func (r *Renderer) PullRequests(data PRData) string {
	if !data.GithubService {
		noGitHub := []string{
			TitleStyle.Render("GitHub Integration"),
			"",
			"GitHub is not connected. To enable PR functionality:",
			"",
			"1. Set your GitHub token:",
			"   export GITHUB_TOKEN=your_token",
			"",
			"2. Make sure your repository has a GitHub remote",
			"",
			"Press 'g' to return to the commit graph.",
		}
		return strings.Join(noGitHub, "\n")
	}

	if data.Repository == nil || len(data.Repository.PRs) == 0 {
		return "No pull requests found.\n\nPress 'r' to refresh."
	}

	var lines []string
	lines = append(lines, TitleStyle.Render("Pull Requests"))
	lines = append(lines, "")

	// Show selected PR details at the TOP (fixed section)
	if data.SelectedPR >= 0 && data.SelectedPR < len(data.Repository.PRs) {
		pr := data.Repository.PRs[data.SelectedPR]
		detailsBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1).
			Render(fmt.Sprintf(
				"%s #%d: %s\n%s\nBase: %s ← Head: %s",
				lipgloss.NewStyle().Bold(true).Render("Selected:"),
				pr.Number,
				pr.Title,
				lipgloss.NewStyle().Foreground(ColorMuted).Render(pr.URL),
				pr.BaseBranch,
				pr.HeadBranch,
			))
		lines = append(lines, detailsBox)
		lines = append(lines, "")
	}

	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Press Enter/o to open in browser"))
	lines = append(lines, "")

	// PR list (scrollable section)
	for i, pr := range data.Repository.PRs {
		prefix := "  "
		style := CommitStyle
		if i == data.SelectedPR {
			prefix = "► "
			style = CommitSelectedStyle
		}

		// State indicator with color
		var stateIndicator string
		switch pr.State {
		case "open":
			stateIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("#2ea44f")).Render("●")
		case "closed":
			stateIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("#cb2431")).Render("●")
		case "merged":
			stateIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("#6f42c1")).Render("●")
		default:
			stateIndicator = "○"
		}

		prLine := fmt.Sprintf("%s%s #%d %s",
			prefix,
			stateIndicator,
			pr.Number,
			pr.Title,
		)

		lines = append(lines, r.Zone.Mark(ZonePR(i), style.Render(prLine)))
	}

	return strings.Join(lines, "\n")
}

// CreatePR renders the create PR view
func (r *Renderer) CreatePR(data CreatePRData) string {
	var lines []string
	lines = append(lines, TitleStyle.Render("Create Pull Request"))
	lines = append(lines, "")

	if !data.GithubService {
		lines = append(lines, "GitHub is not connected.")
		lines = append(lines, "Please configure GitHub in Settings (press ',').")
		lines = append(lines, "")
		lines = append(lines, "Press Esc to go back.")
		return strings.Join(lines, "\n")
	}

	if data.HeadBranch == "" {
		lines = append(lines, "No bookmark found on the selected commit.")
		lines = append(lines, "Create a bookmark first using jj bookmark create.")
		lines = append(lines, "")
		lines = append(lines, "Tip: Use the Jira tab (i) to create a branch from a ticket.")
		lines = append(lines, "")
		lines = append(lines, "Press Esc to go back.")
		return strings.Join(lines, "\n")
	}

	// Show branch info
	lines = append(lines, fmt.Sprintf("Head: %s → Base: %s",
		lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(data.HeadBranch),
		lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(data.BaseBranch),
	))
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Tab to switch fields, Ctrl+S to submit, Esc to cancel"))
	lines = append(lines, "")

	// Title field
	titleLabel := "Title:"
	titleStyle := lipgloss.NewStyle()
	if data.FocusedField == 0 {
		titleStyle = titleStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, titleStyle.Render(titleLabel))
	lines = append(lines, r.Zone.Mark(ZonePRTitle, "  "+data.TitleInput))
	lines = append(lines, "")

	// Body field
	bodyLabel := "Description:"
	bodyStyle := lipgloss.NewStyle()
	if data.FocusedField == 1 {
		bodyStyle = bodyStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, bodyStyle.Render(bodyLabel))
	lines = append(lines, r.Zone.Mark(ZonePRBody, "  "+data.BodyInput))
	lines = append(lines, "")
	lines = append(lines, "")

	// Action buttons
	submitButton := r.Zone.Mark(ZonePRSubmit, ButtonStyle.Render("Create PR (Ctrl+S)"))
	cancelButton := r.Zone.Mark(ZonePRCancel, ButtonStyle.Render("Cancel (Esc)"))
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, submitButton, " ", cancelButton))

	return strings.Join(lines, "\n")
}

