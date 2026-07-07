package render

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone"
)

func TestMarkNilManager(t *testing.T) {
	if got := Mark(nil, "zone-id", "content"); got != "content" {
		t.Fatalf("Mark(nil) = %q, want %q", got, "content")
	}
}

func TestMarkWithManager(t *testing.T) {
	zm := zone.New()
	got := Mark(zm, "zone-id", "content")
	if got == "content" {
		t.Fatalf("Mark with manager should wrap content in zone markers, got unchanged %q", got)
	}
	if !strings.Contains(got, "content") {
		t.Fatalf("Mark output should still contain the original content, got %q", got)
	}
}

func TestTruncateEllipsis(t *testing.T) {
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{"empty", "", 10, ""},
		{"short unchanged", "hello", 10, "hello"},
		{"exact fit", "hello", 5, "hello"},
		{"ascii truncated", "hello world", 5, "hello..."},
		{"zero max", "hello", 0, ""},
		{"negative max", "hello", -1, ""},
		{"matches byte slice pattern", strings.Repeat("a", 200), 150, strings.Repeat("a", 150) + "..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TruncateEllipsis(tt.s, tt.max); got != tt.want {
				t.Errorf("TruncateEllipsis(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
			}
		})
	}
}

// TestTruncateEllipsisWideRunes verifies the helper never splits a wide (CJK) rune
// and measures by display width, not byte count.
func TestTruncateEllipsisWideRunes(t *testing.T) {
	// Each of these CJK characters is 2 display cells wide.
	s := "你好世界你好" // width 12
	got := TruncateEllipsis(s, 6)
	// Width 6 fits exactly 3 wide runes ("你好世"), then "...".
	if got != "你好世..." {
		t.Fatalf("TruncateEllipsis wide-rune = %q, want %q", got, "你好世...")
	}
	// Result must contain only whole runes (valid UTF-8, no partial byte splits).
	if !isValidTruncation(got) {
		t.Fatalf("TruncateEllipsis produced invalid rune boundary: %q", got)
	}
	// Odd width should not split a 2-cell rune: width 5 fits 2 wide runes (width 4).
	if got := TruncateEllipsis(s, 5); got != "你好..." {
		t.Fatalf("TruncateEllipsis odd width = %q, want %q", got, "你好...")
	}
}

// TestTruncateEllipsisANSI verifies that embedded ANSI escape sequences (styling)
// are preserved and not counted toward the display width.
func TestTruncateEllipsisANSI(t *testing.T) {
	styled := lipgloss.NewStyle().Bold(true).Render("hello world")
	got := TruncateEllipsis(styled, 5)
	// Visible width of the truncated portion should be 5 ("hello"), plus the "..." tail.
	if ansi.StringWidth(got) != len("hello...") {
		t.Fatalf("TruncateEllipsis ANSI visible width = %d, want %d (%q)", ansi.StringWidth(got), len("hello..."), got)
	}
	if !strings.Contains(ansi.Strip(got), "hello") {
		t.Fatalf("TruncateEllipsis ANSI should retain visible text, got stripped %q", ansi.Strip(got))
	}
}

func isValidTruncation(s string) bool {
	// A valid truncation contains only whole runes; strings.ToValidUTF8 is a no-op
	// when every byte is part of a complete rune.
	return strings.ToValidUTF8(s, "\uFFFD") == s
}

func TestSeparator(t *testing.T) {
	// Wide enough: width-4 dashes.
	if got := Separator(84); got != strings.Repeat("─", 80) {
		t.Errorf("Separator(84) length = %d, want 80", lipgloss.Width(got))
	}
	// Too narrow: falls back to 80.
	if got := Separator(10); got != strings.Repeat("─", 80) {
		t.Errorf("Separator(10) = fallback, got length %d", lipgloss.Width(got))
	}
	// Exactly at the clamp boundary (width-4 == 20).
	if got := Separator(24); got != strings.Repeat("─", 20) {
		t.Errorf("Separator(24) length = %d, want 20", lipgloss.Width(got))
	}
	// Just below the clamp boundary (width-4 == 19) falls back.
	if got := Separator(23); got != strings.Repeat("─", 80) {
		t.Errorf("Separator(23) should fall back to 80, got length %d", lipgloss.Width(got))
	}
}

func TestSafeWidth(t *testing.T) {
	tests := []struct {
		total, padding, want int
	}{
		{100, 4, 96},
		{4, 4, 0},
		{2, 4, 0},
		{0, 0, 0},
		{10, 2, 8},
	}
	for _, tt := range tests {
		if got := SafeWidth(tt.total, tt.padding); got != tt.want {
			t.Errorf("SafeWidth(%d, %d) = %d, want %d", tt.total, tt.padding, got, tt.want)
		}
	}
}
