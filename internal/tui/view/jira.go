package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Jira renders the Jira tickets view
func (r *Renderer) Jira(data JiraData) string {
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
		return strings.Join(noJira, "\n")
	}

	if len(data.Tickets) == 0 {
		return "No assigned tickets found.\n\nPress 'r' to refresh."
	}

	var lines []string
	lines = append(lines, TitleStyle.Render("Assigned Jira Tickets"))
	lines = append(lines, "")

	// Show selected ticket details at the TOP (fixed section)
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
		lines = append(lines, detailsBox)
		lines = append(lines, "")

		// Action button right after details
		lines = append(lines, r.Zone.Mark(ZoneJiraCreateBranch, ButtonStyle.Render("Create Branch (Enter)")))
		lines = append(lines, "")
	}

	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Select a ticket to create a branch:"))
	lines = append(lines, "")

	// Ticket list (scrollable section)
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

		lines = append(lines, r.Zone.Mark(ZoneJiraTicket(i), style.Render(ticketLine)))
	}

	return strings.Join(lines, "\n")
}

