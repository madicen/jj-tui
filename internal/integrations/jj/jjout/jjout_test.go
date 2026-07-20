package jjout

import (
	"reflect"
	"testing"
)

func TestSplitLines(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"blank lines dropped", "\n\n  \n\t\n", nil},
		{"trims and drops", "  a \n\nb\n c \r\n", []string{"a", "b", "c"}},
		{"single", "only", []string{"only"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SplitLines(tc.in)
			if len(got) == 0 && len(tc.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("SplitLines(%q) = %#v, want %#v", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseCommitInfo(t *testing.T) {
	cases := []struct {
		in         string
		wantChange string
		wantShort  string
	}{
		{"abc def rest of desc", "abc", "def"},
		{"onlychange", "onlychange", "onlychange"},
		{"  spaced   xyz  ", "spaced", "xyz"},
		{"", "", ""},
		{"   ", "", ""},
	}
	for _, tc := range cases {
		change, short := ParseCommitInfo(tc.in)
		if change != tc.wantChange || short != tc.wantShort {
			t.Errorf("ParseCommitInfo(%q) = (%q,%q), want (%q,%q)", tc.in, change, short, tc.wantChange, tc.wantShort)
		}
	}
}

func TestExtractErrorMessage(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"error line preferred", "Warning: x\nError: boom happened\nHint: try this", "boom happened"},
		{"first meaningful line", "Warning: skip\nHint: skip\nsomething useful", "something useful"},
		{"nothing", "Warning: only\nHint: only\n\n", ""},
		{"empty", "", ""},
		{
			name: "internal error with caused by chain",
			in: `Internal error: Unexpected error from backend
Caused by:
1: Could not write object of type commit
2: Signing error
3: GPG failed with exit status: 2:
gpg: signing failed: No pinentry
Hint: something else`,
			want: `Internal error: Unexpected error from backend
Caused by:
1: Could not write object of type commit
2: Signing error
3: GPG failed with exit status: 2:
gpg: signing failed: No pinentry`,
		},
		{
			name: "error with caused by stops at blank",
			in:   "Error: push failed\nCaused by:\n1: network down\n\nHint: retry",
			want: "push failed\nCaused by:\n1: network down",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExtractErrorMessage(tc.in); got != tc.want {
				t.Errorf("ExtractErrorMessage(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestIsSigningPinentryFailure(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"Internal error: Unexpected error from backend\nCaused by:\n2: Signing error\ngpg: signing failed: No pinentry", true},
		{"signing error", true},
		{"gpg: signing failed: No pinentry", true},
		{"failed to push: remote rejected", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := IsSigningPinentryFailure(tc.in); got != tc.want {
			t.Errorf("IsSigningPinentryFailure(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
