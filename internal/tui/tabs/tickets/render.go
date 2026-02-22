package tickets

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

func mark(z *zone.Manager, id, content string) string {
	if z == nil {
		return content
	}
	return z.Mark(id, content)
}

func (m *Model) renderTickets() string {
	if !m.jiraService {
		noTickets := []string{
			styles.TitleStyle.Render("Ticket Integration"),
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
		return strings.Join(noTickets, "\n")
	}

	if len(m.ticketList) == 0 {
		return "No assigned tickets found.\n\nPress Ctrl+r to refresh."
	}

	var headerLines []string

	if m.selectedTicket >= 0 && m.selectedTicket < len(m.ticketList) {
		ticket := m.ticketList[m.selectedTicket]
		displayKey := ticket.DisplayKey
		if displayKey == "" {
			displayKey = ticket.Key
		}

		var detailLines []string
		detailLines = append(detailLines, fmt.Sprintf("%s %s",
			lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render(displayKey),
			ticket.Summary,
		))
		detailLines = append(detailLines, fmt.Sprintf("Type: %s  |  Priority: %s  |  Status: %s",
			ticket.Type, ticket.Priority, ticket.Status,
		))
		if ticket.Description != "" {
			desc := ticket.Description
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
		actionButtons = append(actionButtons,
			mark(m.zoneManager, mouse.ZoneJiraCreateBranch, styles.ButtonStyle.Render("Create Branch (Enter)")),
			mark(m.zoneManager, mouse.ZoneJiraOpenBrowser, styles.ButtonStyle.Render("Open in Browser (o)")),
		)

		if len(m.availableTransitions) > 0 && !m.transitionInProgress {
			if m.statusChangeMode {
				highlightedBtnStyle := lipgloss.NewStyle().
					Background(lipgloss.Color("#BD93F9")).
					Foreground(lipgloss.Color("#000000")).
					Padding(0, 1).
					Bold(true)
				actionButtons = append(actionButtons,
					mark(m.zoneManager, mouse.ZoneJiraChangeStatus, highlightedBtnStyle.Render("Change Status (c)")),
				)
				headerLines = append(headerLines, strings.Join(actionButtons, " "))
				headerLines = append(headerLines, "")

				var statusButtons []string
				for i, t := range m.availableTransitions {
					var shortcut string
					btnStyle := styles.ButtonStyle
					lowerName := strings.ToLower(t.Name)
					isNotStarted := strings.Contains(lowerName, "not") && strings.Contains(lowerName, "start")
					isInProgress := strings.Contains(lowerName, "progress") ||
						(strings.Contains(lowerName, "start") && !strings.Contains(lowerName, "not start") && !strings.Contains(lowerName, "not_start"))

					if isNotStarted {
						shortcut = " (N)"
						btnStyle = lipgloss.NewStyle().Background(lipgloss.Color("#6272A4")).Foreground(lipgloss.Color("#FFFFFF")).Padding(0, 1).Bold(true)
					} else if isInProgress {
						shortcut = " (i)"
						btnStyle = lipgloss.NewStyle().Background(lipgloss.Color("#FFB86C")).Foreground(lipgloss.Color("#000000")).Padding(0, 1).Bold(true)
					} else if strings.Contains(lowerName, "done") || strings.Contains(lowerName, "complete") || strings.Contains(lowerName, "resolve") {
						shortcut = " (D)"
						btnStyle = lipgloss.NewStyle().Background(lipgloss.Color("#50FA7B")).Foreground(lipgloss.Color("#000000")).Padding(0, 1).Bold(true)
					} else if strings.Contains(lowerName, "block") {
						shortcut = " (B)"
						btnStyle = lipgloss.NewStyle().Background(lipgloss.Color("#FF5555")).Foreground(lipgloss.Color("#FFFFFF")).Padding(0, 1).Bold(true)
					}
					zoneID := mouse.ZoneJiraTransition + fmt.Sprintf("%d", i)
					statusButtons = append(statusButtons, mark(m.zoneManager, zoneID, btnStyle.Render(t.Name+shortcut)))
				}
				headerLines = append(headerLines, "  "+strings.Join(statusButtons, "   "))
			} else {
				actionButtons = append(actionButtons,
					mark(m.zoneManager, mouse.ZoneJiraChangeStatus, styles.ButtonStyle.Render("Change Status (c)")),
				)
				headerLines = append(headerLines, strings.Join(actionButtons, " "))
			}
		} else if m.transitionInProgress {
			headerLines = append(headerLines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Italic(true).Render("Updating status..."))
		} else {
			headerLines = append(headerLines, strings.Join(actionButtons, " "))
		}
		headerLines = append(headerLines, separator)
	}

	headerLines = append(headerLines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Select a ticket to create a branch:"))

	var listLines []string
	for i, ticket := range m.ticketList {
		prefix := "  "
		style := styles.CommitStyle
		if i == m.selectedTicket {
			prefix = "► "
			style = styles.CommitSelectedStyle
		}
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
			statusStyle = lipgloss.NewStyle().Foreground(styles.ColorMuted)
		}
		displayKey := ticket.DisplayKey
		if displayKey == "" {
			displayKey = ticket.Key
		}
		ticketLine := fmt.Sprintf("%s%s %s %s",
			prefix,
			lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render(displayKey),
			statusStyle.Render("["+ticket.Status+"]"),
			ticket.Summary,
		)
		listLines = append(listLines, mark(m.zoneManager, mouse.ZoneJiraTicket(i), style.Render(ticketLine)))
	}

	fixedHeader := strings.Join(headerLines, "\n")
	headerLineCount := len(headerLines)
	listHeight := m.height - headerLineCount
	if listHeight <= 0 {
		listHeight = 0
	}
	totalListLines := len(listLines)
	maxListOffset := 0
	if totalListLines > listHeight {
		maxListOffset = totalListLines - listHeight
	}
	if m.listYOffset > maxListOffset {
		m.listYOffset = maxListOffset
	}
	if m.listYOffset < 0 {
		m.listYOffset = 0
	}
	if m.selectedTicket >= 0 && m.selectedTicket < totalListLines {
		if m.selectedTicket < m.listYOffset {
			m.listYOffset = m.selectedTicket
		} else if m.selectedTicket >= m.listYOffset+listHeight {
			m.listYOffset = m.selectedTicket - listHeight + 1
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
	outLines := strings.Split(out, "\n")
	for len(outLines) < m.height {
		outLines = append(outLines, "")
	}
	if len(outLines) > m.height {
		outLines = outLines[:m.height]
	}
	return strings.Join(outLines, "\n")
}
