package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal/tui/mouse"
)

// Note: ZoneJiraTransition is a prefix - full zone ID is ZoneJiraTransition + index

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

		// Separator line style (same as graph tab)
		separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
		separatorWidth := data.Width - 4
		if separatorWidth < 20 {
			separatorWidth = 80 // fallback
		}
		separator := separatorStyle.Render(strings.Repeat("─", separatorWidth))

		// Actions section with separators (like Graph tab)
		headerLines = append(headerLines, separator)
		headerLines = append(headerLines, "Actions:")

		// Build action buttons
		var actionButtons []string

		// Primary actions
		actionButtons = append(actionButtons,
			r.Mark(mouse.ZoneJiraCreateBranch, ButtonStyle.Render("Create Branch (Enter)")),
			r.Mark(mouse.ZoneJiraOpenBrowser, ButtonStyle.Render("Open in Browser (o)")),
		)

		// Status change button - collapsed or expanded
		if len(data.AvailableTransitions) > 0 && !data.TransitionInProgress {
			if data.StatusChangeMode {
				// Expanded: show highlighted "Change Status" button and status options
				highlightedBtnStyle := lipgloss.NewStyle().
					Background(lipgloss.Color("#BD93F9")).
					Foreground(lipgloss.Color("#000000")).
					Padding(0, 1).
					Bold(true)
				actionButtons = append(actionButtons,
					r.Mark(mouse.ZoneJiraChangeStatus, highlightedBtnStyle.Render("Change Status (c)")),
				)
				headerLines = append(headerLines, strings.Join(actionButtons, " "))
				headerLines = append(headerLines, "") // Blank line between rows

				// Status options on a new line with even spacing
				var statusButtons []string
				for i, t := range data.AvailableTransitions {
					var shortcut string
					btnStyle := ButtonStyle

					// Check for common transition patterns
					lowerName := strings.ToLower(t.Name)

					// Not Started: contains "not" and "start"
					isNotStarted := strings.Contains(lowerName, "not") && strings.Contains(lowerName, "start")

					// In Progress: contains "progress" OR ("start" but not "not start")
					isInProgress := strings.Contains(lowerName, "progress") ||
						(strings.Contains(lowerName, "start") && !strings.Contains(lowerName, "not start") && !strings.Contains(lowerName, "not_start"))

					if isNotStarted {
						shortcut = " (N)"
						btnStyle = lipgloss.NewStyle().
							Background(lipgloss.Color("#6272A4")).
							Foreground(lipgloss.Color("#FFFFFF")).
							Padding(0, 1).
							Bold(true)
					} else if isInProgress {
						shortcut = " (i)"
						btnStyle = lipgloss.NewStyle().
							Background(lipgloss.Color("#FFB86C")).
							Foreground(lipgloss.Color("#000000")).
							Padding(0, 1).
							Bold(true)
					} else if strings.Contains(lowerName, "done") || strings.Contains(lowerName, "complete") || strings.Contains(lowerName, "resolve") {
						shortcut = " (D)"
						btnStyle = lipgloss.NewStyle().
							Background(lipgloss.Color("#50FA7B")).
							Foreground(lipgloss.Color("#000000")).
							Padding(0, 1).
							Bold(true)
					} else if strings.Contains(lowerName, "block") {
						shortcut = " (B)"
						btnStyle = lipgloss.NewStyle().
							Background(lipgloss.Color("#FF5555")).
							Foreground(lipgloss.Color("#FFFFFF")).
							Padding(0, 1).
							Bold(true)
					}

					zoneID := mouse.ZoneJiraTransition + fmt.Sprintf("%d", i)
					btn := r.Mark(zoneID, btnStyle.Render(t.Name+shortcut))
					statusButtons = append(statusButtons, btn)
				}
				headerLines = append(headerLines, "  "+strings.Join(statusButtons, "   ")) // More spacing between buttons
			} else {
				// Collapsed: just show "Change Status" button
				actionButtons = append(actionButtons,
					r.Mark(mouse.ZoneJiraChangeStatus, ButtonStyle.Render("Change Status (c)")),
				)
				headerLines = append(headerLines, strings.Join(actionButtons, " "))
			}
		} else if data.TransitionInProgress {
			headerLines = append(headerLines, lipgloss.NewStyle().Foreground(ColorMuted).Italic(true).Render("Updating status..."))
		} else {
			headerLines = append(headerLines, strings.Join(actionButtons, " "))
		}
		headerLines = append(headerLines, separator)
	}

	headerLines = append(headerLines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Select a ticket to create a branch:"))

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

		listLines = append(listLines, r.Mark(mouse.ZoneJiraTicket(i), style.Render(ticketLine)))
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
