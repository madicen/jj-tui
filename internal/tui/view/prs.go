package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// PullRequests renders the PR list view with split header/list for scrolling
func (r *Renderer) PullRequests(data PRData) PRResult {
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
		content := strings.Join(noGitHub, "\n")
		return PRResult{FullContent: content}
	}

	if data.Repository == nil || len(data.Repository.PRs) == 0 {
		content := "No pull requests found.\n\nPress Ctrl+r to refresh."
		return PRResult{FullContent: content}
	}

	// Build fixed header section
	var headerLines []string
	headerLines = append(headerLines, TitleStyle.Render("Pull Requests"))
	headerLines = append(headerLines, "")

	// Show selected PR details in the fixed header
	if data.SelectedPR >= 0 && data.SelectedPR < len(data.Repository.PRs) {
		pr := data.Repository.PRs[data.SelectedPR]

		// Build details content
		var detailLines []string
		detailLines = append(detailLines, fmt.Sprintf("%s #%d: %s",
			lipgloss.NewStyle().Bold(true).Render("Selected:"),
			pr.Number,
			pr.Title,
		))
		detailLines = append(detailLines, lipgloss.NewStyle().Foreground(ColorMuted).Render(pr.URL))
		detailLines = append(detailLines, fmt.Sprintf("Base: %s ← Head: %s", pr.BaseBranch, pr.HeadBranch))

		// Always show description line to prevent layout shift
		if pr.Body != "" {
			// Truncate description if too long
			desc := pr.Body
			// Replace newlines with spaces for single-line display
			desc = strings.ReplaceAll(desc, "\n", " ")
			desc = strings.ReplaceAll(desc, "\r", "")
			if len(desc) > 150 {
				desc = desc[:150] + "..."
			}
			detailLines = append(detailLines, lipgloss.NewStyle().Foreground(ColorMuted).Render(desc))
		} else {
			detailLines = append(detailLines, lipgloss.NewStyle().Foreground(ColorMuted).Italic(true).Render("(No description)"))
		}

		detailsBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1).
			Render(strings.Join(detailLines, "\n"))
		headerLines = append(headerLines, detailsBox)
		headerLines = append(headerLines, "")

		// Add Open in Browser button
		openBrowserBtn := r.Zone.Mark(ZonePROpenBrowser, ButtonStyle.Render("Open in Browser (o)"))
		headerLines = append(headerLines, openBrowserBtn)
		headerLines = append(headerLines, "")
	}

	// Build scrollable list section
	var listLines []string
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

		listLines = append(listLines, r.Zone.Mark(ZonePR(i), style.Render(prLine)))
	}

	fixedHeader := strings.Join(headerLines, "\n")
	scrollableList := strings.Join(listLines, "\n")
	fullContent := fixedHeader + "\n" + scrollableList

	return PRResult{
		FixedHeader:    fixedHeader,
		ScrollableList: scrollableList,
		FullContent:    fullContent,
	}
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
	lines = append(lines, r.Zone.Mark(ZonePRTitle, data.TitleInput))
	lines = append(lines, "")

	// Body field
	bodyLabel := "Description:"
	bodyStyle := lipgloss.NewStyle()
	if data.FocusedField == 1 {
		bodyStyle = bodyStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, bodyStyle.Render(bodyLabel))
	lines = append(lines, r.Zone.Mark(ZonePRBody, data.BodyInput))
	lines = append(lines, "")
	lines = append(lines, "")

	// Action buttons
	submitButton := r.Zone.Mark(ZonePRSubmit, ButtonStyle.Render("Create PR (Ctrl+S)"))
	cancelButton := r.Zone.Mark(ZonePRCancel, ButtonStyle.Render("Cancel (Esc)"))
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, submitButton, " ", cancelButton))

	return strings.Join(lines, "\n")
}

