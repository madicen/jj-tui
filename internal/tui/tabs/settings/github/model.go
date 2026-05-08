package github

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
)

// Model represents the GitHub settings sub-tab.
//
// focusedField semantics on the GitHub panel:
//
//	0 — token text input
//	1 — Only My PRs toggle
//	2 — Show Merged PRs toggle
//	3 — Show Closed PRs toggle
//	4 — (PR Limit / Refresh row; doesn't accept text input)
//	5 — origin URL text input (Repository remote section)
//
// The token input is mirrored into the focusedField=0 case; the origin URL input is mirrored
// into focusedField=5. Toggles 1–3 are navigated with j/k and toggled with space; 4 exists so
// j/k can still move past the toggle row to reach 5.
type Model struct {
	tokenSource       string // config.GitHubTokenSource* — where to read the API token
	tokenInput        textinput.Model
	showMerged        bool
	showClosed        bool
	onlyMine          bool
	prLimit           int
	prRefreshInterval int
	focusedField      int

	// Repository remote section (action-oriented; not persisted as config). originInput holds
	// the next-URL the user wants to apply to `origin`; ghPrivate is the visibility flag for
	// the "Create new GitHub repo" button. currentOrigin caches the live origin URL so the
	// view can show "(none)" / the existing URL above the input without an extra IO call per
	// frame; main refreshes it on Settings open and after any successful remote operation.
	originInput   textinput.Model
	ghPrivate     bool
	currentOrigin string
}

// MaxFocusedField is the highest valid focusedField index for the GitHub tab. Used by parent
// tab navigation (Tab cycling) so callers don't hardcode the literal.
const MaxFocusedField = 5

// NewModel creates a new GitHub settings model
func NewModel() Model {
	tokenInput := textinput.New()
	tokenInput.Placeholder = "GitHub Personal Access Token"
	tokenInput.CharLimit = 256
	tokenInput.Width = 50
	tokenInput.EchoMode = textinput.EchoPassword
	tokenInput.EchoCharacter = '•'
	tokenInput.Focus()

	originInput := textinput.New()
	originInput.Placeholder = "git@github.com:owner/repo.git or https://github.com/owner/repo.git"
	originInput.CharLimit = 512
	originInput.Width = 50

	return Model{
		tokenSource:       config.GitHubTokenSourceSaved,
		tokenInput:        tokenInput,
		showMerged:        true,
		showClosed:        true,
		onlyMine:          false,
		prLimit:           100,
		prRefreshInterval: 120,
		focusedField:      0,
		originInput:       originInput,
		ghPrivate:         true, // Match the welcome-screen default; users can flip with Ctrl+v.
	}
}

// NewModelFromConfig creates a model initialized from config and env.
func NewModelFromConfig(cfg *config.Config) Model {
	m := NewModel()
	if cfg != nil {
		m.tokenSource = cfg.GitHubTokenSourceOrDefault()
		m.showMerged = cfg.ShowMergedPRs()
		m.showClosed = cfg.ShowClosedPRs()
		m.onlyMine = cfg.OnlyMyPRs()
		m.prLimit = cfg.PRLimit()
		m.prRefreshInterval = cfg.PRRefreshInterval()
		if m.tokenSource == config.GitHubTokenSourceSaved {
			m.tokenInput.SetValue(cfg.GitHubToken)
		} else {
			m.tokenInput.SetValue("")
		}
	}
	return m
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Only handle nav and space (toggles) here; other keys go to the focused text input.
		switch msg.String() {
		case "j", "down", "k", "up", " ":
			return m.handleKeyMsg(msg)
		}
	}

	var cmd tea.Cmd
	switch {
	case m.focusedField == 0 && m.tokenSource == config.GitHubTokenSourceSaved:
		m.tokenInput, cmd = m.tokenInput.Update(msg)
	case m.focusedField == 5:
		m.originInput, cmd = m.originInput.Update(msg)
	}
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
		if m.focusedField < MaxFocusedField {
			m.focusedField++
			m.refocus()
		}
		return m, nil
	case "k", "up":
		if m.focusedField > 0 {
			m.focusedField--
			m.refocus()
		}
		return m, nil
	case " ":
		// Toggle boolean options
		switch m.focusedField {
		case 1:
			m.showMerged = !m.showMerged
		case 2:
			m.showClosed = !m.showClosed
		case 3:
			m.onlyMine = !m.onlyMine
		}
		return m, nil
	}
	return m, nil
}

// refocus updates Focus()/Blur() on the two text inputs based on the current focusedField. Kept
// in one place so j/k navigation and SetFocusedField stay consistent.
func (m *Model) refocus() {
	m.tokenInput.Blur()
	m.originInput.Blur()
	switch m.focusedField {
	case 0:
		if m.tokenSource == config.GitHubTokenSourceSaved {
			m.tokenInput.Focus()
		}
	case 5:
		m.originInput.Focus()
	}
}

// Accessors

// GetToken returns the GitHub token
func (m *Model) GetToken() string {
	return m.tokenInput.Value()
}

// SetToken sets the GitHub token
func (m *Model) SetToken(token string) {
	m.tokenInput.SetValue(token)
}

// GetTokenSource returns github_token_source (saved | env | gh_cli).
func (m *Model) GetTokenSource() string {
	if m.tokenSource == "" {
		return config.GitHubTokenSourceSaved
	}
	return m.tokenSource
}

// SetTokenSource sets where jj-tui reads the API token from.
func (m *Model) SetTokenSource(src string) {
	m.tokenSource = config.NormalizeGitHubTokenSource(src)
	if m.tokenSource != config.GitHubTokenSourceSaved {
		m.tokenInput.Blur()
	}
}

// GetShowMerged returns whether to show merged PRs
func (m *Model) GetShowMerged() bool {
return m.showMerged
}

// SetShowMerged sets whether to show merged PRs
func (m *Model) SetShowMerged(show bool) {
m.showMerged = show
}

// GetShowClosed returns whether to show closed PRs
func (m *Model) GetShowClosed() bool {
return m.showClosed
}

// SetShowClosed sets whether to show closed PRs
func (m *Model) SetShowClosed(show bool) {
m.showClosed = show
}

// GetOnlyMine returns whether to show only own PRs
func (m *Model) GetOnlyMine() bool {
return m.onlyMine
}

// SetOnlyMine sets whether to show only own PRs
func (m *Model) SetOnlyMine(only bool) {
m.onlyMine = only
}

// GetPRLimit returns the PR limit
func (m *Model) GetPRLimit() int {
return m.prLimit
}

// SetPRLimit sets the PR limit
func (m *Model) SetPRLimit(limit int) {
m.prLimit = limit
}

// GetRefreshInterval returns the refresh interval
func (m *Model) GetRefreshInterval() int {
	return m.prRefreshInterval
}

// SetRefreshInterval sets the refresh interval
func (m *Model) SetRefreshInterval(interval int) {
	m.prRefreshInterval = interval
}

// GetInputViews returns the view strings for the parent's flat input list. We deliberately
// return ONLY the token view (not the origin URL): the parent's BuildRenderData and downstream
// renderJira / renderCodecks / etc. depend on a fixed global index where index 0 = token,
// 1..8 = jira, 9..12 = codecks, etc. Adding a second entry here would shift every later index
// and break those renders. The origin URL view is rendered directly from the sub-model in
// renderGitHub via GetOriginInputView instead.
func (m *Model) GetInputViews() []string {
	return []string{m.tokenInput.View()}
}

// GetOriginInputView returns the origin URL text input's view. Exposed separately so
// renderGitHub can render the new input without sharing the parent's flat-input indexing.
func (m *Model) GetOriginInputView() string {
	return m.originInput.View()
}

// GetFocusedField returns the focused input index (0..MaxFocusedField).
func (m *Model) GetFocusedField() int {
	return m.focusedField
}

// SetFocusedField sets the focused input index (0..MaxFocusedField). Values outside that range
// are clamped.
func (m *Model) SetFocusedField(i int) {
	if i < 0 {
		i = 0
	}
	if i > MaxFocusedField {
		i = MaxFocusedField
	}
	m.focusedField = i
	m.refocus()
}

// SetInputWidth sets both text input widths so the token and origin URL fields stay visually
// aligned with the rest of the settings layout on resize.
func (m *Model) SetInputWidth(w int) {
	m.tokenInput.Width = w
	m.originInput.Width = w
}

// GetOriginURL returns the URL the user has typed into the origin input field.
func (m *Model) GetOriginURL() string {
	return m.originInput.Value()
}

// SetOriginURL replaces whatever the user typed (used when seeding the field with the live
// `origin` URL on Settings open and after a successful Apply).
func (m *Model) SetOriginURL(url string) {
	m.originInput.SetValue(url)
}

// GetCurrentOrigin returns the cached live `origin` URL ("" if not configured).
func (m *Model) GetCurrentOrigin() string {
	return m.currentOrigin
}

// SetCurrentOrigin updates the cached live `origin` URL displayed above the input.
func (m *Model) SetCurrentOrigin(url string) {
	m.currentOrigin = url
}

// GetGhPrivate reports whether the "Create new GitHub repo" button will pass `--private`.
func (m *Model) GetGhPrivate() bool {
	return m.ghPrivate
}

// SetGhPrivate sets the visibility flag for the "Create new GitHub repo" button.
func (m *Model) SetGhPrivate(p bool) {
	m.ghPrivate = p
}

// ToggleGhPrivate flips the visibility flag (bound to Ctrl+v / the visibility button).
func (m *Model) ToggleGhPrivate() {
	m.ghPrivate = !m.ghPrivate
}

// FocusOriginInput focuses the origin URL input directly. Used by zone clicks on the input box
// without going through j/k navigation.
func (m *Model) FocusOriginInput() {
	m.focusedField = 5
	m.refocus()
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
// GitHub settings don't depend on repository
}
