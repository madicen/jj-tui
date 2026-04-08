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
