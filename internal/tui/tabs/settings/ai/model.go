package ai

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	bubbledropdown "github.com/madicen/bubble-dropdown"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/tui/form"
	"github.com/madicen/jj-tui/internal/tui/form/dropdown"
)

// Field indices into the shared form (also the parent's local focus order).
const (
	fieldBaseURL = iota
	fieldModel
	fieldAPIKey
	fieldProfileName
)

// aiProviderValues maps the provider dropdown indices to their config values;
// aiProviderLabels are the user-facing strings shown in the dropdown.
var (
	aiProviderValues = []string{"openai_compatible", "gemini", "ollama"}
	aiProviderLabels = []string{
		"OpenAI-compatible (Chat Completions)",
		"Google Gemini (Generative Language API)",
		"Ollama (local Chat Completions)",
	}
)

// aiProviderIndex returns the dropdown index for a provider value (0 when unknown).
func aiProviderIndex(value string) int {
	for i, v := range aiProviderValues {
		if v == value {
			return i
		}
	}
	return 0
}

// Timeout stepper bounds for the AI generation HTTP timeout (seconds).
// Lower bound matches the implicit floor in (*config.Config).AITimeout (anything ≤0
// falls back to the 60s default), and the upper bound caps practical local-model
// cold-start waits. Step is small enough to feel responsive on a stepper button.
const (
	// AITimeoutMinSeconds is the smallest timeout the stepper can produce. Stays
	// >0 so the saved value isn't silently coerced to the 60s default by AITimeout().
	AITimeoutMinSeconds = 10
	// AITimeoutMaxSeconds bounds the stepper at 10 minutes; cold-start of a large
	// local Ollama model on a slow disk is the usual reason to push this high.
	AITimeoutMaxSeconds = 600
	// AITimeoutStepSeconds is the +/- click delta. 10s lets you go from 60→120s
	// (the README's "local first-call cold start" guidance) in 6 clicks.
	AITimeoutStepSeconds = 10
	// AITimeoutDefaultSeconds matches (*config.Config).AITimeout's nil/0 fallback
	// so the UI doesn't lie about what an unset config does.
	AITimeoutDefaultSeconds = 60
)

// Model represents the AI settings sub-tab (LLM + evolog split defaults).
// The five "live" inputs (provider, base URL, model, API key, timeout) edit
// the currently selected profile; switching the selected profile reloads those
// inputs from the new profile's saved values.
type Model struct {
	aiEnabled              bool
	aiProvider             string
	form                   form.Model
	aiTimeoutSeconds       int // clamped to [AITimeoutMinSeconds, AITimeoutMaxSeconds]; mirrors cfg.AITimeout() default
	evologDescribeDefault  bool
	evologFileSplitEnabled bool
	evologHunkSplitEnabled bool
	evologMultiStepwise    bool
	evologMultiMax         int // 1..config.EvologAIMultiSplitHardMax

	// AI profile management: profiles holds every saved profile (snapshots, not
	// references to cfg.AIProfiles). selectedIdx points to the one being edited
	// (its values are mirrored to aiProvider/form fields/aiTimeoutSeconds above).
	// activeName tracks the persistently active profile so the row can be marked
	// with ● and used as the default.
	profiles    []config.AIProfile
	selectedIdx int
	activeName  string

	// providerDropdown replaces the old radio rows for the LLM provider.
	providerDropdown *dropdown.Field
}

// NewModel creates an AI settings model with defaults.
func NewModel() Model {
	aiURL := textinput.New()
	aiURL.Placeholder = "https://api.openai.com/v1 or http://127.0.0.1:11434/v1"
	aiURL.CharLimit = 200
	aiURL.Width = 60

	aiModel := textinput.New()
	aiModel.Placeholder = "e.g. gpt-4o-mini, llama3.2, or qwen2.5:1.5b (Ollama)"
	aiModel.CharLimit = 120
	aiModel.Width = 60

	aiKey := textinput.New()
	aiKey.Placeholder = "API key (stored in config.json unless env overrides)"
	aiKey.CharLimit = 400
	aiKey.Width = 60
	aiKey.EchoMode = textinput.EchoPassword
	aiKey.EchoCharacter = '•'

	aiName := textinput.New()
	aiName.Placeholder = "profile name (e.g. fast, smart, local)"
	aiName.CharLimit = 60
	aiName.Width = 30

	return Model{
		aiEnabled:              false,
		aiProvider:             "openai_compatible",
		evologFileSplitEnabled: true,
		evologHunkSplitEnabled: true,
		evologMultiMax:         config.EvologAIMultiSplitHardMax,
		aiTimeoutSeconds:       AITimeoutDefaultSeconds,
		form:                   form.New(aiURL, aiModel, aiKey, aiName),
		profiles: []config.AIProfile{
			{Name: config.DefaultAIProfileName, Provider: "openai_compatible"},
		},
		selectedIdx: 0,
		activeName:  config.DefaultAIProfileName,
		providerDropdown: dropdown.New(
			bubbledropdown.WithOptions(aiProviderLabels),
		),
	}
}

// NewModelFromConfig initializes from config.
func NewModelFromConfig(cfg *config.Config) Model {
	m := NewModel()
	if cfg == nil {
		return m
	}
	m.aiEnabled = cfg.AIGenerationEnabled()
	m.profiles = cfg.AIProfileList()
	m.activeName = cfg.ActiveAIProfile().Name
	m.selectedIdx = 0
	for i, p := range m.profiles {
		if strings.EqualFold(p.Name, m.activeName) {
			m.selectedIdx = i
			break
		}
	}
	m.loadSelectedProfileIntoInputs()
	m.evologDescribeDefault = cfg.DefaultEvologPostSplitDescribe()
	m.evologFileSplitEnabled = cfg.EvologAIFilePhaseEnabled()
	m.evologHunkSplitEnabled = cfg.EvologAIHunkPhaseEnabled()
	m.evologMultiStepwise = cfg.EvologAIMultiSplitStepwise()
	m.evologMultiMax = cfg.EvologAIMultiSplitMaxCap()
	return m
}

// loadSelectedProfileIntoInputs mirrors the selected profile onto the live
// edit fields (provider, base URL, model, API key, timeout, name). Called on
// init and whenever the user picks a different profile from the list.
func (m *Model) loadSelectedProfileIntoInputs() {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.profiles) {
		m.selectedIdx = 0
	}
	if len(m.profiles) == 0 {
		return
	}
	p := m.profiles[m.selectedIdx]
	m.aiProvider = config.NormalizeAIProvider(p.Provider)
	if m.providerDropdown != nil {
		m.providerDropdown.SetSelectedIndex(aiProviderIndex(m.aiProvider))
	}
	m.form.SetValue(fieldBaseURL, p.BaseURL)
	m.form.SetValue(fieldModel, p.Model)
	if m.aiProvider == "ollama" {
		if strings.TrimSpace(p.BaseURL) == "" {
			m.form.SetValue(fieldBaseURL, config.OllamaDefaultChatBaseURL)
		}
		if strings.TrimSpace(p.Model) == "" {
			m.form.SetValue(fieldModel, config.OllamaDefaultModel)
		}
	}
	m.form.SetValue(fieldAPIKey, p.APIKey)
	m.form.SetValue(fieldProfileName, p.Name)
	if p.TimeoutSeconds > 0 {
		m.aiTimeoutSeconds = clampAITimeout(p.TimeoutSeconds)
	} else {
		m.aiTimeoutSeconds = AITimeoutDefaultSeconds
	}
}

// commitInputsToSelectedProfile snapshots the current input/textbox values back
// into the selected profile. Called before SetSelectedProfile, before
// AddProfile/Delete, and on Save so the in-memory profile list never lags the
// visible inputs.
func (m *Model) commitInputsToSelectedProfile() {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.profiles) {
		return
	}
	timeout := 0
	if t := clampAITimeout(m.aiTimeoutSeconds); t > 0 && t != AITimeoutDefaultSeconds {
		timeout = t
	}
	name := strings.TrimSpace(m.form.Value(fieldProfileName))
	if name == "" {
		name = m.profiles[m.selectedIdx].Name
	}
	m.profiles[m.selectedIdx] = config.AIProfile{
		Name:           name,
		Provider:       config.NormalizeAIProvider(m.aiProvider),
		BaseURL:        strings.TrimSpace(m.form.Value(fieldBaseURL)),
		Model:          strings.TrimSpace(m.form.Value(fieldModel)),
		APIKey:         strings.TrimSpace(m.form.Value(fieldAPIKey)),
		TimeoutSeconds: timeout,
	}
}

// clampAITimeout snaps an arbitrary second count into the stepper's allowed range.
func clampAITimeout(v int) int {
	if v < AITimeoutMinSeconds {
		return AITimeoutMinSeconds
	}
	if v > AITimeoutMaxSeconds {
		return AITimeoutMaxSeconds
	}
	return v
}

// Update forwards to the focused text input; when the profile-name field is
// focused, also mirrors the typed name into the selected profile row.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	cmd := m.form.Update(msg)
	if m.form.Focused() == fieldProfileName && m.selectedIdx >= 0 && m.selectedIdx < len(m.profiles) {
		name := strings.TrimSpace(m.form.Value(fieldProfileName))
		if name != "" {
			prevName := m.profiles[m.selectedIdx].Name
			m.profiles[m.selectedIdx].Name = name
			if strings.EqualFold(m.activeName, prevName) {
				m.activeName = name
			}
		}
	}
	return m, cmd
}

// GetAIEnabled returns whether AI assist is enabled.
func (m *Model) GetAIEnabled() bool { return m.aiEnabled }

// SetAIEnabled sets the AI assist toggle.
func (m *Model) SetAIEnabled(v bool) { m.aiEnabled = v }

// ToggleAIEnabled flips the AI assist toggle.
func (m *Model) ToggleAIEnabled() { m.aiEnabled = !m.aiEnabled }

func (m *Model) GetEvologDescribeAfterSplitDefault() bool { return m.evologDescribeDefault }

func (m *Model) ToggleEvologDescribeAfterSplitDefault() {
	m.evologDescribeDefault = !m.evologDescribeDefault
}

func (m *Model) GetEvologFileSplitEnabled() bool { return m.evologFileSplitEnabled }

func (m *Model) ToggleEvologFileSplitEnabled() { m.evologFileSplitEnabled = !m.evologFileSplitEnabled }

func (m *Model) GetEvologHunkSplitEnabled() bool { return m.evologHunkSplitEnabled }

func (m *Model) ToggleEvologHunkSplitEnabled() { m.evologHunkSplitEnabled = !m.evologHunkSplitEnabled }

func (m *Model) GetEvologMultiStepwise() bool { return m.evologMultiStepwise }

func (m *Model) ToggleEvologMultiStepwise() { m.evologMultiStepwise = !m.evologMultiStepwise }

func (m *Model) GetEvologMultiMax() int { return m.evologMultiMax }

func (m *Model) IncEvologMultiMax() {
	if m.evologMultiMax < config.EvologAIMultiSplitHardMax {
		m.evologMultiMax++
	}
}

func (m *Model) DecEvologMultiMax() {
	if m.evologMultiMax > 1 {
		m.evologMultiMax--
	}
}

// GetAITimeoutSeconds returns the configured HTTP timeout for LLM requests, in
// seconds. Always within [AITimeoutMinSeconds, AITimeoutMaxSeconds].
func (m *Model) GetAITimeoutSeconds() int {
	return clampAITimeout(m.aiTimeoutSeconds)
}

// SetAITimeoutSeconds sets the AI HTTP timeout (clamped to the stepper bounds).
// Used by tests and any direct setter; UI uses Inc/DecAITimeout for stepper clicks.
func (m *Model) SetAITimeoutSeconds(v int) {
	m.aiTimeoutSeconds = clampAITimeout(v)
}

// IncAITimeout bumps the AI HTTP timeout up by AITimeoutStepSeconds (clamped).
func (m *Model) IncAITimeout() {
	m.aiTimeoutSeconds = clampAITimeout(m.aiTimeoutSeconds + AITimeoutStepSeconds)
}

// DecAITimeout decreases the AI HTTP timeout by AITimeoutStepSeconds (clamped).
func (m *Model) DecAITimeout() {
	m.aiTimeoutSeconds = clampAITimeout(m.aiTimeoutSeconds - AITimeoutStepSeconds)
}

// GetAIProvider returns the selected provider id.
func (m *Model) GetAIProvider() string {
	return strings.TrimSpace(m.aiProvider)
}

// SetAIProvider sets provider id and applies Ollama presets when switching to ollama with empty fields.
func (m *Model) SetAIProvider(s string) {
	prev := strings.TrimSpace(m.aiProvider)
	m.aiProvider = strings.TrimSpace(s)
	switch strings.ToLower(m.aiProvider) {
	case "gemini":
		m.aiProvider = "gemini"
	case "ollama":
		m.aiProvider = "ollama"
	default:
		m.aiProvider = "openai_compatible"
	}
	if m.aiProvider == "ollama" && prev != "ollama" {
		if strings.TrimSpace(m.form.Value(fieldBaseURL)) == "" {
			m.form.SetValue(fieldBaseURL, config.OllamaDefaultChatBaseURL)
		}
		if strings.TrimSpace(m.form.Value(fieldModel)) == "" {
			m.form.SetValue(fieldModel, config.OllamaDefaultModel)
		}
	}
	if m.providerDropdown != nil {
		m.providerDropdown.SetSelectedIndex(aiProviderIndex(m.aiProvider))
	}
}

// ProviderDropdown returns the LLM provider dropdown (for rendering and
// overlay). It syncs the accent so the panel tracks the live theme primary color.
func (m *Model) ProviderDropdown() *bubbledropdown.Dropdown {
	return m.providerDropdown.Dropdown()
}

// DropdownOpen reports whether the provider dropdown panel is open.
func (m *Model) DropdownOpen() bool { return m.providerDropdown.Open() }

// SetZoneManager wires the bubblezone manager into the provider dropdown.
func (m *Model) SetZoneManager(zm *zone.Manager) {
	m.providerDropdown.SetZoneManager(zm)
}

// UpdateDropdown forwards a message to the provider dropdown and, on selection,
// applies the chosen provider value. Returns any tea.Cmd the dropdown emits.
func (m *Model) UpdateDropdown(msg tea.Msg) tea.Cmd {
	return m.providerDropdown.Update(msg, func(i int) {
		if i >= 0 && i < len(aiProviderValues) {
			m.SetAIProvider(aiProviderValues[i])
		}
	})
}

// GetAIBaseURL returns the configured API base URL field.
func (m *Model) GetAIBaseURL() string {
	return strings.TrimSpace(m.form.Value(fieldBaseURL))
}

// GetAIModel returns the configured model field.
func (m *Model) GetAIModel() string {
	return strings.TrimSpace(m.form.Value(fieldModel))
}

// GetAIAPIKey returns the key field (may be empty).
func (m *Model) GetAIAPIKey() string {
	return strings.TrimSpace(m.form.Value(fieldAPIKey))
}

// GetInputViews returns API URL, model, and API key views (global input indices 16–18 on the AI tab).
// Profile name is rendered separately via GetProfileNameInputView.
func (m *Model) GetInputViews() []string {
	views := m.form.Views()
	return views[:3]
}

// GetFocusedField returns 0–3 for the AI text inputs.
func (m *Model) GetFocusedField() int {
	return m.form.Focused()
}

// SetFocusedField focuses one of the AI inputs (0–3). Returns tea.Cmd from Focus().
// 0 = base URL, 1 = model, 2 = API key, 3 = profile name.
func (m *Model) SetFocusedField(i int) tea.Cmd {
	return m.form.Focus(i)
}

// SetInputWidth sets widths for AI text fields. Profile name stays half-width.
func (m *Model) SetInputWidth(w int) {
	if w < 40 {
		w = 40
	}
	m.form.SetWidth(w)
	nameWidth := w / 2
	if nameWidth < 20 {
		nameWidth = 20
	}
	if in := m.form.Input(fieldProfileName); in != nil {
		in.Width = nameWidth
	}
}

// Profiles returns the current in-memory profile list (snapshots, not
// references). Save uses this to write the full list into config. As a side
// effect, the current input values are committed back to the selected profile
// so the returned snapshot reflects unsaved edits.
func (m *Model) Profiles() []config.AIProfile {
	m.commitInputsToSelectedProfile()
	out := make([]config.AIProfile, len(m.profiles))
	copy(out, m.profiles)
	return out
}

// ProfileCount returns the number of profiles without committing input edits.
// Use this for purely structural needs (e.g. enumerating row zone ids) so
// the count call doesn't itself mutate the selected profile.
func (m *Model) ProfileCount() int {
	return len(m.profiles)
}

// ActiveProfileName returns the persistently active profile's name.
func (m *Model) ActiveProfileName() string {
	return m.activeName
}

// SelectedIndex returns the index of the profile being edited.
func (m *Model) SelectedIndex() int {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.profiles) {
		return 0
	}
	return m.selectedIdx
}

// SelectedName returns the name of the profile being edited.
func (m *Model) SelectedName() string {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.profiles) {
		return ""
	}
	return m.profiles[m.selectedIdx].Name
}

// SelectProfile commits the current input values to the previously-selected
// profile, then switches to the profile at idx and reloads its values into
// the inputs. No-op when idx is out of range.
func (m *Model) SelectProfile(idx int) {
	if idx < 0 || idx >= len(m.profiles) || idx == m.selectedIdx {
		return
	}
	m.commitInputsToSelectedProfile()
	m.selectedIdx = idx
	m.loadSelectedProfileIntoInputs()
}

// SetActiveProfile marks the named profile as the persistent active one.
// Does not change which profile is being edited.
func (m *Model) SetActiveProfile(name string) {
	for _, p := range m.profiles {
		if strings.EqualFold(strings.TrimSpace(p.Name), strings.TrimSpace(name)) {
			m.activeName = p.Name
			return
		}
	}
}

// SetActiveByIndex marks the profile at idx as active.
func (m *Model) SetActiveByIndex(idx int) {
	if idx < 0 || idx >= len(m.profiles) {
		return
	}
	m.activeName = m.profiles[idx].Name
}

// AddProfile commits the current inputs, appends a new profile with an
// auto-generated unique name based on the current selection, and selects it
// for editing.
func (m *Model) AddProfile() {
	m.commitInputsToSelectedProfile()
	base := "profile"
	if m.selectedIdx >= 0 && m.selectedIdx < len(m.profiles) {
		base = m.profiles[m.selectedIdx].Name
	}
	name := uniqueProfileName(m.profiles, base+" copy")
	template := config.AIProfile{Name: name, Provider: "openai_compatible"}
	if m.selectedIdx >= 0 && m.selectedIdx < len(m.profiles) {
		template = m.profiles[m.selectedIdx]
		template.Name = name
	}
	m.profiles = append(m.profiles, template)
	m.selectedIdx = len(m.profiles) - 1
	m.loadSelectedProfileIntoInputs()
}

// DeleteSelectedProfile removes the currently-selected profile. Returns an
// error string for the status message when the deletion is not allowed (last
// remaining profile). When the deleted profile was active, the first remaining
// profile becomes active.
func (m *Model) DeleteSelectedProfile() string {
	if len(m.profiles) <= 1 {
		return "Cannot delete the last AI profile"
	}
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.profiles) {
		return ""
	}
	wasActive := strings.EqualFold(m.activeName, m.profiles[m.selectedIdx].Name)
	m.profiles = append(m.profiles[:m.selectedIdx], m.profiles[m.selectedIdx+1:]...)
	if m.selectedIdx >= len(m.profiles) {
		m.selectedIdx = len(m.profiles) - 1
	}
	if wasActive {
		m.activeName = m.profiles[0].Name
	}
	m.loadSelectedProfileIntoInputs()
	return ""
}

// CycleSelected moves the editing cursor by delta through the profile list,
// wrapping at the ends.
func (m *Model) CycleSelected(delta int) {
	if len(m.profiles) < 2 {
		return
	}
	m.commitInputsToSelectedProfile()
	n := len(m.profiles)
	m.selectedIdx = ((m.selectedIdx+delta)%n + n) % n
	m.loadSelectedProfileIntoInputs()
}

// GetProfileNameInputView returns the rendered profile-name textinput for the
// settings view layer. Kept here so the editor row can include a Mark/Zone.
func (m *Model) GetProfileNameInputView() string {
	return m.form.Input(fieldProfileName).View()
}

// SetProfileNameInputValue replaces the profile-name textinput's contents.
// Used by tests and any flow that wants to set the in-editor profile name
// without typing keystroke-by-keystroke.
func (m *Model) SetProfileNameInputValue(s string) {
	m.form.SetValue(fieldProfileName, s)
}

// CommitInputs is the externally-callable form of commitInputsToSelectedProfile.
// Used by the settings actions layer when building params for Save.
func (m *Model) CommitInputs() {
	m.commitInputsToSelectedProfile()
}

// uniqueProfileName returns base when free, or "base 2", "base 3", ... until
// it finds a name no profile in profiles uses.
func uniqueProfileName(profiles []config.AIProfile, base string) string {
	candidate := strings.TrimSpace(base)
	if candidate == "" {
		candidate = "profile"
	}
	exists := func(name string) bool {
		for _, p := range profiles {
			if strings.EqualFold(strings.TrimSpace(p.Name), name) {
				return true
			}
		}
		return false
	}
	if !exists(candidate) {
		return candidate
	}
	for i := 2; i < 999; i++ {
		c := fmt.Sprintf("%s %d", candidate, i)
		if !exists(c) {
			return c
		}
	}
	return candidate
}
