package settings

import (
	"strings"
	"testing"
)

// TestRenderRepositoryRemote_NoOrigin verifies that the new Repository remote panel renders the
// expected affordances when no `origin` is configured: the explanatory copy, "(none configured)"
// label, the URL input row, and Add origin / Remove (n/a) buttons. This serves as a regression
// fence so README content stays in sync with the actual UI; if any of these strings disappear,
// the README reference becomes a lie and the test catches it.
func TestRenderRepositoryRemote_NoOrigin(t *testing.T) {
	t.Parallel()

	r := renderCtx{} // no zone manager: mark() falls through unchanged.
	data := RenderData{
		CurrentOrigin:   "",
		OriginInputView: "[urlinput]",
		GhAvailable:     true,
		GhRepoPrivate:   true,
		FocusedField:    0,
	}
	out := strings.Join(r.renderRepositoryRemote(data), "\n")

	want := []string{
		"Repository remote",
		"(none configured)",
		"Remote URL:",
		"[urlinput]",
		"Add origin (^enter)",
		"[Remove origin (n/a)]", // disabled-style label when nothing to remove
		// Push buttons are visible even when origin is missing, but rendered with the
		// "set origin first" hint so the user knows where to start.
		"Push current bookmark (p) — set origin first",
		"Push all bookmarks (P) — set origin first",
		"Or create a brand-new GitHub repo",
		"Create new GitHub repo (g)",
		"Visibility: Private  (^v)",
		"and then pushes all", // copy advertising the auto-push behaviour
	}
	for _, s := range want {
		if !strings.Contains(out, s) {
			t.Errorf("renderRepositoryRemote (no origin) missing fragment %q\nfull output:\n%s", s, out)
		}
	}
}

// TestRenderRepositoryRemote_HasOrigin covers the populated-origin variant: the live URL is
// shown, the Apply button switches its label, and the remove button is enabled.
func TestRenderRepositoryRemote_HasOrigin(t *testing.T) {
	t.Parallel()

	r := renderCtx{}
	data := RenderData{
		CurrentOrigin:   "git@github.com:owner/repo.git",
		OriginInputView: "[urlinput]",
		GhAvailable:     true,
		GhRepoPrivate:   false,
		FocusedField:    5,
	}
	out := strings.Join(r.renderRepositoryRemote(data), "\n")

	want := []string{
		"git@github.com:owner/repo.git",
		"Update origin (^enter)",
		"Remove origin (^x)",
		"Visibility: Public  (^v)",
		// With origin configured, the push buttons render their actionable labels (no hint).
		"Push current bookmark (p)",
		"Push all bookmarks (P)",
	}
	for _, s := range want {
		if !strings.Contains(out, s) {
			t.Errorf("renderRepositoryRemote (with origin) missing fragment %q\nfull output:\n%s", s, out)
		}
	}
	// Disabled-look "Remove (n/a)" must NOT appear when an origin is configured.
	if strings.Contains(out, "(n/a)") {
		t.Errorf("renderRepositoryRemote (with origin) should not show the disabled (n/a) label; got:\n%s", out)
	}
	// Likewise, the push buttons must NOT render the "set origin first" hint when origin
	// exists; if the gating logic regresses this catches it.
	if strings.Contains(out, "set origin first") {
		t.Errorf("renderRepositoryRemote (with origin) should not show the disabled push hint; got:\n%s", out)
	}
}

// TestRenderRepositoryRemote_NoGhCli verifies the "gh CLI not found" hint is rendered in place
// of the Create / Visibility buttons when gh isn't on PATH.
func TestRenderRepositoryRemote_NoGhCli(t *testing.T) {
	t.Parallel()

	r := renderCtx{}
	data := RenderData{
		CurrentOrigin:   "",
		OriginInputView: "[urlinput]",
		GhAvailable:     false,
	}
	out := strings.Join(r.renderRepositoryRemote(data), "\n")

	if !strings.Contains(out, "`gh` CLI not found in PATH") {
		t.Errorf("expected gh-not-found hint when GhAvailable=false; got:\n%s", out)
	}
	if strings.Contains(out, "Create new GitHub repo (g)") {
		t.Errorf("Create button should be hidden when gh is unavailable; got:\n%s", out)
	}
}
