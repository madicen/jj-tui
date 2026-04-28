package llm

import (
	"strings"
	"testing"
)

func TestFailureHints_Rate429Body(t *testing.T) {
	h := failureHints(429, `{"error":{"type":"rate_limit_exceeded"}}`)
	if h == "" || !strings.Contains(strings.ToLower(h), "rate") {
		t.Fatalf("got %q", h)
	}
}

func TestFailureHints_Quota(t *testing.T) {
	h := failureHints(200, `{"error":{"type":"insufficient_quota","message":"You exceeded your current quota"}}`)
	if h == "" || !strings.Contains(strings.ToLower(h), "quota") {
		t.Fatalf("got %q", h)
	}
}

func TestFailureHints_GeminiResource(t *testing.T) {
	h := failureHints(429, "RESOURCE_EXHAUSTED: Resource has been exhausted")
	if h == "" {
		t.Fatal("expected hint")
	}
	if !strings.Contains(strings.ToLower(h), "resource") {
		t.Fatalf("got %q", h)
	}
}

func TestFormatLLMHTTPUserError_429(t *testing.T) {
	s := formatLLMHTTPUserError("LLM", 429, "{}")
	if !strings.Contains(s, "429") || !strings.Contains(strings.ToLower(s), "rate") {
		t.Fatalf("got %q", s)
	}
}
