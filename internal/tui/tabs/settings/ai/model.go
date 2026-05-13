package ai

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/config"
)

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
type Model struct {
	aiEnabled              bool
	aiProvider             string
	aiBaseURLInput         textinput.Model
	aiModelInput           textinput.Model
	aiAPIKeyInput          textinput.Model
	focusedField           int // 0 = base URL, 1 = model, 2 = API key
	aiTimeoutSeconds       int // clamped to [AITimeoutMinSeconds, AITimeoutMaxSeconds]; mirrors cfg.AITimeout() default
	evologDescribeDefault  bool
	evologFileSplitEnabled bool
	evologHunkSplitEnabled bool
	evologMultiStepwise    bool
	evologMultiMax         int // 1..config.EvologAIMultiSplitHardMax
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

	return Model{
		aiEnabled:              false,
		aiProvider:             "openai_compatible",
		evologFileSplitEnabled: true,
		evologHunkSplitEnabled: true,
		evologMultiMax:         config.EvologAIMultiSplitHardMax,
		aiTimeoutSeconds:       AITimeoutDefaultSeconds,
		aiBaseURLInput:         aiURL,
		aiModelInput:           aiModel,
		aiAPIKeyInput:          aiKey,
		focusedField:           0,
	}
}

// NewModelFromConfig initializes from config.
func NewModelFromConfig(cfg *config.Config) Model {
	m := NewModel()
	if cfg == nil {
		return m
	}
	m.aiEnabled = cfg.AIGenerationEnabled()
	m.aiProvider = cfg.AIProviderOrDefault()
	m.aiBaseURLInput.SetValue(cfg.AIBaseURL)
	m.aiModelInput.SetValue(cfg.AIModel)
	if cfg.AIProviderOrDefault() == "ollama" {
		if strings.TrimSpace(cfg.AIBaseURL) == "" {
			m.aiBaseURLInput.SetValue(config.OllamaDefaultChatBaseURL)
		}
		if strings.TrimSpace(cfg.AIModel) == "" {
			m.aiModelInput.SetValue(config.OllamaDefaultModel)
		}
	}
	m.aiAPIKeyInput.SetValue(cfg.AIAPIKey)
	m.evologDescribeDefault = cfg.DefaultEvologPostSplitDescribe()
	m.evologFileSplitEnabled = cfg.EvologAIFilePhaseEnabled()
	m.evologHunkSplitEnabled = cfg.EvologAIHunkPhaseEnabled()
	m.evologMultiStepwise = cfg.EvologAIMultiSplitStepwise()
	m.evologMultiMax = cfg.EvologAIMultiSplitMaxCap()
	// Derive the stepper value from AITimeout() so the UI displays the *effective*
	// value (60s) for users whose config has the field unset, instead of showing
	// 0s and silently meaning "default". cfg.AITimeout always returns >0.
	m.aiTimeoutSeconds = clampAITimeout(int(cfg.AITimeout().Seconds()))
	return m
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

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update forwards to the focused text input.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch m.focusedField {
	case 0:
		var cmd tea.Cmd
		m.aiBaseURLInput, cmd = m.aiBaseURLInput.Update(msg)
		return m, cmd
	case 1:
		var cmd tea.Cmd
		m.aiModelInput, cmd = m.aiModelInput.Update(msg)
		return m, cmd
	case 2:
		var cmd tea.Cmd
		m.aiAPIKeyInput, cmd = m.aiAPIKeyInput.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}

// View is unused; parent renders.
func (m Model) View() string { return "" }

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
		if strings.TrimSpace(m.aiBaseURLInput.Value()) == "" {
			m.aiBaseURLInput.SetValue(config.OllamaDefaultChatBaseURL)
		}
		if strings.TrimSpace(m.aiModelInput.Value()) == "" {
			m.aiModelInput.SetValue(config.OllamaDefaultModel)
		}
	}
}

// GetAIBaseURL returns the configured API base URL field.
func (m *Model) GetAIBaseURL() string {
	return strings.TrimSpace(m.aiBaseURLInput.Value())
}

// GetAIModel returns the configured model field.
func (m *Model) GetAIModel() string {
	return strings.TrimSpace(m.aiModelInput.Value())
}

// GetAIAPIKey returns the key field (may be empty).
func (m *Model) GetAIAPIKey() string {
	return strings.TrimSpace(m.aiAPIKeyInput.Value())
}

// GetInputViews returns API URL, model, and API key views (global input indices 16–18 on the AI tab).
func (m *Model) GetInputViews() []string {
	return []string{
		m.aiBaseURLInput.View(),
		m.aiModelInput.View(),
		m.aiAPIKeyInput.View(),
	}
}

// GetFocusedField returns 0–2 for the three AI text inputs.
func (m *Model) GetFocusedField() int {
	return m.focusedField
}

// SetFocusedField focuses one of the AI inputs (0–2). Returns tea.Cmd from Focus().
func (m *Model) SetFocusedField(i int) tea.Cmd {
	if i < 0 {
		i = 0
	}
	if i > 2 {
		i = 2
	}
	m.focusedField = i
	m.aiBaseURLInput.Blur()
	m.aiModelInput.Blur()
	m.aiAPIKeyInput.Blur()
	switch m.focusedField {
	case 0:
		return m.aiBaseURLInput.Focus()
	case 1:
		return m.aiModelInput.Focus()
	default:
		return m.aiAPIKeyInput.Focus()
	}
}

// SetInputWidth sets widths for AI text fields.
func (m *Model) SetInputWidth(w int) {
	if w < 40 {
		w = 40
	}
	m.aiBaseURLInput.Width = w
	m.aiModelInput.Width = w
	m.aiAPIKeyInput.Width = w
}
