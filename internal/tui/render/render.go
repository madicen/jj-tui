// Package render provides small, pure helpers shared across the TUI view layer:
// nil-safe click-zone marking, rune/ANSI-safe truncation, horizontal separators,
// and simple width math. These were previously copy-pasted into per-tab
// view_helpers.go files; centralizing them keeps behavior consistent and removes
// duplication.
package render

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone"
)

// separator tuning shared by every list/detail tab. width-separatorPadding is the
// rule length; when the terminal is too narrow (or width is unset) we fall back to
// separatorFallbackWidth so the rule never collapses to nothing.
const (
	separatorPadding       = 4
	separatorMinWidth      = 20
	separatorFallbackWidth = 80
)

// Mark wraps content in a bubblezone click zone. When zm is nil (e.g. zones are
// disabled or the model has no manager yet) it returns content unchanged. This is
// the nil-safe replacement for the per-tab local mark helpers.
func Mark(zm *zone.Manager, id, content string) string {
	if zm == nil {
		return content
	}
	return zm.Mark(id, content)
}

// TruncateEllipsis truncates s to a maximum display width of max cells, appending
// "..." when (and only when) truncation actually occurs. It is rune- and ANSI-aware
// (via x/ansi), so it never splits a multi-byte rune or an escape sequence the way
// the old byte-slicing `s[:max]+"..."` pattern could. For plain ASCII input it is
// byte-for-byte identical to that pattern.
func TruncateEllipsis(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if ansi.StringWidth(s) <= maxWidth {
		return s
	}
	return ansi.Truncate(s, maxWidth, "") + "..."
}

// Separator returns a horizontal rule of box-drawing dashes sized to width minus the
// shared padding, clamped to a fixed fallback width when the result would be too
// small. Callers apply their own styling.
func Separator(width int) string {
	w := width - separatorPadding
	if w < separatorMinWidth {
		w = separatorFallbackWidth
	}
	return strings.Repeat("─", w)
}

// SafeWidth returns total minus padding, clamped to a minimum of zero so it is safe
// to pass to strings.Repeat / lipgloss width calls.
func SafeWidth(total, padding int) int {
	if w := total - padding; w > 0 {
		return w
	}
	return 0
}
