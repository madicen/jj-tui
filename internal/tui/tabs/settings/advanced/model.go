package advanced

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
)

// Model represents the Advanced settings sub-tab
type Model struct {
	branchLimitInput  textinput.Model
	sanitizeBookmarks bool
	ticketProvider    string // explicit: "jira", "codecks", "github_issues", or ""
	autoInProgress    bool
	confirmingCleanup string // "" = not confirming, "delete_bookmarks", "abandon_old_commits"
	focusedField      int
	statusMessage     string
}

// NewModel creates a new Advanced settings model
func NewModel() Model {
	branchLimitInput := textinput.New()
	branchLimitInput.Placeholder = "50"
	branchLimitInput.CharLimit = 3
	branchLimitInput.Width = 10
	branchLimitInput.SetValue("50")
	branchLimitInput.Focus()

	return Model{
		branchLimitInput:  branchLimitInput,
		sanitizeBookmarks: true,
		autoInProgress:    true,
		focusedField:      0,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	var cmd tea.Cmd
	m.branchLimitInput, cmd = m.branchLimitInput.Update(msg)
	return m, cmd
}

// View renders the model
func (m Model) View() string {
	return "" // Rendered by parent
}

// handleKeyMsg handles keyboard input
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.focusedField < 4 {
			if m.focusedField == 0 {
				m.branchLimitInput.Blur()
			}
			m.focusedField++
		}
		return m, nil
	case "k", "up":
		if m.focusedField > 0 {
			m.focusedField--
			if m.focusedField == 0 {
				m.branchLimitInput.Focus()
			}
		}
		return m, nil
	case " ":
		// Toggle boolean options
		if m.focusedField == 1 {
			m.sanitizeBookmarks = !m.sanitizeBookmarks
		} else if m.focusedField == 2 {
			m.autoInProgress = !m.autoInProgress
		}
		return m, nil
	}
	return m, nil
}

// Accessors

// GetBranchLimit returns the branch limit
func (m *Model) GetBranchLimit() string {
	return m.branchLimitInput.Value()
}

// SetBranchLimit sets the branch limit
func (m *Model) SetBranchLimit(limit string) {
	m.branchLimitInput.SetValue(limit)
}

// GetSanitizeBookmarks returns whether to sanitize bookmark names
func (m *Model) GetSanitizeBookmarks() bool {
	return m.sanitizeBookmarks
}

// SetSanitizeBookmarks sets whether to sanitize bookmark names
func (m *Model) SetSanitizeBookmarks(sanitize bool) {
	m.sanitizeBookmarks = sanitize
}

// GetTicketProvider returns the ticket provider
func (m *Model) GetTicketProvider() string {
	return m.ticketProvider
}

// SetTicketProvider sets the ticket provider
func (m *Model) SetTicketProvider(provider string) {
	m.ticketProvider = provider
}

// GetAutoInProgress returns whether to auto-set ticket to "In Progress"
func (m *Model) GetAutoInProgress() bool {
	return m.autoInProgress
}

// SetAutoInProgress sets whether to auto-set ticket to "In Progress"
func (m *Model) SetAutoInProgress(auto bool) {
	m.autoInProgress = auto
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
	// Advanced settings don't depend on repository
}
