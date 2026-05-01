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

// evologOutcomeOverlayBodyLines is the scrollable body height inside the outcome preview overlay.
const evologOutcomeOverlayBodyLines = 18

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
	// pendingExpectedChain / pendingPrecomputedDescribe* mirror the last AI suggest (chain preview LLM).
	pendingExpectedChain         []aitab.EvologSplitExpectedChainStep
	pendingPrecomputedParentDesc string
	pendingPrecomputedChildDesc  string
	noSplitConfirm               noSplitConfirm // double-Enter on same row after AI no_split

	// outcome preview overlay (p): synthetic graph + @- → @ diff summary
	outcomePreviewOpen    bool
	outcomePreviewSeq     int
	outcomePreviewScroll  int
	outcomePreviewSummary []string
	outcomePreviewErr     string
	outcomePreviewLoading bool
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
	m.pendingExpectedChain = nil
	m.pendingPrecomputedParentDesc = ""
	m.pendingPrecomputedChildDesc = ""
	m.noSplitConfirm.reset()
	m.resetOutcomePreview()
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
	m.pendingExpectedChain = nil
	m.pendingPrecomputedParentDesc = ""
	m.pendingPrecomputedChildDesc = ""
	m.noSplitConfirm.reset()
	m.resetOutcomePreview()
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

func (m Model) canOpenOutcomePreview() bool {
	return !m.loading && m.loadErr == "" && len(m.entries) > 0 && !m.suggestLoading
}

// canRunSuggestedSplit is true when Enter would start a split (parent row below tip is selected).
func (m Model) canRunSuggestedSplit() bool {
	return len(m.entries) > 0 && m.selectedIdx > 0 && m.selectedIdx < len(m.entries)
}

// commitSplitOrArmNoSplit runs the same logic as Enter on the modal: arm no_split confirm, or NavigatePerformEvologSplit.
// closeOverlay is true when the outcome preview should close (armed no_split or split started).
func (m Model) commitSplitOrArmNoSplit() (Model, tea.Cmd, bool) {
	if len(m.entries) == 0 || m.selectedIdx < 0 || m.selectedIdx >= len(m.entries) {
		return m, nil, false
	}
	if m.selectedIdx == 0 {
		return m, nil, false
	}
	if noSplitFirstEnterOnlyArms(m.suggestNoSplit, m.selectedIdx, m.noSplitConfirm) {
		m.noSplitConfirm.armed = true
		return m, nil, true
	}
	m.noSplitConfirm.reset()
	return m, m.performSplitNavigateCmd(), true
}

func (m Model) hasAISplitPlan() bool {
	if len(m.pendingExpectedChain) > 0 || len(m.pendingFilesFirst) > 0 || len(m.pendingHunkPeelRounds) > 0 {
		return true
	}
	if len(m.pendingMultiBaseIDs) > 0 {
		return true
	}
	return strings.TrimSpace(m.pendingPrecomputedChildDesc) != "" || strings.TrimSpace(m.pendingPrecomputedParentDesc) != ""
}

// openOutcomePreviewWithLoad opens the outcome overlay and schedules jj diff --summary @- → @.
func (m Model) openOutcomePreviewWithLoad() (Model, tea.Cmd) {
	if !m.canOpenOutcomePreview() {
		return m, nil
	}
	m.outcomePreviewSeq++
	seq := m.outcomePreviewSeq
	m.outcomePreviewOpen = true
	m.outcomePreviewScroll = 0
	m.outcomePreviewLoading = true
	m.outcomePreviewErr = ""
	m.outcomePreviewSummary = nil
	return m, EvologOutcomePreviewRequestedCmd(seq)
}

func (m Model) outcomePreviewWrapW() int {
	modalW := min(m.termW-4, 120)
	if modalW < 72 {
		modalW = 72
	}
	return max(40, modalW-10)
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
	// Stepwise mode normally runs one FAQ base at a time so the user can review evolog between steps.
	// When the plan preview overlay is open, Enter promises a full run (same as the preview); do not
	// peel off remainder here — that path left the modal up, dropped file/hunk automation for later
	// steps, and cleared precomputed descriptions before post-split describe could run.
	if m.suggestCfg != nil && m.suggestCfg.EvologAIMultiSplitStepwise() && len(bases) > 1 && !m.outcomePreviewOpen {
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
		Kind:                            state.NavigatePerformEvologSplit,
		EvologBookmarkName:              m.bookmarkName,
		EvologTipChangeID:               m.tipChangeID,
		EvologTipCommitHint:             m.tipCommitHint,
		EvologBaseCommitID:              baseID,
		EvologDescribeAfterSplit:        m.describeAfterSplit,
		EvologPrecomputedDescribeParent: m.pendingPrecomputedParentDesc,
		EvologPrecomputedDescribeChild:  m.pendingPrecomputedChildDesc,
		EvologFilesetsFirst:             filesets,
		EvologHunkPeelRounds:            hunkPeels,
		EvologMultiBaseCommitIDs:        runBases,
		EvologStepwiseRemainder:         remainder,
	}.Cmd()
}

// SetPendingMultiSplitIDs updates AI multi-split bases shown after a stepwise step (main calls after reload).
func (m *Model) SetPendingMultiSplitIDs(ids []string) {
	m.pendingMultiBaseIDs = append([]string(nil), ids...)
	// Outcome preview was for the full plan; after a partial FAQ step it is no longer trustworthy.
	m.pendingExpectedChain = nil
	m.pendingPrecomputedParentDesc = ""
	m.pendingPrecomputedChildDesc = ""
	m.resetOutcomePreview()
}

func (m *Model) resetOutcomePreview() {
	m.outcomePreviewOpen = false
	m.outcomePreviewSeq = 0
	m.outcomePreviewScroll = 0
	m.outcomePreviewSummary = nil
	m.outcomePreviewErr = ""
	m.outcomePreviewLoading = false
}

// ResetOutcomePreviewForPerformSplit closes the plan preview and invalidates in-flight preview loads.
// Main calls this when starting NavigatePerformEvologSplit so UI state matches the keyboard Enter path.
func (m *Model) ResetOutcomePreviewForPerformSplit() { m.resetOutcomePreview() }

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
		m.pendingExpectedChain = nil
		m.pendingPrecomputedParentDesc = ""
		m.pendingPrecomputedChildDesc = ""
		m.noSplitConfirm.reset()
		m.resetOutcomePreview()
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

	case EvologOutcomePreviewLoadedMsg:
		if msg.Seq != m.outcomePreviewSeq {
			return m, nil
		}
		m.outcomePreviewLoading = false
		if msg.Err != nil {
			m.outcomePreviewErr = msg.Err.Error()
			m.outcomePreviewSummary = nil
		} else {
			m.outcomePreviewErr = ""
			m.outcomePreviewSummary = append([]string(nil), msg.Lines...)
		}
		m.outcomePreviewScroll = 0
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
			m.pendingExpectedChain = nil
			m.pendingPrecomputedParentDesc = ""
			m.pendingPrecomputedChildDesc = ""
			m.noSplitConfirm.reset()
			m.resetOutcomePreview()
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
		m.pendingExpectedChain = append([]aitab.EvologSplitExpectedChainStep(nil), msg.ExpectedOutcomeChain...)
		m.pendingPrecomputedParentDesc = strings.TrimSpace(msg.PrecomputedDescribeParent)
		m.pendingPrecomputedChildDesc = strings.TrimSpace(msg.PrecomputedDescribeChild)
		m.resetOutcomePreview()
		if msg.NoSplit {
			m.noSplitConfirm.onAISuggestNoSplit(m.selectedIdx)
			return m, nil
		}
		m.noSplitConfirm.reset()
		if msg.PickIndex >= 1 && msg.PickIndex < len(m.entries) {
			m.selectedIdx = msg.PickIndex
			m = m.syncListScroll()
			m, diffCmd := m.refreshDiffPreview()
			if !m.suggestNoSplit {
				m2, prevCmd := m.openOutcomePreviewWithLoad()
				m = m2
				if diffCmd != nil {
					return m, tea.Batch(diffCmd, prevCmd)
				}
				return m, prevCmd
			}
			return m, diffCmd
		}
		m.suggestErrLine = "Model returned an invalid row index"
		return m, nil

	case tea.MouseMsg:
		isWheel := tea.MouseEvent(msg).IsWheel() || msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown
		if m.outcomePreviewOpen && isWheel && !m.loading && m.loadErr == "" {
			wrapW := m.outcomePreviewWrapW()
			total := len(m.buildOutcomePreviewAllLines(wrapW))
			maxScroll := max(0, total-evologOutcomeOverlayBodyLines)
			if msg.Button == tea.MouseButtonWheelUp {
				m.outcomePreviewScroll = max(0, m.outcomePreviewScroll-3)
			} else {
				m.outcomePreviewScroll = min(maxScroll, m.outcomePreviewScroll+3)
			}
			return m, nil
		}
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
	if m.outcomePreviewOpen {
		wrapW := m.outcomePreviewWrapW()
		total := len(m.buildOutcomePreviewAllLines(wrapW))
		maxScroll := max(0, total-evologOutcomeOverlayBodyLines)
		switch msg.String() {
		case "esc", "q":
			m.outcomePreviewOpen = false
			m.outcomePreviewScroll = 0
			return m, nil
		case "j", "down":
			m.outcomePreviewScroll = min(maxScroll, m.outcomePreviewScroll+1)
			return m, nil
		case "k", "up":
			m.outcomePreviewScroll = max(0, m.outcomePreviewScroll-1)
			return m, nil
		case "pgdown", "ctrl+f":
			m.outcomePreviewScroll = min(maxScroll, m.outcomePreviewScroll+evologOutcomeOverlayBodyLines)
			return m, nil
		case "pgup", "ctrl+b":
			m.outcomePreviewScroll = max(0, m.outcomePreviewScroll-evologOutcomeOverlayBodyLines)
			return m, nil
		case "enter":
			upd, cmd, closeOverlay := m.commitSplitOrArmNoSplit()
			if closeOverlay {
				upd.outcomePreviewOpen = false
				upd.outcomePreviewScroll = 0
			}
			return upd, cmd
		default:
			return m, nil
		}
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
		upd, cmd, _ := m.commitSplitOrArmNoSplit()
		return upd, cmd
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
		m.pendingExpectedChain = nil
		m.pendingPrecomputedParentDesc = ""
		m.pendingPrecomputedChildDesc = ""
		m.resetOutcomePreview()
		return m, nil
	case "p":
		if m.outcomePreviewOpen {
			m.outcomePreviewOpen = false
			m.outcomePreviewScroll = 0
			return m, nil
		}
		return m.openOutcomePreviewWithLoad()
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
	case "s", "ctrl+g":
		if m.outcomePreviewOpen {
			m.outcomePreviewOpen = false
			m.outcomePreviewScroll = 0
		}
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
	ids = append(ids, mouse.ZoneEvologSplitSuggest, mouse.ZoneEvologSplitConfirm, mouse.ZoneEvologSplitCancel, mouse.ZoneEvologSplitViewPatch, mouse.ZoneEvologSplitOutcomePreview)
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
		// Match keyboard Enter: when the outcome preview overlay is open, use commitSplitOrArmNoSplit
		// so the overlay closes and stale preview async messages are sequenced like the key path.
		if m.outcomePreviewOpen {
			upd, cmd, closeOverlay := m.commitSplitOrArmNoSplit()
			if closeOverlay {
				upd.outcomePreviewOpen = false
				upd.outcomePreviewScroll = 0
			}
			return upd, cmd
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
	if zoneID == mouse.ZoneEvologSplitOutcomePreview {
		upd, cmd := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
		return upd, cmd
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

func formatOutcomeChainStep(st aitab.EvologSplitExpectedChainStep) string {
	lbl := strings.TrimSpace(st.Label)
	d := strings.TrimSpace(st.Description)
	switch {
	case lbl != "" && d != "":
		return lbl + ": " + d
	case lbl != "":
		return lbl
	default:
		return d
	}
}

// buildOutcomePreviewAllLines returns plain-text lines for the outcome overlay (scrollable body).
func (m Model) buildOutcomePreviewAllLines(wrapW int) []string {
	if wrapW < 24 {
		wrapW = 24
	}
	textW := max(8, wrapW-6)
	var out []string
	out = append(out, "Planned linear stack (newest at top, after full automation)")
	out = append(out, "FAQ step: new work is parented on the evolog row you pick in this modal (j/k), not on the main bookmark unless that row is on main.")
	out = append(out, "")

	wc := strings.TrimSpace(m.pendingPrecomputedChildDesc)
	if wc == "" {
		wc = "working copy @ (after plan)"
	}
	out = append(out, "  @  "+runewidth.Truncate(wc, textW, "…"))

	for i := len(m.pendingExpectedChain) - 1; i >= 0; i-- {
		out = append(out, "  │")
		chunk := formatOutcomeChainStep(m.pendingExpectedChain[i])
		out = append(out, "  o  "+runewidth.Truncate(chunk, textW, "…"))
	}

	if len(m.pendingFilesFirst) > 0 {
		out = append(out, "  │")
		preview := strings.Join(m.pendingFilesFirst, ", ")
		out = append(out, "  o  file split — "+runewidth.Truncate(preview, max(8, textW-16), "…"))
	}
	if len(m.pendingHunkPeelRounds) > 0 {
		for ri := len(m.pendingHunkPeelRounds) - 1; ri >= 0; ri-- {
			round := m.pendingHunkPeelRounds[ri]
			var rp []string
			for p, k := range round {
				rp = append(rp, fmt.Sprintf("%s:%d", p, k))
			}
			sort.Strings(rp)
			line := fmt.Sprintf("hunk peel round %d — %s", ri+1, strings.Join(rp, ", "))
			out = append(out, "  │")
			out = append(out, "  o  "+runewidth.Truncate(line, textW, "…"))
		}
	}

	out = append(out, "  │")
	parent := strings.TrimSpace(m.pendingPrecomputedParentDesc)
	if parent == "" {
		parent = "parent @- (remainder / described parent)"
	}
	baseHint := ""
	if m.selectedIdx >= 0 && m.selectedIdx < len(m.entries) {
		e := m.entries[m.selectedIdx]
		sum := runewidth.Truncate(strings.TrimSpace(e.Summary), 32, "…")
		baseHint = fmt.Sprintf("  ·  base %s %s", e.CommitIDShort, sum)
	}
	out = append(out, "  ◆  "+runewidth.Truncate(parent+baseHint, textW, "…"))

	out = append(out, "")
	out = append(out, "Working copy — jj diff --summary `@-` -> `@`:")
	if m.outcomePreviewLoading {
		out = append(out, "  (loading…)")
		return out
	}
	if m.outcomePreviewErr != "" {
		out = append(out, "  error: "+runewidth.Truncate(m.outcomePreviewErr, wrapW-4, "…"))
		return out
	}
	if len(m.outcomePreviewSummary) == 0 {
		out = append(out, "  (no lines — empty diff or not loaded)")
		return out
	}
	for _, ln := range m.outcomePreviewSummary {
		ln = strings.TrimRight(ln, "\r")
		if strings.TrimSpace(ln) == "" {
			out = append(out, "")
			continue
		}
		out = append(out, "  "+runewidth.Truncate(ln, max(8, wrapW-4), "…"))
	}
	return out
}

// renderOutcomePreviewOverlay draws a scrollable box on top of the split modal.
func (m Model) renderOutcomePreviewOverlay(base string, innerContentW int) string {
	wrapW := innerContentW
	if wrapW < 32 {
		wrapW = m.outcomePreviewWrapW()
	}
	all := m.buildOutcomePreviewAllLines(wrapW)
	viewH := evologOutcomeOverlayBodyLines
	maxScroll := max(0, len(all)-viewH)
	scroll := m.outcomePreviewScroll
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}
	end := min(len(all), scroll+viewH)
	window := all[scroll:end]
	if len(window) < viewH {
		pad := make([]string, viewH-len(window))
		window = append(window, pad...)
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8BE9FD"))
	muted := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	atSty := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#50FA7B"))
	oSty := lipgloss.NewStyle().Foreground(styles.ColorSecondary)
	diaSty := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	pipe := muted.Render("  │")

	var styled []string
	for _, ln := range window {
		s := strings.TrimRight(ln, " ")
		if s == "" {
			styled = append(styled, "")
			continue
		}
		switch {
		case strings.HasPrefix(s, "  @  "):
			rest := strings.TrimPrefix(s, "  @  ")
			styled = append(styled, "  "+atSty.Render("@")+"  "+rest)
		case strings.HasPrefix(s, "  o  "):
			rest := strings.TrimPrefix(s, "  o  ")
			styled = append(styled, "  "+oSty.Render("o")+"  "+rest)
		case strings.HasPrefix(s, "  ◆  "):
			rest := strings.TrimPrefix(s, "  ◆  ")
			styled = append(styled, "  "+diaSty.Render("◆")+"  "+rest)
		case strings.TrimSpace(s) == "│":
			styled = append(styled, pipe)
		default:
			styled = append(styled, s)
		}
	}

	scrollHint := ""
	if maxScroll > 0 {
		scrollHint = fmt.Sprintf("scroll %d–%d of %d", scroll+1, min(len(all), scroll+viewH), len(all))
	}
	runHint := ""
	if m.canRunSuggestedSplit() {
		runHint = "Enter — run full suggested split (same as modal) · "
	}
	footer1 := muted.Render(runHint + "Esc / q · j/k · wheel · " + scrollHint)
	footer2 := muted.Render("Peels: --insert-before when @ has one child")
	body := strings.Join(styled, "\n")
	title := titleStyle.Render("Plan (before split)") + "\n" + muted.Render("After split: Graph (g) shows what jj did — compare there to this view.")
	inner := title + "\n" + body + "\n" + footer1 + "\n" + footer2

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorMuted).
		Background(styles.HeaderBarBackground).
		Padding(0, 1).
		Width(wrapW + 4).
		Render(inner)

	return overlay.OverlayViewInCenterInMain(base, box)
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
		rows = append(rows, muted.Render(runewidth.Truncate("Calling AI (split plan + outcome preview)…", maxLine, "…")))
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
	return overlay.OverlayViewInCenterInMain(base, box)
}

// View renders the modal.
func (m Model) View() string {
	if !m.shown {
		return ""
	}
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8BE9FD"))
	muted := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	modalW := min(m.termW-4, 120)
	if modalW < 72 {
		modalW = 72
	}
	innerW := max(48, modalW-6)

	if m.loading {
		headerRow := styles.SpreadRow(innerW, titleStyle.Render("Split"), "")
		var lines []string
		lines = append(lines, headerRow)
		lines = append(lines, "")
		lines = append(lines, muted.Render("Loading jj evolog…"))
		box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(styles.ColorMuted).Padding(1, 2).Width(modalW)
		return box.Render(strings.Join(lines, "\n"))
	}
	if m.loadErr != "" {
		headerRow := styles.SpreadRow(innerW, titleStyle.Render("Split"), "")
		var lines []string
		lines = append(lines, headerRow)
		lines = append(lines, "")
		if strings.TrimSpace(m.bookmarkName) != "" {
			lines = append(lines, muted.Render(fmt.Sprintf("Bookmark: %s  ·  change: %s", m.bookmarkName, m.tipChangeID)))
		} else {
			lines = append(lines, muted.Render(fmt.Sprintf("Change: %s", m.tipChangeID)))
		}
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#F85149")).Render("Error: "+m.loadErr))
		lines = append(lines, "")
		lines = append(lines, muted.Render("Esc or Enter to close"))
		box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(styles.ColorMuted).Padding(1, 2).Width(modalW)
		return box.Render(strings.Join(lines, "\n"))
	}

	var lines []string
	aiReady := m.suggestCfg != nil && m.suggestCfg.AIConfiguredForGeneration() && len(m.entries) >= 2 && !m.suggestLoading
	var genChip string
	switch {
	case m.suggestLoading:
		genChip = lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(styles.AIGenerateMark + " …")
	case aiReady:
		genChip = m.mark(mouse.ZoneEvologSplitSuggest, styles.AIGenerateChip())
	default:
		genChip = lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(styles.AIGenerateMark)
	}
	headerRow := styles.SpreadRow(innerW, titleStyle.Render("Split"), genChip)
	lines = append(lines, headerRow)
	lines = append(lines, "")
	if strings.TrimSpace(m.bookmarkName) != "" {
		lines = append(lines, muted.Render(fmt.Sprintf("Bookmark: %s  ·  change: %s", m.bookmarkName, m.tipChangeID)))
	} else {
		lines = append(lines, muted.Render(fmt.Sprintf("Change: %s (no bookmark — only this revision is moved)", m.tipChangeID)))
	}
	lines = append(lines, muted.Render("Pick a parent row; the new commit keeps the tip’s tree."))
	lines = append(lines, "")
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
		patchBtn = m.mark(mouse.ZoneEvologSplitViewPatch, styles.ButtonStyle.Render("Patch (o)"))
	} else {
		patchBtn = lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Patch (o)")
	}
	var previewBtn string
	if m.canOpenOutcomePreview() {
		previewBtn = m.mark(mouse.ZoneEvologSplitOutcomePreview, styles.ButtonStyle.Render("Preview (p)"))
	} else {
		previewBtn = lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Preview (p)")
	}
	var confirm string
	if noSplitFirstEnterOnlyArms(m.suggestNoSplit, m.selectedIdx, m.noSplitConfirm) {
		confirm = m.mark(mouse.ZoneEvologSplitConfirm, styles.ButtonStyle.Render("Confirm (Enter)"))
	} else {
		confirm = m.mark(mouse.ZoneEvologSplitConfirm, styles.ButtonStyle.Render("Split (Enter)"))
	}
	cancel := m.mark(mouse.ZoneEvologSplitCancel, styles.ButtonSecondaryStyle.Render("Cancel (Esc)"))
	buttonRow := lipgloss.JoinHorizontal(lipgloss.Left, patchBtn, "  ", previewBtn, "  ", confirm, "  ", cancel)
	lines = append(lines, buttonRow)
	lines = append(lines, "")
	textWrapW := max(16, modalW-6)
	if m.suggestNoSplit {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#E3B341")).Render("AI: no split")+"  "+
			muted.Render("p — current WC files · Enter again here to split anyway, or j/k for another row"))
	} else if !m.suggestNoSplit && (m.hasAISplitPlan() || strings.TrimSpace(m.suggestRationale) != "") {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#BD93F9")).Render("AI plan")+"  "+
			muted.Render("p toggles preview · c clears plan"))
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
		lines = append(lines, muted.Render("d — AI describe after split: "+dstate))
	}
	lines = append(lines, "")
	lines = append(lines, muted.Render("Scroll: j/k · PgUp/Dn · wheel"))
	lines = append(lines, muted.Render("o patch · p plan (Enter runs split while preview open) · s / ✧^g AI suggest · d toggle post-split describe · c clear AI plan"))
	lines = append(lines, muted.Render("Pick a parent below the tip (not the tip row)."))

	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(styles.ColorMuted).Padding(1, 2).Width(modalW)
	out := box.Render(strings.Join(lines, "\n"))
	if m.outcomePreviewOpen {
		out = m.renderOutcomePreviewOverlay(out, max(48, modalW-10))
	}
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
