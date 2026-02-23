package prs

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

// mark wraps zone.Mark; if zoneManager is nil returns content unchanged
func mark(z *zone.Manager, id, content string) string {
	if z == nil {
		return content
	}
	return z.Mark(id, content)
}

// renderPRs renders the PR list view (list-only scroll; details fixed)
func (m *Model) renderPRs() string {
	if !m.githubService {
		noGitHub := []string{
			styles.TitleStyle.Render("GitHub Integration"),
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

	if m.repository == nil || len(m.repository.PRs) == 0 {
		return "No pull requests found.\n\nPress Ctrl+r to refresh."
	}

	var headerLines []string

	if m.selectedPR >= 0 && m.selectedPR < len(m.repository.PRs) {
		pr := m.repository.PRs[m.selectedPR]

		var detailLines []string
		detailLines = append(detailLines, fmt.Sprintf("%s #%d: %s",
			lipgloss.NewStyle().Bold(true).Render("Selected:"),
			pr.Number,
			pr.Title,
		))
		detailLines = append(detailLines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(pr.URL))
		detailLines = append(detailLines, fmt.Sprintf("Base: %s ← Head: %s", pr.BaseBranch, pr.HeadBranch))

		var checkPart, reviewPart string
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

		if pr.Body != "" {
			desc := strings.ReplaceAll(pr.Body, "\n", " ")
			desc = strings.ReplaceAll(desc, "\r", "")
			if len(desc) > 150 {
				desc = desc[:150] + "..."
			}
			detailLines = append(detailLines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(desc))
		} else {
			detailLines = append(detailLines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Italic(true).Render("(No description)"))
		}

		detailsBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.ColorPrimary).
			Padding(0, 1).
			Render(strings.Join(detailLines, "\n"))
		headerLines = append(headerLines, detailsBox)

		separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
		separatorWidth := m.width - 4
		if separatorWidth < 20 {
			separatorWidth = 80
		}
		separator := separatorStyle.Render(strings.Repeat("─", separatorWidth))
		headerLines = append(headerLines, separator)
		headerLines = append(headerLines, "Actions:")

		var actionButtons []string
		actionButtons = append(actionButtons, mark(m.zoneManager, mouse.ZonePROpenBrowser, styles.ButtonStyle.Render("Open in Browser (o)")))
		if pr.State == "open" {
			actionButtons = append(actionButtons,
				mark(m.zoneManager, mouse.ZonePRMerge, styles.ButtonStyle.Render("Merge (M)")),
				mark(m.zoneManager, mouse.ZonePRClose, styles.ButtonStyle.Render("Close (X)")),
			)
		}
		headerLines = append(headerLines, strings.Join(actionButtons, " "))
		headerLines = append(headerLines, separator)
	}

	var listLines []string
	for i, pr := range m.repository.PRs {
		prefix := "  "
		style := styles.CommitStyle
		if i == m.selectedPR {
			prefix = "► "
			style = styles.CommitSelectedStyle
		}
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
			prefix, stateIndicator, checkIndicator, reviewIndicator, pr.Number, pr.Title)
		listLines = append(listLines, mark(m.zoneManager, mouse.ZonePR(i), style.Render(prLine)))
	}

	fixedHeader := strings.Join(headerLines, "\n")
	headerLineCount := strings.Count(fixedHeader, "\n") + 1
	listHeight := m.height - headerLineCount
	if listHeight <= 0 {
		listHeight = 0
	}
	totalListLines := len(listLines)
	maxListOffset := 0
	if totalListLines > listHeight {
		maxListOffset = totalListLines - listHeight
	}
	// Clamp listYOffset
	if m.listYOffset > maxListOffset {
		m.listYOffset = maxListOffset
	}
	if m.listYOffset < 0 {
		m.listYOffset = 0
	}
	// Keep selection in view only when selection changed via key/click (so mouse scroll can move selection off screen)
	if m.scrollToSelectedPR {
		m.scrollToSelectedPR = false
		if m.selectedPR >= 0 && m.selectedPR < totalListLines {
			if m.selectedPR < m.listYOffset {
				m.listYOffset = m.selectedPR
			} else if m.selectedPR >= m.listYOffset+listHeight {
				m.listYOffset = m.selectedPR - listHeight + 1
			}
		}
	}
	start := m.listYOffset
	end := start + listHeight
	if end > totalListLines {
		end = totalListLines
	}
	var visibleList string
	if start < end {
		visibleList = strings.Join(listLines[start:end], "\n")
	} else {
		visibleList = ""
	}
	out := fixedHeader + "\n" + visibleList
	// Pad to height lines so main view doesn't scroll
	outLines := strings.Split(out, "\n")
	for len(outLines) < m.height {
		outLines = append(outLines, "")
	}
	if len(outLines) > m.height {
		outLines = outLines[:m.height]
	}
	return strings.Join(outLines, "\n")
}
