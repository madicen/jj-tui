package evologsplit

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	overlay "github.com/madicen/bubble-overlay"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	aitab "github.com/madicen/jj-tui/internal/tui/ai"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/styles"
	"github.com/mattn/go-runewidth"
)

// evologAIWrapMaxLines caps wrapped AI rationale / errors so short terminals stay usable.
const evologAIWrapMaxLines = 18

// Model is an experimental evolog-driven split picker (FAQ: move recent work into a new change).
type Model struct {
	shown         bool
	loading       bool
	loadErr       string
	bookmarkName  string
	tipChangeID   string
	tipCommitHint string
	entries       []jj.EvologEntry
	selectedIdx   int
	listScrollTop int
	// term size (SetDimensions from main)
	termW, termH     int
	listViewportRows int
	// diff: selected row vs the list row above (newer neighbor), evolog order is newest-first
	diffSeq     int
	diffLoading bool
	diffErr     string
	diffFiles   []jj.ChangedFile
	diffGitRaw  string // cached jj --git for the step; opened via overlay (o)
	zoneManager *zone.Manager

	suggestCfg       *config.Config
	suggestReqID     int
	suggestLoading   bool
	suggestSpinIdx   int       // mini-dot frame for overlay (driven by OverlaySpinTickMsg, not bubbles spinner)
	suggestStartedAt time.Time // wall clock when AI suggest began (for overlay hints)
	// suggestPrep* / suggestPhase: JJ batched diff summaries vs LLM (shown on suggest overlay).
	suggestPrepJJDone  int
	suggestPrepJJTotal int
	suggestPhase       string // "", "jj", "llm"
	suggestRationale   string
	suggestErrLine     string
	// describeAfterSplit: user toggle (d); sent on NavigatePerformEvologSplit.
	describeAfterSplit    bool
	suggestNoSplit        bool // last AI recommendation
	pendingFilesFirst     []string
	pendingHunkPeelRounds []map[string]int // AI hunk peel plan (each map = one jj split); nil when empty
	pendingMultiBaseIDs   []string
	noSplitConfirm        noSplitConfirm // double-Enter on same row after AI no_split
}

// NewModel creates the evolog split modal. zoneManager may be nil.
func NewModel(zm *zone.Manager) Model {
	return Model{
		zoneManager:      zm,
		listViewportRows: 6,
		termW:            100,
		termH:            24,
	}
}

func evologSuggestMiniDotFrame(idx int) string {
	fr := spinner.MiniDot.Frames
	if len(fr) == 0 {
		return "·"
	}
	idx %= len(fr)
	if idx < 0 {
		idx = 0
	}
	return fr[idx]
}

func (m *Model) SetZoneManager(zm *zone.Manager) { m.zoneManager = zm }

// WithSuggestConfig stores config for optional AI split suggestions (main sets when opening modal).
func (m Model) WithSuggestConfig(cfg *config.Config) Model {
	m.suggestCfg = cfg
	return m
}

// EvologEntries returns a copy of loaded evolog rows for async AI commands.
func (m Model) EvologEntries() []jj.EvologEntry {
	return append([]jj.EvologEntry(nil), m.entries...)
}

// SetDimensions records terminal size and viewport row count for scrolling.
func (m Model) SetDimensions(w, h int) Model {
	if w < 1 {
		w = 80
	}
	if h < 1 {
		h = 24
	}
	m.termW, m.termH = w, h
	vr := (h - 16) / 2
	if vr < 3 {
		vr = 3
	}
	if vr > 16 {
		vr = 16
	}
	m.listViewportRows = vr
	return m
}

// Show resets state for a new session (caller should batch LoadEvologCmd).
// describeAfterSplitDefault seeds the post-split AI describe toggle when AI is configured (user can still press d).
func (m *Model) Show(commit internal.Commit, bookmarkName string, describeAfterSplitDefault bool) {
	m.shown = true
	m.loading = true
	m.loadErr = ""
	m.bookmarkName = bookmarkName
	m.tipChangeID = commit.ChangeID
	m.tipCommitHint = commit.ID
	m.entries = nil
	m.selectedIdx = 0
	m.listScrollTop = 0
	m.diffSeq = 0
	m.diffLoading = false
	m.diffErr = ""
	m.diffFiles = nil
	m.diffGitRaw = ""
	m.suggestReqID = 0
	m.suggestLoading = false
	m.suggestSpinIdx = 0
	m.suggestStartedAt = time.Time{}
	m.suggestPrepJJDone = 0
	m.suggestPrepJJTotal = 0
	m.suggestPhase = ""
	m.suggestRationale = ""
	m.suggestErrLine = ""
	m.describeAfterSplit = describeAfterSplitDefault &&
		m.suggestCfg != nil && m.suggestCfg.AIConfiguredForGeneration()
	m.suggestNoSplit = false
	m.pendingFilesFirst = nil
	m.pendingHunkPeelRounds = nil
	m.pendingMultiBaseIDs = nil
	m.noSplitConfirm.reset()
}

// Hide clears the modal.
func (m *Model) Hide() {
	m.shown = false
	m.loading = false
	m.loadErr = ""
	m.entries = nil
	m.diffFiles = nil
	m.diffErr = ""
	m.diffLoading = false
	m.diffGitRaw = ""
	m.suggestLoading = false
	m.suggestSpinIdx = 0
	m.suggestStartedAt = time.Time{}
	m.suggestPrepJJDone = 0
	m.suggestPrepJJTotal = 0
	m.suggestPhase = ""
	m.suggestRationale = ""
	m.suggestErrLine = ""
	m.describeAfterSplit = false
	m.suggestNoSplit = false
	m.pendingFilesFirst = nil
	m.pendingHunkPeelRounds = nil
	m.pendingMultiBaseIDs = nil
	m.noSplitConfirm.reset()
}

// IsShown reports whether the modal is active.
func (m *Model) IsShown() bool { return m.shown }

// SuggestLoading is true while the async AI evolog-split suggestion is running.
func (m Model) SuggestLoading() bool { return m.suggestLoading }

// SuggestReqID is the id for the in-flight suggest request (stale async messages are ignored).
func (m Model) SuggestReqID() int { return m.suggestReqID }

// WithSuggestPrepProgress updates JJ-summary vs LLM phase for the suggest overlay.
func (m Model) WithSuggestPrepProgress(jjDone, jjTotal int, phase string) Model {
	m.suggestPrepJJDone = jjDone
	m.suggestPrepJJTotal = jjTotal
	m.suggestPhase = phase
	return m
}

func (m Model) mark(id, s string) string {
	if m.zoneManager == nil {
		return s
	}
	return m.zoneManager.Mark(id, s)
}

func (m Model) syncListScroll() Model {
	vr := m.listViewportRows
	if vr < 1 {
		vr = 5
	}
	maxScroll := max(0, len(m.entries)-vr)
	if m.listScrollTop > maxScroll {
		m.listScrollTop = maxScroll
	}
	if m.selectedIdx < m.listScrollTop {
		m.listScrollTop = m.selectedIdx
	}
	if m.selectedIdx >= m.listScrollTop+vr {
		m.listScrollTop = m.selectedIdx - vr + 1
	}
	if m.listScrollTop < 0 {
		m.listScrollTop = 0
	}
	return m
}

// refreshDiffPreview bumps the diff generation; loads jj diff when the selected row has a newer
// neighbor above it in the list (index selectedIdx-1). Row 0 is the tip — no neighbor, so no load.
func (m Model) refreshDiffPreview() (Model, tea.Cmd) {
	m.diffSeq++
	m.diffErr = ""
	if len(m.entries) == 0 || m.selectedIdx <= 0 || m.selectedIdx >= len(m.entries) {
		m.diffLoading = false
		m.diffFiles = nil
		m.diffErr = ""
		m.diffGitRaw = ""
		return m, nil
	}
	m.diffLoading = true
	m.diffErr = ""
	m.diffGitRaw = ""
	return m, EvologDiffLoadRequestedCmd()
}

// DiffSnapshotForLoad returns seq and revisions for the in-flight diff (after refreshDiffPreview).
// Diffs the selected commit against the previous list entry (newer), not cumulative vs tip.
// When selectedIdx >= 2, prevStepFrom/prevStepTo are the older→newer pair for the list row above the
// selected row (used to hide files whose patch is identical to the prior evolog step).
func (m Model) DiffSnapshotForLoad() (seq int, fromCommitID, toCommitID, prevStepFrom, prevStepTo string, ok bool) {
	if len(m.entries) == 0 || m.selectedIdx < 0 || m.selectedIdx >= len(m.entries) || m.selectedIdx == 0 {
		return 0, "", "", "", "", false
	}
	prev := strings.TrimSpace(m.entries[m.selectedIdx-1].CommitID)
	sel := strings.TrimSpace(m.entries[m.selectedIdx].CommitID)
	if prev == "" || sel == "" {
		return 0, "", "", "", "", false
	}
	psf, pst := "", ""
	if m.selectedIdx >= 2 {
		psf = strings.TrimSpace(m.entries[m.selectedIdx-1].CommitID)
		pst = strings.TrimSpace(m.entries[m.selectedIdx-2].CommitID)
		if psf == "" || pst == "" {
			psf, pst = "", ""
		}
	}
	return m.diffSeq, sel, prev, psf, pst, true
}

func (m Model) canOpenStepPatchOverlay() bool {
	return m.selectedIdx > 0 &&
		len(m.entries) > m.selectedIdx &&
		!m.diffLoading &&
		m.diffErr == "" &&
		strings.TrimSpace(m.diffGitRaw) != ""
}

func (m Model) openStepPatchNavigate() tea.Cmd {
	if !m.canOpenStepPatchOverlay() {
		return nil
	}
	sel := m.entries[m.selectedIdx]
	prev := m.entries[m.selectedIdx-1]
	sub := fmt.Sprintf("%s → %s", sel.CommitIDShort, prev.CommitIDShort)
	return state.NavigateTarget{
		Kind:                    state.NavigateOpenFileDiff,
		FileDiffRawGit:          m.diffGitRaw,
		FileDiffOverlayTitle:    "Evolog step",
		FileDiffOverlaySubtitle: sub,
	}.Cmd()
}

func (m Model) performSplitNavigateCmd() tea.Cmd {
	bases := append([]string(nil), m.pendingMultiBaseIDs...)
	filesets := append([]string(nil), m.pendingFilesFirst...)
	hunkPeels := cloneHunkPeelRounds(m.pendingHunkPeelRounds)
	var remainder []string
	runBases := bases
	if m.suggestCfg != nil && m.suggestCfg.EvologAIMultiSplitStepwise() && len(bases) > 1 {
		remainder = append([]string(nil), bases[1:]...)
		runBases = []string{bases[0]}
		if len(remainder) > 0 {
			filesets = nil
			hunkPeels = nil
		}
	}
	baseID := m.entries[m.selectedIdx].CommitID
	if len(runBases) == 1 {
		baseID = runBases[0]
	}
	return state.NavigateTarget{
		Kind:                     state.NavigatePerformEvologSplit,
		EvologBookmarkName:       m.bookmarkName,
		EvologTipChangeID:        m.tipChangeID,
		EvologTipCommitHint:      m.tipCommitHint,
		EvologBaseCommitID:       baseID,
		EvologDescribeAfterSplit: m.describeAfterSplit,
		EvologFilesetsFirst:      filesets,
		EvologHunkPeelRounds:     hunkPeels,
		EvologMultiBaseCommitIDs: runBases,
		EvologStepwiseRemainder:  remainder,
	}.Cmd()
}

// SetPendingMultiSplitIDs updates AI multi-split bases shown after a stepwise step (main calls after reload).
func (m *Model) SetPendingMultiSplitIDs(ids []string) {
	m.pendingMultiBaseIDs = append([]string(nil), ids...)
}

// Update handles keys, zone clicks, mouse wheel, and async messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.shown {
		return m, nil
	}
	switch msg := msg.(type) {
	case OverlaySpinTickMsg:
		if !m.suggestLoading {
			return m, nil
		}
		if n := len(spinner.MiniDot.Frames); n > 0 {
			m.suggestSpinIdx = (m.suggestSpinIdx + 1) % n
		}
		return m, OverlaySpinCmd()

	case EvologLoadedMsg:
		m.loading = false
		m.suggestLoading = false
		m.suggestSpinIdx = 0
		m.suggestStartedAt = time.Time{}
		m.suggestPrepJJDone = 0
		m.suggestPrepJJTotal = 0
		m.suggestPhase = ""
		m.suggestRationale = ""
		m.suggestErrLine = ""
		m.suggestNoSplit = false
		m.pendingFilesFirst = nil
		m.pendingHunkPeelRounds = nil
		m.pendingMultiBaseIDs = nil
		m.noSplitConfirm.reset()
		if msg.Err != nil {
			m.loadErr = msg.Err.Error()
			return m, nil
		}
		m.entries = msg.Entries
		if len(m.entries) > 0 {
			if len(m.entries) > 1 {
				m.selectedIdx = 1
			} else {
				m.selectedIdx = 0
			}
		}
		m.listScrollTop = 0
		m = m.syncListScroll()
		return m.refreshDiffPreview()

	case EvologSplitDiffLoadedMsg:
		if msg.Seq != m.diffSeq {
			return m, nil
		}
		m.diffLoading = false
		if msg.Err != nil {
			m.diffErr = msg.Err.Error()
			m.diffFiles = nil
			m.diffGitRaw = ""
		} else {
			m.diffErr = ""
			m.diffFiles = msg.Files
			m.diffGitRaw = msg.GitDiff
		}
		return m, nil

	case aitab.EvologSplitSuggestMsg:
		if msg.ReqID != m.suggestReqID {
			return m, nil
		}
		m.suggestLoading = false
		m.suggestSpinIdx = 0
		m.suggestStartedAt = time.Time{}
		m.suggestPrepJJDone = 0
		m.suggestPrepJJTotal = 0
		m.suggestPhase = ""
		m.suggestErrLine = ""
		if msg.Err != nil {
			m.suggestErrLine = msg.Err.Error()
			m.suggestRationale = ""
			m.suggestNoSplit = false
			m.pendingFilesFirst = nil
			m.pendingHunkPeelRounds = nil
			m.pendingMultiBaseIDs = nil
			m.noSplitConfirm.reset()
			return m, nil
		}
		m.suggestNoSplit = msg.NoSplit
		m.pendingFilesFirst = append([]string(nil), msg.FilesForFirstCommit...)
		if len(msg.HunkPeelRounds) > 0 {
			m.pendingHunkPeelRounds = cloneHunkPeelRounds(msg.HunkPeelRounds)
		} else {
			m.pendingHunkPeelRounds = nil
		}
		m.pendingMultiBaseIDs = append([]string(nil), msg.MultiSplitBaseCommitIDs...)
		m.suggestRationale = msg.Rationale
		if msg.NoSplit {
			m.noSplitConfirm.onAISuggestNoSplit(m.selectedIdx)
			return m, nil
		}
		m.noSplitConfirm.reset()
		if msg.PickIndex >= 1 && msg.PickIndex < len(m.entries) {
			m.selectedIdx = msg.PickIndex
			m = m.syncListScroll()
			return m.refreshDiffPreview()
		}
		m.suggestErrLine = "Model returned an invalid row index"
		return m, nil

	case tea.MouseMsg:
		isWheel := tea.MouseEvent(msg).IsWheel() || msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown
		if !isWheel || m.loading || m.loadErr != "" || len(m.entries) == 0 {
			break
		}
		vr := m.listViewportRows
		if vr < 1 {
			vr = 5
		}
		maxScroll := max(0, len(m.entries)-vr)
		if msg.Button == tea.MouseButtonWheelUp {
			m.listScrollTop = max(0, m.listScrollTop-1)
		} else {
			m.listScrollTop = min(maxScroll, m.listScrollTop+1)
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case zone.MsgZoneInBounds:
		if m.zoneManager != nil {
			if zid := m.resolveClickedZone(msg); zid != "" {
				return m.handleZoneClick(zid)
			}
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.loading {
		if msg.String() == "esc" {
			m.shown = false
			m.loading = false
			return m, state.NavigateTarget{Kind: state.NavigateBackToGraph}.Cmd()
		}
		return m, nil
	}
	if m.loadErr != "" {
		if msg.String() == "esc" || msg.String() == "enter" {
			m.shown = false
			return m, state.NavigateTarget{Kind: state.NavigateBackToGraph}.Cmd()
		}
		return m, nil
	}
	vr := m.listViewportRows
	if vr < 1 {
		vr = 5
	}
	maxScroll := max(0, len(m.entries)-vr)

	switch msg.String() {
	case "esc":
		m.shown = false
		return m, state.NavigateTarget{Kind: state.NavigateBackToGraph, StatusMessage: "Split cancelled"}.Cmd()
	case "enter":
		if len(m.entries) == 0 || m.selectedIdx < 0 || m.selectedIdx >= len(m.entries) {
			return m, nil
		}
		if noSplitFirstEnterOnlyArms(m.suggestNoSplit, m.selectedIdx, m.noSplitConfirm) {
			m.noSplitConfirm.armed = true
			return m, nil
		}
		m.noSplitConfirm.reset()
		return m, m.performSplitNavigateCmd()
	case "d":
		if m.suggestCfg != nil && m.suggestCfg.AIConfiguredForGeneration() {
			m.describeAfterSplit = !m.describeAfterSplit
		}
		return m, nil
	case "c":
		if len(m.pendingFilesFirst) > 0 {
			m.pendingFilesFirst = nil
		}
		if len(m.pendingHunkPeelRounds) > 0 {
			m.pendingHunkPeelRounds = nil
		}
		return m, nil
	case "o":
		cmd := m.openStepPatchNavigate()
		if cmd == nil {
			return m, nil
		}
		return m, cmd
	case "j", "down":
		if m.selectedIdx < len(m.entries)-1 {
			m.selectedIdx++
			m.noSplitConfirm.onSelectedIdxChange(m.selectedIdx)
			m = m.syncListScroll()
			return m.refreshDiffPreview()
		}
		return m, nil
	case "k", "up":
		if m.selectedIdx > 0 {
			m.selectedIdx--
			m.noSplitConfirm.onSelectedIdxChange(m.selectedIdx)
			m = m.syncListScroll()
			return m.refreshDiffPreview()
		}
		return m, nil
	case "pgdown", "ctrl+f":
		m.listScrollTop = min(maxScroll, m.listScrollTop+vr)
		return m, nil
	case "pgup", "ctrl+b":
		m.listScrollTop = max(0, m.listScrollTop-vr)
		return m, nil
	case "s":
		if m.suggestLoading || len(m.entries) < 2 {
			return m, nil
		}
		if m.suggestCfg == nil || !m.suggestCfg.AIConfiguredForGeneration() {
			m.suggestErrLine = "Enable AI (Settings → Advanced) and set an API key"
			return m, nil
		}
		m.suggestReqID++
		rid := m.suggestReqID
		m.suggestLoading = true
		m.suggestStartedAt = time.Now()
		m.suggestPrepJJTotal = aitab.EvologSplitJJStepTotal(len(m.entries))
		m.suggestPrepJJDone = 0
		m.suggestPhase = "jj"
		m.suggestErrLine = ""
		m.suggestRationale = ""
		return m, tea.Batch(
			func() tea.Msg { return EvologSplitSuggestRequestedMsg{ReqID: rid} },
			func() tea.Msg { return OverlaySpinTickMsg{Time: time.Now()} },
		)
	}
	return m, nil
}

func (m Model) ZoneIDs() []string {
	ids := make([]string, 0, len(m.entries)+4)
	for i := range m.entries {
		ids = append(ids, mouse.ZoneEvologSplitEntry(i))
	}
	ids = append(ids, mouse.ZoneEvologSplitSuggest, mouse.ZoneEvologSplitConfirm, mouse.ZoneEvologSplitCancel, mouse.ZoneEvologSplitViewPatch)
	return ids
}

func (m Model) resolveClickedZone(msg zone.MsgZoneInBounds) string {
	if msg.Zone == nil {
		return ""
	}
	for _, id := range m.ZoneIDs() {
		z := m.zoneManager.Get(id)
		if z != nil && z.InBounds(msg.Event) {
			return id
		}
	}
	return ""
}

func (m Model) handleZoneClick(zoneID string) (Model, tea.Cmd) {
	const prefix = "zone:evologsplit:entry:"
	if strings.HasPrefix(zoneID, prefix) {
		idx, err := strconv.Atoi(strings.TrimPrefix(zoneID, prefix))
		if err == nil && idx >= 0 && idx < len(m.entries) {
			m.selectedIdx = idx
			m.noSplitConfirm.onSelectedIdxChange(m.selectedIdx)
			m = m.syncListScroll()
			return m.refreshDiffPreview()
		}
	}
	if zoneID == mouse.ZoneEvologSplitSuggest {
		upd, cmd := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		return upd, cmd
	}
	if zoneID == mouse.ZoneEvologSplitConfirm {
		if len(m.entries) == 0 || m.selectedIdx < 0 || m.selectedIdx >= len(m.entries) {
			return m, nil
		}
		if noSplitFirstEnterOnlyArms(m.suggestNoSplit, m.selectedIdx, m.noSplitConfirm) {
			m.noSplitConfirm.armed = true
			return m, nil
		}
		m.noSplitConfirm.reset()
		return m, m.performSplitNavigateCmd()
	}
	if zoneID == mouse.ZoneEvologSplitCancel {
		m.shown = false
		return m, state.NavigateTarget{Kind: state.NavigateBackToGraph, StatusMessage: "Split cancelled"}.Cmd()
	}
	if zoneID == mouse.ZoneEvologSplitViewPatch {
		cmd := m.openStepPatchNavigate()
		if cmd == nil {
			return m, nil
		}
		return m, cmd
	}
	return m, nil
}

func formatSplitModalFileLine(f jj.ChangedFile, pathMax int) string {
	p := runewidth.Truncate(f.Path, max(8, pathMax), "…")
	st, ch := styles.GetStatusStyle(f.Status)
	return fmt.Sprintf("%s %s%s", st.Render(ch), p, styles.DiffStatsSuffix(f.LinesAdded, f.LinesRemoved, f.StatsOK))
}

// buildEvologSplitRightColumn returns exactly vr+1 lines (title + vr body rows) so layout
// matches the history column; the last body row is always used (overflow count or "—").
func buildEvologSplitRightColumn(m Model, vr, rightW int, muted lipgloss.Style) []string {
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F85149"))
	title := muted.Render("Files vs row above (step →)")
	if vr < 1 {
		vr = 1
	}
	body := make([]string, vr)

	if m.diffLoading {
		body[0] = muted.Render("Loading diff…")
		return append([]string{title}, body...)
	}
	if m.diffErr != "" {
		body[0] = errStyle.Render(runewidth.Truncate(m.diffErr, max(4, rightW-2), "…"))
		return append([]string{title}, body...)
	}
	if m.selectedIdx == 0 && len(m.entries) > 0 {
		body[0] = muted.Render("(tip row — move down for step diff)")
		return append([]string{title}, body...)
	}
	if len(m.diffFiles) == 0 {
		body[0] = muted.Render("(no file changes in this step)")
		return append([]string{title}, body...)
	}

	if vr == 1 {
		if len(m.diffFiles) == 1 {
			body[0] = formatSplitModalFileLine(m.diffFiles[0], rightW-4)
		} else {
			body[0] = muted.Render(fmt.Sprintf("… +%d files", len(m.diffFiles)))
		}
		return append([]string{title}, body...)
	}

	fileSlots := vr - 1
	for i := 0; i < fileSlots && i < len(m.diffFiles); i++ {
		body[i] = formatSplitModalFileLine(m.diffFiles[i], rightW-4)
	}
	last := vr - 1
	extra := len(m.diffFiles) - fileSlots
	if extra > 0 {
		body[last] = muted.Render(fmt.Sprintf("… +%d more", extra))
	} else {
		body[last] = muted.Render("—")
	}
	return append([]string{title}, body...)
}

func formatSuggestWaitDuration(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Second {
		return "0s"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%02ds", m, s)
}

// renderEvologSuggestSpinnerOverlay draws a centered box on the modal; elapsed drives “still working” hints.
func renderEvologSuggestSpinnerOverlay(base, spinGlyph string, elapsed time.Duration, maxLine int, jjDone, jjTotal int, phase string) string {
	if maxLine < 36 {
		maxLine = 36
	}
	muted := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	spin := lipgloss.NewStyle().Foreground(styles.ColorSecondary).Render(spinGlyph)
	row1 := lipgloss.JoinHorizontal(lipgloss.Center, spin, " ", muted.Render("Analyzing evolog with AI…"))
	var rows []string
	rows = append(rows, row1)
	if phase == "llm" {
		rows = append(rows, muted.Render(runewidth.Truncate("Calling AI model…", maxLine, "…")))
	} else if phase == "jj" && jjTotal > 0 {
		t := fmt.Sprintf("JJ diff summaries: %d / %d", jjDone, jjTotal)
		rows = append(rows, muted.Render(runewidth.Truncate(t, maxLine, "…")))
	}
	if elapsed >= 12*time.Second {
		t := "Elapsed " + formatSuggestWaitDuration(elapsed) + " — jj diff prep + LLM can take minutes on large histories."
		rows = append(rows, muted.Render(runewidth.Truncate(t, maxLine, "…")))
	}
	if elapsed >= 75*time.Second {
		t := "If the spinner moves, the app is still working (not frozen)."
		rows = append(rows, muted.Render(runewidth.Truncate(t, maxLine, "…")))
	}
	if elapsed >= 3*time.Minute {
		t := "Still no reply — when this finishes, check API/network; raise ai_timeout_seconds (Settings → Advanced) if errors mention timeout."
		rows = append(rows, muted.Render(runewidth.Truncate(t, maxLine, "…")))
	}
	inner := strings.Join(rows, "\n")
	box := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(styles.ColorMuted).
		Background(styles.HeaderBarBackground).
		Padding(0, 1).
		Render(inner)
	baseLines := strings.Split(base, "\n")
	bh := len(baseLines)
	bw := 0
	for _, l := range baseLines {
		if w := lipgloss.Width(l); w > bw {
			bw = w
		}
	}
	boxLines := strings.Split(box, "\n")
	h := len(boxLines)
	mw := 0
	for _, l := range boxLines {
		if w := lipgloss.Width(l); w > mw {
			mw = w
		}
	}
	top := max((bh-h)/2, 0)
	left := max((bw-mw)/2, 0)
	return overlay.OverlayView(base, box, bw, bh, top, left)
}

// View renders the modal.
func (m Model) View() string {
	if !m.shown {
		return ""
	}
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8BE9FD"))
	muted := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	var lines []string
	lines = append(lines, titleStyle.Render("Split"))
	lines = append(lines, "")
	if strings.TrimSpace(m.bookmarkName) != "" {
		lines = append(lines, muted.Render(fmt.Sprintf("Bookmark: %s  ·  change: %s", m.bookmarkName, m.tipChangeID)))
	} else {
		lines = append(lines, muted.Render(fmt.Sprintf("Change: %s (no bookmark — only this revision is moved)", m.tipChangeID)))
	}
	lines = append(lines, muted.Render("Pick a parent revision; the new commit gets the current tip’s tree."))
	lines = append(lines, "")

	if m.loading {
		lines = append(lines, "Loading jj evolog…")
		box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(styles.ColorMuted).Padding(1, 2).Width(min(m.termW-4, 120))
		return box.Render(strings.Join(lines, "\n"))
	}
	if m.loadErr != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#F85149")).Render("Error: "+m.loadErr))
		lines = append(lines, "")
		lines = append(lines, muted.Render("Esc or Enter to close"))
		box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(styles.ColorMuted).Padding(1, 2).Width(min(m.termW-4, 120))
		return box.Render(strings.Join(lines, "\n"))
	}

	modalW := min(m.termW-4, 120)
	if modalW < 72 {
		modalW = 72
	}
	leftW := max(38, modalW*42/100)
	rightW := modalW - leftW - 3
	if rightW < 28 {
		rightW = 28
		leftW = modalW - rightW - 3
	}

	vr := m.listViewportRows
	if vr < 1 {
		vr = 5
	}

	selStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary)
	normal := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	cidStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))

	var leftLines []string
	leftLines = append(leftLines, muted.Render("History (j/k · PgUp/PgDn scroll)"))
	end := min(m.listScrollTop+vr, len(m.entries))
	for i := m.listScrollTop; i < end; i++ {
		e := m.entries[i]
		prefix := "  "
		style := normal
		if i == m.selectedIdx {
			prefix = "► "
			style = selStyle
		}
		sum := e.Summary
		maxSum := leftW - 22
		if maxSum < 8 {
			maxSum = 8
		}
		sum = runewidth.Truncate(sum, maxSum, "…")
		line := fmt.Sprintf("%s%s  %s", prefix, cidStyle.Render(e.CommitIDShort), style.Render(sum))
		leftLines = append(leftLines, m.mark(mouse.ZoneEvologSplitEntry(i), line))
	}
	for len(leftLines) < vr+1 {
		leftLines = append(leftLines, "")
	}
	leftBlock := lipgloss.NewStyle().Width(leftW).Render(strings.Join(leftLines, "\n"))

	rightLines := buildEvologSplitRightColumn(m, vr, rightW, muted)
	rightBlock := lipgloss.NewStyle().Width(rightW).Render(strings.Join(rightLines, "\n"))

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, "  ", rightBlock)
	lines = append(lines, body)
	lines = append(lines, "")
	var patchBtn string
	if m.canOpenStepPatchOverlay() {
		patchBtn = m.mark(mouse.ZoneEvologSplitViewPatch, styles.ButtonStyle.Render("View patch (o)"))
	} else {
		patchBtn = lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("View patch (o)")
	}
	aiReady := m.suggestCfg != nil && m.suggestCfg.AIConfiguredForGeneration() && len(m.entries) >= 2 && !m.suggestLoading
	var suggestBtn string
	switch {
	case m.suggestLoading:
		suggestBtn = lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Suggesting…")
	case aiReady:
		suggestBtn = m.mark(mouse.ZoneEvologSplitSuggest, styles.ButtonStyle.Render("Suggest split (s)"))
	default:
		suggestBtn = lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Suggest (s) — AI off")
	}
	var confirm string
	if noSplitFirstEnterOnlyArms(m.suggestNoSplit, m.selectedIdx, m.noSplitConfirm) {
		confirm = m.mark(mouse.ZoneEvologSplitConfirm, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Split here (Enter again to confirm)"))
	} else {
		confirm = m.mark(mouse.ZoneEvologSplitConfirm, styles.ButtonStyle.Render("Split here (Enter)"))
	}
	cancel := m.mark(mouse.ZoneEvologSplitCancel, styles.ButtonSecondaryStyle.Render("Cancel (Esc)"))
	lines = append(lines, patchBtn+"  "+suggestBtn)
	lines = append(lines, confirm+"  "+cancel)
	lines = append(lines, "")
	textWrapW := max(16, modalW-6)
	if m.suggestNoSplit && strings.TrimSpace(m.suggestRationale) != "" {
		lines = append(lines, renderEvologModalWrapped(
			"AI: no split recommended — "+m.suggestRationale,
			textWrapW,
			evologAIWrapMaxLines,
			lipgloss.NewStyle().Foreground(lipgloss.Color("#E3B341")),
			muted,
		))
		if noSplitFirstEnterOnlyArms(m.suggestNoSplit, m.selectedIdx, m.noSplitConfirm) {
			lines = append(lines, muted.Render("Press Enter again to split at this row, or j/k to pick another row."))
		}
	} else if m.suggestRationale != "" {
		lines = append(lines, renderEvologModalWrapped(
			"AI: "+m.suggestRationale,
			textWrapW,
			evologAIWrapMaxLines,
			lipgloss.NewStyle().Foreground(styles.ColorSecondary),
			muted,
		))
	}
	if m.suggestErrLine != "" {
		lines = append(lines, renderEvologModalWrapped(
			m.suggestErrLine,
			textWrapW,
			evologAIWrapMaxLines,
			lipgloss.NewStyle().Foreground(lipgloss.Color("#F85149")),
			muted,
		))
	}
	if m.suggestCfg != nil && m.suggestCfg.AIConfiguredForGeneration() {
		dstate := "off"
		if m.describeAfterSplit {
			dstate = "on"
		}
		lines = append(lines, muted.Render("d — AI describe @- and @ after split: "+dstate))
	}
	if len(m.pendingMultiBaseIDs) > 1 {
		lines = append(lines, muted.Render(fmt.Sprintf("AI plan: %d sequential FAQ splits (deepest first)", len(m.pendingMultiBaseIDs))))
		if m.suggestCfg != nil && m.suggestCfg.EvologAIMultiSplitStepwise() {
			lines = append(lines, muted.Render("Stepwise multi-split: one FAQ base per Enter. Disable it in Settings → Advanced to run every FAQ base in one Enter (batch)."))
		}
	}
	if len(m.pendingFilesFirst) > 0 {
		preview := strings.Join(m.pendingFilesFirst, ", ")
		fileLine := "AI file split (after FAQ): " + preview + " (c clears)"
		lines = append(lines, renderEvologModalWrapped(fileLine, textWrapW, evologAIWrapMaxLines, muted, muted))
	}
	if len(m.pendingHunkPeelRounds) > 0 {
		var parts []string
		for ri, round := range m.pendingHunkPeelRounds {
			var rp []string
			for p, k := range round {
				rp = append(rp, fmt.Sprintf("%s:%d", p, k))
			}
			sort.Strings(rp)
			parts = append(parts, fmt.Sprintf("r%d:%s", ri+1, strings.Join(rp, ",")))
		}
		prev := strings.Join(parts, " · ")
		hunkLine := fmt.Sprintf("AI hunk peel (%d round(s) after FAQ): %s (c clears)", len(m.pendingHunkPeelRounds), prev)
		lines = append(lines, renderEvologModalWrapped(hunkLine, textWrapW, evologAIWrapMaxLines, muted, muted))
	}
	lines = append(lines, "")
	lines = append(lines, muted.Render("Wheel scrolls history · o step diff · s AI suggest · d post-split describe · c clear AI file/hunk plan · Do not pick the tip row as base"))

	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(styles.ColorMuted).Padding(1, 2).Width(modalW)
	out := box.Render(strings.Join(lines, "\n"))
	if m.suggestLoading {
		var elapsed time.Duration
		if !m.suggestStartedAt.IsZero() {
			elapsed = time.Since(m.suggestStartedAt)
		}
		out = renderEvologSuggestSpinnerOverlay(out, evologSuggestMiniDotFrame(m.suggestSpinIdx), elapsed, modalW-4, m.suggestPrepJJDone, m.suggestPrepJJTotal, m.suggestPhase)
	}
	return out
}

func cloneHunkPeelRounds(in []map[string]int) []map[string]int {
	if len(in) == 0 {
		return nil
	}
	out := make([]map[string]int, len(in))
	for i, m := range in {
		out[i] = make(map[string]int, len(m))
		for k, v := range m {
			out[i][k] = v
		}
	}
	return out
}

// renderEvologModalWrapped word-wraps plain text to wrapW and limits how many
// lines are shown so the split modal stays readable (replaces one-line truncation).
func renderEvologModalWrapped(text string, wrapW, maxLines int, sty lipgloss.Style, omitNote lipgloss.Style) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if wrapW < 12 {
		wrapW = 12
	}
	out := sty.Width(wrapW).Render(text)
	if maxLines <= 0 {
		return out
	}
	ls := strings.Split(out, "\n")
	if len(ls) <= maxLines {
		return out
	}
	return strings.Join(ls[:maxLines], "\n") + "\n" +
		omitNote.Render(fmt.Sprintf("… +%d line(s) — widen terminal or enlarge window for more", len(ls)-maxLines))
}
