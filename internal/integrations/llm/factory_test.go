package llm

import (
	"testing"

	"github.com/madicen/jj-tui/internal/config"
)

func TestNewProviderForProfile_OpenAI(t *testing.T) {
	cfg := &config.Config{}
	p := config.AIProfile{Name: "p", Provider: "openai_compatible", Model: "gpt-4o-mini", APIKey: "sk-test"}
	provider, err := NewProviderForProfile(p, cfg)
	if err != nil {
		t.Fatalf("NewProviderForProfile: %v", err)
	}
	op, ok := provider.(*OpenAICompatibleProvider)
	if !ok {
		t.Fatalf("expected OpenAICompatibleProvider, got %T", provider)
	}
	if op.client.Model != "gpt-4o-mini" {
		t.Fatalf("model: %q", op.client.Model)
	}
	if op.client.APIKey != "sk-test" {
		t.Fatalf("api key: %q", op.client.APIKey)
	}
	if op.client.BaseURL != "https://api.openai.com/v1" {
		t.Fatalf("default base url: %q", op.client.BaseURL)
	}
}

func TestNewProviderForProfile_GeminiRequiresKey(t *testing.T) {
	cfg := &config.Config{}
	_, err := NewProviderForProfile(config.AIProfile{Name: "g", Provider: "gemini"}, cfg)
	if err == nil {
		t.Fatal("expected error: missing API key")
	}
}

func TestNewProviderForProfile_Gemini(t *testing.T) {
	p := config.AIProfile{Name: "g", Provider: "gemini", Model: "gemini-2.5-flash", APIKey: "G-KEY"}
	provider, err := NewProviderForProfile(p, &config.Config{})
	if err != nil {
		t.Fatalf("NewProviderForProfile: %v", err)
	}
	gp, ok := provider.(*GeminiProvider)
	if !ok {
		t.Fatalf("expected GeminiProvider, got %T", provider)
	}
	if gp.APIKey != "G-KEY" {
		t.Fatalf("api key: %q", gp.APIKey)
	}
	if gp.Model != "gemini-2.5-flash" {
		t.Fatalf("model: %q", gp.Model)
	}
}

func TestNewProviderForProfile_OllamaPlaceholderBearer(t *testing.T) {
	cfg := &config.Config{}
	provider, err := NewProviderForProfile(
		config.AIProfile{Name: "local", Provider: "ollama"},
		cfg,
	)
	if err != nil {
		t.Fatalf("NewProviderForProfile: %v", err)
	}
	op, ok := provider.(*OpenAICompatibleProvider)
	if !ok {
		t.Fatalf("expected OpenAICompatibleProvider for ollama, got %T", provider)
	}
	if op.client.APIKey != config.OllamaOpenAICompatiblePlaceholderKey {
		t.Fatalf("ollama bearer: got %q", op.client.APIKey)
	}
	if op.client.BaseURL != config.OllamaDefaultChatBaseURL {
		t.Fatalf("ollama base: got %q", op.client.BaseURL)
	}
}

func TestNewProviderForProfile_EnvKeyFallback(t *testing.T) {
	t.Setenv(config.EnvAIAPIKey, "env-key")
	cfg := &config.Config{}
	provider, err := NewProviderForProfile(
		config.AIProfile{Name: "p", Provider: "openai_compatible", Model: "gpt-4o-mini"},
		cfg,
	)
	if err != nil {
		t.Fatalf("env key fallback: %v", err)
	}
	if provider.(*OpenAICompatibleProvider).client.APIKey != "env-key" {
		t.Fatalf("expected env-key, got %q", provider.(*OpenAICompatibleProvider).client.APIKey)
	}
}

// TestNewProviderForProfile_IgnoresCfgFlatFields ensures the profile's fields
// are used even when the cfg's flat AI* fields point to a different setup.
func TestNewProviderForProfile_IgnoresCfgFlatFields(t *testing.T) {
	cfg := &config.Config{
		AIProvider: "ollama",
		AIBaseURL:  "http://127.0.0.1:11434/v1",
		AIModel:    "qwen2.5:1.5b",
	}
	override := config.AIProfile{Name: "remote", Provider: "openai_compatible", Model: "gpt-4o", APIKey: "sk-x"}
	provider, err := NewProviderForProfile(override, cfg)
	if err != nil {
		t.Fatalf("NewProviderForProfile: %v", err)
	}
	op := provider.(*OpenAICompatibleProvider)
	if op.client.Model != "gpt-4o" {
		t.Fatalf("model from profile, got %q", op.client.Model)
	}
	if op.client.BaseURL != "https://api.openai.com/v1" {
		t.Fatalf("base from profile defaults, got %q", op.client.BaseURL)
	}
	if op.client.APIKey != "sk-x" {
		t.Fatalf("api key from profile, got %q", op.client.APIKey)
	}
}
