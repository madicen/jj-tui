// Package mock provides mock implementations for demo/testing purposes
package mock

import (
	"context"

	"github.com/madicen/jj-tui/internal/tickets"
)

// TicketService is a mock ticket service that returns demo data
type TicketService struct {
	provider string
	tickets  []tickets.Ticket
}

// NewTicketService creates a new mock ticket service with demo data
func NewTicketService(provider string) *TicketService {
	var ticketList []tickets.Ticket

	switch provider {
	case "jira":
		ticketList = jiraTickets()
	case "codecks":
		ticketList = codecksTickets()
	case "github_issues":
		ticketList = githubIssuesTickets()
	default:
		ticketList = jiraTickets() // Default to Jira-style
	}

	return &TicketService{
		provider: provider,
		tickets:  ticketList,
	}
}

// GetAssignedTickets returns demo tickets
func (s *TicketService) GetAssignedTickets(ctx context.Context) ([]tickets.Ticket, error) {
	return s.tickets, nil
}

// GetTicket returns a single ticket by key
func (s *TicketService) GetTicket(ctx context.Context, key string) (*tickets.Ticket, error) {
	for _, t := range s.tickets {
		if t.Key == key || t.DisplayKey == key {
			return &t, nil
		}
	}
	return nil, nil
}

// GetTicketURL returns a demo URL for a ticket
func (s *TicketService) GetTicketURL(ticket tickets.Ticket) string {
	switch s.provider {
	case "jira":
		return "https://demo.atlassian.net/browse/" + ticket.Key
	case "codecks":
		return "https://demo.codecks.io/card/" + ticket.Key
	case "github_issues":
		return "https://github.com/demo/repo/issues/" + ticket.Key
	default:
		return "https://example.com/ticket/" + ticket.Key
	}
}

// GetProviderName returns the provider name
func (s *TicketService) GetProviderName() string {
	switch s.provider {
	case "jira":
		return "Jira"
	case "codecks":
		return "Codecks"
	case "github_issues":
		return "GitHub Issues"
	default:
		return "Demo Tickets"
	}
}

// GetAvailableTransitions returns demo transitions
func (s *TicketService) GetAvailableTransitions(ctx context.Context, ticketKey string) ([]tickets.Transition, error) {
	// Find the ticket to check its status
	for _, t := range s.tickets {
		if t.Key == ticketKey || t.DisplayKey == ticketKey {
			switch t.Status {
			case "To Do", "Open", "Backlog":
				return []tickets.Transition{
					{ID: "in_progress", Name: "Start Progress"},
					{ID: "done", Name: "Done"},
				}, nil
			case "In Progress", "Started":
				return []tickets.Transition{
					{ID: "done", Name: "Done"},
					{ID: "todo", Name: "Back to To Do"},
				}, nil
			case "Done", "Closed", "Resolved":
				return []tickets.Transition{
					{ID: "reopen", Name: "Reopen"},
				}, nil
			}
		}
	}
	return []tickets.Transition{}, nil
}

// TransitionTicket updates the ticket status in the mock
func (s *TicketService) TransitionTicket(ctx context.Context, ticketKey string, transitionID string) error {
	// Find and update the ticket status
	for i, t := range s.tickets {
		if t.Key == ticketKey || t.DisplayKey == ticketKey {
			switch transitionID {
			case "in_progress":
				s.tickets[i].Status = "In Progress"
			case "done":
				s.tickets[i].Status = "Done"
			case "todo":
				s.tickets[i].Status = "To Do"
			case "reopen":
				s.tickets[i].Status = "To Do"
			}
			break
		}
	}
	return nil
}

// jiraTickets returns demo Jira-style tickets
func jiraTickets() []tickets.Ticket {
	return []tickets.Ticket{
		{
			Key:         "PROJ-142",
			DisplayKey:  "PROJ-142",
			Summary:     "Add dark mode support to dashboard",
			Status:      "In Progress",
			Priority:    "High",
			Type:        "Story",
			Description: "Users have requested a dark mode option for the dashboard to reduce eye strain during night usage.",
		},
		{
			Key:         "PROJ-139",
			DisplayKey:  "PROJ-139",
			Summary:     "Fix pagination bug in search results",
			Status:      "To Do",
			Priority:    "Critical",
			Type:        "Bug",
			Description: "Search results pagination breaks when filtering by date range.",
		},
		{
			Key:         "PROJ-135",
			DisplayKey:  "PROJ-135",
			Summary:     "Implement user profile settings page",
			Status:      "In Progress",
			Priority:    "Medium",
			Type:        "Story",
			Description: "Create a new settings page where users can update their profile information.",
		},
		{
			Key:         "PROJ-128",
			DisplayKey:  "PROJ-128",
			Summary:     "Add export to CSV functionality",
			Status:      "To Do",
			Priority:    "Low",
			Type:        "Feature",
			Description: "Allow users to export their data to CSV format for offline analysis.",
		},
		{
			Key:         "PROJ-121",
			DisplayKey:  "PROJ-121",
			Summary:     "Update authentication flow for SSO",
			Status:      "Done",
			Priority:    "High",
			Type:        "Story",
			Description: "Integrate with corporate SSO provider for seamless authentication.",
		},
		{
			Key:         "PROJ-118",
			DisplayKey:  "PROJ-118",
			Summary:     "Performance optimization for large datasets",
			Status:      "To Do",
			Priority:    "Medium",
			Type:        "Task",
			Description: "Improve rendering performance when displaying tables with 10k+ rows.",
		},
	}
}

// codecksTickets returns demo Codecks-style tickets
func codecksTickets() []tickets.Ticket {
	return []tickets.Ticket{
		{
			Key:         "card-uuid-1",
			DisplayKey:  "$12u",
			Summary:     "Add particle effects to player abilities",
			Status:      "Started",
			Priority:    "Highest",
			Type:        "Card",
			Description: "Visual polish pass - add particle effects when player uses special abilities.",
		},
		{
			Key:         "card-uuid-2",
			DisplayKey:  "$12v",
			Summary:     "Fix collision detection on slopes",
			Status:      "Open",
			Priority:    "High",
			Type:        "Card",
			Description: "Player clips through terrain on steep slopes. Needs physics adjustment.",
		},
		{
			Key:         "card-uuid-3",
			DisplayKey:  "$12w",
			Summary:     "Implement save/load system",
			Status:      "Started",
			Priority:    "High",
			Type:        "Card",
			Description: "Core feature - allow players to save and load their game progress.",
		},
		{
			Key:         "card-uuid-4",
			DisplayKey:  "$12x",
			Summary:     "Add sound effects for UI interactions",
			Status:      "Open",
			Priority:    "Normal",
			Type:        "Card",
			Description: "Polish pass - add subtle sound effects for button clicks and menu navigation.",
		},
		{
			Key:         "card-uuid-5",
			DisplayKey:  "$12y",
			Summary:     "Create tutorial level",
			Status:      "Done",
			Priority:    "Normal",
			Type:        "Card",
			Description: "Design and implement an introductory level that teaches basic mechanics.",
		},
	}
}

// githubIssuesTickets returns demo GitHub Issues-style tickets
func githubIssuesTickets() []tickets.Ticket {
	return []tickets.Ticket{
		{
			Key:         "#47",
			DisplayKey:  "#47",
			Summary:     "Support for custom themes",
			Status:      "Open",
			Priority:    "enhancement",
			Type:        "Feature",
			Description: "Allow users to define custom color themes via config file.",
		},
		{
			Key:         "#45",
			DisplayKey:  "#45",
			Summary:     "Crash when repository has no commits",
			Status:      "Open",
			Priority:    "bug",
			Type:        "Bug",
			Description: "App crashes with nil pointer when opening an empty repository.",
		},
		{
			Key:         "#42",
			DisplayKey:  "#42",
			Summary:     "Add keyboard shortcut reference card",
			Status:      "Open",
			Priority:    "documentation",
			Type:        "Documentation",
			Description: "Create a printable reference card with all keyboard shortcuts.",
		},
		{
			Key:         "#38",
			DisplayKey:  "#38",
			Summary:     "Improve startup time",
			Status:      "Open",
			Priority:    "performance",
			Type:        "Feature",
			Description: "App takes 2+ seconds to start on large repositories.",
		},
		{
			Key:         "#35",
			DisplayKey:  "#35",
			Summary:     "Add support for multiple remotes",
			Status:      "Closed",
			Priority:    "enhancement",
			Type:        "Feature",
			Description: "Currently only supports 'origin' remote, should support multiple.",
		},
	}
}

// Ensure TicketService implements tickets.Service
var _ tickets.Service = (*TicketService)(nil)

