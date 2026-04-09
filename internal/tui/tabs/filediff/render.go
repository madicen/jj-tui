package filediff

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal/tui/styles"
	"github.com/mattn/go-runewidth"
)

// StyleGitUnifiedDiff applies per-line background colors to git unified diff output
// (from `jj diff --git --color never`). Other formats are returned unchanged.
func StyleGitUnifiedDiff(text string, contentWidth int) string {
	if contentWidth < 8 {
		contentWidth = 8
	}
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return normalized
	}
	if !strings.HasPrefix(lines[0], "diff --git") {
		return normalized
	}

	fg := lipgloss.Color("#F8F8F2")
	addSt := lipgloss.NewStyle().Background(lipgloss.Color("#1B4332")).Foreground(fg)
	delSt := lipgloss.NewStyle().Background(lipgloss.Color("#4A232C")).Foreground(fg)
	ctxSt := lipgloss.NewStyle().Background(lipgloss.Color("#21222C")).Foreground(fg)
	metaSt := lipgloss.NewStyle().Background(lipgloss.Color("#2D303E")).Foreground(styles.ColorMuted)
	hunkSt := lipgloss.NewStyle().Background(lipgloss.Color("#44475A")).Foreground(lipgloss.Color("#8BE9FD")).Bold(true)

	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			out = append(out, "")
			continue
		}
		st := lineStyle(line, addSt, delSt, ctxSt, metaSt, hunkSt)
		out = append(out, st.Width(contentWidth).Render(truncateVisual(line, contentWidth)))
	}
	return strings.Join(out, "\n")
}

func lineStyle(line string, addSt, delSt, ctxSt, metaSt, hunkSt lipgloss.Style) lipgloss.Style {
	switch {
	case strings.HasPrefix(line, "diff --git "),
		strings.HasPrefix(line, "index "),
		strings.HasPrefix(line, "--- "),
		strings.HasPrefix(line, "+++ "),
		strings.HasPrefix(line, "new file mode "),
		strings.HasPrefix(line, "deleted file mode "),
		strings.HasPrefix(line, "similarity index "),
		strings.HasPrefix(line, "rename from "),
		strings.HasPrefix(line, "rename to "),
		strings.HasPrefix(line, "Binary files "):
		return metaSt
	case strings.HasPrefix(line, "@@"):
		return hunkSt
	case strings.HasPrefix(line, "+"):
		return addSt
	case strings.HasPrefix(line, "-"):
		return delSt
	case strings.HasPrefix(line, " "):
		return ctxSt
	default:
		return metaSt
	}
}

func truncateVisual(s string, maxWidth int) string {
	if maxWidth < 1 {
		return ""
	}
	var b strings.Builder
	w := 0
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if w+rw > maxWidth {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	return b.String()
}
