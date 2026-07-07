package advanced

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	bubbledropdown "github.com/madicen/bubble-dropdown"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/tui/form"
	"github.com/madicen/jj-tui/internal/tui/form/dropdown"
)

// Field indices into the shared form (0 = graph revset, 1 = custom editor).
const (
	fieldGraphRevset = iota
	fieldCustomEditor
)

// Model represents the Advanced settings sub-tab (sanitize bookmarks, graph revset, external editor, cleanup).
type Model struct {
	sanitizeBookmarks    bool
	confirmingCleanup    string
	form                 form.Model
	externalEditorPreset int // 0..8 — see externalEditorPresetLabels

	// editorDropdown replaces the old radio rows for picking the external editor
	// preset. The selected index maps 1:1 onto externalEditorPreset.
	editorDropdown *dropdown.Field
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
		form:              form.New(revsetInput, customIn),
		editorDropdown: dropdown.New(
			bubbledropdown.WithOptions(ExternalEditorPresetLabels),
			bubbledropdown.WithMaxVisible(len(ExternalEditorPresetLabels)),
		),
	}
}

// NewModelFromConfig creates a model initialized from config.
func NewModelFromConfig(cfg *config.Config) Model {
	m := NewModel()
	if cfg != nil {
		m.sanitizeBookmarks = cfg.ShouldSanitizeBookmarkNames()
		m.form.SetValue(fieldGraphRevset, cfg.GraphRevset)
		m.form.SetValue(fieldCustomEditor, cfg.ExternalFileEditorCustom)
		m.externalEditorPreset = presetIndexFromConfig(cfg.ExternalFileEditor)
	}
	m.editorDropdown.SetSelectedIndex(m.externalEditorPreset)
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
	return m, m.form.Update(msg)
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
	return m.form.Value(fieldGraphRevset)
}

// SetGraphRevset sets the graph revset string
func (m *Model) SetGraphRevset(s string) {
	m.form.SetValue(fieldGraphRevset, s)
}

// GetConfirmingCleanup returns the current cleanup confirmation type ("", "delete_bookmarks", "abandon_old_commits")
func (m *Model) GetConfirmingCleanup() string {
	return m.confirmingCleanup
}

// SetConfirmingCleanup sets the cleanup confirmation type
func (m *Model) SetConfirmingCleanup(s string) {
	m.confirmingCleanup = s
}

// GetInputViews returns graph revset and custom editor views (global input indices 14–15 on the Advanced tab).
func (m *Model) GetInputViews() []string {
	return m.form.Views()
}

// GetFocusedField returns the focused input index (0 = graph revset, 1 = custom editor).
func (m *Model) GetFocusedField() int {
	return m.form.Focused()
}

// SetFocusedField sets the focused input index.
// Returns the tea.Cmd from Focus() so the cursor is shown; caller must return it from Update.
func (m *Model) SetFocusedField(i int) tea.Cmd {
	return m.form.Focus(i)
}

// SetInputWidth sets input widths (minimum 40 so the field and cursor are visible).
func (m *Model) SetInputWidth(w int) {
	if w < 40 {
		w = 40
	}
	m.form.SetWidth(w)
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
	if m.editorDropdown != nil {
		m.editorDropdown.SetSelectedIndex(i)
	}
}

// EditorDropdown returns the external-editor preset dropdown (for rendering and
// overlay). It syncs the accent so the panel tracks the live theme primary color.
func (m *Model) EditorDropdown() *bubbledropdown.Dropdown {
	return m.editorDropdown.Dropdown()
}

// DropdownOpen reports whether the editor preset dropdown panel is open.
func (m *Model) DropdownOpen() bool { return m.editorDropdown.Open() }

// SetZoneManager wires the bubblezone manager into the editor dropdown.
func (m *Model) SetZoneManager(zm *zone.Manager) {
	m.editorDropdown.SetZoneManager(zm)
}

// UpdateDropdown forwards a message to the editor dropdown and, on selection,
// applies the chosen preset index. Returns any tea.Cmd the dropdown emits.
func (m *Model) UpdateDropdown(msg tea.Msg) tea.Cmd {
	return m.editorDropdown.Update(msg, func(i int) {
		m.SetExternalEditorPreset(i)
	})
}

// SavedExternalEditor returns config strings to persist.
func (m *Model) SavedExternalEditor() (preset string, custom string) {
	i := m.externalEditorPreset
	if i < 0 || i >= len(externalEditorPresetConfig) {
		return config.ExternalEditorNone, strings.TrimSpace(m.form.Value(fieldCustomEditor))
	}
	return externalEditorPresetConfig[i], strings.TrimSpace(m.form.Value(fieldCustomEditor))
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {}
