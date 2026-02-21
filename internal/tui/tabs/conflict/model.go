package conflict

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal"
)

// Model represents the bookmark conflict resolution view
type Model struct {
	shown          bool
	bookmarkName   string
	localCommitID  string
	remoteCommitID string
	localSummary   string
	remoteSummary  string
	selectedOption int // 0=Keep Local, 1=Reset to Remote
	statusMessage  string
}

// NewModel creates a new Conflict model
func NewModel() Model {
	return Model{
		shown:          false,
		selectedOption: 0,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the Conflict modal
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.shown {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}
	return m, nil
}

// View renders the Conflict modal
func (m Model) View() string {
	if !m.shown {
		return ""
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("3")).
		Padding(1, 2).
		MaxWidth(70)

	content := "Bookmark Conflict: " + m.bookmarkName + "\n\n"
	content += "Local: " + m.localSummary + "\n"
	content += "Remote: " + m.remoteSummary + "\n\n"
	content += "Options:\n"

	localStyle := lipgloss.NewStyle()
	remoteStyle := lipgloss.NewStyle()
	if m.selectedOption == 0 {
		localStyle = localStyle.Bold(true).Foreground(lipgloss.Color("2"))
	}
	if m.selectedOption == 1 {
		remoteStyle = remoteStyle.Bold(true).Foreground(lipgloss.Color("2"))
	}

	content += localStyle.Render("> Keep Local") + "\n"
	content += remoteStyle.Render("> Reset to Remote") + "\n\n"
	content += "(Use j/k to select, enter to apply, esc to cancel)"

	return style.Render(content)
}

// handleKeyMsg handles keyboard input
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.shown = false
		return m, nil
	case "enter":
		// Signal to resolve - handled outside
		return m, nil
	case "j", "down":
		if m.selectedOption < 1 {
			m.selectedOption++
		}
		return m, nil
	case "k", "up":
		if m.selectedOption > 0 {
			m.selectedOption--
		}
		return m, nil
	}
	return m, nil
}

// Accessors

// IsShown returns whether the modal is displayed
func (m *Model) IsShown() bool {
	return m.shown
}

// Show displays the conflict modal
func (m *Model) Show(bookmarkName, localID, remoteID, localSummary, remoteSummary string) {
	m.shown = true
	m.bookmarkName = bookmarkName
	m.localCommitID = localID
	m.remoteCommitID = remoteID
	m.localSummary = localSummary
	m.remoteSummary = remoteSummary
	m.selectedOption = 0
}

// Hide hides the modal
func (m *Model) Hide() {
	m.shown = false
}

// GetSelectedOption returns "keep_local" or "reset_remote"
func (m *Model) GetSelectedOption() string {
	if m.selectedOption == 0 {
		return "keep_local"
	}
	return "reset_remote"
}

// GetBookmarkName returns the conflicted bookmark name
func (m *Model) GetBookmarkName() string {
	return m.bookmarkName
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
	// Conflict modal doesn't use repository directly
}
