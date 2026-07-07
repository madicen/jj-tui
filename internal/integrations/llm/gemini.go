package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/madicen/jj-tui/internal/integrations/httpapi"
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

	targetURL := u.String()
	// Reuse the shared httpapi base for request construction; Gemini authenticates
	// via the ?key= query param set above, so no auth header decoration is needed.
	hc := &httpapi.Client{
		Provider: "gemini",
		HTTP:     p.HTTPClient,
		Decorate: func(req *http.Request) {
			req.Header.Set("Content-Type", "application/json")
		},
	}
	respBody, err := withLLMHTTPRetry(ctx, "gemini", func(reqCtx context.Context) (*http.Response, error) {
		return hc.Do(reqCtx, http.MethodPost, targetURL, bytes.NewReader(raw))
	})
	if err != nil {
		return "", err
	}
	return parseGeminiGenerateBody(respBody)
}

func parseGeminiGenerateBody(respBody []byte) (string, error) {
	var parsed geminiGenerateResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("gemini response JSON: %w", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", fmt.Errorf("%s", formatGeminiAPIError(parsed.Error.Message, respBody))
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
