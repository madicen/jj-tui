package github

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/config"
)

// These tests cover the new origin-remote-management behaviour added to the GitHub settings
// sub-model. The data-layer tea.Cmds (ApplyOriginCmd / RemoveOriginCmd / CreateGhRepoCmd /
// PushBookmarksCmd) shell out to `jj` and `gh` and live in package data; their pure-logic
// branches are exercised in `internal/tui/data/git_remote_test.go`. The tests below focus on
// the in-memory state surface this package owns: focusedField, the two text inputs, the
// visibility flag, and the cached origin URL displayed above the input.

// --- GetOriginURL / SetOriginURL ------------------------------------------------------------

// TestSetGetOriginURL_HappyPath rounds the value through the textinput (which is what the parent
// reads when dispatching NavigateRemoteApply), confirming the accessor pair is consistent.
func TestSetGetOriginURL_HappyPath(t *testing.T) {
	t.Parallel()
	m := NewModel()
	m.SetOriginURL("git@github.com:owner/repo.git")
	if got, want := m.GetOriginURL(), "git@github.com:owner/repo.git"; got != want {
		t.Errorf("GetOriginURL() = %q, want %q", got, want)
	}
}

// TestSetOriginURL_MalformedInput verifies the model accepts (and round-trips) malformed-looking
// strings without trying to validate them. URL validation is deferred to the data layer / jj
// itself so users can paste anything and see jj's actual error rather than a UI-side rewording.
func TestSetOriginURL_MalformedInput(t *testing.T) {
	t.Parallel()
	cases := []string{
		"",                      // empty: meaningful "user wants to clear" signal
		"   leading-and-trailing  ", // textinput preserves whitespace as typed
		"not a url",             // jj surfaces this on Apply
		"https://",              // syntactically valid scheme but no host
	}
	for _, in := range cases {
		m := NewModel()
		m.SetOriginURL(in)
		if got := m.GetOriginURL(); got != in {
			t.Errorf("SetOriginURL(%q): GetOriginURL() = %q, want round-trip", in, got)
		}
	}
}

// TestSetOriginURL_FailureScenario_LongInputClampedByCharLimit verifies the textinput's CharLimit
// (set to 512 in NewModel) actually clamps overlong input. Without this, a user pasting a
// pathologically long string could wreck the rendering.
func TestSetOriginURL_FailureScenario_LongInputClampedByCharLimit(t *testing.T) {
	t.Parallel()
	m := NewModel()
	overlong := strings.Repeat("a", 1024)
	m.SetOriginURL(overlong)
	got := m.GetOriginURL()
	if len(got) > 512 {
		t.Errorf("SetOriginURL did not respect CharLimit: got length %d, want <= 512", len(got))
	}
}

// --- GetCurrentOrigin / SetCurrentOrigin ----------------------------------------------------

// TestSetGetCurrentOrigin_HappyPath: cached value updates and is observable via the getter (used
// by renderRepositoryRemote to display "Current origin: …").
func TestSetGetCurrentOrigin_HappyPath(t *testing.T) {
	t.Parallel()
	m := NewModel()
	if got := m.GetCurrentOrigin(); got != "" {
		t.Errorf("default GetCurrentOrigin() = %q, want \"\" so the view shows (none configured)", got)
	}
	m.SetCurrentOrigin("https://github.com/foo/bar.git")
	if got, want := m.GetCurrentOrigin(), "https://github.com/foo/bar.git"; got != want {
		t.Errorf("GetCurrentOrigin() = %q, want %q", got, want)
	}
}

// TestSetCurrentOrigin_MalformedInput: same passthrough story as SetOriginURL. The cached value
// is whatever `jj git remote list` reports, so we don't second-guess it.
func TestSetCurrentOrigin_MalformedInput(t *testing.T) {
	t.Parallel()
	m := NewModel()
	m.SetCurrentOrigin("\t\nweird\t")
	if got := m.GetCurrentOrigin(); got != "\t\nweird\t" {
		t.Errorf("SetCurrentOrigin should preserve raw value; got %q", got)
	}
}

// TestSetCurrentOrigin_FailureScenario_DoesNotMutateInput: setting the cached display value must
// not bleed into the editable input field, otherwise pressing Apply right after Settings open
// would re-Apply the URL the user hadn't actually edited yet on top of a remote already at that
// URL — wasteful at best, and confusing at worst when paired with `jj git fetch` errors.
func TestSetCurrentOrigin_FailureScenario_DoesNotMutateInput(t *testing.T) {
	t.Parallel()
	m := NewModel()
	m.SetOriginURL("user-typed-url")
	m.SetCurrentOrigin("ignored-by-input")
	if got, want := m.GetOriginURL(), "user-typed-url"; got != want {
		t.Errorf("SetCurrentOrigin leaked into origin input: GetOriginURL() = %q, want %q", got, want)
	}
}

// --- GetGhPrivate / SetGhPrivate / ToggleGhPrivate ------------------------------------------

// TestGhPrivate_DefaultIsTrue: NewModel must default to private to match the welcome-screen
// behaviour and the README claim that "Visibility: Private" is the starting state.
func TestGhPrivate_DefaultIsTrue(t *testing.T) {
	t.Parallel()
	m := NewModel()
	if !m.GetGhPrivate() {
		t.Errorf("expected ghPrivate=true by default, got false")
	}
}

// TestSetGhPrivate_HappyPath: setter overrides regardless of starting state.
func TestSetGhPrivate_HappyPath(t *testing.T) {
	t.Parallel()
	m := NewModel()
	m.SetGhPrivate(false)
	if m.GetGhPrivate() {
		t.Errorf("SetGhPrivate(false) didn't take effect")
	}
	m.SetGhPrivate(true)
	if !m.GetGhPrivate() {
		t.Errorf("SetGhPrivate(true) didn't take effect")
	}
}

// TestToggleGhPrivate_FlipsTwice: two toggles return to the original state. This guards the
// Ctrl+v / visibility-button path so a double-tap doesn't drift the value.
func TestToggleGhPrivate_FlipsTwice(t *testing.T) {
	t.Parallel()
	m := NewModel()
	start := m.GetGhPrivate()
	m.ToggleGhPrivate()
	if m.GetGhPrivate() == start {
		t.Errorf("ToggleGhPrivate didn't flip the value")
	}
	m.ToggleGhPrivate()
	if m.GetGhPrivate() != start {
		t.Errorf("ToggleGhPrivate twice should return to start (%v), got %v", start, m.GetGhPrivate())
	}
}

// TestToggleGhPrivate_FailureScenario_NoCrossContamination: toggling visibility must not perturb
// the origin input, focused field, or current origin cache. Without this guard a regression
// could re-couple unrelated panels and produce hard-to-debug visual glitches.
func TestToggleGhPrivate_FailureScenario_NoCrossContamination(t *testing.T) {
	t.Parallel()
	m := NewModel()
	m.SetOriginURL("https://example.com/repo.git")
	m.SetCurrentOrigin("https://example.com/repo.git")
	m.SetFocusedField(5)
	before := struct {
		url, current string
		field        int
	}{m.GetOriginURL(), m.GetCurrentOrigin(), m.GetFocusedField()}
	m.ToggleGhPrivate()
	if m.GetOriginURL() != before.url || m.GetCurrentOrigin() != before.current || m.GetFocusedField() != before.field {
		t.Errorf("ToggleGhPrivate touched unrelated state: before=%+v after=(url=%q current=%q field=%d)",
			before, m.GetOriginURL(), m.GetCurrentOrigin(), m.GetFocusedField())
	}
}

// --- SetFocusedField --------------------------------------------------------------------------

// TestSetFocusedField_HappyPath cycles through the documented index range (0..MaxFocusedField).
func TestSetFocusedField_HappyPath(t *testing.T) {
	t.Parallel()
	m := NewModel()
	for i := 0; i <= MaxFocusedField; i++ {
		m.SetFocusedField(i)
		if got := m.GetFocusedField(); got != i {
			t.Errorf("SetFocusedField(%d): GetFocusedField() = %d", i, got)
		}
	}
}

// TestSetFocusedField_MalformedInput exercises the documented clamping for out-of-range values
// (negative below, MaxFocusedField+1 and a much larger number above). Without clamping a
// caller passing a stale global index could focus a non-existent field and make all keystrokes
// disappear into the void.
func TestSetFocusedField_MalformedInput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want int
	}{
		{-1, 0},
		{-99999, 0},
		{MaxFocusedField + 1, MaxFocusedField},
		{1000, MaxFocusedField},
	}
	for _, tc := range tests {
		m := NewModel()
		m.SetFocusedField(tc.in)
		if got := m.GetFocusedField(); got != tc.want {
			t.Errorf("SetFocusedField(%d): GetFocusedField() = %d, want %d", tc.in, got, tc.want)
		}
	}
}

// TestSetFocusedField_FailureScenario_FocusFollowsField: after focusing field 5 (origin URL) the
// origin input must actually receive .Focus() and the token input must be blurred — that's the
// invariant `refocus` exists to enforce. If this regresses, keystrokes typed while clicking the
// origin URL field would silently land in the token input.
func TestSetFocusedField_FailureScenario_FocusFollowsField(t *testing.T) {
	t.Parallel()
	m := NewModel()
	m.SetFocusedField(5)
	if !m.originInput.Focused() {
		t.Errorf("focusedField=5 should focus the origin URL input")
	}
	if m.tokenInput.Focused() {
		t.Errorf("focusedField=5 should blur the token input")
	}
	m.SetFocusedField(0)
	if !m.tokenInput.Focused() {
		t.Errorf("focusedField=0 should focus the token input (when token source is saved)")
	}
	if m.originInput.Focused() {
		t.Errorf("focusedField=0 should blur the origin URL input")
	}
}

// TestSetFocusedField_TokenBlurredWhenSourceIsExternal: when the token source is env / gh-cli
// (i.e. there's no editable saved token), focusedField=0 must NOT focus the token input. This
// guard prevents accidental keystrokes leaking into a hidden, unused field.
func TestSetFocusedField_TokenBlurredWhenSourceIsExternal(t *testing.T) {
	t.Parallel()
	m := NewModel()
	m.SetTokenSource(config.GitHubTokenSourceEnv)
	m.SetFocusedField(0)
	if m.tokenInput.Focused() {
		t.Errorf("token source=env should leave token input blurred even on focusedField=0")
	}
}

// --- FocusOriginInput ------------------------------------------------------------------------

// TestFocusOriginInput_HappyPath jumps focus directly to the URL field (used by zone clicks).
func TestFocusOriginInput_HappyPath(t *testing.T) {
	t.Parallel()
	m := NewModel()
	m.FocusOriginInput()
	if got, want := m.GetFocusedField(), 5; got != want {
		t.Errorf("FocusOriginInput: focusedField = %d, want %d", got, want)
	}
	if !m.originInput.Focused() {
		t.Errorf("FocusOriginInput must focus the origin URL input")
	}
}

// TestFocusOriginInput_FailureScenario_FromAnyStartingState verifies the jump works regardless
// of where the user was before — replicating the click-from-anywhere zone behaviour.
func TestFocusOriginInput_FailureScenario_FromAnyStartingState(t *testing.T) {
	t.Parallel()
	for start := 0; start <= MaxFocusedField; start++ {
		m := NewModel()
		m.SetFocusedField(start)
		m.FocusOriginInput()
		if got := m.GetFocusedField(); got != 5 {
			t.Errorf("FocusOriginInput from focusedField=%d landed on %d, want 5", start, got)
		}
		if !m.originInput.Focused() {
			t.Errorf("FocusOriginInput from focusedField=%d did not focus the input", start)
		}
	}
}

// --- j/k navigation extends to MaxFocusedField ----------------------------------------------

// TestHandleKeyMsg_JKExtendsThroughOriginField walks `j` from 0 up to MaxFocusedField then `k`
// back down, asserting that field 5 is reachable (it's the new origin URL field) and that
// neither key escapes the documented range.
func TestHandleKeyMsg_JKExtendsThroughOriginField(t *testing.T) {
	t.Parallel()
	m := NewModel()
	for i := 0; i < MaxFocusedField; i++ {
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = next
		if got, want := m.GetFocusedField(), i+1; got != want {
			t.Errorf("j from %d landed at %d, want %d", i, got, want)
		}
	}
	// One more j must not advance past MaxFocusedField.
	pinned, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if got := pinned.GetFocusedField(); got != MaxFocusedField {
		t.Errorf("j past MaxFocusedField (%d) should pin; got %d", MaxFocusedField, got)
	}
	m = pinned
	for i := MaxFocusedField; i > 0; i-- {
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
		m = next
		if got, want := m.GetFocusedField(), i-1; got != want {
			t.Errorf("k from %d landed at %d, want %d", i, got, want)
		}
	}
	// One more k must not go below 0.
	pinned2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if got := pinned2.GetFocusedField(); got != 0 {
		t.Errorf("k below 0 should pin; got %d", got)
	}
}

// TestHandleKeyMsg_FailureScenario_SpaceOnlyTogglesIntendedField: the toggle row (1=showMerged,
// 2=showClosed, 3=onlyMine) must respond to space, but space pressed on field 0 (token input)
// or field 5 (origin URL) must not flip any boolean. A regression here would silently change
// PR filter visibility every time the user typed a space inside the URL field.
func TestHandleKeyMsg_FailureScenario_SpaceOnlyTogglesIntendedField(t *testing.T) {
	t.Parallel()
	m := NewModel()
	originalMerged := m.GetShowMerged()
	originalClosed := m.GetShowClosed()
	originalMine := m.GetOnlyMine()

	m.SetFocusedField(5) // origin URL
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = next
	if m.GetShowMerged() != originalMerged ||
		m.GetShowClosed() != originalClosed ||
		m.GetOnlyMine() != originalMine {
		t.Errorf("space on focusedField=5 must not toggle PR filters; before=(%v,%v,%v) after=(%v,%v,%v)",
			originalMerged, originalClosed, originalMine,
			m.GetShowMerged(), m.GetShowClosed(), m.GetOnlyMine())
	}

	// Sanity: space on field 1 still toggles showMerged.
	m.SetFocusedField(1)
	next2, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = next2
	if m.GetShowMerged() == originalMerged {
		t.Errorf("space on focusedField=1 should have toggled showMerged")
	}
}

// --- Update routing of keystrokes to the focused text input ---------------------------------

// TestUpdate_TypingRoutesToOriginInputWhenFocused: with focusedField=5, typed runes must
// accumulate in the origin URL field (mirrors what bubbletea sends per keystroke). The earlier
// regression we shipped was: typing into the URL field also silently appended to the token,
// because the focusedField=5 case wasn't routed; this test pins the fix.
func TestUpdate_TypingRoutesToOriginInputWhenFocused(t *testing.T) {
	t.Parallel()
	m := NewModel()
	m.SetFocusedField(5)
	m.SetOriginURL("") // start clean
	for _, r := range "git@x:y.git" {
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = next
	}
	if got, want := m.GetOriginURL(), "git@x:y.git"; got != want {
		t.Errorf("typed runes into origin URL: got %q, want %q", got, want)
	}
	if m.GetToken() != "" {
		t.Errorf("typing into origin URL should NOT mutate the token; got %q", m.GetToken())
	}
}

// TestUpdate_TypingRoutesToTokenInputWhenFocused: symmetric guard for focusedField=0. Without
// this, a regression to the routing logic might silently break the token entry flow.
//
// NOTE: the test string deliberately avoids j/k/space because those are intercepted as
// navigation/toggle keys at the top of Update before being handed off to the token input —
// see TestUpdate_FailureScenario_TokenInputCannotReceive_jkSpace for the explicit pin of that
// limitation. Real GitHub PATs (`ghp_…`) are alphanumeric so users typing one character at a
// time *can* hit a 'j' or 'k'; that's a pre-existing wart of this tab, not something this set
// of changes introduced or aims to fix.
func TestUpdate_TypingRoutesToTokenInputWhenFocused(t *testing.T) {
	t.Parallel()
	m := NewModel()
	// NewModel defaults to focusedField=0 + tokenSource=Saved; that's the editable case.
	for _, r := range "ghp_abcXYZ012" {
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = next
	}
	if got, want := m.GetToken(), "ghp_abcXYZ012"; got != want {
		t.Errorf("typed runes into token: got %q, want %q", got, want)
	}
	if m.GetOriginURL() != "" {
		t.Errorf("typing into token should NOT mutate origin URL; got %q", m.GetOriginURL())
	}
}

// TestUpdate_FailureScenario_TokenInputCannotReceive_jkSpace documents an existing limitation
// of the GitHub settings panel: j/k/space are intercepted as navigation/toggle keys at the top
// of Update before reaching either text input, so users cannot type those characters via the
// keyboard input flow. Origin URL inputs face the same constraint. This test exists to pin
// the current behaviour explicitly so a future routing-rationalisation PR can find it via
// test failure rather than user reports.
func TestUpdate_FailureScenario_TokenInputCannotReceive_jkSpace(t *testing.T) {
	t.Parallel()
	for _, key := range []string{"j", "k", " "} {
		m := NewModel()
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		m = next
		if m.GetToken() != "" {
			t.Errorf("unexpected: %q reached token input as %q (existing j/k/space carve-out regressed)", key, m.GetToken())
		}
	}
}

// TestUpdate_FailureScenario_TypingDroppedWhenNothingTextEditableFocused: on toggle-row fields
// (1..3) typed runes other than space / j / k / up / down are discarded. Behaviour is what
// shields stray keystrokes from leaking into the wrong input.
func TestUpdate_FailureScenario_TypingDroppedWhenNothingTextEditableFocused(t *testing.T) {
	t.Parallel()
	m := NewModel()
	m.SetFocusedField(2) // showClosed toggle row
	beforeToken := m.GetToken()
	beforeURL := m.GetOriginURL()
	for _, r := range "garbage" {
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = next
	}
	if m.GetToken() != beforeToken {
		t.Errorf("toggle-row typing leaked into token: %q", m.GetToken())
	}
	if m.GetOriginURL() != beforeURL {
		t.Errorf("toggle-row typing leaked into origin URL: %q", m.GetOriginURL())
	}
}

// --- GetInputViews / GetOriginInputView -----------------------------------------------------

// TestGetInputViews_ReturnsTokenOnly: the parent settings model concatenates GetInputViews()
// across all sub-tabs into a flat global index (token=0, jira=1..8, codecks=9..12, ...). Adding
// the origin URL view to this slice would shift every later index and break every other tab's
// click handlers, so GetInputViews MUST stay at length 1.
func TestGetInputViews_ReturnsTokenOnly(t *testing.T) {
	t.Parallel()
	m := NewModel()
	views := m.GetInputViews()
	if len(views) != 1 {
		t.Fatalf("GetInputViews() returned %d entries (regression: would shift global indices), want 1", len(views))
	}
	// The token's textinput renders something non-empty even when the value is empty (it shows
	// placeholder/cursor). Just confirm it's a string we can render.
	if views[0] == "" {
		t.Errorf("GetInputViews()[0] is empty; expected the token input view")
	}
}

// TestGetOriginInputView_NotEmpty: the view is rendered directly from the sub-model in
// renderGitHub, so it must always return something printable even with no value typed.
func TestGetOriginInputView_NotEmpty(t *testing.T) {
	t.Parallel()
	m := NewModel()
	if v := m.GetOriginInputView(); v == "" {
		t.Errorf("GetOriginInputView() returned empty string with no value (placeholder should show)")
	}
	m.SetOriginURL("https://github.com/owner/repo.git")
	if v := m.GetOriginInputView(); !strings.Contains(v, "https://github.com/owner/repo.git") {
		// Note: the rendered view includes ANSI styles. The substring check is enough to
		// confirm the value made it through to the output; we don't need to strip the styles.
		t.Errorf("GetOriginInputView() did not contain the value; got %q", v)
	}
}

// --- SetInputWidth (extended to cover both inputs) ------------------------------------------

// TestSetInputWidth_AppliesToBothInputs: on resize, widths must be propagated to both the token
// and origin URL inputs so the visual alignment in renderGitHub stays consistent.
func TestSetInputWidth_AppliesToBothInputs(t *testing.T) {
	t.Parallel()
	m := NewModel()
	m.SetInputWidth(72)
	if m.tokenInput.Width != 72 {
		t.Errorf("tokenInput.Width = %d, want 72", m.tokenInput.Width)
	}
	if m.originInput.Width != 72 {
		t.Errorf("originInput.Width = %d, want 72", m.originInput.Width)
	}
}

// TestSetInputWidth_FailureScenario_ZeroWidthStillStored: a zero width is unusual but we accept
// it (the parent layout occasionally clamps; the textinput handles 0 by hiding visible runs).
// What we DON'T want is a panic or a silent rejection that desyncs the two inputs.
func TestSetInputWidth_FailureScenario_ZeroWidthStillStored(t *testing.T) {
	t.Parallel()
	m := NewModel()
	m.SetInputWidth(0)
	if m.tokenInput.Width != 0 || m.originInput.Width != 0 {
		t.Errorf("SetInputWidth(0): token=%d origin=%d, want both 0", m.tokenInput.Width, m.originInput.Width)
	}
}
