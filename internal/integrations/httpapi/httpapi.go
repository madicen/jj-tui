// Package httpapi holds a small shared HTTP client used by the ticket-provider
// integrations (Jira, Codecks). It centralises the boilerplate those services
// previously hand-rolled: base-URL normalization, request building with a
// per-request auth decorator, executing the request against an *http.Client,
// reading the body, and mapping a non-2xx status into a consistent error.
package httpapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// NormalizeBaseURL trims a trailing slash so an endpoint path can be
// concatenated directly onto the base.
func NormalizeBaseURL(base string) string {
	return strings.TrimSuffix(base, "/")
}

// Client wraps an *http.Client with an auth decorator and shared error mapping.
// It intentionally exposes low-level primitives (Do) alongside convenience
// helpers (DoRead, EnsureOK) so callers with different body-handling needs can
// pick the right level.
type Client struct {
	// Provider names the integration for error messages, e.g. "jira".
	Provider string
	// HTTP is the underlying client; http.DefaultClient is used when nil.
	HTTP *http.Client
	// Decorate applies authentication and any required headers to every
	// outgoing request. It may be nil.
	Decorate func(*http.Request)
}

// Do builds a request for url with the given method/body, applies the auth
// decorator, and executes it. The caller owns resp.Body.
func (c *Client) Do(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	if c.Decorate != nil {
		c.Decorate(req)
	}
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	return hc.Do(req)
}

// StatusError formats a consistent "<provider> API error (status N): body"
// error.
func (c *Client) StatusError(resp *http.Response, body []byte) error {
	return fmt.Errorf("%s API error (status %d): %s", c.Provider, resp.StatusCode, string(body))
}

// EnsureOK returns a StatusError when resp is not 200 OK, reading the body for
// the message. On 200 it returns nil and leaves resp.Body unread so the caller
// can still stream-decode it.
func (c *Client) EnsureOK(resp *http.Response) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return c.StatusError(resp, body)
}

// DoRead executes the request, reads and closes the body, and returns the raw
// bytes. Any non-200 status yields a StatusError carrying the body.
func (c *Client) DoRead(ctx context.Context, method, url string, body io.Reader) ([]byte, error) {
	resp, err := c.Do(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.StatusError(resp, respBody)
	}
	return respBody, nil
}
