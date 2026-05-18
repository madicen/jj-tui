package llm

import (
	"fmt"
	"strings"
	"time"

	"github.com/madicen/jj-tui/internal/config"
)

// NewProviderForConfig builds the LLM provider from the active AI profile of cfg.
// Equivalent to NewProviderForProfile(cfg.ActiveAIProfile(), cfg).
func NewProviderForConfig(cfg *config.Config) (Provider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil config")
	}
	return NewProviderForProfile(cfg.ActiveAIProfile(), cfg)
}

// NewProviderForProfile builds the LLM provider from a specific AI profile.
// fallbackCfg is used only to resolve env-based API key overrides
// (JJ_TUI_AI_API_KEY via config.EffectiveAIAPIKey) and to compute defaults
// (e.g. Ollama placeholder bearer for unkeyed local setups, default timeout).
// Pass cfg the profile came from, or any *config.Config when overriding;
// nil is also accepted but disables the env-key shortcut.
func NewProviderForProfile(p config.AIProfile, fallbackCfg *config.Config) (Provider, error) {
	provider := config.NormalizeAIProvider(p.Provider)
	model := resolveProfileModel(p)
	timeout := profileTimeout(p, fallbackCfg)

	switch provider {
	case "gemini":
		key := profileAPIKey(p, fallbackCfg)
		if key == "" {
			return nil, fmt.Errorf("missing API key")
		}
		return NewGeminiProvider(key, model, timeout), nil
	default:
		key, err := profileOpenAICompatibleBearer(p, fallbackCfg, provider)
		if err != nil {
			return nil, err
		}
		return NewOpenAICompatibleProvider(profileBaseURL(p, provider), key, model, timeout), nil
	}
}

// profileAPIKey returns the API key for the profile, falling back to the
// JJ_TUI_AI_API_KEY env via fallbackCfg when the profile has no key set.
func profileAPIKey(p config.AIProfile, fallbackCfg *config.Config) string {
	if k := strings.TrimSpace(p.APIKey); k != "" {
		return k
	}
	if fallbackCfg != nil {
		return config.EffectiveAIAPIKey(fallbackCfg)
	}
	return ""
}

// profileBaseURL returns the API base URL for the profile, applying
// provider-specific defaults when the profile has none set.
func profileBaseURL(p config.AIProfile, provider string) string {
	s := strings.TrimSpace(p.BaseURL)
	if s != "" {
		return strings.TrimSuffix(s, "/")
	}
	switch provider {
	case "ollama":
		return config.OllamaDefaultChatBaseURL
	default:
		return "https://api.openai.com/v1"
	}
}

// resolveProfileModel returns the model id for the profile, applying
// provider-specific defaults when the profile has none set.
func resolveProfileModel(p config.AIProfile) string {
	if s := strings.TrimSpace(p.Model); s != "" {
		return s
	}
	switch config.NormalizeAIProvider(p.Provider) {
	case "gemini":
		return "gemini-2.5-flash"
	case "ollama":
		return config.OllamaDefaultModel
	default:
		return "gpt-4o-mini"
	}
}

// profileTimeout returns the HTTP timeout for the profile, defaulting to the
// cfg-wide AITimeout when the profile's TimeoutSeconds is unset.
func profileTimeout(p config.AIProfile, fallbackCfg *config.Config) time.Duration {
	if p.TimeoutSeconds > 0 {
		return time.Duration(p.TimeoutSeconds) * time.Second
	}
	if fallbackCfg != nil {
		return fallbackCfg.AITimeout()
	}
	return 60 * time.Second
}

// profileOpenAICompatibleBearer returns the Bearer token for OpenAI-compatible
// chat requests using this profile (with env-key fallback and Ollama placeholder).
func profileOpenAICompatibleBearer(p config.AIProfile, fallbackCfg *config.Config, provider string) (string, error) {
	if k := profileAPIKey(p, fallbackCfg); k != "" {
		return k, nil
	}
	if provider == "ollama" {
		return config.OllamaOpenAICompatiblePlaceholderKey, nil
	}
	if config.IsOllamaOpenAICompatibleBaseURL(profileBaseURL(p, provider)) {
		return config.OllamaOpenAICompatiblePlaceholderKey, nil
	}
	return "", fmt.Errorf("missing API key")
}
