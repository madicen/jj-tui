package llm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const llmMaxHTTPRetries = 6

func isRetryableLLMHTTPStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests, http.StatusInternalServerError,
		http.StatusBadGateway, http.StatusServiceUnavailable:
		return true
	default:
		return false
	}
}

func isRetryableNetErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var ne net.Error
	if errors.As(err, &ne) {
		return ne.Timeout() || ne.Temporary()
	}
	return false
}

// parseRetryAfterWait parses Retry-After as delta-seconds or HTTP-date (RFC 7231).
func parseRetryAfterWait(header string) time.Duration {
	header = strings.TrimSpace(header)
	if header == "" {
		return 0
	}
	if n, err := strconv.Atoi(header); err == nil && n >= 0 {
		return time.Duration(n) * time.Second
	}
	if t, err := http.ParseTime(header); err == nil {
		d := time.Until(t)
		if d < 0 {
			return 0
		}
		return d
	}
	return 0
}

func retryBackoff(attempt int) time.Duration {
	d := time.Second
	for i := 0; i < attempt && d < 30*time.Second; i++ {
		d *= 2
	}
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	return d
}

func mergeRetryWait(retryAfterHeader string, attempt int) time.Duration {
	fromHeader := parseRetryAfterWait(retryAfterHeader)
	b := retryBackoff(attempt)
	w := b
	if fromHeader > w {
		w = fromHeader
	}
	if w > 2*time.Minute {
		w = 2 * time.Minute
	}
	if w < 250*time.Millisecond {
		w = 250 * time.Millisecond
	}
	return w
}

func sleepContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// withLLMHTTPRetry runs doReq until a 2xx response body is returned, or gives up on non-retryable HTTP / exhausted retries.
func withLLMHTTPRetry(ctx context.Context, opName string, doReq func(context.Context) (*http.Response, error)) ([]byte, error) {
	var lastStatus int
	var lastSnippet string
	for attempt := 0; attempt < llmMaxHTTPRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		resp, err := doReq(ctx)
		if err != nil {
			if attempt < llmMaxHTTPRetries-1 && isRetryableNetErr(err) {
				if err := sleepContext(ctx, mergeRetryWait("", attempt)); err != nil {
					return nil, err
				}
				continue
			}
			return nil, err
		}
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return body, nil
		}
		lastStatus = resp.StatusCode
		lastSnippet = truncate(string(body), 500)
		if !isRetryableLLMHTTPStatus(resp.StatusCode) || attempt == llmMaxHTTPRetries-1 {
			return nil, fmt.Errorf("%s", formatLLMHTTPUserError(opName, resp.StatusCode, lastSnippet))
		}
		if err := sleepContext(ctx, mergeRetryWait(resp.Header.Get("Retry-After"), attempt)); err != nil {
			return nil, err
		}
	}
	return nil, fmt.Errorf("%s", formatLLMExhaustedRetriesError(opName, lastStatus, lastSnippet))
}
