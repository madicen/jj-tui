package llm

import (
	"fmt"

	"github.com/madicen/jj-tui/internal/config"
)

// NewProviderForConfig builds the LLM provider from persisted settings.
func NewProviderForConfig(cfg *config.Config) (Provider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil config")
	}
	key := config.EffectiveAIAPIKey(cfg)
	if key == "" {
		return nil, fmt.Errorf("missing API key")
	}
	timeout := cfg.AITimeout()
	model := cfg.AIModelResolved()

	switch cfg.AIProviderOrDefault() {
	case "gemini":
		return NewGeminiProvider(key, model, timeout), nil
	default:
		return NewOpenAICompatibleProvider(cfg.AIBaseURLResolved(), key, model, timeout), nil
	}
}
