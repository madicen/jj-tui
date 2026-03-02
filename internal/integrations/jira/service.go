package jira

import (
	"bytes"
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

	svc := &Service{
		baseURL:  baseURL,
		username: username,
		token:    token,
		client:   &http.Client{},
	}

	// Verify the token has proper permissions by checking BROWSE_PROJECTS
	if err := svc.checkPermissions(); err != nil {
		return nil, err
	}

	return svc, nil
}

// checkPermissions verifies the API token has necessary permissions
func (s *Service) checkPermissions() error {
	ctx := context.Background()
	
	// Check if we have BROWSE_PROJECTS permission
	resp, err := s.doRequest(ctx, "GET", "/rest/api/3/mypermissions?permissions=BROWSE_PROJECTS", nil)
	if err != nil {
		return fmt.Errorf("failed to check permissions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("authentication failed - check your Jira credentials")
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to check permissions (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Permissions map[string]struct {
			HavePermission bool `json:"havePermission"`
		} `json:"permissions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode permissions response: %w", err)
	}

	browseProjects, ok := result.Permissions["BROWSE_PROJECTS"]
	if !ok {
		return fmt.Errorf("could not determine BROWSE_PROJECTS permission")
	}

	if !browseProjects.HavePermission {
		return fmt.Errorf("API token lacks 'Browse Projects' permission - regenerate your token at https://id.atlassian.com/manage-profile/security/api-tokens")
	}

	return nil
}

// buildJQL constructs the JQL query with optional project and custom filters
func (s *Service) buildJQL() string {
	var conditions []string

	// Base condition: assigned to current user
	conditions = append(conditions, fmt.Sprintf("assignee = \"%s\"", s.username))

	// Optional: filter by project(s) — use JIRA_PROJECT_FILTER for list, fall back to JIRA_PROJECT
	projectFilter := os.Getenv("JIRA_PROJECT_FILTER")
	if projectFilter == "" {
		projectFilter = os.Getenv("JIRA_PROJECT")
	}
	if projectFilter != "" {
		// Support comma-separated projects (e.g., "PROJ,TEAM")
		projects := strings.Split(projectFilter, ",")
		for i, p := range projects {
			projects[i] = strings.TrimSpace(p)
		}
		if len(projects) == 1 {
			conditions = append(conditions, fmt.Sprintf("project = \"%s\"", projects[0]))
		} else {
			// Multiple projects: project IN ("PROJ", "TEAM")
			quotedProjects := make([]string, len(projects))
			for i, p := range projects {
				quotedProjects[i] = fmt.Sprintf("\"%s\"", p)
			}
			conditions = append(conditions, fmt.Sprintf("project IN (%s)", strings.Join(quotedProjects, ", ")))
		}
	}

	// Optional: custom JQL filter
	if customJQL := os.Getenv("JIRA_JQL"); customJQL != "" {
		conditions = append(conditions, customJQL)
	}

	// Combine all conditions with AND, then add ORDER BY
	jql := strings.Join(conditions, " AND ") + " ORDER BY updated DESC"
	return jql
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
	// Build JQL query with optional filters
	jql := s.buildJQL()

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
		return nil, fmt.Errorf("jira API error (status %d): %s", resp.StatusCode, string(bodyBytes))
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
		return nil, fmt.Errorf("jira API error (status %d): %s", resp.StatusCode, string(bodyBytes))
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
func (s *Service) GetTicketURL(ticket tickets.Ticket) string {
	return s.baseURL + "/browse/" + ticket.Key
}

// GetBaseURL returns the Jira base URL
func (s *Service) GetBaseURL() string {
	return s.baseURL
}

// GetProviderName returns the name of this provider
func (s *Service) GetProviderName() string {
	return "Jira"
}

// transitionsResponse represents the response from Jira transitions API
type transitionsResponse struct {
	Transitions []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		To   struct {
			Name string `json:"name"`
		} `json:"to"`
	} `json:"transitions"`
}

// GetAvailableTransitions returns the available status transitions for a Jira issue
func (s *Service) GetAvailableTransitions(ctx context.Context, ticketKey string) ([]tickets.Transition, error) {
	endpoint := "/rest/api/3/issue/" + ticketKey + "/transitions"

	resp, err := s.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get transitions for %s: %w", ticketKey, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jira API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result transitionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode transitions response: %w", err)
	}

	transitions := make([]tickets.Transition, 0, len(result.Transitions))
	for _, t := range result.Transitions {
		transitions = append(transitions, tickets.Transition{
			ID:   t.ID,
			Name: t.Name,
		})
	}

	return transitions, nil
}

// TransitionTicket executes a transition on a Jira issue
func (s *Service) TransitionTicket(ctx context.Context, ticketKey string, transitionID string) error {
	endpoint := "/rest/api/3/issue/" + ticketKey + "/transitions"

	// Build the transition request body
	body := fmt.Sprintf(`{"transition":{"id":"%s"}}`, transitionID)

	resp, err := s.doRequest(ctx, "POST", endpoint, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to transition %s: %w", ticketKey, err)
	}
	defer resp.Body.Close()

	// Jira returns 204 No Content on successful transition
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("jira transition failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// createIssueRequest is the body for POST /rest/api/3/issue
type createIssueRequest struct {
	Fields struct {
		Project     map[string]string `json:"project"`
		Summary     string            `json:"summary"`
		IssueType   map[string]string `json:"issuetype"`
		Description *adfDocument      `json:"description,omitempty"`
	} `json:"fields"`
}

// adfDocument is Atlassian Document Format for description
type adfDocument struct {
	Type    string     `json:"type"`
	Version int        `json:"version"`
	Content []adfBlock `json:"content"`
}

type adfBlock struct {
	Type    string     `json:"type"`
	Content []adfText  `json:"content,omitempty"`
}

type adfText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// createIssueResponse is the response from POST /rest/api/3/issue
type createIssueResponse struct {
	Key string `json:"key"`
	ID  string `json:"id"`
}

// CanCreateTicket returns true if Jira project is configured (JIRA_PROJECT or first project from API).
func (s *Service) CanCreateTicket() bool {
	project := os.Getenv("JIRA_PROJECT")
	return project != ""
}

// CreateTicket creates a new Jira issue. Uses JIRA_PROJECT for project key and JIRA_ISSUE_TYPE for type (default "Task").
func (s *Service) CreateTicket(ctx context.Context, input *tickets.CreateTicketInput) (*tickets.Ticket, error) {
	if input == nil || strings.TrimSpace(input.Summary) == "" {
		return nil, fmt.Errorf("summary is required")
	}
	projectKey := os.Getenv("JIRA_PROJECT")
	if projectKey == "" {
		return nil, fmt.Errorf("JIRA_PROJECT is required to create issues")
	}
	issueType := os.Getenv("JIRA_ISSUE_TYPE")
	if issueType == "" {
		issueType = "Task"
	}
	reqBody := createIssueRequest{}
	reqBody.Fields.Project = map[string]string{"key": strings.TrimSpace(projectKey)}
	reqBody.Fields.IssueType = map[string]string{"name": issueType}
	reqBody.Fields.Summary = strings.TrimSpace(input.Summary)
	if input.Description != "" {
		reqBody.Fields.Description = &adfDocument{
			Type:    "doc",
			Version: 1,
			Content: []adfBlock{{
				Type: "paragraph",
				Content: []adfText{{Type: "text", Text: strings.TrimSpace(input.Description)}},
			}},
		}
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := s.doRequest(ctx, "POST", "/rest/api/3/issue", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create issue: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jira API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}
	var created createIssueResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	// Fetch full ticket so we return consistent Ticket fields
	return s.GetTicket(ctx, created.Key)
}
func IsConfigured() bool {
	return os.Getenv("JIRA_URL") != "" &&
		os.Getenv("JIRA_USER") != "" &&
		os.Getenv("JIRA_TOKEN") != ""
}
