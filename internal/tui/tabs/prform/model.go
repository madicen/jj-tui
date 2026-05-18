package prform

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/tui/genmenu"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/state"
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
	// Long-press AI profile picker over the Generate chip; same structure used in
	// the descedit, bookmark, and ticketform modals.
	genMenu       genmenu.State
	profiles      []config.AIProfile
	activeProfile string
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
	// Handle request messages (main forwards these to us)
	switch msg.(type) {
	case CancelRequestedMsg:
		m.shown = false
		m.Reset()
		m.genMenu.Reset()
		return m, state.NavigateTarget{Kind: state.NavigateBackFromPRForm, StatusMessage: "PR creation cancelled"}.Cmd()
	case SubmitRequestedMsg:
		return m, state.NavigateTarget{Kind: state.NavigateSubmitPR}.Cmd()
	}
	switch msg := msg.(type) {
	case genmenu.TickMsg:
		m.genMenu.OpenIfMatches(msg)
		return m, nil
	case tea.MouseMsg:
		return m.handleMouseForMenu(msg)
	case zone.MsgZoneInBounds:
		if m.genMenu.IsShown() {
			return m, nil
		}
		m.genMenu.CancelPress()
		if m.zoneManager != nil {
			if zoneID := m.resolveClickedZone(msg); zoneID != "" {
				return m.handleZoneClick(zoneID)
			}
		}
		return m, nil
	case tea.KeyMsg:
		if m.genMenu.IsShown() && msg.String() == "esc" {
			m.genMenu.Close()
			return m, nil
		}
		return m.handleKeyMsg(msg)
	}

	if m.focusedField == 0 {
		var cmd tea.Cmd
		m.titleInput, cmd = m.titleInput.Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	m.bodyInput, cmd = m.bodyInput.Update(msg)
	return m, cmd
}

// handleMouseForMenu drives the long-press AI profile picker for the Generate chip.
// See descedit/model.go for the shared design notes.
func (m Model) handleMouseForMenu(msg tea.MouseMsg) (Model, tea.Cmd) {
	if m.zoneManager == nil {
		return m, nil
	}
	if m.genMenu.IsShown() {
		switch msg.Action {
		case tea.MouseActionMotion, tea.MouseActionPress:
			m.genMenu.UpdateHover(m.zoneManager, msg, len(m.profiles))
			return m, nil
		case tea.MouseActionRelease:
			if msg.Button != tea.MouseButtonLeft {
				return m, nil
			}
			idx := m.genMenu.HitTestRelease(m.zoneManager, msg, len(m.profiles))
			if idx >= 0 && idx < len(m.profiles) {
				return m, state.NavigateTarget{
					Kind:              state.NavigateGeneratePRForm,
					AIOverrideProfile: m.profiles[idx].Name,
				}.Cmd()
			}
			return m, nil
		}
		return m, nil
	}
	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button != tea.MouseButtonLeft {
			return m, nil
		}
		z := m.zoneManager.Get(mouse.ZonePRGenerate)
		if z != nil && z.InBounds(msg) && len(m.profiles) > 0 {
			return m, m.genMenu.BeginPress(mouse.ZonePRGenerate, msg)
		}
	case tea.MouseActionMotion:
		m.genMenu.OnMotion(m.zoneManager, msg)
	}
	return m, nil
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

	contentW := m.bodyInput.Width()
	if contentW < 12 {
		contentW = 60
	}
	genChip := mark(mouse.ZonePRGenerate, styles.AIGenerateChip())
	headerRow := styles.SpreadRow(contentW, titleStyle.Render("Create Pull Request"), genChip)

	branchLine := subtitleStyle.Render(fmt.Sprintf("Branch: %s → %s", m.baseBranch, m.headBranch))
	titleInput := mark(mouse.ZonePRTitle, m.titleInput.View())
	bodyInput := mark(mouse.ZonePRBody, m.bodyInput.View())
	submitBtn := mark(mouse.ZonePRSubmit, buttonStyle.Render("Create PR (Ctrl+S)"))
	cancelBtn := mark(mouse.ZonePRCancel, buttonStyle.Render("Cancel (Esc)"))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		headerRow,
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

// handleKeyMsg handles keyboard input; returns request cmds for main to handle cancel/submit.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, CancelRequestedCmd()
	case "ctrl+g":
		return m, state.NavigateTarget{Kind: state.NavigateGeneratePRForm}.Cmd()
	case "ctrl+s", "ctrl+enter":
		return m, SubmitRequestedCmd()
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

// ZoneIDs returns the zone IDs this modal uses when rendering. Used to resolve clicks.
func (m Model) ZoneIDs() []string {
	return []string{mouse.ZonePRTitle, mouse.ZonePRBody, mouse.ZonePRSubmit, mouse.ZonePRGenerate, mouse.ZonePRCancel}
}

func (m Model) resolveClickedZone(msg zone.MsgZoneInBounds) string {
	if m.zoneManager == nil || msg.Zone == nil {
		return ""
	}
	// bubblezone's AnyInBoundsAndUpdate calls Update once per overlapping zone (sorted by id).
	// Use the zone from this message so each dispatch targets the correct control; do not
	// re-scan "first InBounds" or every overlapping hit would act like the topmost zone.
	for _, id := range m.ZoneIDs() {
		if z := m.zoneManager.Get(id); z != nil && z == msg.Zone {
			return id
		}
	}
	return ""
}

// handleZoneClick handles a zone click by zone ID (called from Update after resolve). Returns (updated model, cmd).
func (m Model) handleZoneClick(zoneID string) (Model, tea.Cmd) {
	switch zoneID {
	case mouse.ZonePRTitle:
		m.SetFocusedField(0)
		return m, nil
	case mouse.ZonePRBody:
		m.SetFocusedField(1)
		return m, nil
	case mouse.ZonePRSubmit:
		return m, SubmitRequestedCmd()
	case mouse.ZonePRGenerate:
		return m, state.NavigateTarget{Kind: state.NavigateGeneratePRForm}.Cmd()
	case mouse.ZonePRCancel:
		return m, CancelRequestedCmd()
	}
	return m, nil
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

// SetAIProfiles updates the profile list shown by the long-press menu and the
// active profile mark. See descedit/model.go SetAIProfiles for design notes.
func (m *Model) SetAIProfiles(profiles []config.AIProfile, activeProfile string) {
	m.profiles = profiles
	m.activeProfile = activeProfile
}

// MenuState returns a pointer to the long-press menu state so main can render
// the popover overlay or check IsShown when laying out the view.
func (m *Model) MenuState() *genmenu.State {
	return &m.genMenu
}

// MenuOverlay returns the rendered popover (empty when hidden) and its (x, y) anchor.
func (m *Model) MenuOverlay() (string, int, int) {
	if !m.genMenu.IsShown() {
		return "", 0, 0
	}
	view := genmenu.Render(m.zoneManager, m.profiles, m.activeProfile, m.genMenu.HoverIndex())
	x, y := m.genMenu.MouseAnchor()
	return view, x, y
}
