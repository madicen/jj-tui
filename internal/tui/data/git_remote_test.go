package data

import (
	"context"
	"strings"
	"testing"
)

// These tests cover the error-side branches of the Apply / Create / Remove / Push origin
// cmds. The happy paths shell out to `jj` and `gh` and are exercised in the integration test
// suite (`integration_tests/`); here we focus on the defensive checks each cmd performs before
// any subprocess invocation, since those are the branches where a regression would silently
// degrade the UX (e.g. a missing nil-service guard producing a panic instead of an error
// message).

// runCmd helper invokes a tea.Cmd's func and asserts the result is the expected message type.
func runCmd[T any](t *testing.T, cmd func() any) T {
	t.Helper()
	raw := cmd()
	v, ok := raw.(T)
	if !ok {
		t.Fatalf("cmd returned %T, want %T", raw, *new(T))
	}
	return v
}

// --- ApplyOriginCmd ---------------------------------------------------------------------------

// TestApplyOriginCmd_EmptyURL_ReturnsRoutingError verifies the "you cleared the field on a
// repo with no origin" path returns an actionable error rather than blindly running
// `jj git remote add origin ""`. Main routes empty URL + existing origin to RemoveOriginCmd
// instead; this case (no origin AND empty URL) deserves a clear error.
func TestApplyOriginCmd_EmptyURL_ReturnsRoutingError(t *testing.T) {
	t.Parallel()
	cmd := ApplyOriginCmd(nil, "")
	if cmd == nil {
		t.Fatalf("ApplyOriginCmd returned nil cmd; expected a func that emits an error message")
	}
	msg := runCmd[RemoteOpResultMsg](t, func() any { return cmd() })
	if msg.Err == nil {
		t.Fatalf("expected error for empty URL, got nil")
	}
	if !strings.Contains(msg.Err.Error(), "remote URL is empty") {
		t.Errorf("error should mention empty URL, got %q", msg.Err)
	}
	if msg.Op != RemoteOpApply {
		t.Errorf("Op = %v, want %v", msg.Op, RemoteOpApply)
	}
}

// TestApplyOriginCmd_MalformedInput_WhitespaceOnly is the "user pasted a tab/space accidentally
// and pressed Apply" case. The cmd must trim and treat it as empty, surfacing the same error
// rather than running `jj git remote add origin "  \t  "`.
func TestApplyOriginCmd_MalformedInput_WhitespaceOnly(t *testing.T) {
	t.Parallel()
	cmd := ApplyOriginCmd(nil, "   \t  \n  ")
	msg := runCmd[RemoteOpResultMsg](t, func() any { return cmd() })
	if msg.Err == nil {
		t.Fatalf("expected error for whitespace-only URL, got nil")
	}
	if !strings.Contains(msg.Err.Error(), "remote URL is empty") {
		t.Errorf("whitespace-only URL should map to empty-URL error path; got %q", msg.Err)
	}
}

// TestApplyOriginCmd_FailureScenario_NilService verifies that a nil jj service never panics —
// it returns a structured error so the UI can surface "jj service unavailable" rather than
// crashing the whole TUI. Realistically this path only fires in tests / startup races, but it
// matters because nil-deref crashes are the worst-class UX failure.
func TestApplyOriginCmd_FailureScenario_NilService(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ApplyOriginCmd(nil, valid-url) panicked: %v", r)
		}
	}()
	cmd := ApplyOriginCmd(nil, "git@github.com:owner/repo.git")
	msg := runCmd[RemoteOpResultMsg](t, func() any { return cmd() })
	if msg.Err == nil {
		t.Fatalf("expected error from nil service, got nil")
	}
}

// --- RemoveOriginCmd --------------------------------------------------------------------------

// TestRemoveOriginCmd_NoOriginConfigured returns a clear "no origin to remove" error rather
// than falling through to `jj git remote remove origin` and surfacing jj's less-friendly
// wording. Handles the user clicking Remove on an unconfigured repo.
func TestRemoveOriginCmd_NoOriginConfigured(t *testing.T) {
	t.Parallel()
	cmd := RemoveOriginCmd(nil)
	if cmd == nil {
		t.Fatalf("RemoveOriginCmd returned nil cmd")
	}
	msg := runCmd[RemoteOpResultMsg](t, func() any { return cmd() })
	if msg.Err == nil {
		t.Fatalf("expected error when no origin configured, got nil")
	}
	if !strings.Contains(msg.Err.Error(), "no `origin` remote to remove") {
		t.Errorf("error should mention no origin to remove, got %q", msg.Err)
	}
	if msg.Op != RemoteOpRemove {
		t.Errorf("Op = %v, want %v", msg.Op, RemoteOpRemove)
	}
}

// TestRemoveOriginCmd_FailureScenario_NilService confirms no panic. The nil-service path lands
// in readOriginURL which short-circuits to ""; the "no origin" branch then fires.
func TestRemoveOriginCmd_FailureScenario_NilService(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RemoveOriginCmd(nil) panicked: %v", r)
		}
	}()
	msg := runCmd[RemoteOpResultMsg](t, func() any { return RemoveOriginCmd(nil)() })
	if msg.Err == nil {
		t.Errorf("expected an error message; got nil")
	}
}

// --- CreateGhRepoCmd --------------------------------------------------------------------------

// TestCreateGhRepoCmd_GhMissingFromPath returns a "gh CLI not found" error before any other
// step. Pinning this matters because the panel renders a different copy when gh is missing,
// and the cmd must agree with the panel's preflight check; a regression here would let users
// click Create when no button was rendered.
func TestCreateGhRepoCmd_GhMissingFromPath(t *testing.T) {
	// NOTE: deliberately not t.Parallel() — t.Setenv requires sequential execution. The pair
	// (this test + TestCreateGhRepoCmd_FailureScenario_NilServiceAfterPreflight) run serially
	// because they both stub PATH; everything else in this file is parallel-safe.
	t.Setenv("PATH", "")
	cmd := CreateGhRepoCmd(nil, "myrepo", true)
	msg := runCmd[RemoteOpResultMsg](t, func() any { return cmd() })
	if msg.Err == nil {
		t.Fatalf("expected gh-missing error, got nil")
	}
	if !strings.Contains(msg.Err.Error(), "gh CLI not found in PATH") {
		t.Errorf("error should mention gh missing, got %q", msg.Err)
	}
	if msg.Op != RemoteOpCreateGh {
		t.Errorf("Op = %v, want %v", msg.Op, RemoteOpCreateGh)
	}
}

// TestCreateGhRepoCmd_FailureScenario_NilServiceAfterPreflight: with gh missing the cmd returns
// before touching the jj service, so this test only meaningfully runs when gh is on PATH. Use
// a clearly-impossible PATH to consistently exercise the early-return path; the assertion is
// about the absence of a panic.
func TestCreateGhRepoCmd_FailureScenario_NilServiceAfterPreflight(t *testing.T) {
	// Deliberately not t.Parallel() — see TestCreateGhRepoCmd_GhMissingFromPath note above.
	t.Setenv("PATH", "")
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("CreateGhRepoCmd(nil) panicked: %v", r)
		}
	}()
	_ = runCmd[RemoteOpResultMsg](t, func() any { return CreateGhRepoCmd(nil, "name", true)() })
}

// --- PushBookmarksCmd -------------------------------------------------------------------------

// TestPushBookmarksCmd_NoOriginConfigured returns a friendly error pointing the user at the
// Apply / Create flow instead of executing `jj git push` against a non-existent remote. This
// is the most common pre-Apply user mistake.
func TestPushBookmarksCmd_NoOriginConfigured_PushAll(t *testing.T) {
	t.Parallel()
	cmd := PushBookmarksCmd(nil, true)
	if cmd == nil {
		t.Fatalf("PushBookmarksCmd returned nil cmd")
	}
	msg := runCmd[PushResultMsg](t, func() any { return cmd() })
	if msg.Err == nil {
		t.Fatalf("expected error when no origin configured, got nil")
	}
	if !strings.Contains(msg.Err.Error(), "no `origin` remote configured") {
		t.Errorf("error should point user at Apply/Create, got %q", msg.Err)
	}
	if !msg.All {
		t.Errorf("All flag should round-trip; got false")
	}
}

// TestPushBookmarksCmd_NoOriginConfigured_PushCurrent: same guard as the all-bookmarks case.
// Both push variants must short-circuit on missing origin so the failure modes are symmetric.
func TestPushBookmarksCmd_NoOriginConfigured_PushCurrent(t *testing.T) {
	t.Parallel()
	msg := runCmd[PushResultMsg](t, func() any { return PushBookmarksCmd(nil, false)() })
	if msg.Err == nil {
		t.Fatalf("expected error, got nil")
	}
	if msg.All {
		t.Errorf("All flag = true, want false")
	}
}

// TestPushBookmarksCmd_FailureScenario_NilService: defensive nil-deref guard. Nil service
// surfaces via readOriginURL → "" → the "no origin" branch fires; we must never panic.
func TestPushBookmarksCmd_FailureScenario_NilService(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PushBookmarksCmd(nil) panicked: %v", r)
		}
	}()
	msg := runCmd[PushResultMsg](t, func() any { return PushBookmarksCmd(nil, true)() })
	if msg.Err == nil {
		t.Errorf("expected an error message; got nil")
	}
}

// --- listLocalBookmarks -----------------------------------------------------------------------

// TestListLocalBookmarks_NilService returns an empty slice rather than panicking. The caller
// (CreateGhRepoCmd's auto-push branch) treats an empty result as "nothing to push", which is
// the exact intent for the welcome-screen first-init case.
func TestListLocalBookmarks_NilService(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("listLocalBookmarks(nil) panicked: %v", r)
		}
	}()
	got := listLocalBookmarks(context.Background(), nil)
	if got != nil {
		t.Errorf("listLocalBookmarks(ctx, nil) = %v, want nil", got)
	}
}

// --- readOriginURL ----------------------------------------------------------------------------

// TestReadOriginURL_NilService returns ("", err) — the caller treats any error as "no origin",
// so this is functionally equivalent to "(none configured)" in the panel. Important: it must
// not panic.
func TestReadOriginURL_NilService(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("readOriginURL(ctx, nil) panicked: %v", r)
		}
	}()
	url, err := readOriginURL(context.Background(), nil)
	if url != "" {
		t.Errorf("readOriginURL(ctx, nil) URL = %q, want \"\"", url)
	}
	if err == nil {
		t.Errorf("readOriginURL(ctx, nil) err = nil, want non-nil")
	}
}

// --- RemoteOp / RemoteOpResultMsg / PushResultMsg zero values --------------------------------

// TestRemoteOpResultMsg_ZeroValues verifies the zero value of the result struct corresponds to
// "RemoteOpApply success with no changes" — a sensible-looking default. This is a regression
// guard: if someone reorders the RemoteOp constants and RemoteOpApply stops being iota=0, every
// place that constructs RemoteOpResultMsg{} without explicitly setting Op would silently
// flip to a different op kind.
func TestRemoteOpResultMsg_ZeroValues(t *testing.T) {
	t.Parallel()
	var msg RemoteOpResultMsg
	if msg.Op != RemoteOpApply {
		t.Errorf("zero-value Op = %v, want %v (constant ordering changed?)", msg.Op, RemoteOpApply)
	}
	if msg.Err != nil || msg.PushErr != nil {
		t.Errorf("zero-value should have no errors")
	}
	if msg.PushedCount != 0 || len(msg.PushedNames) != 0 {
		t.Errorf("zero-value should have empty push outcome")
	}
}
