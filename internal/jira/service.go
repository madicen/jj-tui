package jira

import (
	"context"
	"fmt"
	"os"

	"github.com/andygrunwald/go-jira"
)

// Ticket represents a Jira issue
type Ticket struct {
	Key         string
	Summary     string
	Status      string
	Priority    string
	Type        string
	Description string
}

// Service handles Jira API interactions
type Service struct {
	client   *jira.Client
	baseURL  string
	username string
}

// NewService creates a new Jira service
// Requires environment variables: JIRA_URL, JIRA_USER, JIRA_TOKEN
func NewService() (*Service, error) {
	baseURL := os.Getenv("JIRA_URL")
	username := os.Getenv("JIRA_USER")
	token := os.Getenv("JIRA_TOKEN")

	if baseURL == "" {
		return nil, fmt.Errorf("JIRA_URL environment variable not set")
	}
	if username == "" {
		return nil, fmt.Errorf("JIRA_USER environment variable not set")
	}
	if token == "" {
		return nil, fmt.Errorf("JIRA_TOKEN environment variable not set")
	}

	tp := jira.BasicAuthTransport{
		Username: username,
		Password: token,
	}

	client, err := jira.NewClient(tp.Client(), baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create Jira client: %w", err)
	}

	return &Service{
		client:   client,
		baseURL:  baseURL,
		username: username,
	}, nil
}

// GetAssignedTickets fetches tickets assigned to the current user
func (s *Service) GetAssignedTickets(ctx context.Context) ([]Ticket, error) {
	// JQL to find issues assigned to the current user that are not done
	jql := fmt.Sprintf("assignee = \"%s\" AND status != Done ORDER BY updated DESC", s.username)

	issues, _, err := s.client.Issue.Search(jql, &jira.SearchOptions{
		MaxResults: 50,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search issues: %w", err)
	}

	tickets := make([]Ticket, 0, len(issues))
	for _, issue := range issues {
		ticket := Ticket{
			Key:     issue.Key,
			Summary: issue.Fields.Summary,
		}

		if issue.Fields.Status != nil {
			ticket.Status = issue.Fields.Status.Name
		}
		if issue.Fields.Priority != nil {
			ticket.Priority = issue.Fields.Priority.Name
		}
		if issue.Fields.Type.Name != "" {
			ticket.Type = issue.Fields.Type.Name
		}
		if issue.Fields.Description != "" {
			ticket.Description = issue.Fields.Description
		}

		tickets = append(tickets, ticket)
	}

	return tickets, nil
}

// GetTicket fetches a single ticket by key
func (s *Service) GetTicket(ctx context.Context, key string) (*Ticket, error) {
	issue, _, err := s.client.Issue.Get(key, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue %s: %w", key, err)
	}

	ticket := &Ticket{
		Key:     issue.Key,
		Summary: issue.Fields.Summary,
	}

	if issue.Fields.Status != nil {
		ticket.Status = issue.Fields.Status.Name
	}
	if issue.Fields.Priority != nil {
		ticket.Priority = issue.Fields.Priority.Name
	}
	if issue.Fields.Type.Name != "" {
		ticket.Type = issue.Fields.Type.Name
	}
	if issue.Fields.Description != "" {
		ticket.Description = issue.Fields.Description
	}

	return ticket, nil
}

// IsConfigured returns true if Jira environment variables are set
func IsConfigured() bool {
	return os.Getenv("JIRA_URL") != "" &&
		os.Getenv("JIRA_USER") != "" &&
		os.Getenv("JIRA_TOKEN") != ""
}
