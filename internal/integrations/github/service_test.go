package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	gogithub "github.com/google/go-github/v66/github"
)

// These tests cover the new error-summary / retry-decision / default-branch helpers added
// alongside the "failed to create PR: 422 Validation Failed [Field:base]" fix. They focus on
// the small chunks of logic that don't require a real GitHub API: pure transformations of
// error shapes, plus the cached HTTP path mocked with httptest. The happy-path Create / List
// flows shell out through the live REST client and are exercised in integration_tests/.

// --- summarize422 -----------------------------------------------------------------------------

// TestSummarize422_PlainError returns the underlying message when err isn't a go-github
// ErrorResponse. This is the fallback path for non-API errors that still happen to bubble up
// through the same code path (e.g. URL parse failures, transport timeouts).
func TestSummarize422_PlainError(t *testing.T) {
	t.Parallel()
	got := summarize422(errors.New("network blip"))
	if got != "network blip" {
		t.Errorf("summarize422(plain) = %q, want %q", got, "network blip")
	}
}

// TestSummarize422_ErrorResponseWithoutFields uses the parent ErrorResponse Message when the
// structured Errors array is empty. GitHub sometimes returns 422 with only a top-level
// message and no per-field array, so this case must still produce something readable.
func TestSummarize422_ErrorResponseWithoutFields(t *testing.T) {
	t.Parallel()
	resp := &gogithub.ErrorResponse{
		Response: &http.Response{StatusCode: 422, Request: &http.Request{Method: "POST", URL: mustURL(t, "https://api.github.com/repos/x/y/pulls")}},
		Message:  "Validation Failed",
	}
	got := summarize422(resp)
	if got == "" {
		t.Fatalf("summarize422 returned empty string for ErrorResponse without fields")
	}
	if !strings.Contains(got, "Validation Failed") {
		t.Errorf("expected message to contain top-level Message, got %q", got)
	}
}

// TestSummarize422_ErrorResponseWithFields formats each per-field error into a readable
// segment. This is the case that motivated the helper — the user's reported error
// `[{Resource:PullRequest Field:base Code:invalid Message:}]` should turn into something
// scannable like "PullRequest.base=invalid".
func TestSummarize422_ErrorResponseWithFields(t *testing.T) {
	t.Parallel()
	resp := &gogithub.ErrorResponse{
		Response: &http.Response{StatusCode: 422, Request: &http.Request{Method: "POST", URL: mustURL(t, "https://api.github.com/repos/x/y/pulls")}},
		Message:  "Validation Failed",
		Errors: []gogithub.Error{
			{Resource: "PullRequest", Field: "base", Code: "invalid"},
		},
	}
	got := summarize422(resp)
	if !strings.Contains(got, "PullRequest.base=invalid") {
		t.Errorf("expected formatted segment, got %q", got)
	}
	if !strings.Contains(got, "Validation Failed") {
		t.Errorf("expected top-level Message, got %q", got)
	}
}

// TestSummarize422_PreservesPerErrorMessage retains the per-field Message when GitHub
// supplies one (e.g. "head and base must be different"). The Message often carries the most
// actionable wording, so the formatter must not drop it.
func TestSummarize422_PreservesPerErrorMessage(t *testing.T) {
	t.Parallel()
	resp := &gogithub.ErrorResponse{
		Response: &http.Response{StatusCode: 422, Request: &http.Request{Method: "POST", URL: mustURL(t, "https://api.github.com/repos/x/y/pulls")}},
		Message:  "Validation Failed",
		Errors: []gogithub.Error{
			{Resource: "PullRequest", Field: "head", Code: "invalid", Message: "head sha can't be blank"},
		},
	}
	got := summarize422(resp)
	if !strings.Contains(got, "head sha can't be blank") {
		t.Errorf("expected per-field Message to be preserved, got %q", got)
	}
}

// --- isHeadRefRetryable -----------------------------------------------------------------------

// TestIsHeadRefRetryable_BaseError refuses to retry when the structured error array points at
// the base field. Retrying a base-related 422 just spins for 15s and surfaces the same error;
// this is the regression that the user reported.
func TestIsHeadRefRetryable_BaseError(t *testing.T) {
	t.Parallel()
	resp := &gogithub.ErrorResponse{
		Response: &http.Response{StatusCode: 422, Request: &http.Request{Method: "POST", URL: mustURL(t, "https://api.github.com/repos/x/y/pulls")}},
		Errors:   []gogithub.Error{{Resource: "PullRequest", Field: "base", Code: "invalid"}},
	}
	if isHeadRefRetryable(resp, summarize422(resp)) {
		t.Errorf("base-field 422 must not be retried")
	}
}

// TestIsHeadRefRetryable_HeadRefError retries the legacy "head ref needs owner:branch" case.
// This is the original purpose of the retry path and must keep working.
func TestIsHeadRefRetryable_HeadRefError(t *testing.T) {
	t.Parallel()
	err := errors.New("422 Validation Failed: not all refs are reachable")
	if !isHeadRefRetryable(err, err.Error()) {
		t.Errorf("head/refs 422 should be retryable")
	}
}

// TestIsHeadRefRetryable_NonValidationError treats unrelated errors as non-retryable so we
// don't over-retry on unrelated 4xx/5xx (the legacy code only retried on 422).
func TestIsHeadRefRetryable_NonValidationError(t *testing.T) {
	t.Parallel()
	err := errors.New("connection refused")
	if isHeadRefRetryable(err, err.Error()) {
		t.Errorf("non-validation error should not be retryable via the head-ref path")
	}
}

// TestIsHeadRefRetryable_BaseInDetailString catches the case where the API returns base=...
// inside a freeform string rather than the structured Errors array. This is defensive — if
// summarize422 has already reduced the error to its detail line, isHeadRefRetryable should
// still pick out the base signal.
func TestIsHeadRefRetryable_BaseInDetailString(t *testing.T) {
	t.Parallel()
	err := errors.New("opaque")
	detail := "Validation Failed: PullRequest.base=invalid"
	if isHeadRefRetryable(err, detail) {
		t.Errorf("detail string mentioning .base= should not be retryable")
	}
}

// --- GetDefaultBranch -------------------------------------------------------------------------

// TestGetDefaultBranch_HappyPathAndCache verifies the helper returns the API-supplied default
// branch and caches it (a second call must not hit the server). The cache is the whole point
// of the helper — without it, every PR-form open would issue a /repos/{owner}/{repo} request.
func TestGetDefaultBranch_HappyPathAndCache(t *testing.T) {
	t.Parallel()
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		// go-github's REST URL for repo lookup is /repos/{owner}/{repo}
		if r.Method != http.MethodGet || r.URL.Path != "/repos/owner/repo" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"default_branch":"trunk"}`)
	}))
	defer server.Close()

	svc := newTestServiceWithBaseURL(t, "owner", "repo", server.URL)
	for i := 0; i < 3; i++ {
		got, err := svc.GetDefaultBranch(context.Background())
		if err != nil {
			t.Fatalf("call %d: GetDefaultBranch err = %v", i, err)
		}
		if got != "trunk" {
			t.Errorf("call %d: GetDefaultBranch = %q, want %q", i, got, "trunk")
		}
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("GetDefaultBranch hit the server %d times, want 1 (caching broken)", got)
	}
}

// TestGetDefaultBranch_EmptyResponseDoesNotCache: an empty default_branch happens for repos
// with no commits. We must not cache "" — the user is likely about to push a first branch
// and should be able to retry the lookup once they do.
func TestGetDefaultBranch_EmptyResponseDoesNotCache(t *testing.T) {
	t.Parallel()
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"default_branch":""}`)
	}))
	defer server.Close()

	svc := newTestServiceWithBaseURL(t, "owner", "repo", server.URL)
	for i := 0; i < 2; i++ {
		got, err := svc.GetDefaultBranch(context.Background())
		if err != nil {
			t.Fatalf("call %d: GetDefaultBranch err = %v", i, err)
		}
		if got != "" {
			t.Errorf("call %d: GetDefaultBranch = %q, want empty", i, got)
		}
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Errorf("empty response should not be cached; got %d hits, want 2", got)
	}
}

// TestGetDefaultBranch_AuthErrorPropagates verifies a 401 surfaces as AuthError so the UI's
// existing reauth flow can pick it up. Without this branch, the lookup would fail silently
// and the form would show a stale default branch with no clear next step for the user.
func TestGetDefaultBranch_AuthErrorPropagates(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"message":"Bad credentials"}`)
	}))
	defer server.Close()

	svc := newTestServiceWithBaseURL(t, "owner", "repo", server.URL)
	got, err := svc.GetDefaultBranch(context.Background())
	if err == nil {
		t.Fatalf("expected error for 401 response, got nil (default=%q)", got)
	}
	if !IsAuthError(err) {
		t.Errorf("expected IsAuthError=true, got %q", err)
	}
}

// TestGetDefaultBranch_NilService returns a clear error rather than panicking. This matches
// the defensive-nil-guard pattern the rest of the codebase uses (data/git_remote_test.go has
// the same pattern for its commands) and keeps the failure mode actionable.
func TestGetDefaultBranch_NilService(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("GetDefaultBranch on nil service panicked: %v", r)
		}
	}()
	var s *Service
	got, err := s.GetDefaultBranch(context.Background())
	if err == nil {
		t.Fatalf("expected error for nil service, got nil (default=%q)", got)
	}
}

// --- helpers ----------------------------------------------------------------------------------

// newTestServiceWithBaseURL wires a Service to point at a test HTTP server. Used by the
// GetDefaultBranch tests so we can mock GitHub API responses without exporting any of the
// service internals to other packages.
func newTestServiceWithBaseURL(t *testing.T, owner, repo, baseURL string) *Service {
	t.Helper()
	svc, err := NewServiceWithToken(owner, repo, "fake-token")
	if err != nil {
		t.Fatalf("NewServiceWithToken: %v", err)
	}
	parsed, err := url.Parse(baseURL + "/")
	if err != nil {
		t.Fatalf("invalid test URL: %v", err)
	}
	svc.client.BaseURL = parsed
	return svc
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("invalid URL %q: %v", raw, err)
	}
	return u
}
