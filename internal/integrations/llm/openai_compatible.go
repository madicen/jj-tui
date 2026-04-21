package llm

import (
	"context"
	"time"
)

// OpenAICompatibleProvider wraps Client to satisfy Provider.
type OpenAICompatibleProvider struct {
	client *Client
}

func NewOpenAICompatibleProvider(baseURL, apiKey, model string, timeout time.Duration) Provider {
	return &OpenAICompatibleProvider{client: NewClient(baseURL, apiKey, model, timeout)}
}

func (p *OpenAICompatibleProvider) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return p.client.Complete(ctx, systemPrompt, userPrompt)
}
