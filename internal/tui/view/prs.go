package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tui/mouse"
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

		// Always show status line to prevent layout shift
		var checkPart, reviewPart string

		// CI status
		switch pr.CheckStatus {
		case internal.CheckStatusSuccess:
			checkPart = lipgloss.NewStyle().Foreground(lipgloss.Color("#2ea44f")).Render("✓ Checks passed")
		case internal.CheckStatusFailure:
			checkPart = lipgloss.NewStyle().Foreground(lipgloss.Color("#cb2431")).Render("✗ Checks failed")
		case internal.CheckStatusPending:
			checkPart = lipgloss.NewStyle().Foreground(lipgloss.Color("#dbab09")).Render("○ Checks pending")
		default:
			checkPart = lipgloss.NewStyle().Foreground(lipgloss.Color("#6a737d")).Render("· No checks")
		}

		// Review status (using text symbols for consistent terminal rendering)
		switch pr.ReviewStatus {
		case internal.ReviewStatusApproved:
			reviewPart = lipgloss.NewStyle().Foreground(lipgloss.Color("#2ea44f")).Render("✓ Approved")
		case internal.ReviewStatusChangesRequested:
			reviewPart = lipgloss.NewStyle().Foreground(lipgloss.Color("#cb2431")).Render("✗ Changes requested")
		case internal.ReviewStatusPending:
			reviewPart = lipgloss.NewStyle().Foreground(lipgloss.Color("#dbab09")).Render("○ Review pending")
		default:
			reviewPart = lipgloss.NewStyle().Foreground(lipgloss.Color("#6a737d")).Render("· No reviews")
		}

		detailLines = append(detailLines, checkPart+"  │  "+reviewPart)

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

		// Actions section with separators (like Graph and Tickets tabs)
		separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
		separatorWidth := data.Width - 4
		if separatorWidth < 20 {
			separatorWidth = 80
		}
		separator := separatorStyle.Render(strings.Repeat("─", separatorWidth))

		headerLines = append(headerLines, separator)
		headerLines = append(headerLines, "Actions:")

		// Build action buttons
		var actionButtons []string
		openBrowserBtn := r.Mark(mouse.ZonePROpenBrowser, ButtonStyle.Render("Open in Browser (o)"))
		actionButtons = append(actionButtons, openBrowserBtn)

		// Only show merge/close for open PRs
		if pr.State == "open" {
			mergeBtn := r.Mark(mouse.ZonePRMerge, ButtonStyle.Render("Merge (M)"))
			closeBtn := r.Mark(mouse.ZonePRClose, ButtonStyle.Render("Close (X)"))
			actionButtons = append(actionButtons, mergeBtn, closeBtn)
		}

		headerLines = append(headerLines, strings.Join(actionButtons, " "))
		headerLines = append(headerLines, separator)
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

		// CI Check status indicator
		var checkIndicator string
		switch pr.CheckStatus {
		case internal.CheckStatusSuccess:
			checkIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("#2ea44f")).Render("✓")
		case internal.CheckStatusFailure:
			checkIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("#cb2431")).Render("✗")
		case internal.CheckStatusPending:
			checkIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("#dbab09")).Render("○")
		default:
			checkIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("#6a737d")).Render("·")
		}

		// Review status indicator
		var reviewIndicator string
		switch pr.ReviewStatus {
		case internal.ReviewStatusApproved:
			reviewIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("#2ea44f")).Render("👍")
		case internal.ReviewStatusChangesRequested:
			reviewIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("#cb2431")).Render("📝")
		case internal.ReviewStatusPending:
			reviewIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("#dbab09")).Render("⏳")
		default:
			reviewIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("#6a737d")).Render("·")
		}

		prLine := fmt.Sprintf("%s%s %s%s #%d %s",
			prefix,
			stateIndicator,
			checkIndicator,
			reviewIndicator,
			pr.Number,
			pr.Title,
		)

		listLines = append(listLines, r.Mark(mouse.ZonePR(i), style.Render(prLine)))
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
	lines = append(lines, r.Mark(mouse.ZonePRTitle, data.TitleInput))
	lines = append(lines, "")

	// Body field
	bodyLabel := "Description:"
	bodyStyle := lipgloss.NewStyle()
	if data.FocusedField == 1 {
		bodyStyle = bodyStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, bodyStyle.Render(bodyLabel))
	lines = append(lines, r.Mark(mouse.ZonePRBody, data.BodyInput))
	lines = append(lines, "")
	lines = append(lines, "")

	// Action buttons
	submitButton := r.Mark(mouse.ZonePRSubmit, ButtonStyle.Render("Create PR (Ctrl+S)"))
	cancelButton := r.Mark(mouse.ZonePRCancel, ButtonStyle.Render("Cancel (Esc)"))
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, submitButton, " ", cancelButton))

	return strings.Join(lines, "\n")
}
