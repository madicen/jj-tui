package error

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal"
)

// Model represents the error display modal
type Model struct {
	err           error
	isJJRepoError bool   // true if "not a jj repository"
	currentPath   string // Path where jj init would be run
	copied        bool   // true if error was just copied to clipboard
	repository    *internal.Repository
	statusMessage string
}

// NewModel creates a new Error model
func NewModel() Model {
	return Model{
		isJJRepoError: false,
		copied:        false,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the Error modal
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}
	return m, nil
}

// View renders the Error modal
func (m Model) View() string {
	if m.err == nil {
		return ""
	}

	if m.isJJRepoError {
		// Welcome screen for "not a jj repo"
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2)

		message := fmt.Sprintf(
			"Not a jj repository\n\nPress 'i' to initialize, 'esc' to dismiss\nPath: %s",
			m.currentPath,
		)
		return style.Render(message)
	}

	// Regular error modal - return content that should be centered by parent
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("1")).
		Padding(1, 2).
		MaxWidth(70)

	var buttons string
	if m.copied {
		buttons = " ✓ Copied to clipboard"
	}
	message := fmt.Sprintf("Error: %v\n\nPress 'esc' to dismiss, 'ctrl+r' to retry, 'c' to copy%s", m.err, buttons)
	return style.Render(message)
}

// handleKeyMsg handles keyboard input for the error modal
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+q", "ctrl+c":
		// These should be handled at a higher level (app should quit)
		return m, nil
	case "ctrl+r":
		// Clear error and trigger refresh - signal should be handled outside
		m.err = nil
		m.copied = false
		return m, nil
	case "esc":
		// Dismiss error
		m.err = nil
		m.copied = false
		return m, nil
	case "c":
		// Copy error to clipboard (not for welcome screen)
		if !m.isJJRepoError && m.err != nil {
			// This would be copyToClipboard action - handled outside
			m.copied = true
			return m, nil
		}
	case "i":
		// Initialize jj repo if in welcome screen
		if m.isJJRepoError {
			// This would be runJJInit action - handled outside
			return m, nil
		}
	}
	return m, nil
}

// Accessors

// GetError returns the current error
func (m *Model) GetError() error {
	return m.err
}

// SetError sets the current error
func (m *Model) SetError(err error, isJJRepo bool, path string) {
	m.err = err
	m.isJJRepoError = isJJRepo
	m.currentPath = path
	m.copied = false
}

// ClearError clears the error
func (m *Model) ClearError() {
	m.err = nil
	m.copied = false
}

// IsCopied returns whether the error was just copied
func (m *Model) IsCopied() bool {
	return m.copied
}

// IsJJRepoError returns true if this is a "not a jj repo" error
func (m *Model) IsJJRepoError() bool {
	return m.isJJRepoError
}

// GetCurrentPath returns the current path
func (m *Model) GetCurrentPath() string {
	return m.currentPath
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
	m.repository = repo
}
