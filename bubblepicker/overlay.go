// Package bubblepicker provides OverlayView for compositing a modal over a main view
// so only the modal rectangle is replaced; the rest of the main view stays visible.

package bubblepicker

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// OverlayView composites modalView on top of mainView. Only the rectangle at (top, left)
// with the modal's size is replaced; all other cells show the main view. Returns a single
// string with viewHeight lines, each viewWidth cells wide (padding/truncation as needed).
//
// Main and modal strings may contain ANSI (e.g. from lipgloss); overlay uses display-cell
// width so alignment is correct. Use this when you want the modal to "float" over the
// main view without hiding the surrounding content.
func OverlayView(mainView, modalView string, viewWidth, viewHeight, top, left int) string {
	mainLines := strings.Split(mainView, "\n")
	modalLines := strings.Split(modalView, "\n")
	if len(modalLines) == 0 {
		var out []string
		for row := 0; row < viewHeight; row++ {
			line := ""
			if row < len(mainLines) {
				line = mainLines[row]
			}
			out = append(out, padOrTruncateLine(line, viewWidth))
		}
		return strings.Join(out, "\n")
	}
	modalH := len(modalLines)
	modalW := 0
	for _, l := range modalLines {
		if w := lipgloss.Width(l); w > modalW {
			modalW = w
		}
	}
	// Clamp so modal fits
	if left+modalW > viewWidth {
		left = max(0, viewWidth-modalW)
	}
	if left < 0 {
		left = 0
	}
	if top+modalH > viewHeight {
		top = max(0, viewHeight-modalH)
	}
	if top < 0 {
		top = 0
	}

	var out []string
	for row := 0; row < viewHeight; row++ {
		mainLine := ""
		if row < len(mainLines) {
			mainLine = mainLines[row]
		}
		if row < top || row >= top+modalH {
			out = append(out, padOrTruncateLine(mainLine, viewWidth))
			continue
		}
		modalLine := modalLines[row-top]
		combined := overlayLine(mainLine, modalLine, left, modalW, viewWidth)
		out = append(out, combined)
	}
	return strings.Join(out, "\n")
}

// overlayLine returns mainLine with modalLine overlaid at column left for modalW cells.
// When mainLine has fewer than left cells (e.g. main view has fewer rows), prefix is
// padded so the modal stays aligned at column left.
func overlayLine(mainLine, modalLine string, left, modalW, viewWidth int) string {
	prefix := prefixCells(mainLine, left)
	if w := widthCells(prefix); w < left {
		prefix += strings.Repeat(" ", left-w)
	}
	suffix := skipCells(mainLine, left+modalW)
	line := prefix + modalLine + suffix
	return padOrTruncateLine(line, viewWidth)
}

// prefixCells returns the prefix of s that spans at most n display cells (ANSI preserved).
func prefixCells(s string, n int) string {
	if n <= 0 {
		return ""
	}
	var cellCount int
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) {
				c := s[j]
				if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' {
					j++
					break
				}
				j++
			}
			i = j
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		w := runewidth.RuneWidth(r)
		if cellCount+w > n {
			break
		}
		cellCount += w
		i += size
	}
	return s[:i]
}

// skipCells returns the substring of s starting after the first n display cells (ANSI preserved).
func skipCells(s string, n int) string {
	var cellCount int
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) {
				c := s[j]
				if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' {
					j++
					break
				}
				j++
			}
			i = j
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		w := runewidth.RuneWidth(r)
		if cellCount >= n {
			return s[i:]
		}
		cellCount += w
		i += size
	}
	return ""
}

func widthCells(s string) int {
	var n int
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) {
				c := s[j]
				if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' {
					j++
					break
				}
				j++
			}
			i = j
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		n += runewidth.RuneWidth(r)
		i += size
	}
	return n
}

func padOrTruncateLine(line string, viewWidth int) string {
	w := widthCells(line)
	if w < viewWidth {
		return line + strings.Repeat(" ", viewWidth-w)
	}
	if w > viewWidth {
		return prefixCells(line, viewWidth)
	}
	return line
}

