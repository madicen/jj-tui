// Package tickets provides a common interface for ticket/issue tracking services
package tickets

import "context"

// Ticket represents a generic ticket from any provider
type Ticket struct {
	Key         string // Full ID used for API calls and URLs
	DisplayKey  string // Short ID for display (e.g., "#51" for Codecks, "PROJ-123" for Jira)
	Summary     string
	Status      string
	Priority    string
	Type        string
	Description string
}

// Service is the interface that all ticket providers must implement
type Service interface {
	// GetAssignedTickets returns tickets assigned to the current user
	GetAssignedTickets(ctx context.Context) ([]Ticket, error)

	// GetTicket returns a single ticket by key
	GetTicket(ctx context.Context, key string) (*Ticket, error)

	// GetTicketURL returns the browser URL for a ticket
	GetTicketURL(ticketKey string) string

	// GetProviderName returns the name of the ticket provider (e.g., "Jira", "Codecks")
	GetProviderName() string
}

// Provider represents a ticket provider type
type Provider string

const (
	ProviderNone    Provider = ""
	ProviderJira    Provider = "jira"
	ProviderCodecks Provider = "codecks"
)

