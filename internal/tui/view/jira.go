package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Jira renders the Jira tickets view with split header/list for scrolling
func (r *Renderer) Jira(data JiraData) JiraResult {
	if !data.JiraService {
		noJira := []string{
			TitleStyle.Render("Jira Integration"),
			"",
			"Jira is not connected. To enable Jira functionality:",
			"",
			"1. Set your Jira credentials:",
			"   export JIRA_URL=https://your-domain.atlassian.net",
			"   export JIRA_USER=your-email@example.com",
			"   export JIRA_TOKEN=your_api_token",
			"",
			"2. Get your API token from:",
			"   https://id.atlassian.com/manage-profile/security/api-tokens",
			"",
			"Press 'g' to return to the commit graph.",
		}
		content := strings.Join(noJira, "\n")
		return JiraResult{FullContent: content}
	}

	if len(data.Tickets) == 0 {
		content := "No assigned tickets found.\n\nPress 'r' to refresh."
		return JiraResult{FullContent: content}
	}

	// Build fixed header section
	var headerLines []string
	headerLines = append(headerLines, TitleStyle.Render("Assigned Jira Tickets"))
	headerLines = append(headerLines, "")

	// Show selected ticket details in the fixed header
	if data.SelectedTicket >= 0 && data.SelectedTicket < len(data.Tickets) {
		ticket := data.Tickets[data.SelectedTicket]

		// Build details content
		var detailLines []string
		detailLines = append(detailLines, fmt.Sprintf("%s %s",
			lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(ticket.Key),
			ticket.Summary,
		))
		detailLines = append(detailLines, fmt.Sprintf("Type: %s  |  Priority: %s  |  Status: %s",
			ticket.Type, ticket.Priority, ticket.Status,
		))
		if ticket.Description != "" {
			// Truncate description if too long
			desc := ticket.Description
			if len(desc) > 150 {
				desc = desc[:150] + "..."
			}
			detailLines = append(detailLines, lipgloss.NewStyle().Foreground(ColorMuted).Render(desc))
		}

		detailsBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1).
			Render(strings.Join(detailLines, "\n"))
		headerLines = append(headerLines, detailsBox)
		headerLines = append(headerLines, "")

		// Action button in header
		headerLines = append(headerLines, r.Zone.Mark(ZoneJiraCreateBranch, ButtonStyle.Render("Create Branch (Enter)")))
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

		// Status indicator with color
		var statusStyle lipgloss.Style
		switch strings.ToLower(ticket.Status) {
		case "to do", "open", "backlog":
			statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4"))
		case "in progress", "in review":
			statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C"))
		case "done", "closed", "resolved":
			statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B"))
		default:
			statusStyle = lipgloss.NewStyle().Foreground(ColorMuted)
		}

		ticketLine := fmt.Sprintf("%s%s %s %s",
			prefix,
			lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(ticket.Key),
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

