package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultGeminiBase = "https://generativelanguage.googleapis.com/v1beta"

// GeminiProvider calls Google Generative Language API generateContent.
type GeminiProvider struct {
	BaseURL    string // e.g. https://generativelanguage.googleapis.com/v1beta
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

// NewGeminiProvider returns a provider for Gemini. apiKey is a Google AI Studio / Gemini API key.
// model is the model id segment (e.g. gemini-2.5-flash).
func NewGeminiProvider(apiKey, model string, timeout time.Duration) *GeminiProvider {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	base := strings.TrimSuffix(strings.TrimSpace(defaultGeminiBase), "/")
	return &GeminiProvider{
		BaseURL: base,
		APIKey:  strings.TrimSpace(apiKey),
		Model:   strings.TrimSpace(model),
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

type geminiGenerateRequest struct {
	SystemInstruction *geminiContent  `json:"systemInstruction,omitempty"`
	Contents          []geminiContent `json:"contents"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerateResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (p *GeminiProvider) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if p == nil || p.APIKey == "" {
		return "", fmt.Errorf("gemini: missing API key")
	}
	model := p.Model
	if model == "" {
		model = "gemini-2.5-flash"
	}
	base := p.BaseURL
	if base == "" {
		base = defaultGeminiBase
	}
	base = strings.TrimSuffix(strings.TrimSpace(base), "/")

	u, err := url.Parse(base + "/models/" + url.PathEscape(model) + ":generateContent")
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("key", p.APIKey)
	u.RawQuery = q.Encode()

	body := geminiGenerateRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: userPrompt}}},
		},
	}
	if strings.TrimSpace(systemPrompt) != "" {
		body.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: systemPrompt}},
		}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := p.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("gemini HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 500))
	}
	var parsed geminiGenerateResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("gemini response JSON: %w", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", fmt.Errorf("gemini API error: %s", parsed.Error.Message)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: empty response")
	}
	out := strings.TrimSpace(parsed.Candidates[0].Content.Parts[0].Text)
	if out == "" {
		return "", fmt.Errorf("gemini: empty text")
	}
	return out, nil
}
