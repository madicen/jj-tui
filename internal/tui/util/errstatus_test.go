package util

import (
	"errors"
	"strings"
	"testing"
)

func TestStatusStringFromError(t *testing.T) {
	if got := StatusStringFromError(nil, 50); got != "" {
		t.Errorf("nil: got %q", got)
	}
	long := errors.New("abcdefghijklmnopqrst")
	// min width is 8; use 12 so we truncate after clamp
	got := StatusStringFromError(long, 12)
	want := "abcdefghijk…"
	if got != want {
		t.Errorf("truncate: got %q want %q", got, want)
	}
	multi := errors.New("line1\nline2")
	got2 := StatusStringFromError(multi, 100)
	if got2 != "line1 line2" {
		t.Errorf("newlines: got %q", got2)
	}
}

// TestIsMissingOriginError exercises the matcher used by push-side error wrappers to decide
// whether to append the "Set up origin in Settings → GitHub" hint. The matcher is substring-
// based against jj/git error wording and intentionally case-insensitive: jj and git aren't
// strict about quote style ("'origin'" vs "\"origin\""), so the matcher accepts both.
func TestIsMissingOriginError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want bool
	}{
		// Happy path: the canonical jj/git error for a missing origin remote.
		{"missing origin single-quoted", errors.New("No git remote named 'origin'"), true},
		// Some jj versions / locales produce double-quoted output; matcher must accept both.
		{"missing origin double-quoted", errors.New(`No git remote named "origin"`), true},
		// Case-insensitivity: the wrapper lowercases before matching so we can pin both forms.
		{"missing origin uppercased", errors.New("NO GIT REMOTE NAMED 'ORIGIN'"), true},
		// readOriginURL surfaces this when `jj git remote list` returns nothing; treated as the
		// same condition because the user-visible fix is identical.
		{"no remotes found", errors.New("no git remotes found"), true},

		// Failure / negative cases — none of these should match, otherwise we'd append a
		// confusing hint to unrelated errors.
		{"nil", nil, false},
		{"unrelated push error", errors.New("permission denied"), false},
		{"empty string", errors.New(""), false},
		{"different remote name", errors.New("No git remote named 'upstream'"), false},
		// Substring of an unrelated message that happens to contain "origin": must NOT match
		// because that would surface the hint on, say, "rebased onto origin/main".
		{"contains origin but not the error", errors.New("rebased onto origin/main"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := IsMissingOriginError(tc.err); got != tc.want {
				t.Errorf("IsMissingOriginError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// TestMissingOriginHint pins the exact wording surfaced to users so a copy change can't drift
// silently. The hint must be a leading-newline string when matching (so it appends cleanly on
// its own line below the raw error) and "" otherwise.
func TestMissingOriginHint(t *testing.T) {
	t.Parallel()
	t.Run("happy_match", func(t *testing.T) {
		t.Parallel()
		hint := MissingOriginHint(errors.New("No git remote named 'origin'"))
		if !strings.HasPrefix(hint, "\n") {
			t.Errorf("hint should start with a newline so it appends on its own line, got %q", hint)
		}
		if !strings.Contains(hint, "Settings → GitHub → Repository remote") {
			t.Errorf("hint should mention the in-app fix path, got %q", hint)
		}
	})
	t.Run("malformed_input_nil_returns_empty", func(t *testing.T) {
		t.Parallel()
		if got := MissingOriginHint(nil); got != "" {
			t.Errorf("nil err should return \"\"; got %q", got)
		}
	})
	t.Run("failure_scenario_unrelated_error_returns_empty", func(t *testing.T) {
		t.Parallel()
		// A push error not caused by missing origin must not get the hint appended; otherwise
		// users would chase a non-issue and lose trust in the hint.
		if got := MissingOriginHint(errors.New("ssh: connect to host github.com port 22: Connection timed out")); got != "" {
			t.Errorf("unrelated network error should not get the hint; got %q", got)
		}
	})
}
