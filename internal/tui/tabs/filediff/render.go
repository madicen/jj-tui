package filediff

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal/tui/styles"
	"github.com/mattn/go-runewidth"
)

// unifiedHunkHeader matches git unified diff hunk lines, e.g. @@ -10,6 +10,7 @@
var unifiedHunkHeader = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// StyleGitUnifiedDiff applies per-line background colors to git unified diff output
// (from `jj diff --git --color never`). When the text looks like a git diff, each line
// gets an old/new line-number gutter; other formats are returned unchanged.
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

	const gutterW = 12
	gap := strings.Repeat(" ", gutterW)

	fg := lipgloss.Color("#F8F8F2")
	addSt := lipgloss.NewStyle().Background(lipgloss.Color("#1B4332")).Foreground(fg)
	delSt := lipgloss.NewStyle().Background(lipgloss.Color("#4A232C")).Foreground(fg)
	ctxSt := lipgloss.NewStyle().Background(lipgloss.Color("#21222C")).Foreground(fg)
	metaSt := lipgloss.NewStyle().Background(lipgloss.Color("#2D303E")).Foreground(styles.ColorMuted)
	hunkSt := lipgloss.NewStyle().Background(lipgloss.Color("#44475A")).Foreground(lipgloss.Color("#8BE9FD")).Bold(true)

	out := make([]string, 0, len(lines))
	var oldLine, newLine int
	inHunk := false

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			inHunk = false
		}

		if line == "" {
			out = append(out, styleGitDiffLine("", gap, contentWidth, metaSt))
			continue
		}

		if strings.HasPrefix(line, "@@") {
			m := unifiedHunkHeader.FindStringSubmatch(line)
			if m != nil {
				o0, _ := strconv.Atoi(m[1])
				n0, _ := strconv.Atoi(m[3])
				oldLine, newLine = o0, n0
				inHunk = true
			}
			out = append(out, styleGitDiffLine(line, gap, contentWidth, hunkSt))
			continue
		}

		if !inHunk || isGitDiffMetaLine(line) {
			out = append(out, styleGitDiffLine(line, gap, contentWidth, metaSt))
			continue
		}

		switch {
		case strings.HasPrefix(line, " "):
			g := gutterPair(oldLine, newLine, true, true, gutterW)
			oldLine++
			newLine++
			out = append(out, styleGitDiffLine(line, g, contentWidth, ctxSt))
		case strings.HasPrefix(line, "-"):
			g := gutterPair(oldLine, 0, true, false, gutterW)
			oldLine++
			out = append(out, styleGitDiffLine(line, g, contentWidth, delSt))
		case strings.HasPrefix(line, "+"):
			g := gutterPair(0, newLine, false, true, gutterW)
			newLine++
			out = append(out, styleGitDiffLine(line, g, contentWidth, addSt))
		default:
			out = append(out, styleGitDiffLine(line, gap, contentWidth, metaSt))
		}
	}
	return strings.Join(out, "\n")
}

func isGitDiffMetaLine(line string) bool {
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
		strings.HasPrefix(line, "Binary files "),
		strings.HasPrefix(line, "\\"):
		return true
	default:
		return false
	}
}

func gutterPair(oldN, newN int, hasOld, hasNew bool, targetW int) string {
	left, right := "    ", "    "
	if hasOld {
		left = fmt.Sprintf("%4d", oldN)
	}
	if hasNew {
		right = fmt.Sprintf("%4d", newN)
	}
	s := left + "│" + right + " "
	for runewidth.StringWidth(s) < targetW {
		s += " "
	}
	return s
}

func styleGitDiffLine(line, gutter string, contentWidth int, st lipgloss.Style) string {
	gw := runewidth.StringWidth(gutter)
	avail := contentWidth - gw
	if avail < 1 {
		avail = 1
	}
	body := truncateVisual(line, avail)
	return st.Width(contentWidth).Render(gutter + body)
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
