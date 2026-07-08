package graph

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	overlay "github.com/madicen/bubble-overlay"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/keys"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/mousedouble"
	"github.com/madicen/jj-tui/internal/tui/render"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/styles"
	"github.com/madicen/jj-tui/internal/tui/util"
	"github.com/mattn/go-runewidth"
)

// GraphModel represents the state of the Graph tab
type GraphModel struct {
	zoneManager *zone.Manager
	repository  *internal.Repository

	keys keys.GraphKeyMap

	width          int
	height         int
	selectedCommit int

	// Changed files for selected commit
	changedFiles         []jj.ChangedFile
	changedFilesCommitID string // Which commit the files are for
	selectedFile         int    // Index of selected file in changed files list (-1 = none)

	viewport      viewport.Model // Main viewport (graph or other content)
	filesViewport viewport.Model // Secondary viewport for changed files in graph view
	graphFocused  bool           // True if graph viewport has focus, false if files viewport

	// Scroll-to-selection: only adjust viewport when selection changed via keys/click (not on every frame, so mouse scroll isn't overridden)
	scrollToSelectedCommit bool
	scrollToSelectedFile   bool

	// Rebase mode state
	selectionMode      SelectionMode
	rebaseSourceCommit int // Index of commit being rebased
	// duplicateMode reuses the rebase destination picker to duplicate the source
	// commit onto the chosen destination instead of rebasing it.
	duplicateMode bool

	// Merge mode state: index of the commit being merged into (the destination/target).
	mergeTargetCommit int

	// Click-drag rebase: press on commit row, release on another (not used with keyboard rebase mode).
	rebasePressAnchor   int // commit index at mouse-down (-1 = none); does not affect styling until drag starts
	rebaseDragSource    int // set when pointer leaves press row (motion) so simple clicks do not look like rebase
	rebaseDragHoverDest int

	// Long-press context menu for changed files.
	longPressFileIndex int // file index at mouse-down (-1 = none)
	longPressPressID   int // monotonic counter; tick msg carries this to match against stale ticks
	longPressMouseX    int // terminal X at press
	longPressMouseY    int // terminal Y at press
	contextMenu        *ContextMenuState

	// Long-press context menu for commit rows.
	longPressCommitIndex   int // commit index at mouse-down (-1 = none)
	longPressCommitPressID int
	longPressCommitMouseX  int
	longPressCommitMouseY  int
	commitContextMenu      *CommitContextMenuState

	// Mouse: press generation for overlapping zone dedupe; double-click on rows.
	mousePressGen  uint64
	zoneOverlap    mousedouble.OverlapRelease
	rowDoubleClick mousedouble.DoubleClick

	// confirm holds a pending destructive-op confirmation (abandon / backout) when the
	// ui.confirm_destructive toggle is on. While set, the graph shows a y/n prompt and
	// swallows other keys until the user confirms (y) or cancels (n/Esc). See confirm.go.
	confirm *destructiveConfirm

	// annotate holds the scrollable blame overlay (`B` on a changed file). While
	// shown it owns navigation keys (j/k/Enter/Esc) — see annotate.go.
	annotate *annotateView

	// filterInput is the `/` revset search overlay (P4.4).
	filterInput revsetFilterInput

	// filterQuery / filterError mirror app state for header rendering in View().
	filterQuery string
	filterError string
}

// SelectionMode indicates what the user is selecting commits for
type SelectionMode int

const (
	SelectionNormal            SelectionMode = iota // Normal selection
	SelectionRebaseDestination                      // Selecting destination for rebase
	SelectionMergeSource                            // Selecting source to merge into the target commit
)

// ChangedFile represents a file changed in a commit
type ChangedFile struct {
	Path         string
	Status       string // M=modified, A=added, D=deleted
	LinesAdded   int
	LinesRemoved int
	StatsOK      bool
}

// GraphData contains data needed for commit graph rendering
type GraphData struct {
	Repository         *internal.Repository
	SelectedCommit     int
	InRebaseMode       bool            // True when selecting rebase destination
	DuplicateMode      bool            // True when the destination picker is duplicating (not rebasing)
	RebaseSourceCommit int             // Index of commit being rebased
	InMergeMode        bool            // True when selecting source to merge into the target
	MergeTargetCommit  int             // Index of commit being merged into
	OpenPRBranches     map[string]bool // Map of branch names that have open PRs
	CommitPRBranch     map[int]string  // Maps commit index to PR branch it can push to (including descendants)
	CommitBookmark     map[int]string  // Maps commit index to bookmark it can create a PR with (including descendants)
	ChangedFiles       []ChangedFile   // Changed files for the selected commit
	GraphFocused       bool            // True if graph pane has focus
	SelectedFile       int             // Index of selected file in changed files list
	FilterQuery        string          // Active revset search display text (empty = none)
	FilterError        string          // Inline jj revset error for the active filter attempt
	// RebaseDragSource / RebaseDragHoverDest: mouse drag rebase (-1 = none)
	RebaseDragSource    int
	RebaseDragHoverDest int
}

func NewGraphModel(zoneManager *zone.Manager) GraphModel {
	// Create viewports with default size and wheel enabled so mouse scroll works before first click or WindowSizeMsg.
	const defaultW, defaultH = 80, 12
	vp := viewport.New(defaultW, defaultH)
	vp.MouseWheelEnabled = true
	filesVp := viewport.New(defaultW, defaultH)
	filesVp.MouseWheelEnabled = true
	return GraphModel{
		zoneManager:          zoneManager,
		keys:                 keys.DefaultGraphKeyMap(nil),
		graphFocused:         true, // default to graph pane focused so j/k navigate commits and wheel scrolls graph
		viewport:             vp,
		filesViewport:        filesVp,
		rebasePressAnchor:    -1,
		rebaseDragSource:     -1,
		rebaseDragHoverDest:  -1,
		mergeTargetCommit:    -1,
		longPressFileIndex:   -1,
		longPressCommitIndex: -1,
		filterInput:          newRevsetFilterInput(),
	}
}

func (m GraphModel) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

// SetKeyMap replaces the graph keybindings (PLAN(P5.1): config overrides).
func (m *GraphModel) SetKeyMap(km keys.GraphKeyMap) {
	m.keys = km
}

// Update uses a pointer receiver so scroll state is modified in place on the main model's graphTabModel.
func (m *GraphModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ChangedFilesLoadedMsg:
		m.SetChangedFiles(msg.Files, msg.CommitID)
		return m, nil

	case AnnotateLoadedMsg:
		m.SetAnnotateResult(msg.Seq, msg.Lines, msg.Err)
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.viewport.Width == 0 {
			h := max(1, m.height/2)
			m.viewport = viewport.New(max(1, m.width), h)
			m.viewport.MouseWheelEnabled = true
			m.filesViewport = viewport.New(max(1, m.width), h)
			m.filesViewport.MouseWheelEnabled = true
		}
		return m, nil

	case tea.KeyMsg:
		updated, req, directCmd := m.handleKeyMsg(msg)
		*m = updated
		if req != nil {
			return m, req.Cmd()
		}
		return m, directCmd

	case tea.MouseMsg:
		if tea.MouseEvent(msg).IsWheel() {
			delta := 3
			isUp := msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelLeft
			if m.graphFocused {
				if isUp {
					m.viewport.ScrollUp(delta)
				} else {
					m.viewport.ScrollDown(delta)
				}
			} else {
				if isUp {
					m.filesViewport.ScrollUp(delta)
				} else {
					m.filesViewport.ScrollDown(delta)
				}
			}
			return m, nil
		}
		if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			mousedouble.OnLeftPress(&m.mousePressGen)
		}
		return m, nil

	case zone.MsgZoneInBounds:
		updated, req, directCmd := m.handleZoneClick(msg)
		*m = updated
		if req != nil {
			return m, req.Cmd()
		}
		return m, directCmd

	case LongPressTickMsg:
		if msg.PressID == m.longPressPressID && m.longPressFileIndex >= 0 {
			m.contextMenu = &ContextMenuState{
				FileIndex: m.longPressFileIndex,
				MouseX:    m.longPressMouseX,
				MouseY:    m.longPressMouseY,
				PressID:   msg.PressID,
				HoverItem: -1,
			}
			m.selectedFile = m.longPressFileIndex
			m.scrollToSelectedFile = true
		}
		return m, nil

	case CommitLongPressTickMsg:
		if msg.PressID == m.longPressCommitPressID && m.longPressCommitIndex >= 0 {
			m.commitContextMenu = &CommitContextMenuState{
				CommitIndex: m.longPressCommitIndex,
				MouseX:      m.longPressCommitMouseX,
				MouseY:      m.longPressCommitMouseY,
				PressID:     msg.PressID,
				HoverItem:   -1,
			}
			m.selectedCommit = m.longPressCommitIndex
			m.scrollToSelectedCommit = true
		}
		return m, nil
	}

	return m, nil
}

// UpdateWithApp processes msg with app state so requests are applied internally (status, follow-ups, cmd).
// Main calls this when in graph view so the graph mutates app and returns the cmd instead of sending Request.
func (m *GraphModel) UpdateWithApp(msg tea.Msg, app *state.AppState) (GraphModel, tea.Cmd) {
	if app == nil {
		updated, cmd := m.Update(msg)
		if g, ok := updated.(*GraphModel); ok {
			return *g, cmd
		}
		return *m, nil
	}
	switch msg := msg.(type) {
	case LongPressTickMsg:
		if msg.PressID == m.longPressPressID && m.longPressFileIndex >= 0 {
			m.contextMenu = &ContextMenuState{
				FileIndex: m.longPressFileIndex,
				MouseX:    m.longPressMouseX,
				MouseY:    m.longPressMouseY,
				PressID:   msg.PressID,
				HoverItem: -1,
			}
			m.selectedFile = m.longPressFileIndex
			m.scrollToSelectedFile = true
		}
		return *m, nil

	case CommitLongPressTickMsg:
		if msg.PressID == m.longPressCommitPressID && m.longPressCommitIndex >= 0 {
			m.commitContextMenu = &CommitContextMenuState{
				CommitIndex: m.longPressCommitIndex,
				MouseX:      m.longPressCommitMouseX,
				MouseY:      m.longPressCommitMouseY,
				PressID:     msg.PressID,
				HoverItem:   -1,
			}
			m.selectedCommit = m.longPressCommitIndex
			m.scrollToSelectedCommit = true
			m.rebasePressAnchor = -1
			m.rebaseDragSource = -1
			m.rebaseDragHoverDest = -1
		}
		return *m, nil

	case tea.MouseMsg:
		if tea.MouseEvent(msg).IsWheel() {
			updated, cmd := m.Update(msg)
			if g, ok := updated.(*GraphModel); ok {
				return *g, cmd
			}
			return *m, cmd
		}
		if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			mousedouble.OnLeftPress(&m.mousePressGen)
		}
		m.handleRebaseDragMouse(msg)
		if cmd := m.handleFileLongPress(msg); cmd != nil {
			return *m, cmd
		}
		if cmd := m.handleCommitLongPress(msg); cmd != nil {
			return *m, cmd
		}
		return *m, nil

	case tea.KeyMsg:
		// The search overlay owns the keyboard while open.
		if m.filterInput.open {
			updated, cmd := m.handleFilterInputKey(msg, app)
			*m = updated
			return *m, cmd
		}
		// A pending destructive confirmation owns the keyboard until resolved.
		if m.confirm != nil {
			if req, run := m.resolveConfirm(msg, app); run {
				ctx := BuildRequestContextFromApp(app, m)
				res := HandleRequest(req, ctx)
				return *m, ApplyResult(res, m, ctx, app)
			}
			return *m, nil
		}
		if key.Matches(msg, m.keys.SearchFilter) && m.graphFocused {
			return *m, m.beginFilterInput(app)
		}
		if key.Matches(msg, m.keys.CancelSelection) && m.contextMenu == nil && m.commitContextMenu == nil &&
			m.selectionMode == SelectionNormal && app != nil && strings.TrimSpace(app.GraphFilterRevset) != "" {
			m.ClearFilterDisplay()
			return *m, clearGraphFilter(app)
		}
		updated, req, directCmd := m.handleKeyMsg(msg)
		*m = updated
		if req != nil {
			if m.maybeConfirmDestructive(*req, app) {
				return *m, nil
			}
			ctx := BuildRequestContextFromApp(app, m)
			res := HandleRequest(*req, ctx)
			return *m, ApplyResult(res, m, ctx, app)
		}
		return *m, directCmd

	case zone.MsgZoneInBounds:
		// Ignore stray clicks while a destructive confirmation is pending (it is keyboard-only).
		if m.confirm != nil {
			return *m, nil
		}
		updated, req, directCmd := m.handleZoneClick(msg)
		*m = updated
		if req != nil {
			if m.maybeConfirmDestructive(*req, app) {
				return *m, nil
			}
			ctx := BuildRequestContextFromApp(app, m)
			res := HandleRequest(*req, ctx)
			return *m, ApplyResult(res, m, ctx, app)
		}
		return *m, directCmd
	}
	// Other message types (WindowSize, ChangedFilesLoadedMsg): no app needed, use Update
	updated, cmd := m.Update(msg)
	if g, ok := updated.(*GraphModel); ok {
		return *g, cmd
	}
	return *m, cmd
}

// paneZoneContent pads each line to the given width so the zone spans the full pane width and
// clicks on the right half of the screen register. Uses lipgloss.Width for measurement (strips
// ANSI/zone markers) and runewidth for padding so the rendered width is correct.
func paneZoneContent(content string, width int) string {
	if width <= 0 || content == "" {
		return content
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w < width {
			lines[i] = line + runewidth.FillRight("", width-w)
		}
	}
	return strings.Join(lines, "\n")
}

// graphLineIndexForCommit returns the line index in the graph content for the given commit index.
// Matches the layout in data.go: each commit uses 1 line plus len(commit.GraphLines) connector lines.
func graphLineIndexForCommit(commits []internal.Commit, commitIndex int) int {
	if commitIndex <= 0 {
		return 0
	}
	lineIdx := 0
	for j := 0; j < commitIndex && j < len(commits); j++ {
		lineIdx += 1 + len(commits[j].GraphLines)
	}
	return lineIdx
}

// View uses a pointer receiver so viewport YOffset updates (scroll-to-selection) persist on the model.
func (m *GraphModel) View() string {
	// Graph view with split panes: graph (scrollable) | actions (fixed) | files (scrollable)
	graphResult := m.getGraphResult()

	// Use a minimum actions height during loading to keep layout stable
	actionsContent := graphResult.ActionsBar
	if actionsContent == "" {
		actionsContent = "Actions:"
	}
	actionsHeight := strings.Count(actionsContent, "\n") + 1

	// Content area layout: graph pane + separator + actions + separator + files pane = m.height
	// So graphHeight + filesHeight = m.height - actionsHeight - 2 (the two separator lines)
	availableHeight := max(m.height-actionsHeight-2, 6)

	// Split height: 50% for graph, 50% for files (changed files list uses full available space)
	graphHeight := (availableHeight * 50) / 100
	filesHeight := availableHeight - graphHeight
	graphHeight = max(graphHeight, 3)
	filesHeight = max(filesHeight, 3)

	graphVisible := max(graphHeight, 2)

	// Set up graph viewport (store content and scroll state; we slice manually to preserve zone markup)
	m.viewport.Height = graphVisible
	savedGraphOffset := m.viewport.YOffset
	if graphResult.GraphContent != "" {
		m.viewport.SetContent(graphResult.GraphContent)
	}
	m.viewport.YOffset = savedGraphOffset
	maxGraphOffset := max(m.viewport.TotalLineCount()-graphVisible, 0)
	m.viewport.YOffset = max(min(m.viewport.YOffset, maxGraphOffset), 0)

	// Keep selected commit visible only when selection changed via keys/click (so mouse scroll isn't overridden)
	// GraphContent has a header at line 0 ("Graph (Tab to switch):"), so selected commit is at content line lineIdx+1
	if m.scrollToSelectedCommit {
		m.scrollToSelectedCommit = false
		if m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
			lineIdx := graphLineIndexForCommit(m.repository.Graph.Commits, m.selectedCommit)
			contentLine := lineIdx + 1
			if contentLine < m.viewport.YOffset {
				// Selection above view: scroll up one line
				m.viewport.YOffset = max(0, m.viewport.YOffset-1)
			} else if contentLine >= m.viewport.YOffset+graphVisible {
				// Selection below view: scroll so it appears on last visible line (don't scroll before it goes off)
				m.viewport.YOffset = min(contentLine-(graphVisible-1), maxGraphOffset)
			}
		}
	}

	// Slice graph content manually so ZoneCommit(i) etc. are preserved (viewport.View() would corrupt them)
	graphLines := strings.Split(graphResult.GraphContent, "\n")
	gYOff := m.viewport.YOffset
	if gYOff < 0 {
		gYOff = 0
	}
	gEnd := min(gYOff+graphVisible, len(graphLines))
	gStart := min(gYOff, gEnd)
	var visibleGraph string
	if gStart < gEnd {
		visibleGraph = strings.Join(graphLines[gStart:gEnd], "\n")
	}
	// Pad to full graphVisible height so the graph pane always uses its full 50% of vertical space
	visibleGraphLines := strings.Split(visibleGraph, "\n")
	for len(visibleGraphLines) < graphVisible {
		visibleGraphLines = append(visibleGraphLines, "")
	}
	if len(visibleGraphLines) > graphVisible {
		visibleGraphLines = visibleGraphLines[:graphVisible]
	}
	visibleGraph = strings.Join(visibleGraphLines, "\n")
	graphPane := m.zoneManager.Mark(mouse.ZoneGraphPane, paneZoneContent(visibleGraph, m.width))

	// Set up files pane - slice content manually to preserve ZoneChangedFile(i) markup
	m.filesViewport.Height = filesHeight
	filesContent := graphResult.FilesContent
	if filesContent == "" {
		// Already loaded for this commit and there are no changed files?
		loadedForSelected := m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) &&
			m.changedFilesCommitID == m.repository.Graph.Commits[m.selectedCommit].ChangeID
		if loadedForSelected && len(m.changedFiles) == 0 {
			filesContent = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("  No changed files in this commit.")
		} else {
			filesContent = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("  Loading changed files...")
		}
	}
	filesLines := strings.Split(filesContent, "\n")
	totalFilesLines := len(filesLines)
	maxFilesOffset := max(totalFilesLines-filesHeight, 0)
	savedFilesOffset := max(min(m.filesViewport.YOffset, maxFilesOffset), 0)
	m.filesViewport.SetContent(filesContent)
	m.filesViewport.YOffset = savedFilesOffset

	// Keep selected file visible only when selection changed via keys/click (so mouse scroll isn't overridden)
	if m.scrollToSelectedFile {
		m.scrollToSelectedFile = false
		if len(m.changedFiles) > 0 && m.selectedFile >= 0 && m.selectedFile < len(m.changedFiles) &&
			graphResult.FileIndexToLineIndex != nil && m.selectedFile < len(graphResult.FileIndexToLineIndex) {
			lineIdx := graphResult.FileIndexToLineIndex[m.selectedFile]
			if lineIdx >= 0 {
				if lineIdx < m.filesViewport.YOffset {
					m.filesViewport.YOffset = max(0, lineIdx)
				} else if lineIdx >= m.filesViewport.YOffset+filesHeight {
					m.filesViewport.YOffset = min(lineIdx-filesHeight+1, maxFilesOffset)
				}
			}
		}
	}

	m.filesViewport.YOffset = max(min(m.filesViewport.YOffset, maxFilesOffset), 0)
	fYOff := m.filesViewport.YOffset
	if fYOff < 0 {
		fYOff = 0
	}
	fEnd := min(fYOff+filesHeight, len(filesLines))
	fStart := min(fYOff, fEnd)
	var visibleFiles string
	if fStart < fEnd {
		visibleFiles = strings.Join(filesLines[fStart:fEnd], "\n")
	}
	// Pad to full filesHeight so the files pane always uses its full 50% of vertical space
	visibleFilesLines := strings.Split(visibleFiles, "\n")
	for len(visibleFilesLines) < filesHeight {
		visibleFilesLines = append(visibleFilesLines, "")
	}
	if len(visibleFilesLines) > filesHeight {
		visibleFilesLines = visibleFilesLines[:filesHeight]
	}
	visibleFiles = strings.Join(visibleFilesLines, "\n")
	filesPane := m.zoneManager.Mark(mouse.ZoneFilesPane, paneZoneContent(visibleFiles, m.width))

	// Simple separator line
	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#444444")).
		Render(strings.Repeat("─", render.SafeWidth(m.width, 2)))

	v := lipgloss.JoinVertical(
		lipgloss.Left,
		graphPane,
		separator,
		actionsContent,
		separator,
		filesPane,
	)

	if m.contextMenu != nil {
		isMutable := false
		if m.selectedCommit >= 0 && m.repository != nil && m.selectedCommit < len(m.repository.Graph.Commits) {
			isMutable = !m.repository.Graph.Commits[m.selectedCommit].Immutable
		}
		menuView := m.renderContextMenu(isMutable)
		v = overlay.OverlayViewAtPoint(v, menuView, m.width, m.height, m.contextMenu.MouseY, m.contextMenu.MouseX)
	}

	if m.commitContextMenu != nil {
		isMutable := m.commitMenuIsMutable()
		firstParentImm := m.commitMenuFirstParentImmutable()
		menuView := m.renderCommitContextMenu(isMutable, firstParentImm)
		v = overlay.OverlayViewAtPoint(v, menuView, m.width, m.height, m.commitContextMenu.MouseY, m.commitContextMenu.MouseX)
	}

	if m.confirm != nil {
		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.ColorPrimary).
			Padding(0, 1).
			Render(m.confirm.prompt)
		v = overlay.OverlayViewInCenterWithOffset(v, box, m.width, m.height, 0, 0)
	}

	if m.AnnotateShown() {
		box := m.renderAnnotateOverlay()
		v = overlay.OverlayViewInCenterWithOffset(v, box, m.width, m.height, 0, 0)
	}

	v = m.overlayFilterInput(v)

	return v
}

// SetFilterDisplay updates the active-filter header state (mirrors app.GraphFilter*).
func (m *GraphModel) SetFilterDisplay(query, errMsg string) {
	m.filterQuery = query
	m.filterError = errMsg
}

// ClearFilterDisplay clears the filter header.
func (m *GraphModel) ClearFilterDisplay() {
	m.filterQuery = ""
	m.filterError = ""
}

// getGraphResult returns the GraphResult for the commit graph view
func (m *GraphModel) getGraphResult() GraphResult {
	return m.Graph(m.buildGraphData())
}

// buildGraphData builds the GraphData for the commit graph
func (m *GraphModel) buildGraphData() GraphData {
	// Build a map of branches that have open PRs
	openPRBranches := make(map[string]bool)
	if m.repository != nil {
		for _, pr := range m.repository.PRs {
			if pr.State == "open" {
				openPRBranches[pr.HeadBranch] = true
			}
		}
	}

	// Build a map of commit index -> PR branch name for commits that can push to a PR
	// This includes commits with the bookmark AND their descendants
	commitPRBranch := make(map[int]string)
	if m.repository != nil && len(m.repository.Graph.Commits) > 0 {
		// First, find commits that directly have PR bookmarks
		commitIDToIndex := make(map[string]int)
		for i, commit := range m.repository.Graph.Commits {
			commitIDToIndex[commit.ID] = i
			commitIDToIndex[commit.ChangeID] = i
			// Check if this commit has a PR bookmark
			for _, branch := range commit.Branches {
				local := util.LocalBookmarkName(branch)
				if openPRBranches[branch] || openPRBranches[local] {
					commitPRBranch[i] = local
					break
				}
			}
		}

		// Now propagate PR branch info to descendants (commits whose parents have PR branches)
		// We iterate multiple times to handle chains of descendants
		changed := true
		for changed {
			changed = false
			for i, commit := range m.repository.Graph.Commits {
				if commitPRBranch[i] != "" {
					continue // Already has a PR branch
				}
				// Check if any parent has a PR branch
				for _, parentID := range commit.Parents {
					if parentIdx, ok := commitIDToIndex[parentID]; ok {
						if branch := commitPRBranch[parentIdx]; branch != "" {
							commitPRBranch[i] = branch
							changed = true
							break
						}
					}
				}
			}
		}
	}

	// Build a map of commit index -> bookmark name for commits that can create a PR
	// This includes commits with bookmarks (that don't have open PRs) AND their descendants
	commitBookmark := make(map[int]string)
	if m.repository != nil && len(m.repository.Graph.Commits) > 0 {
		commitIDToIndex := make(map[string]int)
		for i, commit := range m.repository.Graph.Commits {
			commitIDToIndex[commit.ID] = i
			commitIDToIndex[commit.ChangeID] = i
			// Check if this commit has a bookmark without an open PR
			for _, branch := range commit.Branches {
				local := util.LocalBookmarkName(branch)
				if local == "" || isDefaultBranch(local) {
					continue
				}
				if !openPRBranches[branch] && !openPRBranches[local] {
					commitBookmark[i] = local
					break
				}
			}
		}

		// Propagate bookmark info to descendants
		changed := true
		for changed {
			changed = false
			for i, commit := range m.repository.Graph.Commits {
				if commitBookmark[i] != "" {
					continue
				}
				for _, parentID := range commit.Parents {
					if parentIdx, ok := commitIDToIndex[parentID]; ok {
						branch := commitBookmark[parentIdx]
						if branch != "" && !isDefaultBranch(branch) {
							commitBookmark[i] = branch
							changed = true
							break
						}
					}
				}
			}
		}
	}

	// Convert changed files to view format
	changedFiles := make([]ChangedFile, 0, len(m.changedFiles))
	for _, f := range m.changedFiles {
		changedFiles = append(changedFiles, ChangedFile{
			Path:         f.Path,
			Status:       f.Status,
			LinesAdded:   f.LinesAdded,
			LinesRemoved: f.LinesRemoved,
			StatsOK:      f.StatsOK,
		})
	}

	return GraphData{
		Repository:          m.repository,
		SelectedCommit:      m.selectedCommit,
		InRebaseMode:        m.selectionMode == SelectionRebaseDestination,
		DuplicateMode:       m.duplicateMode && m.selectionMode == SelectionRebaseDestination,
		RebaseSourceCommit:  m.rebaseSourceCommit,
		InMergeMode:         m.selectionMode == SelectionMergeSource,
		MergeTargetCommit:   m.mergeTargetCommit,
		OpenPRBranches:      openPRBranches,
		CommitPRBranch:      commitPRBranch,
		CommitBookmark:      commitBookmark,
		ChangedFiles:        changedFiles,
		GraphFocused:        m.graphFocused,
		SelectedFile:        m.selectedFile,
		FilterQuery:         m.filterQuery,
		FilterError:         m.filterError,
		RebaseDragSource:    m.rebaseDragSource,
		RebaseDragHoverDest: m.rebaseDragHoverDest,
	}
}

// GetCreatePRBranch returns the branch name that would be used for Create PR for the
// currently selected commit, or "" if not applicable. Used to block Create PR on main/master.
func (m *GraphModel) GetCreatePRBranch() string {
	data := m.buildGraphData()
	return data.CommitBookmark[m.selectedCommit]
}

// OnRepositoryLoaded recomputes graph-derived state (selection, rebase drag) from
// the newly-loaded repository. It implements tab.RepositoryAware (P2.8): the root
// no longer fans a cached copy out to every tab; it invokes this hook only on tabs
// that need to recompute on load.
func (m *GraphModel) OnRepositoryLoaded(repo *internal.Repository) {
	if repo == nil {
		return
	}
	oldCommitID := m.changedFilesCommitID
	m.repository = repo
	commits := repo.Graph.Commits
	if oldCommitID != "" && len(commits) > 0 {
		found := false
		for i, c := range commits {
			if c.ChangeID == oldCommitID {
				m.selectedCommit = i
				found = true
				break
			}
		}
		if !found {
			m.selectedCommit = 0
			m.changedFilesCommitID = ""
			m.changedFiles = nil
		}
	}
	if m.selectedCommit >= len(commits) {
		m.selectedCommit = max(0, len(commits)-1)
	}
	if m.rebaseDragSource >= len(commits) || m.rebasePressAnchor >= len(commits) {
		m.rebasePressAnchor = -1
		m.rebaseDragSource = -1
		m.rebaseDragHoverDest = -1
	}
}

// SetDimensions sets the width and height and lazy-inits viewports if needed.
func (m *GraphModel) SetDimensions(width, height int) {
	m.width = width
	m.height = height
	if m.viewport.Width == 0 && width > 0 && height > 0 {
		h := max(1, height/2)
		m.viewport = viewport.New(max(1, width), h)
		m.viewport.MouseWheelEnabled = true
		m.filesViewport = viewport.New(max(1, width), h)
		m.filesViewport.MouseWheelEnabled = true
	}
}

// SetChangedFiles updates the changed files for the selected commit.
// Files are sorted in tree-display order (depth-first: dirs then files at each level, each sorted)
// so that selection index order matches the tree display order, preventing scroll/selection from jumping.
func (m *GraphModel) SetChangedFiles(files []jj.ChangedFile, commitID string) {
	accept := (commitID == m.changedFilesCommitID) ||
		(m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) &&
			commitID == m.repository.Graph.Commits[m.selectedCommit].ChangeID)
	if !accept {
		return
	}
	if m.changedFilesCommitID != commitID {
		m.changedFilesCommitID = commitID
	}
	// Sort in same order as tree: at each path level, dirs before files, then alphabetically within each
	sorted := make([]jj.ChangedFile, len(files))
	copy(sorted, files)
	sort.Slice(sorted, func(i, j int) bool { return changedFileTreeOrderLess(sorted[i].Path, sorted[j].Path) })
	m.changedFiles = sorted
	m.selectedFile = 0
	m.scrollToSelectedFile = true
}

// changedFileTreeOrderLess compares two file paths in the order the file tree displays them:
// depth-first, with directories before files at each level, and alphabetical within dirs and within files.
func changedFileTreeOrderLess(a, b string) bool {
	pa := strings.Split(a, "/")
	pb := strings.Split(b, "/")
	for i := 0; i < len(pa) && i < len(pb); i++ {
		if i == len(pa)-1 && i == len(pb)-1 {
			return pa[i] < pb[i]
		}
		if i == len(pa)-1 {
			return false // a is file at this level, b has more (dir) → b's subtree first
		}
		if i == len(pb)-1 {
			return true // b is file at this level, a has more (dir) → a's subtree first
		}
		if pa[i] != pb[i] {
			return pa[i] < pb[i]
		}
	}
	return len(pa) < len(pb)
}

// SelectCommit selects a commit by index.
func (m *GraphModel) SelectCommit(idx int) {
	if m.repository != nil && idx >= 0 && idx < len(m.repository.Graph.Commits) {
		m.selectedCommit = idx
		// Clear changed-files state until LoadChangedFiles completes; otherwise we'd show "No changed files" before load
		m.changedFilesCommitID = ""
		m.changedFiles = nil
		m.selectedFile = 0
	}
}

// SetSelectionMode sets the selection mode.
func (m *GraphModel) SetSelectionMode(mode SelectionMode) {
	m.selectionMode = mode
}

// SetRebaseSourceCommit sets the commit index being rebased.
func (m *GraphModel) SetRebaseSourceCommit(idx int) {
	m.rebaseSourceCommit = idx
}

// GetSelectionMode returns the current selection mode.
func (m *GraphModel) GetSelectionMode() SelectionMode {
	return m.selectionMode
}

// GetRebaseSourceCommit returns the commit index being rebased.
func (m *GraphModel) GetRebaseSourceCommit() int {
	return m.rebaseSourceCommit
}

// GetSelectedCommit returns the index of the selected commit.
func (m *GraphModel) GetSelectedCommit() int {
	return m.selectedCommit
}

// GetSelectedFile returns the index of the selected file.
func (m *GraphModel) GetSelectedFile() int {
	return m.selectedFile
}

// SetSelectedFile sets the selected file index.
func (m *GraphModel) SetSelectedFile(idx int) {
	if idx >= -1 && idx < len(m.changedFiles) {
		m.selectedFile = idx
		m.scrollToSelectedFile = true
	}
}

// IsGraphFocused returns whether the graph pane has focus.
func (m *GraphModel) IsGraphFocused() bool {
	return m.graphFocused
}

// SetGraphFocused sets whether the graph pane has focus.
func (m *GraphModel) SetGraphFocused(focused bool) {
	m.graphFocused = focused
}

// GetChangedFiles returns the changed files for the selected commit.
func (m *GraphModel) GetChangedFiles() []jj.ChangedFile {
	return m.changedFiles
}

// GetChangedFilesCommitID returns the ChangeID for which changed files are loaded.
func (m *GraphModel) GetChangedFilesCommitID() string {
	return m.changedFilesCommitID
}

// SetViewport sets the graph viewport.
func (m *GraphModel) SetViewport(vp viewport.Model) {
	m.viewport = vp
}

// SetFilesViewport sets the files viewport.
func (m *GraphModel) SetFilesViewport(vp viewport.Model) {
	m.filesViewport = vp
}

// GetViewport returns the graph viewport.
func (m *GraphModel) GetViewport() viewport.Model {
	return m.viewport
}

// GetFilesViewport returns the files viewport.
func (m *GraphModel) GetFilesViewport() viewport.Model {
	return m.filesViewport
}

// StartRebaseMode starts rebase mode.
func (m *GraphModel) StartRebaseMode(sourceCommitIdx int) {
	m.selectionMode = SelectionRebaseDestination
	m.rebaseSourceCommit = sourceCommitIdx
	m.rebasePressAnchor = -1
	m.rebaseDragSource = -1
	m.rebaseDragHoverDest = -1
}

// CancelRebaseMode cancels rebase mode (and the duplicate variant that reuses it).
func (m *GraphModel) CancelRebaseMode() {
	m.selectionMode = SelectionNormal
	m.rebaseSourceCommit = -1
	m.duplicateMode = false
	m.rebasePressAnchor = -1
	m.rebaseDragSource = -1
	m.rebaseDragHoverDest = -1
}

// StartDuplicateMode starts the destination picker in "duplicate" mode: it reuses
// the rebase destination-selection UI/keys/mouse, but confirming duplicates the
// source commit onto the chosen destination instead of rebasing it.
func (m *GraphModel) StartDuplicateMode(sourceCommitIdx int) {
	m.StartRebaseMode(sourceCommitIdx)
	m.duplicateMode = true
}

// GetDuplicateMode reports whether the destination picker is in duplicate mode.
func (m *GraphModel) GetDuplicateMode() bool {
	return m.duplicateMode
}

// IsInRebaseMode returns whether the graph is in rebase mode.
func (m *GraphModel) IsInRebaseMode() bool {
	return m.selectionMode == SelectionRebaseDestination
}

// StartMergeMode starts merge mode with the given target commit (the commit being merged into).
func (m *GraphModel) StartMergeMode(targetCommitIdx int) {
	m.selectionMode = SelectionMergeSource
	m.mergeTargetCommit = targetCommitIdx
}

// CancelMergeMode cancels merge mode.
func (m *GraphModel) CancelMergeMode() {
	m.selectionMode = SelectionNormal
	m.mergeTargetCommit = -1
}

// IsInMergeMode returns whether the graph is in merge mode.
func (m *GraphModel) IsInMergeMode() bool {
	return m.selectionMode == SelectionMergeSource
}

// GetMergeTargetCommit returns the commit index being merged into.
func (m *GraphModel) GetMergeTargetCommit() int {
	return m.mergeTargetCommit
}
