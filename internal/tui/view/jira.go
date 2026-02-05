package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Jira renders the tickets view with split header/list for scrolling
func (r *Renderer) Jira(data JiraData) JiraResult {
	if !data.JiraService {
		noTickets := []string{
			TitleStyle.Render("Ticket Integration"),
			"",
			"No ticket service is connected. Configure either Jira or Codecks in Settings (,):",
			"",
			"Jira Setup:",
			"   export JIRA_URL=https://your-domain.atlassian.net",
			"   export JIRA_USER=your-email@example.com",
			"   export JIRA_TOKEN=your_api_token",
			"",
			"Codecks Setup:",
			"   export CODECKS_SUBDOMAIN=your-team",
			"   export CODECKS_TOKEN=your_token  # (from browser 'at' cookie)",
			"",
			"Press ',' to open Settings, or 'g' to return to the commit graph.",
		}
		content := strings.Join(noTickets, "\n")
		return JiraResult{FullContent: content}
	}

	if len(data.Tickets) == 0 {
		content := "No assigned tickets found.\n\nPress Ctrl+r to refresh."
		return JiraResult{FullContent: content}
	}

	// Build fixed header section
	var headerLines []string
	title := "Assigned Tickets"
	if data.ProviderName != "" {
		title = fmt.Sprintf("Assigned %s Tickets", data.ProviderName)
	}
	headerLines = append(headerLines, TitleStyle.Render(title))
	headerLines = append(headerLines, "")

	// Show selected ticket details in the fixed header
	if data.SelectedTicket >= 0 && data.SelectedTicket < len(data.Tickets) {
		ticket := data.Tickets[data.SelectedTicket]

		// Use DisplayKey if available, otherwise fall back to Key
		displayKey := ticket.DisplayKey
		if displayKey == "" {
			displayKey = ticket.Key
		}

		// Build details content
		var detailLines []string
		detailLines = append(detailLines, fmt.Sprintf("%s %s",
			lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(displayKey),
			ticket.Summary,
		))
		detailLines = append(detailLines, fmt.Sprintf("Type: %s  |  Priority: %s  |  Status: %s",
			ticket.Type, ticket.Priority, ticket.Status,
		))
		// Always show description line to prevent layout shift
		if ticket.Description != "" {
			// Truncate description if too long
			desc := ticket.Description
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

		// Action buttons in header
		createBranchBtn := r.Zone.Mark(ZoneJiraCreateBranch, ButtonStyle.Render("Create Branch (Enter)"))
		openBrowserBtn := r.Zone.Mark(ZoneJiraOpenBrowser, ButtonStyle.Render("Open in Browser (o)"))
		headerLines = append(headerLines, createBranchBtn+"  "+openBrowserBtn)
		headerLines = append(headerLines, "")
	}

	headerLines = append(headerLines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Select a ticket to create a branch:"))
	headerLines = append(headerLines, "")

	// Build scrollable list section
	var listLines []string
	for i, ticket := range data.Tickets {
		prefix := "  "
		style := CommitStyle
		if i == data.SelectedTicket {
			prefix = "► "
			style = CommitSelectedStyle
		}

		// Status indicator with color (supports both Jira and Codecks statuses)
		var statusStyle lipgloss.Style
		switch strings.ToLower(ticket.Status) {
		case "to do", "open", "backlog", "not started":
			statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4"))
		case "in progress", "in review", "started":
			statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C"))
		case "done", "closed", "resolved":
			statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B"))
		case "blocked":
			statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))
		default:
			statusStyle = lipgloss.NewStyle().Foreground(ColorMuted)
		}

		// Use DisplayKey if available, otherwise fall back to Key
		displayKey := ticket.DisplayKey
		if displayKey == "" {
			displayKey = ticket.Key
		}
		ticketLine := fmt.Sprintf("%s%s %s %s",
			prefix,
			lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(displayKey),
			statusStyle.Render("["+ticket.Status+"]"),
			ticket.Summary,
		)

		listLines = append(listLines, r.Zone.Mark(ZoneJiraTicket(i), style.Render(ticketLine)))
	}

	fixedHeader := strings.Join(headerLines, "\n")
	scrollableList := strings.Join(listLines, "\n")
	fullContent := fixedHeader + "\n" + scrollableList

	return JiraResult{
		FixedHeader:    fixedHeader,
		ScrollableList: scrollableList,
		FullContent:    fullContent,
	}
}

