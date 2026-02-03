package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/madicen/jj-tui/internal/tickets"
)

// Service handles Jira API interactions using REST API v3
type Service struct {
	baseURL  string
	username string
	token    string
	client   *http.Client
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

	// Ensure baseURL doesn't have trailing slash
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &Service{
		baseURL:  baseURL,
		username: username,
		token:    token,
		client:   &http.Client{},
	}, nil
}

// searchResponse represents the response from Jira search API v3
type searchResponse struct {
	Issues []struct {
		Key    string `json:"key"`
		Fields struct {
			Summary     string `json:"summary"`
			Description *struct {
				Content []struct {
					Content []struct {
						Text string `json:"text"`
					} `json:"content"`
				} `json:"content"`
			} `json:"description"`
			Status *struct {
				Name string `json:"name"`
			} `json:"status"`
			Priority *struct {
				Name string `json:"name"`
			} `json:"priority"`
			IssueType *struct {
				Name string `json:"name"`
			} `json:"issuetype"`
		} `json:"fields"`
	} `json:"issues"`
	Total int `json:"total"`
}

// doRequest performs an authenticated request to the Jira API
func (s *Service) doRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Response, error) {
	reqURL := s.baseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(s.username, s.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	return s.client.Do(req)
}

// GetAssignedTickets fetches tickets assigned to the current user using API v3
func (s *Service) GetAssignedTickets(ctx context.Context) ([]tickets.Ticket, error) {
	// JQL to find issues assigned to the current user that are not done
	jql := fmt.Sprintf("assignee = \"%s\" AND status != Done ORDER BY updated DESC", s.username)

	// Use the new /rest/api/3/search/jql endpoint
	// Must explicitly request fields - the v3 API returns minimal data by default
	fields := "key,summary,status,priority,issuetype,description"
	endpoint := "/rest/api/3/search/jql?jql=" + url.QueryEscape(jql) + "&maxResults=50&fields=" + fields

	resp, err := s.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search issues: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Jira API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	ticketList := make([]tickets.Ticket, 0, len(result.Issues))
	for _, issue := range result.Issues {
		ticket := tickets.Ticket{
			Key:        issue.Key,
			DisplayKey: issue.Key, // For Jira, display key is the same as the key (e.g., "PROJ-123")
			Summary:    issue.Fields.Summary,
		}

		if issue.Fields.Status != nil {
			ticket.Status = issue.Fields.Status.Name
		}
		if issue.Fields.Priority != nil {
			ticket.Priority = issue.Fields.Priority.Name
		}
		if issue.Fields.IssueType != nil {
			ticket.Type = issue.Fields.IssueType.Name
		}

		// Extract description text from Atlassian Document Format (ADF)
		if issue.Fields.Description != nil && len(issue.Fields.Description.Content) > 0 {
			var descParts []string
			for _, block := range issue.Fields.Description.Content {
				for _, inline := range block.Content {
					if inline.Text != "" {
						descParts = append(descParts, inline.Text)
					}
				}
			}
			ticket.Description = strings.Join(descParts, " ")
		}

		ticketList = append(ticketList, ticket)
	}

	return ticketList, nil
}

// issueResponse represents a single issue from Jira API v3
type issueResponse struct {
	Key    string `json:"key"`
	Fields struct {
		Summary     string `json:"summary"`
		Description *struct {
			Content []struct {
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"content"`
		} `json:"description"`
		Status *struct {
			Name string `json:"name"`
		} `json:"status"`
		Priority *struct {
			Name string `json:"name"`
		} `json:"priority"`
		IssueType *struct {
			Name string `json:"name"`
		} `json:"issuetype"`
	} `json:"fields"`
}

// GetTicket fetches a single ticket by key
func (s *Service) GetTicket(ctx context.Context, key string) (*tickets.Ticket, error) {
	endpoint := "/rest/api/3/issue/" + key

	resp, err := s.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue %s: %w", key, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Jira API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var issue issueResponse
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	ticket := &tickets.Ticket{
		Key:        issue.Key,
		DisplayKey: issue.Key, // For Jira, display key is the same as the key
		Summary:    issue.Fields.Summary,
	}

	if issue.Fields.Status != nil {
		ticket.Status = issue.Fields.Status.Name
	}
	if issue.Fields.Priority != nil {
		ticket.Priority = issue.Fields.Priority.Name
	}
	if issue.Fields.IssueType != nil {
		ticket.Type = issue.Fields.IssueType.Name
	}

	// Extract description text from Atlassian Document Format (ADF)
	if issue.Fields.Description != nil && len(issue.Fields.Description.Content) > 0 {
		var descParts []string
		for _, block := range issue.Fields.Description.Content {
			for _, inline := range block.Content {
				if inline.Text != "" {
					descParts = append(descParts, inline.Text)
				}
			}
		}
		ticket.Description = strings.Join(descParts, " ")
	}

	return ticket, nil
}

// GetTicketURL returns the browser URL for a ticket
func (s *Service) GetTicketURL(ticketKey string) string {
	return s.baseURL + "/browse/" + ticketKey
}

// GetBaseURL returns the Jira base URL
func (s *Service) GetBaseURL() string {
	return s.baseURL
}

// GetProviderName returns the name of this provider
func (s *Service) GetProviderName() string {
	return "Jira"
}

// IsConfigured returns true if Jira environment variables are set
func IsConfigured() bool {
	return os.Getenv("JIRA_URL") != "" &&
		os.Getenv("JIRA_USER") != "" &&
		os.Getenv("JIRA_TOKEN") != ""
}
