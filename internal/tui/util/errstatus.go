package util

import (
	"strings"
)

// StatusStringFromError flattens an error into a single line for the footer status bar.
func StatusStringFromError(err error, maxRunes int) string {
	if err == nil {
		return ""
	}
	if maxRunes < 8 {
		maxRunes = 8
	}
	s := strings.ReplaceAll(strings.TrimSpace(err.Error()), "\n", " ")
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes-1]) + "…"
}

// IsMissingOriginError reports whether err looks like a push/fetch failure caused by no
// `origin` remote being configured. Substring-based against the underlying jj/git error text.
func IsMissingOriginError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "no git remote named 'origin'") ||
		strings.Contains(s, "no git remote named \"origin\"") ||
		strings.Contains(s, "no git remotes found")
}

// MissingOriginHint returns a leading-newline hint pointing the user to the in-app fix when err
// matches IsMissingOriginError; "" otherwise. Used by push wrapping sites that compose errors
// into status strings, so the hint always appears on its own line below the raw error.
func MissingOriginHint(err error) string {
	if !IsMissingOriginError(err) {
		return ""
	}
	return "\nSet up origin in Settings → GitHub → Repository remote"
}
