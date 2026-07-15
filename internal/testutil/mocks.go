// Package testutil provides mock implementations for testing
package testutil

import (
	"context"
	"fmt"

	"github.com/madicen/jj-tui/internal/tickets"
)

// MockTicketService mocks ticket provider interactions (Jira/Codecks)
type MockTicketService struct {
	Tickets        []tickets.Ticket
	ProviderName   string
	BaseURL        string
	GetTicketsFunc func(ctx context.Context) ([]tickets.Ticket, error)
}

// GetAssignedTickets returns mock tickets
func (m *MockTicketService) GetAssignedTickets(ctx context.Context) ([]tickets.Ticket, error) {
	if m.GetTicketsFunc != nil {
		return m.GetTicketsFunc(ctx)
	}
	return m.Tickets, nil
}

// GetTicket returns a specific mock ticket
func (m *MockTicketService) GetTicket(ctx context.Context, key string) (*tickets.Ticket, error) {
	for _, t := range m.Tickets {
		if t.Key == key {
			return &t, nil
		}
	}
	return nil, nil
}

// GetTicketURL returns a mock URL for a ticket
func (m *MockTicketService) GetTicketURL(ticket tickets.Ticket) string {
	return m.BaseURL + "/ticket/" + ticket.Key
}

// GetProviderName returns the provider name
func (m *MockTicketService) GetProviderName() string {
	return m.ProviderName
}

// GetAvailableTransitions returns mock transitions
func (m *MockTicketService) GetAvailableTransitions(ctx context.Context, ticketKey string) ([]tickets.Transition, error) {
	// Return common transitions for testing
	return []tickets.Transition{
		{ID: "21", Name: "In Progress"},
		{ID: "31", Name: "Done"},
	}, nil
}

// TransitionTicket mocks transitioning a ticket
func (m *MockTicketService) TransitionTicket(ctx context.Context, ticketKey string, transitionID string) error {
	// Mock successful transition
	return nil
}

// CanCreateTicket returns true for testing create-ticket flows
func (m *MockTicketService) CanCreateTicket() bool {
	return true
}

// CreateTicket adds a mock ticket and returns it
func (m *MockTicketService) CreateTicket(ctx context.Context, input *tickets.CreateTicketInput) (*tickets.Ticket, error) {
	if input == nil || input.Summary == "" {
		return nil, fmt.Errorf("summary is required")
	}
	t := tickets.Ticket{
		Key:         "MOCK-NEW",
		DisplayKey:  "MOCK-NEW",
		Summary:     input.Summary,
		Description: input.Description,
		Status:      "To Do",
		Type:        "Task",
	}
	m.Tickets = append([]tickets.Ticket{t}, m.Tickets...)
	return &t, nil
}

// NewMockJiraService creates a mock Jira service with sample data
func NewMockJiraService() *MockTicketService {
	return &MockTicketService{
		ProviderName: "Jira",
		BaseURL:      "https://test.atlassian.net",
		Tickets: []tickets.Ticket{
			{Key: "PROJ-123", DisplayKey: "PROJ-123", Summary: "Implement user authentication", Status: "In Progress", Type: "Story", Priority: "High"},
			{Key: "PROJ-124", DisplayKey: "PROJ-124", Summary: "Fix login button styling", Status: "To Do", Type: "Bug", Priority: "Medium"},
			{Key: "PROJ-125", DisplayKey: "PROJ-125", Summary: "Add unit tests", Status: "Done", Type: "Task", Priority: "Low"},
		},
	}
}
