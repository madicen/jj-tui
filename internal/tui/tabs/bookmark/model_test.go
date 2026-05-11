package bookmark

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/madicen/jj-tui/internal"
)

// longDescription mirrors what jj's parser puts into Commit.Summary for a commit with a
// long single-paragraph description (Summary is set to the full description, not just the
// first line). The original bug appeared on exactly this shape: the rounded "Target:" box,
// with no Width set, grew its border to match the entire single line, then the outer
// FrameFormModal's fixed Width chopped that border into stacked horizontal segments
// above and below the text.
const longDescription = "Unified the AI and Hybrid backends into a single `product-crawler-ai` " +
	"image, serving both CRAWL_BACKEND=ai (pure LLM extraction) and " +
	"CRAWL_BACKEND=hybrid (cheap markdown + JSON-LD pass with AI gap-filling). " +
	"The hybrid backend is selected by default but can be overridden at runtime. " +
	"This change simplifies the deployment and reduces the number of images required."

func longDescCommit() internal.Commit {
	return internal.Commit{
		ID:       "01182b12abcdef",
		ShortID:  "01182b12",
		ChangeID: "01182b12",
		Summary:  longDescription,
	}
}

// syncContentWidth applies contentW to both the inner-box pin and the name input width.
// The parent Model does the same on tea.WindowSizeMsg; mirroring it here keeps the test
// faithful to runtime behavior so the max-visible-width assertions don't accidentally pass
// because of an unsynced 50-wide default name input.
func syncContentWidth(m *Model, contentW int) {
	m.SetContentWidth(contentW)
	m.GetNameInput().Width = contentW
}

func newModelWithCommit(t *testing.T, c internal.Commit, contentW int) Model {
	t.Helper()
	m := NewModel(nil)
	m.Show(0, nil)
	m.SetCommitIdx(0)
	m.UpdateRepository(&internal.Repository{
		Graph: internal.CommitGraph{Commits: []internal.Commit{c}},
	})
	if contentW > 0 {
		syncContentWidth(&m, contentW)
	}
	return m
}

func renderRows(view string) []string {
	return strings.Split(ansi.Strip(view), "\n")
}

func maxVisibleWidth(rows []string) (int, string) {
	best := 0
	worst := ""
	for _, r := range rows {
		w := utf8.RuneCountInString(r)
		if w > best {
			best = w
			worst = r
		}
	}
	return best, worst
}

// dashHeavyRows returns the indices of rows that contain a long run of '─' box-drawing
// characters — i.e. anything that looks like a horizontal border row, anchored (╭…╮ / ╰…╯)
// or not. Plain text never carries 10+ consecutive box-drawing dashes (note: this is the
// Unicode '─' U+2500, not ASCII '-'), so this is a safe proxy for "border-shaped row".
//
// In non-Jira mode with no existing bookmarks we expect exactly four such rows once the
// view is wrapped in an outer frame:
//
//	╭───…───╮  ← outer top
//	╭───…───╮  ← inner Target box top
//	╰───…───╯  ← inner Target box bottom
//	╰───…───╯  ← outer bottom
//
// Pre-fix, the inner box's natural border row was ~370 cols wide; lipgloss hard-wrapped it
// into 5–6 fragments above and 5–6 below the text, sending this count well past 4.
func dashHeavyRows(rows []string) []int {
	var idx []int
	for i, r := range rows {
		if strings.Count(r, "─") >= 10 {
			idx = append(idx, i)
		}
	}
	return idx
}

// The unit-level contract of the fix: with SetContentWidth(W), no rendered row of the
// bookmark view exceeds W+2 cols (W content/padding plus 1 border col on each side).
// Pre-fix, the inner box's top/bottom border row was as wide as the longest line in
// commit.Summary (≈370 cols for the long-description fixture), so this assertion would
// have failed by a huge margin.
func TestRender_TargetBox_LongDescription_FitsContentWidth(t *testing.T) {
	const contentW = 56
	m := newModelWithCommit(t, longDescCommit(), contentW)

	view := m.View()
	rows := renderRows(view)
	w, worst := maxVisibleWidth(rows)

	if w > contentW+2 {
		t.Fatalf("widest row is %d cols, expected <= %d (contentWidth+border).\n"+
			"worst row: %q\n--- view ---\n%s", w, contentW+2, worst, ansi.Strip(view))
	}

	// Exactly one ╭…╮ row and one ╰…╯ row — sanity check that the box still has a
	// proper, single, non-fragmented border.
	top, bot := 0, 0
	for _, r := range rows {
		if strings.ContainsRune(r, '╭') && strings.ContainsRune(r, '╮') {
			top++
		}
		if strings.ContainsRune(r, '╰') && strings.ContainsRune(r, '╯') {
			bot++
		}
	}
	if top != 1 || bot != 1 {
		t.Fatalf("expected exactly one top and one bottom row for the Target box, got top=%d bot=%d\n--- view ---\n%s",
			top, bot, ansi.Strip(view))
	}
}

// End-to-end check: wrap m.View() with an outer rounded-border style that mimics the
// FrameFormModal used by the parent model. Pre-fix, the inner box's natural border row
// (≈370 cols) overflowed this outer frame's Width and lipgloss soft-wrapped that row
// across multiple visual lines, producing 4–6 consecutive "horizontal slab" rows above
// and below the text — the artifact in the screenshot.
func TestRender_TargetBox_LongDescription_NoSlabsWhenFramed(t *testing.T) {
	const contentW = 56
	// FrameFormModal computes outerW ≈ innerW + 6, with Padding(1, 2). innerW here is
	// the contentWidth we hand the bookmark Model, so the outer frame fits the inner
	// box exactly. We construct an equivalent local style instead of importing the
	// model package (which would create a circular dependency).
	outerW := contentW + 6
	framed := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Width(outerW).
		Render(newModelWithCommit(t, longDescCommit(), contentW).View())

	rows := renderRows(framed)
	if heavy := dashHeavyRows(rows); len(heavy) != 4 {
		t.Fatalf("framed render has %d dash-heavy rows (rows %v); expected exactly 4 "+
			"(outer top + inner top + inner bottom + outer bottom). Extras indicate the "+
			"inner border was wrapped across multiple lines.\n--- view ---\n%s",
			len(heavy), heavy, ansi.Strip(framed))
	}

	// Each of those four rows should also be anchored to corner characters (╭…╮ or ╰…╯).
	// If any dash-heavy row lacks both corners on its end positions, that's the signature
	// of a wrapped border fragment slipping past the count-of-four check.
	for _, i := range dashHeavyRows(rows) {
		r := strings.TrimSpace(rows[i])
		// Walk past leading │ from a surrounding frame to find the box-drawing prefix.
		stripped := strings.TrimLeft(r, "│ ")
		stripped = strings.TrimRight(stripped, "│ ")
		hasTop := strings.HasPrefix(stripped, "╭") && strings.HasSuffix(stripped, "╮")
		hasBot := strings.HasPrefix(stripped, "╰") && strings.HasSuffix(stripped, "╯")
		if !hasTop && !hasBot {
			t.Fatalf("dash-heavy row %d is not anchored by corners: %q\n--- view ---\n%s",
				i, rows[i], ansi.Strip(framed))
		}
	}
}

// Without SetContentWidth, the fallback (50) in boxWidth() kicks in. The render should
// still be bounded — this guards against future regressions that remove the fallback.
func TestRender_TargetBox_LongDescription_FallbackWidth(t *testing.T) {
	m := newModelWithCommit(t, longDescCommit(), 0) // 0 means "don't call SetContentWidth"

	rows := renderRows(m.View())
	w, worst := maxVisibleWidth(rows)
	if w > 52 { // fallback 50 + 2 border cols
		t.Fatalf("fallback render widest row is %d cols, expected <= 52.\nworst: %q", w, worst)
	}
}

// The Jira-flavored modal renders a different inner box ("Jira Ticket:") via the same
// shared style, so the same Width pin protects it too.
func TestRender_JiraBox_FitsContentWidth(t *testing.T) {
	const contentW = 40
	m := NewModel(nil)
	m.Show(0, nil)
	m.SetFromJira("APP-1234", strings.Repeat("very long jira summary ", 12), "APP-1234")
	syncContentWidth(&m, contentW)

	rows := renderRows(m.View())
	w, worst := maxVisibleWidth(rows)
	if w > contentW+2 {
		t.Fatalf("Jira box widest row is %d cols, expected <= %d.\nworst: %q", w, contentW+2, worst)
	}
}

// A short summary should render verbatim with a single clean border, no width artifacts.
func TestRender_TargetBox_ShortDescription(t *testing.T) {
	c := internal.Commit{ShortID: "abc1", Summary: "Fix flaky test"}
	m := newModelWithCommit(t, c, 56)

	view := ansi.Strip(m.View())
	if !strings.Contains(view, "Target: abc1") {
		t.Fatalf("expected target line in view, got:\n%s", view)
	}
	if !strings.Contains(view, "Fix flaky test") {
		t.Fatalf("expected description in view, got:\n%s", view)
	}
	w, _ := maxVisibleWidth(renderRows(m.View()))
	if w > 58 {
		t.Fatalf("short description widest row is %d cols, expected <= 58", w)
	}
}

// An existing bookmark name that came from a pre-cap repo (e.g. a 90-char branch the
// user created before jj.TruncateBookmarkName existed, or via a tool that bypasses
// this UI) must not blow past the modal's content width. The model displays such names
// with an ellipsis; the underlying m.existingBookmarks slice is untouched so move/click
// resolution still uses the real name.
func TestRender_ExistingBookmarkList_LongNameTruncatedForDisplay(t *testing.T) {
	const contentW = 56
	longName := "feature/" + strings.Repeat("very-long-historical-branch-segment-", 4)
	m := NewModel(nil)
	m.Show(0, []string{"main", longName})
	m.SetCommitIdx(0)
	m.UpdateRepository(&internal.Repository{
		Graph: internal.CommitGraph{Commits: []internal.Commit{{ShortID: "abc1", Summary: "Short"}}},
	})
	syncContentWidth(&m, contentW)

	rows := renderRows(m.View())
	w, worst := maxVisibleWidth(rows)
	if w > contentW+2 {
		t.Fatalf("existing-bookmark row widened the modal: max=%d cols, expected <= %d.\nworst: %q",
			w, contentW+2, worst)
	}

	// The underlying name should still be retrievable in full — display truncation
	// must not have leaked into the model's GetExistingBookmarks state.
	if got := m.GetExistingBookmarks(); len(got) != 2 || got[1] != longName {
		t.Fatalf("existing-bookmark slice mutated: got %v", got)
	}

	// And the rendered view should contain an ellipsis marker, confirming we actually
	// truncated rather than (e.g.) collapsing to a single-char string.
	if !strings.Contains(ansi.Strip(m.View()), "…") {
		t.Fatalf("expected ellipsis in truncated bookmark display; view:\n%s", ansi.Strip(m.View()))
	}
}
