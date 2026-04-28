package llm

import (
	"fmt"
	"net/http"
	"strings"
)

// failureHints returns user-facing sentences when the response looks like throttling, overload, or quota exhaustion.
func failureHints(httpStatus int, blobs ...string) string {
	low := strings.ToLower(strings.Join(blobs, " "))
	var out []string

	if httpStatus == http.StatusTooManyRequests ||
		strings.Contains(low, "rate_limit") ||
		strings.Contains(low, "ratelimit") ||
		strings.Contains(low, "too many requests") ||
		strings.Contains(low, "throttl") {
		out = append(out, "This response looks like rate limiting or throttling. Wait and retry, try a lighter model, or raise provider limits.")
	}
	if httpStatus == http.StatusBadGateway || httpStatus == http.StatusServiceUnavailable ||
		strings.Contains(low, "overloaded") ||
		strings.Contains(low, "unavailable") ||
		strings.Contains(low, "try again later") {
		out = append(out, "The provider may be overloaded or temporarily unavailable (502/503). Retrying after a short wait often helps.")
	}
	if strings.Contains(low, "insufficient_quota") ||
		strings.Contains(low, "billing_hard_limit") ||
		strings.Contains(low, "exceeded your current quota") ||
		(strings.Contains(low, "quota") && strings.Contains(low, "exceeded")) {
		out = append(out, "Billing or usage quota may be exhausted — check the provider dashboard, plan, and API key.")
	}
	if strings.Contains(low, "resource_exhausted") ||
		strings.Contains(low, "resource exhausted") {
		out = append(out, "Resource exhausted (common on free tiers) — shorten prompts, wait, or upgrade the API project.")
	}

	if len(out) == 0 {
		return ""
	}
	seen := make(map[string]struct{})
	var uniq []string
	for _, s := range out {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		uniq = append(uniq, s)
	}
	return strings.Join(uniq, " ")
}

func formatLLMHTTPUserError(opName string, httpStatus int, bodySnippet string) string {
	base := fmt.Sprintf("%s HTTP %d: %s", opName, httpStatus, bodySnippet)
	if hint := failureHints(httpStatus, bodySnippet); hint != "" {
		return hint + "\n\n" + base
	}
	return base
}

func formatLLMExhaustedRetriesError(opName string, lastHTTP int, bodySnippet string) string {
	base := fmt.Sprintf("%s: exhausted retries (last HTTP %d: %s)", opName, lastHTTP, bodySnippet)
	if hint := failureHints(lastHTTP, bodySnippet); hint != "" {
		return hint + "\n\n" + base
	}
	return base
}

func formatOpenAIChatAPIError(message, errorType string, rawBody []byte) string {
	base := fmt.Sprintf("LLM API error: %s", message)
	blobs := []string{message, errorType, string(rawBody)}
	if hint := failureHints(200, blobs...); hint != "" {
		return hint + "\n\n" + base
	}
	return base
}

func formatGeminiAPIError(message string, rawBody []byte) string {
	base := fmt.Sprintf("gemini API error: %s", message)
	if hint := failureHints(200, message, string(rawBody)); hint != "" {
		return hint + "\n\n" + base
	}
	return base
}
