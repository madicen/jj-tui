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
	timeout := cfg.AITimeout()
	model := cfg.AIModelResolved()

	switch cfg.AIProviderOrDefault() {
	case "gemini":
		key := config.EffectiveAIAPIKey(cfg)
		if key == "" {
			return nil, fmt.Errorf("missing API key")
		}
		return NewGeminiProvider(key, model, timeout), nil
	default:
		key, err := cfg.ResolveOpenAICompatibleBearerKey()
		if err != nil {
			return nil, err
		}
		return NewOpenAICompatibleProvider(cfg.AIBaseURLResolved(), key, model, timeout), nil
	}
}
