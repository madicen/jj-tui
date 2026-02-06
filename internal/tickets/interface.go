// Package tickets provides a common interface for ticket/issue tracking services
package tickets

import "context"

// Ticket represents a generic ticket from any provider
type Ticket struct {
	Key         string // Full ID used for API calls and URLs
	DisplayKey  string // Short ID for display (e.g., "$12u" for Codecks, "PROJ-123" for Jira)
	Summary     string
	Status      string
	Priority    string
	Type        string
	Description string
	DeckID      string // Codecks: deck ID for URL construction
}

// Transition represents a possible status transition for a ticket
type Transition struct {
	ID   string // Transition ID (for Jira) or status value (for Codecks)
	Name string // Human-readable name (e.g., "In Progress", "Done")
}

// Service is the interface that all ticket providers must implement
type Service interface {
	// GetAssignedTickets returns tickets assigned to the current user
	GetAssignedTickets(ctx context.Context) ([]Ticket, error)

	// GetTicket returns a single ticket by key
	GetTicket(ctx context.Context, key string) (*Ticket, error)

	// GetTicketURL returns the browser URL for a ticket
	GetTicketURL(ticket Ticket) string

	// GetProviderName returns the name of the ticket provider (e.g., "Jira", "Codecks")
	GetProviderName() string

	// GetAvailableTransitions returns the available status transitions for a ticket
	// Returns nil/empty if transitions are not supported or unavailable
	GetAvailableTransitions(ctx context.Context, ticketKey string) ([]Transition, error)

	// TransitionTicket changes the ticket's status using the given transition
	// For Jira: transitionID is the transition ID
	// For Codecks: transitionID is the target status value (e.g., "started", "done")
	TransitionTicket(ctx context.Context, ticketKey string, transitionID string) error
}

// Provider represents a ticket provider type
type Provider string

const (
	ProviderNone    Provider = ""
	ProviderJira    Provider = "jira"
	ProviderCodecks Provider = "codecks"
)

// Common transition names for convenience
const (
	TransitionInProgress = "in_progress"
	TransitionDone       = "done"
)

