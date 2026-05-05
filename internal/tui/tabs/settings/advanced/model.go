package advanced

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
)

// Model represents the Advanced settings sub-tab (sanitize bookmarks, graph revset, external editor, cleanup).
type Model struct {
	sanitizeBookmarks    bool
	confirmingCleanup    string
	graphRevsetInput     textinput.Model
	customEditorInput    textinput.Model
	focusedField         int // 0 = graph revset, 1 = custom editor
	externalEditorPreset int // 0..8 — see externalEditorPresetLabels
}

// ExternalEditorPresetLabels are UI labels for each editor preset (same order as config values below).
var ExternalEditorPresetLabels = []string{
	"None (disabled)",
	"Cursor",
	"VS Code",
	"Zed",
	"Neovim (nvr — remote)",
	"Emacs (emacsclient)",
	"Sublime Text (subl)",
	"JetBrains (idea)",
	"Custom shell command",
}

var externalEditorPresetConfig = []string{
	config.ExternalEditorNone,
	config.ExternalEditorCursor,
	config.ExternalEditorVSCode,
	config.ExternalEditorZed,
	config.ExternalEditorNeovim,
	config.ExternalEditorEmacs,
	config.ExternalEditorSublime,
	config.ExternalEditorIntelliJ,
	config.ExternalEditorCustom,
}

// NewModel creates a new Advanced settings model
func NewModel() Model {
	revsetInput := textinput.New()
	revsetInput.Placeholder = "e.g. trunk() | (ancestors(@) - ancestors(trunk()))"
	revsetInput.CharLimit = 500
	revsetInput.Width = 60

	customIn := textinput.New()
	customIn.Placeholder = `e.g. cursor -g {path}  or  alacritty -e nvim {path}`
	customIn.CharLimit = 400
	customIn.Width = 60

	return Model{
		sanitizeBookmarks: true,
		confirmingCleanup: "",
		graphRevsetInput:  revsetInput,
		customEditorInput: customIn,
		focusedField:      0,
	}
}

// NewModelFromConfig creates a model initialized from config.
func NewModelFromConfig(cfg *config.Config) Model {
	m := NewModel()
	if cfg != nil {
		m.sanitizeBookmarks = cfg.ShouldSanitizeBookmarkNames()
		m.graphRevsetInput.SetValue(cfg.GraphRevset)
		m.customEditorInput.SetValue(cfg.ExternalFileEditorCustom)
		m.externalEditorPreset = presetIndexFromConfig(cfg.ExternalFileEditor)
	}
	return m
}

func presetIndexFromConfig(s string) int {
	n := config.NormalizeExternalFileEditor(&config.Config{ExternalFileEditor: s})
	for i, v := range externalEditorPresetConfig {
		if v == n {
			return i
		}
	}
	return 0
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages (key handling for inputs; zones handled by parent)
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch m.focusedField {
	case 0:
		var cmd tea.Cmd
		m.graphRevsetInput, cmd = m.graphRevsetInput.Update(msg)
		return m, cmd
	case 1:
		var cmd tea.Cmd
		m.customEditorInput, cmd = m.customEditorInput.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}

// View renders the model
func (m Model) View() string {
	return "" // Rendered by parent
}

// GetSanitizeBookmarks returns whether to sanitize bookmark names
func (m *Model) GetSanitizeBookmarks() bool {
	return m.sanitizeBookmarks
}

// SetSanitizeBookmarks sets whether to sanitize bookmark names
func (m *Model) SetSanitizeBookmarks(sanitize bool) {
	m.sanitizeBookmarks = sanitize
}

// GetGraphRevset returns the graph revset string
func (m *Model) GetGraphRevset() string {
	return m.graphRevsetInput.Value()
}

// SetGraphRevset sets the graph revset string
func (m *Model) SetGraphRevset(s string) {
	m.graphRevsetInput.SetValue(s)
}

// GetConfirmingCleanup returns the current cleanup confirmation type ("", "delete_bookmarks", "abandon_old_commits")
func (m *Model) GetConfirmingCleanup() string {
	return m.confirmingCleanup
}

// SetConfirmingCleanup sets the cleanup confirmation type
func (m *Model) SetConfirmingCleanup(s string) {
	m.confirmingCleanup = s
}

// GetInputViews returns the view strings for graph revset and custom editor (indices 14–15).
func (m *Model) GetInputViews() []string {
	return []string{
		m.graphRevsetInput.View(),
		m.customEditorInput.View(),
	}
}

// GetFocusedField returns the focused input index (0 = graph revset, 1 = custom editor).
func (m *Model) GetFocusedField() int {
	return m.focusedField
}

// SetFocusedField sets the focused input index.
// Returns the tea.Cmd from Focus() so the cursor is shown; caller must return it from Update.
func (m *Model) SetFocusedField(i int) tea.Cmd {
	if i < 0 {
		i = 0
	}
	if i > 1 {
		i = 1
	}
	m.focusedField = i
	m.graphRevsetInput.Blur()
	m.customEditorInput.Blur()
	switch m.focusedField {
	case 0:
		return m.graphRevsetInput.Focus()
	default:
		return m.customEditorInput.Focus()
	}
}

// SetInputWidth sets input widths (minimum 40 so the field and cursor are visible).
func (m *Model) SetInputWidth(w int) {
	if w < 40 {
		w = 40
	}
	m.graphRevsetInput.Width = w
	m.customEditorInput.Width = w
}

// GetExternalEditorPreset returns the selected editor preset index (0..len(ExternalEditorPresetLabels)-1).
func (m *Model) GetExternalEditorPreset() int {
	if m.externalEditorPreset < 0 || m.externalEditorPreset >= len(ExternalEditorPresetLabels) {
		return 0
	}
	return m.externalEditorPreset
}

// SetExternalEditorPreset selects an editor preset by index.
func (m *Model) SetExternalEditorPreset(i int) {
	if i < 0 || i >= len(externalEditorPresetConfig) {
		return
	}
	m.externalEditorPreset = i
}

// SavedExternalEditor returns config strings to persist.
func (m *Model) SavedExternalEditor() (preset string, custom string) {
	i := m.externalEditorPreset
	if i < 0 || i >= len(externalEditorPresetConfig) {
		return config.ExternalEditorNone, strings.TrimSpace(m.customEditorInput.Value())
	}
	return externalEditorPresetConfig[i], strings.TrimSpace(m.customEditorInput.Value())
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {}
