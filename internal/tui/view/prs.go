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

	// Show selected PR details
	if data.SelectedPR >= 0 && data.SelectedPR < len(data.Repository.PRs) {
		pr := data.Repository.PRs[data.SelectedPR]
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Details:"))
		lines = append(lines, fmt.Sprintf("  Base: %s ← Head: %s", pr.BaseBranch, pr.HeadBranch))
		lines = append(lines, fmt.Sprintf("  URL: %s", pr.URL))
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
		lines = append(lines, "Please set GITHUB_TOKEN environment variable.")
		lines = append(lines, "")
		lines = append(lines, "Press Esc to go back.")
		return strings.Join(lines, "\n")
	}

	// Find the current bookmark name for the selected commit
	var branchName string
	var commitSummary string
	if data.Repository != nil && data.SelectedCommit >= 0 && data.SelectedCommit < len(data.Repository.Graph.Commits) {
		commit := data.Repository.Graph.Commits[data.SelectedCommit]
		commitSummary = commit.Summary
		if len(commit.Branches) > 0 {
			branchName = commit.Branches[0]
		}
	}

	if branchName == "" {
		lines = append(lines, "No bookmark found on the selected commit.")
		lines = append(lines, "Create a bookmark first using jj bookmark create.")
		lines = append(lines, "")
		lines = append(lines, "Tip: Use the Jira tab (i) to create a branch from a ticket.")
		lines = append(lines, "")
		lines = append(lines, "Press Esc to go back.")
		return strings.Join(lines, "\n")
	}

	lines = append(lines, fmt.Sprintf("Branch: %s", lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(branchName)))

	// Use branch name as default PR title
	prTitle := branchName
	if commitSummary != "" {
		prTitle = commitSummary
	}
	lines = append(lines, fmt.Sprintf("Title: %s", prTitle))
	lines = append(lines, fmt.Sprintf("Base: %s", "main"))
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("(PR creation form coming soon)"))
	lines = append(lines, "")
	lines = append(lines, "Press Esc to go back.")

	return strings.Join(lines, "\n")
}

