// Package view provides rendering functions for the TUI views.
package view

import (
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen-utilities/jj-tui/v2/internal/models"
)

// Renderer provides rendering capabilities with zone support
type Renderer struct {
	Zone *zone.Manager
}

// New creates a new Renderer
func New(z *zone.Manager) *Renderer {
	return &Renderer{Zone: z}
}

// GraphData contains data needed for commit graph rendering
type GraphData struct {
	Repository     *models.Repository
	SelectedCommit int
}

// PRData contains data needed for PR rendering
type PRData struct {
	Repository    *models.Repository
	SelectedPR    int
	GithubService bool // whether GitHub is connected
}

// JiraData contains data needed for Jira rendering
type JiraData struct {
	Tickets        []JiraTicket
	SelectedTicket int
	JiraService    bool // whether Jira is connected
}

// JiraTicket represents a Jira ticket for rendering
type JiraTicket struct {
	Key         string
	Summary     string
	Status      string
	Type        string
	Priority    string
	Description string
}

// SettingsData contains data needed for settings rendering
type SettingsData struct {
	Inputs        []InputView
	FocusedField  int
	GithubService bool
	JiraService   bool
}

// InputView represents a text input for rendering
type InputView struct {
	View string
}

// DescriptionData contains data needed for description editing
type DescriptionData struct {
	Repository      *models.Repository
	EditingCommitID string
	InputView       string
}

// CreatePRData contains data needed for create PR view
type CreatePRData struct {
	Repository     *models.Repository
	SelectedCommit int
	GithubService  bool
}
