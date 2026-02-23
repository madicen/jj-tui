package prform

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

// Model represents the PR creation dialog
type Model struct {
	zoneManager       *zone.Manager
	shown             bool
	titleInput        textinput.Model
	bodyInput         textarea.Model
	baseBranch        string
	headBranch        string
	focusedField      int  // 0=title, 1=body
	commitIndex       int  // Index of commit PR is being created from
	needsMoveBookmark bool // True if we need to move the bookmark to include all commits
}

// NewModel creates a new PR creation model. zoneManager may be nil (zones will be omitted).
func NewModel(zoneManager *zone.Manager) Model {
	titleInput := textinput.New()
	titleInput.Placeholder = "Pull request title"
	titleInput.CharLimit = 200
	titleInput.Width = 60

	bodyInput := textarea.New()
	bodyInput.Placeholder = "Describe your changes..."
	bodyInput.ShowLineNumbers = false
	bodyInput.SetWidth(60)
	bodyInput.SetHeight(8)

	return Model{
		zoneManager:  zoneManager,
		shown:        false,
		titleInput:   titleInput,
		bodyInput:    bodyInput,
		baseBranch:   "main",
		focusedField: 0,
		commitIndex:  -1,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the PR creation view
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.shown {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	if m.focusedField == 0 {
		var cmd tea.Cmd
		m.titleInput, cmd = m.titleInput.Update(msg)
		return m, cmd
	} else {
		var cmd tea.Cmd
		m.bodyInput, cmd = m.bodyInput.Update(msg)
		return m, cmd
	}
}

// View renders the PR creation dialog
func (m Model) View() string {
	if !m.shown {
		return ""
	}
	return m.renderForm()
}

// renderForm builds the Create PR form UI (title, branch info, inputs, buttons)
func (m Model) renderForm() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8BE9FD"))
	subtitleStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	buttonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#30363d")).
		Padding(0, 1).
		Bold(true)

	mark := func(id, s string) string {
		if m.zoneManager == nil {
			return s
		}
		return m.zoneManager.Mark(id, s)
	}

	branchLine := subtitleStyle.Render(fmt.Sprintf("Branch: %s → %s", m.baseBranch, m.headBranch))
	titleInput := mark(mouse.ZonePRTitle, m.titleInput.View())
	bodyInput := mark(mouse.ZonePRBody, m.bodyInput.View())
	submitBtn := mark(mouse.ZonePRSubmit, buttonStyle.Render("Create PR (Ctrl+S)"))
	cancelBtn := mark(mouse.ZonePRCancel, buttonStyle.Render("Cancel (Esc)"))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render("Create Pull Request"),
		branchLine,
		"",
		"Title:",
		titleInput,
		"",
		"Body:",
		bodyInput,
		"",
		lipgloss.JoinHorizontal(lipgloss.Left, submitBtn, "  ", cancelBtn),
	)
}

// handleKeyMsg handles keyboard input: special keys (esc, tab, ctrl+enter) here; all others go to the focused field.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.shown = false
		m.Reset()
		return m, nil
	case "tab":
		// Switch between title and body
		m.focusedField = (m.focusedField + 1) % 2
		if m.focusedField == 0 {
			m.titleInput.Focus()
			m.bodyInput.Blur()
		} else {
			m.titleInput.Blur()
			m.bodyInput.Focus()
		}
		return m, nil
	case "ctrl+enter":
		// Create PR - handled outside
		return m, nil
	}
	// Forward typing and other keys to the focused input
	if m.focusedField == 0 {
		var cmd tea.Cmd
		m.titleInput, cmd = m.titleInput.Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	m.bodyInput, cmd = m.bodyInput.Update(msg)
	return m, cmd
}

// Accessors

// IsShown returns whether the dialog is displayed
func (m *Model) IsShown() bool {
	return m.shown
}

// Show displays the PR creation dialog
func (m *Model) Show(commitIndex int, baseBranch, headBranch string) {
	m.shown = true
	m.commitIndex = commitIndex
	m.baseBranch = baseBranch
	m.headBranch = headBranch
	m.focusedField = 0
	m.titleInput.Focus()
	m.bodyInput.Blur()
	m.Reset()
}

// Hide hides the dialog
func (m *Model) Hide() {
	m.shown = false
	m.Reset()
}

// Reset clears the form
func (m *Model) Reset() {
	m.titleInput.SetValue("")
	m.bodyInput.SetValue("")
	m.focusedField = 0
	m.needsMoveBookmark = false
}

// GetTitle returns the PR title
func (m *Model) GetTitle() string {
	return m.titleInput.Value()
}

// SetTitle sets the PR title
func (m *Model) SetTitle(title string) {
	m.titleInput.SetValue(title)
}

// GetBody returns the PR body
func (m *Model) GetBody() string {
	return m.bodyInput.Value()
}

// SetBody sets the PR body
func (m *Model) SetBody(body string) {
	m.bodyInput.SetValue(body)
}

// GetBaseBranch returns the base branch
func (m *Model) GetBaseBranch() string {
	return m.baseBranch
}

// SetBaseBranch sets the base branch
func (m *Model) SetBaseBranch(branch string) {
	m.baseBranch = branch
}

// GetHeadBranch returns the head branch
func (m *Model) GetHeadBranch() string {
	return m.headBranch
}

// SetHeadBranch sets the head branch
func (m *Model) SetHeadBranch(branch string) {
	m.headBranch = branch
}

// GetCommitIndex returns the commit index
func (m *Model) GetCommitIndex() int {
	return m.commitIndex
}

// GetFocusedField returns the focused field (0=title, 1=body)
func (m *Model) GetFocusedField() int {
	return m.focusedField
}

// SetFocusedField sets the focused field and syncs Focus/Blur on inputs
func (m *Model) SetFocusedField(i int) {
	if i != 0 && i != 1 {
		return
	}
	m.focusedField = i
	if i == 0 {
		m.titleInput.Focus()
		m.bodyInput.Blur()
	} else {
		m.titleInput.Blur()
		m.bodyInput.Focus()
	}
}

// SetNeedsMoveBookmark sets whether bookmark needs to be moved
func (m *Model) SetNeedsMoveBookmark(needs bool) {
	m.needsMoveBookmark = needs
}

// NeedsMoveBookmark returns whether bookmark needs to be moved
func (m *Model) NeedsMoveBookmark() bool {
	return m.needsMoveBookmark
}

// GetTitleInput returns the title input field
func (m *Model) GetTitleInput() *textinput.Model {
	return &m.titleInput
}

// GetBodyInput returns the body textarea field
func (m *Model) GetBodyInput() *textarea.Model {
	return &m.bodyInput
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
	// PR creation model doesn't use repository directly
}
