package bookmark

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
)

// Model represents the bookmark creation dialog
type Model struct {
	shown               bool
	nameInput           textinput.Model
	commitIdx           int      // Index of commit to create bookmark on
	existingBookmarks   []string // List of existing bookmarks
	selectedBookmarkIdx int      // Index of selected existing bookmark (-1 for new)
	fromJira            bool     // True if creating bookmark from Jira ticket
	jiraTicketKey       string   // Jira ticket key if creating from Jira
	jiraTicketTitle     string   // Jira ticket summary if creating from Jira
	ticketDisplayKey    string   // Short display key (e.g., "$12u" for Codecks)
	bookmarkNameExists  bool     // True if entered name matches an existing bookmark
	statusMessage       string
}

// NewModel creates a new Bookmark model
func NewModel() Model {
	nameInput := textinput.New()
	nameInput.Placeholder = "bookmark-name"
	nameInput.CharLimit = 100
	nameInput.Width = 50
	nameInput.Focus()

	return Model{
		shown:               false,
		nameInput:           nameInput,
		commitIdx:           -1,
		selectedBookmarkIdx: -1,
		fromJira:            false,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the Bookmark creation view
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.shown {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

// View renders the Bookmark creation dialog
func (m Model) View() string {
	if !m.shown {
		return ""
	}
	return "" // Rendering handled by parent for now
}

// handleKeyMsg handles keyboard input
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.shown = false
		m.nameInput.SetValue("")
		return m, nil
	case "enter":
		// Create bookmark - handled outside
		return m, nil
	}
	return m, nil
}

// Accessors

// IsShown returns whether the dialog is displayed
func (m *Model) IsShown() bool {
	return m.shown
}

// Show displays the bookmark creation dialog
func (m *Model) Show(commitIdx int, existingBookmarks []string) {
	m.shown = true
	m.commitIdx = commitIdx
	m.existingBookmarks = existingBookmarks
	m.selectedBookmarkIdx = -1
	m.nameInput.SetValue("")
	m.nameInput.Focus()
	m.bookmarkNameExists = false
}

// Hide hides the dialog
func (m *Model) Hide() {
	m.shown = false
	m.nameInput.SetValue("")
}

// GetBookmarkName returns the entered bookmark name
func (m *Model) GetBookmarkName() string {
	return m.nameInput.Value()
}

// SetBookmarkName sets the bookmark name
func (m *Model) SetBookmarkName(name string) {
	m.nameInput.SetValue(name)
}

// GetCommitIdx returns the commit index
func (m *Model) GetCommitIdx() int {
	return m.commitIdx
}

// SetFromJira sets the jira context
func (m *Model) SetFromJira(ticketKey, ticketTitle, displayKey string) {
	m.fromJira = true
	m.jiraTicketKey = ticketKey
	m.jiraTicketTitle = ticketTitle
	m.ticketDisplayKey = displayKey
}

// ClearJiraContext clears the jira context
func (m *Model) ClearJiraContext() {
	m.fromJira = false
	m.jiraTicketKey = ""
	m.jiraTicketTitle = ""
	m.ticketDisplayKey = ""
}

// IsFromJira returns whether creating from Jira ticket
func (m *Model) IsFromJira() bool {
	return m.fromJira
}

// GetJiraKey returns the Jira ticket key
func (m *Model) GetJiraKey() string {
	return m.jiraTicketKey
}

// SetNameExists sets whether the name already exists
func (m *Model) SetNameExists(exists bool) {
	m.bookmarkNameExists = exists
}

// NameExists returns whether the name already exists
func (m *Model) NameExists() bool {
	return m.bookmarkNameExists
}

// GetNameInput returns the name input field
func (m *Model) GetNameInput() *textinput.Model {
	return &m.nameInput
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
	// Bookmark model doesn't use repository directly
}
