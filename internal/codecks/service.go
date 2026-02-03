// Package codecks handles Codecks API interactions
// API Reference: https://manual.codecks.io/api/
package codecks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/madicen/jj-tui/internal/tickets"
)

// Service handles Codecks API interactions
type Service struct {
	subdomain     string
	token         string
	projectFilter string            // Optional: filter cards by project name
	projectIDs    map[string]string // Map of project name -> project ID
	client        *http.Client
}

// NewService creates a new Codecks service
// Requires environment variables: CODECKS_SUBDOMAIN, CODECKS_TOKEN
// Optional: CODECKS_PROJECT to filter by project name
func NewService() (*Service, error) {
	subdomain := os.Getenv("CODECKS_SUBDOMAIN")
	token := os.Getenv("CODECKS_TOKEN")
	projectFilter := os.Getenv("CODECKS_PROJECT")

	if subdomain == "" {
		return nil, fmt.Errorf("CODECKS_SUBDOMAIN environment variable not set")
	}
	if token == "" {
		return nil, fmt.Errorf("CODECKS_TOKEN environment variable not set")
	}

	svc := &Service{
		subdomain:     subdomain,
		token:         token,
		projectFilter: projectFilter,
		projectIDs:    make(map[string]string),
		client:        &http.Client{},
	}

	// Verify connection and load project list
	if err := svc.loadProjects(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to connect to Codecks: %w", err)
	}

	return svc, nil
}

// projectDecks maps project ID to list of deck IDs
var projectDecks = make(map[string][]string)

// deckToProject maps deck ID to project ID
var deckToProject = make(map[string]string)

// archivedProjects tracks which project IDs are archived
var archivedProjects = make(map[string]bool)

// deletedDecks tracks which deck IDs are deleted
var deletedDecks = make(map[string]bool)

// deckMeta stores deck metadata for URL construction
type deckMeta struct {
	seq   int
	title string
}

// deckMetadata maps deck ID to its metadata
var deckMetadata = make(map[string]deckMeta)

// loadProjects fetches and caches the project list and deck mappings
func (s *Service) loadProjects(ctx context.Context) error {
	// Query projects with their decks, including archived projects and deleted deck status
	// Also fetch deck metadata (accountSeq, title) for URL construction
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"_root": []interface{}{
				map[string]interface{}{
					"account": []interface{}{
						"name",
						map[string]interface{}{
							"projects": []interface{}{
								"id", "name",
								map[string]interface{}{
									"decks": []string{"id", "isDeleted", "accountSeq", "title"},
								},
							},
						},
						map[string]interface{}{
							"archivedProjects": []string{"id", "name"},
						},
					},
				},
			},
		},
	}

	respBody, err := s.doRequest(ctx, query)
	if err != nil {
		return err
	}

	var rawResult map[string]interface{}
	if err := json.Unmarshal(respBody, &rawResult); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Track archived projects from the account's archivedProjects relation
	if account, ok := rawResult["account"].(map[string]interface{}); ok {
		for _, accData := range account {
			if accMap, ok := accData.(map[string]interface{}); ok {
				if archivedList, ok := accMap["archivedProjects"].([]interface{}); ok {
					for _, projID := range archivedList {
						if pid, ok := projID.(string); ok {
							archivedProjects[pid] = true
						}
					}
				}
			}
		}
	}

	// Track deleted decks and store deck metadata for URL construction
	if decksMap, ok := rawResult["deck"].(map[string]interface{}); ok {
		for deckID, deckData := range decksMap {
			if deck, ok := deckData.(map[string]interface{}); ok {
				if isDeleted, ok := deck["isDeleted"].(bool); ok && isDeleted {
					deletedDecks[deckID] = true
				}
				// Store deck metadata for URL construction
				seq := 0
				if seqFloat, ok := deck["accountSeq"].(float64); ok {
					seq = int(seqFloat)
				}
				title := ""
				if t, ok := deck["title"].(string); ok {
					title = t
				}
				deckMetadata[deckID] = deckMeta{seq: seq, title: title}
			}
		}
	}

	// Projects are in the normalized "project" object
	projectsMap, ok := rawResult["project"].(map[string]interface{})
	if ok {
		for projID, projData := range projectsMap {
			// Skip archived projects
			if archivedProjects[projID] {
				continue
			}

			if proj, ok := projData.(map[string]interface{}); ok {
				if name, ok := proj["name"].(string); ok {
					s.projectIDs[name] = projID
				}
				// Get deck list for this project (excluding deleted decks)
				if deckList, ok := proj["decks"].([]interface{}); ok {
					for _, deckID := range deckList {
						if did, ok := deckID.(string); ok {
							// Skip deleted decks
							if deletedDecks[did] {
								continue
							}
							projectDecks[projID] = append(projectDecks[projID], did)
							deckToProject[did] = projID
						}
					}
				}
			}
		}
	}

	return nil
}

// GetProjects returns the list of available project names
func (s *Service) GetProjects() []string {
	projects := make([]string, 0, len(s.projectIDs))
	for name := range s.projectIDs {
		projects = append(projects, name)
	}
	return projects
}

// GetProjectFilter returns the current project filter
func (s *Service) GetProjectFilter() string {
	return s.projectFilter
}

// doRequest performs an authenticated request to the Codecks API
func (s *Service) doRequest(ctx context.Context, body interface{}) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.codecks.io/", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Account", s.subdomain)
	req.Header.Set("X-Auth-Token", s.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Codecks API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// GetAssignedTickets fetches all cards from the account
// Note: Codecks API returns cards in a normalized format with card data in a separate "card" object
// If CODECKS_PROJECT is set, only cards from that project's decks are returned
func (s *Service) GetAssignedTickets(ctx context.Context) ([]tickets.Ticket, error) {
	// Get the project ID to filter by (if configured)
	var filterProjectID string
	if s.projectFilter != "" {
		filterProjectID = s.projectIDs[s.projectFilter]
	}

	// If filtering by project, query cards through the project's decks
	// (The Codecks API doesn't return deckId when querying from account.cards)
	if filterProjectID != "" {
		return s.getCardsFromProject(ctx, filterProjectID)
	}

	// No filter - query all cards from account
	return s.getAllCards(ctx)
}

// getAllCards fetches all cards from the account (no project filter)
func (s *Service) getAllCards(ctx context.Context) ([]tickets.Ticket, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"_root": []interface{}{
				map[string]interface{}{
					"account": []interface{}{
						map[string]interface{}{
							"cards": []string{"title", "status", "priority", "content", "accountSeq", "visibility", "deck"},
						},
					},
				},
			},
		},
	}

	respBody, err := s.doRequest(ctx, query)
	if err != nil {
		return nil, err
	}

	var rawResult map[string]interface{}
	if err := json.Unmarshal(respBody, &rawResult); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	cardsMap, ok := rawResult["card"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format: missing 'card' object")
	}

	ticketList := make([]tickets.Ticket, 0)
	for cardID, cardData := range cardsMap {
		cardMap, ok := cardData.(map[string]interface{})
		if !ok {
			continue
		}

		// Skip archived or deleted cards
		visibility := getString(cardMap, "visibility")
		if visibility == "archived" || visibility == "deleted" {
			continue
		}

		status := getString(cardMap, "status")
		if status == "done" {
			continue
		}

		// Get the accountSeq and encode it as Codecks' short ID format
		accountSeq := getInt(cardMap, "accountSeq")
		displayKey := "$" + encodeShortID(accountSeq)

		ticketList = append(ticketList, tickets.Ticket{
			Key:         cardID,
			DisplayKey:  displayKey,
			Summary:     getString(cardMap, "title"),
			Status:      mapCodecksStatus(status),
			Priority:    mapCodecksPriority(getString(cardMap, "priority")),
			Type:        "Card",
			Description: getString(cardMap, "content"),
			DeckID:      getString(cardMap, "deck"),
		})
	}

	return ticketList, nil
}

// getCardsFromProject fetches cards from a specific project's decks
func (s *Service) getCardsFromProject(ctx context.Context, projectID string) ([]tickets.Ticket, error) {
	// Get the decks for this project
	deckIDs := projectDecks[projectID]
	if len(deckIDs) == 0 {
		return []tickets.Ticket{}, nil
	}

	// Query cards from each deck
	ticketList := make([]tickets.Ticket, 0)
	for _, deckID := range deckIDs {
		cards, err := s.getCardsFromDeck(ctx, deckID)
		if err != nil {
			// Skip decks that fail to load
			continue
		}
		ticketList = append(ticketList, cards...)
	}

	return ticketList, nil
}

// getCardsFromDeck fetches cards from a specific deck
func (s *Service) getCardsFromDeck(ctx context.Context, deckID string) ([]tickets.Ticket, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			fmt.Sprintf("deck(%s)", deckID): []interface{}{
				map[string]interface{}{
					"cards": []string{"title", "status", "priority", "content", "accountSeq", "visibility"},
				},
			},
		},
	}

	respBody, err := s.doRequest(ctx, query)
	if err != nil {
		return nil, err
	}

	var rawResult map[string]interface{}
	if err := json.Unmarshal(respBody, &rawResult); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	cardsMap, ok := rawResult["card"].(map[string]interface{})
	if !ok {
		return []tickets.Ticket{}, nil
	}

	ticketList := make([]tickets.Ticket, 0)
	for cardID, cardData := range cardsMap {
		cardMap, ok := cardData.(map[string]interface{})
		if !ok {
			continue
		}

		// Skip archived or deleted cards
		visibility := getString(cardMap, "visibility")
		if visibility == "archived" || visibility == "deleted" {
			continue
		}

		status := getString(cardMap, "status")
		if status == "done" {
			continue
		}

		// Get the accountSeq and encode it as Codecks' short ID format
		accountSeq := getInt(cardMap, "accountSeq")
		displayKey := "$" + encodeShortID(accountSeq)

		ticketList = append(ticketList, tickets.Ticket{
			Key:         cardID,
			DisplayKey:  displayKey,
			Summary:     getString(cardMap, "title"),
			Status:      mapCodecksStatus(status),
			Priority:    mapCodecksPriority(getString(cardMap, "priority")),
			Type:        "Card",
			Description: getString(cardMap, "content"),
			DeckID:      deckID, // Already known from the query
		})
	}

	return ticketList, nil
}

// getString safely extracts a string from a map
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// getInt safely extracts an integer from a map (handles float64 from JSON)
func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

// slugify converts a string to a URL-friendly slug
// Example: "Add Codecks support to jj-tui" -> "add-codecks-support-to-jj-tui"
func slugify(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)
	// Replace spaces with hyphens
	s = strings.ReplaceAll(s, " ", "-")
	// Remove any characters that aren't alphanumeric or hyphens
	invalidChars := regexp.MustCompile(`[^a-z0-9\-]`)
	s = invalidChars.ReplaceAllString(s, "")
	// Remove multiple consecutive hyphens
	multipleHyphens := regexp.MustCompile(`-+`)
	s = multipleHyphens.ReplaceAllString(s, "-")
	// Trim leading/trailing hyphens
	s = strings.Trim(s, "-")
	return s
}

// encodeShortID converts an accountSeq number to Codecks' short ID format
// Codecks uses bijective base-28 encoding with a custom alphabet that excludes
// confusing characters (0, b, d, l, m, n, p, t) to avoid ambiguity and offensive words.
// An offset of 812 is applied to skip the 1-2 digit ID space:
//   - 812 = "zz" in bijective base-28 (largest 2-digit number)
//   - So accountSeq=1 maps to 813 = "111" (first 3-digit number)
//
// This ensures all card short IDs have at least 3 characters.
func encodeShortID(n int) string {
	if n == 0 {
		return ""
	}

	// Skip the 1-2 digit ID space (1-812 = "1" through "zz")
	// All card IDs start at 3 digits ("111" = 813)
	n = n + 812

	// Codecks alphabet: 28 characters (no 0, b, d, l, m, n, p, t)
	alphabet := "123456789acefghijkoqrsuvwxyz"
	base := 28

	var result []byte
	for n > 0 {
		remainder := n % base
		if remainder == 0 {
			// In bijective numeration, remainder 0 means use highest digit
			// and decrement the quotient
			remainder = base
			n = n/base - 1
		} else {
			n = n / base
		}
		result = append([]byte{alphabet[remainder-1]}, result...)
	}
	return string(result)
}

// mapCodecksStatus maps Codecks status to a readable string
func mapCodecksStatus(status string) string {
	switch status {
	case "not_started":
		return "Not Started"
	case "started":
		return "In Progress"
	case "done":
		return "Done"
	case "blocked":
		return "Blocked"
	default:
		return status
	}
}

// mapCodecksPriority maps Codecks priority codes to readable strings
func mapCodecksPriority(priority string) string {
	switch priority {
	case "a":
		return "Highest"
	case "b":
		return "High"
	case "c":
		return "Medium"
	case "d":
		return "Low"
	case "e":
		return "Lowest"
	default:
		return priority
	}
}

// GetTicket fetches a single card by ID (can be short or full ID)
func (s *Service) GetTicket(ctx context.Context, key string) (*tickets.Ticket, error) {
	// If it's a short ID, we need to find the full ID first
	// For now, we only support full IDs in GetTicket
	query := map[string]interface{}{
		"query": map[string]interface{}{
			fmt.Sprintf("card(%s)", key): []string{
				"title", "content", "status", "priority", "accountSeq", "visibility", "deck",
			},
		},
	}

	respBody, err := s.doRequest(ctx, query)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Response is normalized: card data is in result["card"][key]
	cardsMap, ok := result["card"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("card %s not found", key)
	}

	cardData, ok := cardsMap[key].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("card %s not found", key)
	}

	// Check if card is archived or deleted
	visibility := getString(cardData, "visibility")
	if visibility == "archived" || visibility == "deleted" {
		return nil, fmt.Errorf("card %s is archived", key)
	}

	// Get the accountSeq and encode it as Codecks' short ID format
	accountSeq := getInt(cardData, "accountSeq")
	displayKey := "$" + encodeShortID(accountSeq)

	return &tickets.Ticket{
		Key:         key, // Full GUID used for URLs
		DisplayKey:  displayKey,
		Summary:     getString(cardData, "title"),
		Status:      mapCodecksStatus(getString(cardData, "status")),
		Priority:    mapCodecksPriority(getString(cardData, "priority")),
		Type:        "Card",
		Description: getString(cardData, "content"),
		DeckID:      getString(cardData, "deck"),
	}, nil
}

// GetTicketURL returns the browser URL for a card
// URL format: https://{subdomain}.codecks.io/decks/{deckSeq}-{deckSlug}/card/{shortId}-{cardSlug}
func (s *Service) GetTicketURL(ticket tickets.Ticket) string {
	// Get deck metadata for URL construction
	deckSlug := ""
	if meta, ok := deckMetadata[ticket.DeckID]; ok && meta.seq > 0 {
		deckSlug = fmt.Sprintf("%d-%s", meta.seq, slugify(meta.title))
	}

	// Get the short ID without the "$" prefix
	shortID := strings.TrimPrefix(ticket.DisplayKey, "$")
	cardSlug := fmt.Sprintf("%s-%s", shortID, slugify(ticket.Summary))

	// If we have deck info, use the full URL; otherwise fallback to simple URL
	if deckSlug != "" {
		return fmt.Sprintf("https://%s.codecks.io/decks/%s/card/%s", s.subdomain, deckSlug, cardSlug)
	}
	return fmt.Sprintf("https://%s.codecks.io/card/%s", s.subdomain, cardSlug)
}

// GetProviderName returns the name of this provider
func (s *Service) GetProviderName() string {
	return "Codecks"
}

// GetSubdomain returns the Codecks subdomain
func (s *Service) GetSubdomain() string {
	return s.subdomain
}

// IsConfigured returns true if Codecks environment variables are set
func IsConfigured() bool {
	return os.Getenv("CODECKS_SUBDOMAIN") != "" &&
		os.Getenv("CODECKS_TOKEN") != ""
}
