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
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Press Enter to create a branch from the selected ticket"))
	lines = append(lines, "")

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

	// Show selected ticket details
	if data.SelectedTicket >= 0 && data.SelectedTicket < len(data.Tickets) {
		ticket := data.Tickets[data.SelectedTicket]
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Details:"))
		lines = append(lines, fmt.Sprintf("  Type: %s", ticket.Type))
		lines = append(lines, fmt.Sprintf("  Priority: %s", ticket.Priority))
		if ticket.Description != "" {
			// Truncate description if too long
			desc := ticket.Description
			if len(desc) > 200 {
				desc = desc[:200] + "..."
			}
			lines = append(lines, fmt.Sprintf("  Description: %s", desc))
		}

		// Show action button
		lines = append(lines, "")
		lines = append(lines, "Actions:")
		lines = append(lines, r.Zone.Mark(ZoneJiraCreateBranch, ButtonStyle.Render("Create Branch (Enter)")))
	}

	return strings.Join(lines, "\n")
}

