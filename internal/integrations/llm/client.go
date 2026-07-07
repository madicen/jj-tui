// Package llm provides a minimal OpenAI-compatible chat completions client.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/madicen/jj-tui/internal/integrations/httpapi"
)

// Client calls POST {base}/chat/completions with Bearer auth.
type Client struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

// NewClient returns a client with sensible defaults. baseURL should be like https://api.openai.com/v1 (no trailing slash).
func NewClient(baseURL, apiKey, model string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &Client{
		BaseURL: strings.TrimSuffix(strings.TrimSpace(baseURL), "/"),
		APIKey:  strings.TrimSpace(apiKey),
		Model:   strings.TrimSpace(model),
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// Complete sends a chat completion request and returns the assistant message content (trimmed).
func (c *Client) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if c == nil || c.APIKey == "" {
		return "", fmt.Errorf("LLM client: missing API key")
	}
	if c.BaseURL == "" {
		return "", fmt.Errorf("LLM client: missing base URL")
	}
	model := c.Model
	if model == "" {
		model = "gpt-4o-mini"
	}
	body := chatRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.3,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	url := c.BaseURL + "/chat/completions"
	// Request construction + auth decoration come from the shared httpapi base;
	// retry/backoff and body/status handling stay in withLLMHTTPRetry because the
	// LLM path needs Retry-After honoring and 2xx (not just 200) success.
	hc := &httpapi.Client{
		Provider: "LLM",
		HTTP:     c.HTTPClient,
		Decorate: func(req *http.Request) {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+c.APIKey)
		},
	}
	respBody, err := withLLMHTTPRetry(ctx, "LLM", func(reqCtx context.Context) (*http.Response, error) {
		return hc.Do(reqCtx, http.MethodPost, url, bytes.NewReader(raw))
	})
	if err != nil {
		return "", err
	}
	return parseOpenAIChatCompletionBody(respBody)
}

func parseOpenAIChatCompletionBody(respBody []byte) (string, error) {
	var parsed chatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("LLM response JSON: %w", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", fmt.Errorf("%s", formatOpenAIChatAPIError(parsed.Error.Message, parsed.Error.Type, respBody))
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("LLM: empty response")
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "…"
}
