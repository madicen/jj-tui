package bookmark

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/styles"
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
	jiraBookmarkTitles  map[string]string // Maps bookmark names to formatted PR titles ("KEY - Title")
	ticketBookmarkDisplayKeys map[string]string // Maps bookmark names to ticket short IDs for commit messages
	repository          *internal.Repository
	zoneManager         *zone.Manager
}

// NewModel creates a new Bookmark model. zoneManager may be nil.
func NewModel(zoneManager *zone.Manager) Model {
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
		zoneManager:         zoneManager,
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
	return m.renderBookmark()
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

// GetExistingBookmarks returns the list of existing bookmarks (for move)
func (m *Model) GetExistingBookmarks() []string {
	return m.existingBookmarks
}

// GetSelectedBookmarkIdx returns the selected existing bookmark index (-1 for new)
func (m *Model) GetSelectedBookmarkIdx() int {
	return m.selectedBookmarkIdx
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

// GetJiraTicketTitle returns the Jira ticket summary
func (m *Model) GetJiraTicketTitle() string {
	return m.jiraTicketTitle
}

// GetTicketDisplayKey returns the short display key for commit messages
func (m *Model) GetTicketDisplayKey() string {
	return m.ticketDisplayKey
}

// SetNameExists sets whether the name already exists
func (m *Model) SetNameExists(exists bool) {
	m.bookmarkNameExists = exists
}

// SetExistingBookmarks sets the list of existing bookmarks (synced from main model)
func (m *Model) SetExistingBookmarks(bookmarks []string) {
	m.existingBookmarks = bookmarks
}

// SetCommitIdx sets the commit index for the bookmark target
func (m *Model) SetCommitIdx(idx int) {
	m.commitIdx = idx
}

// SetSelectedBookmarkIdx sets the selected existing bookmark index
func (m *Model) SetSelectedBookmarkIdx(idx int) {
	m.selectedBookmarkIdx = idx
}

// NameExists returns whether the name already exists
func (m *Model) NameExists() bool {
	return m.bookmarkNameExists
}

// GetNameInput returns the name input field
func (m *Model) GetNameInput() *textinput.Model {
	return &m.nameInput
}

// UpdateRepository updates the repository (for rendering commit target)
func (m *Model) UpdateRepository(repo *internal.Repository) {
	m.repository = repo
}

// JiraBookmarkTitles / TicketBookmarkDisplayKeys (for PR title formatting from bookmarks)
func (m *Model) GetJiraBookmarkTitles() map[string]string { return m.jiraBookmarkTitles }
func (m *Model) SetJiraBookmarkTitles(mp map[string]string) {
	if mp != nil {
		m.jiraBookmarkTitles = mp
	} else {
		m.jiraBookmarkTitles = make(map[string]string)
	}
}
func (m *Model) GetTicketBookmarkDisplayKeys() map[string]string { return m.ticketBookmarkDisplayKeys }
func (m *Model) SetTicketBookmarkDisplayKeys(mp map[string]string) {
	if mp != nil {
		m.ticketBookmarkDisplayKeys = mp
	} else {
		m.ticketBookmarkDisplayKeys = make(map[string]string)
	}
}

// SetZoneManager sets the zone manager for clickable elements
func (m *Model) SetZoneManager(z *zone.Manager) {
	m.zoneManager = z
}

func mark(z *zone.Manager, id, content string) string {
	if z == nil {
		return content
	}
	return z.Mark(id, content)
}

func (m Model) renderBookmark() string {
	var lines []string
	if m.fromJira {
		jiraBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.ColorPrimary).
			Padding(0, 1).
			Render(fmt.Sprintf("Jira Ticket: %s\n\nThis will create a new branch from main with the bookmark name below.",
				lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render(m.jiraTicketKey),
			))
		lines = append(lines, jiraBox)
		lines = append(lines, "")
	} else {
		if m.repository != nil && m.commitIdx >= 0 && m.commitIdx < len(m.repository.Graph.Commits) {
			commit := m.repository.Graph.Commits[m.commitIdx]
			commitBox := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(styles.ColorPrimary).
				Padding(0, 1).
				Render(fmt.Sprintf("Target: %s\n%s",
					lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render(commit.ShortID),
					commit.Summary,
				))
			lines = append(lines, commitBox)
			lines = append(lines, "")
		}
		if len(m.existingBookmarks) > 0 {
			lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Move Existing Bookmark:"))
			lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Click or use j/k to select, Enter to move"))
			lines = append(lines, "")
			for i, bookmark := range m.existingBookmarks {
				prefix := "  "
				style := styles.CommitStyle
				if i == m.selectedBookmarkIdx {
					prefix = "► "
					style = styles.CommitSelectedStyle
				}
				bookmarkLine := fmt.Sprintf("%s%s", prefix, bookmark)
				lines = append(lines, mark(m.zoneManager, mouse.ZoneExistingBookmark(i), style.Render(bookmarkLine)))
			}
			lines = append(lines, "")
			lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("─────────────────────────────────"))
			lines = append(lines, "")
		}
	}
	newStyle := lipgloss.NewStyle().Bold(true)
	if m.selectedBookmarkIdx == -1 || m.fromJira {
		newStyle = newStyle.Foreground(styles.ColorPrimary)
	}
	if m.fromJira {
		lines = append(lines, newStyle.Render("Branch/Bookmark Name:"))
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Edit if needed, then press Enter to create"))
	} else {
		lines = append(lines, newStyle.Render("Create New Bookmark:"))
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Type a name and press Enter"))
	}
	lines = append(lines, "")
	inputStyle := lipgloss.NewStyle()
	if m.selectedBookmarkIdx == -1 || m.fromJira {
		inputStyle = inputStyle.Foreground(styles.ColorPrimary)
	}
	lines = append(lines, inputStyle.Render("Name:"))
	lines = append(lines, mark(m.zoneManager, mouse.ZoneBookmarkName, "  "+m.nameInput.View()))
	if m.bookmarkNameExists {
		warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E3B341")).Bold(true)
		lines = append(lines, "")
		lines = append(lines, warningStyle.Render("⚠ A bookmark with this name already exists"))
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("  Creating will move the existing bookmark to this commit"))
	}
	lines = append(lines, "")
	var submitLabel string
	if m.fromJira {
		submitLabel = "Create Branch (Enter)"
	} else if m.selectedBookmarkIdx >= 0 && m.selectedBookmarkIdx < len(m.existingBookmarks) {
		submitLabel = fmt.Sprintf("Move '%s' (Enter)", m.existingBookmarks[m.selectedBookmarkIdx])
	} else {
		submitLabel = "Create (Enter)"
	}
	submitButton := mark(m.zoneManager, mouse.ZoneBookmarkSubmit, styles.ButtonStyle.Render(submitLabel))
	cancelButton := mark(m.zoneManager, mouse.ZoneBookmarkCancel, styles.ButtonStyle.Render("Cancel (Esc)"))
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, submitButton, " ", cancelButton))
	return strings.Join(lines, "\n")
}
