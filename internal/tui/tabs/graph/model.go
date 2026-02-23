package graph

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/mattn/go-runewidth"
)

// GraphModel represents the state of the Graph tab
type GraphModel struct {
	zoneManager *zone.Manager
	repository  *internal.Repository

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

	// Description editing (when view is ViewEditDescription)
	descriptionInput textarea.Model
	editingCommitID  string // Change ID of commit being edited
}

// SelectionMode indicates what the user is selecting commits for
type SelectionMode int

const (
	SelectionNormal            SelectionMode = iota // Normal selection
	SelectionRebaseDestination                      // Selecting destination for rebase
)

// ChangedFile represents a file changed in a commit
type ChangedFile struct {
	Path   string
	Status string // M=modified, A=added, D=deleted
}

// GraphData contains data needed for commit graph rendering
type GraphData struct {
	Repository         *internal.Repository
	SelectedCommit     int
	InRebaseMode       bool            // True when selecting rebase destination
	RebaseSourceCommit int             // Index of commit being rebased
	OpenPRBranches     map[string]bool // Map of branch names that have open PRs
	CommitPRBranch     map[int]string  // Maps commit index to PR branch it can push to (including descendants)
	CommitBookmark     map[int]string  // Maps commit index to bookmark it can create a PR with (including descendants)
	ChangedFiles       []ChangedFile   // Changed files for the selected commit
	GraphFocused       bool            // True if graph pane has focus
	SelectedFile       int             // Index of selected file in changed files list
}

func NewGraphModel(zoneManager *zone.Manager) GraphModel {
	// Create viewports with default size and wheel enabled so mouse scroll works before first click or WindowSizeMsg.
	const defaultW, defaultH = 80, 12
	vp := viewport.New(defaultW, defaultH)
	vp.MouseWheelEnabled = true
	filesVp := viewport.New(defaultW, defaultH)
	filesVp.MouseWheelEnabled = true
	return GraphModel{
		zoneManager:     zoneManager,
		graphFocused:    true, // default to graph pane focused so j/k navigate commits and wheel scrolls graph
		viewport:        vp,
		filesViewport:   filesVp,
	}
}

func (m GraphModel) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

// Update uses a pointer receiver so scroll state is modified in place on the main model's graphTabModel.
func (m *GraphModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Lazy-init viewports so mouse wheel scrolling works (need real viewport, not zero value)
		if m.viewport.Width == 0 {
			h := max(1, m.height/2)
			m.viewport = viewport.New(max(1, m.width), h)
			m.viewport.MouseWheelEnabled = true
			m.filesViewport = viewport.New(max(1, m.width), h)
			m.filesViewport.MouseWheelEnabled = true
		}
		return m, nil

	case tea.KeyMsg:
		updated, cmd := m.handleKeyMsg(msg)
		*m = updated
		return m, cmd

	case tea.MouseMsg:
		// Mouse wheel: use IsWheel() so we accept any encoding the terminal sends (not just X11 4/5)
		if tea.MouseEvent(msg).IsWheel() {
			delta := 3
			// WheelUp/WheelLeft = scroll up, WheelDown/WheelRight = scroll down
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

	case zone.MsgZoneInBounds:
		updated, cmd := m.handleZoneClick(msg.Zone)
		*m = updated
		return m, cmd
	}

	return m, nil
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

	// Split height: 60% for graph, 40% for files
	graphHeight := (availableHeight * 60) / 100
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
	graphPane := m.zoneManager.Mark(mouse.ZoneGraphPane, paneZoneContent(visibleGraph, m.width))

	// Set up files pane - slice content manually to preserve ZoneChangedFile(i) markup
	m.filesViewport.Height = filesHeight
	filesContent := graphResult.FilesContent
	if filesContent == "" {
		filesContent = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("  Loading changed files...")
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
	filesPane := m.zoneManager.Mark(mouse.ZoneFilesPane, paneZoneContent(visibleFiles, m.width))

	// Simple separator line
	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#444444")).
		Render(strings.Repeat("─", max(m.width-2, 0)))

	v := lipgloss.JoinVertical(
		lipgloss.Left,
		graphPane,
		separator,
		actionsContent,
		separator,
		filesPane,
	)

	// Return raw view so the main model can do a single Scan() on the full screen (avoids double-Scan breaking zone positions)
	return v
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
				if openPRBranches[branch] {
					commitPRBranch[i] = branch
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
				if !openPRBranches[branch] {
					commitBookmark[i] = branch
					break
				}
			}
		}

		// Propagate bookmark info to descendants
		changed := true
		for changed {
			changed = false
			for i, commit := range m.repository.Graph.Commits {
				if commitBookmark[i] != "" || commitPRBranch[i] != "" {
					continue // Already has a bookmark or PR branch
				}
				// Check if any parent has a bookmark (without PR)
				for _, parentID := range commit.Parents {
					if parentIdx, ok := commitIDToIndex[parentID]; ok {
						if branch := commitBookmark[parentIdx]; branch != "" {
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
	var changedFiles []ChangedFile
	for _, f := range m.changedFiles {
		changedFiles = append(changedFiles, ChangedFile{
			Path:   f.Path,
			Status: f.Status,
		})
	}

	return GraphData{
		Repository:         m.repository,
		SelectedCommit:     m.selectedCommit,
		InRebaseMode:       m.selectionMode == SelectionRebaseDestination,
		RebaseSourceCommit: m.rebaseSourceCommit,
		OpenPRBranches:     openPRBranches,
		CommitPRBranch:     commitPRBranch,
		CommitBookmark:     commitBookmark,
		ChangedFiles:       changedFiles,
		GraphFocused:       m.graphFocused,
		SelectedFile:       m.selectedFile,
	}
}
