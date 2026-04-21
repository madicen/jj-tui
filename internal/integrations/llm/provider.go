package llm

import "context"

// Provider is a minimal interface for generating text from prompts.
// Implementations may use different backends (OpenAI-compatible, Gemini, etc.).
type Provider interface {
	Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}
