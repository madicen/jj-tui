package graph

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/mouse"
)

type graphModel struct {
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

	// Rebase mode state
	selectionMode      SelectionMode
	rebaseSourceCommit int // Index of commit being rebased
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

func NewGraphModel(zoneManager *zone.Manager) graphModel {
	return graphModel{
		zoneManager: zoneManager,
	}
}

func (m graphModel) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (m graphModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.MouseMsg:
		// Handle mouse wheel scrolling
		if msg.Action == tea.MouseActionPress && (msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown) {
			// For graph view, scroll the focused pane
			if m.graphFocused {
				// Scroll graph pane
				var cmd tea.Cmd
				m.viewport, cmd = m.viewport.Update(msg)
				return m, cmd
			} else {
				// Scroll files pane
				var cmd tea.Cmd
				m.filesViewport, cmd = m.filesViewport.Update(msg)
				return m, cmd
			}
		}

	case zone.MsgZoneInBounds:
		return m.handleZoneClick(msg.Zone)
	}

	return m, nil
}

func (m graphModel) View() string {
	// Graph view with split panes: graph (scrollable) | actions (fixed) | files (scrollable)
	graphResult := m.getGraphResult()

	headerHeight := 1
	statusHeight := 1
	separatorLines := 2 // Two separator lines between sections
	paddingLines := 1   // Padding after header

	// Use a minimum actions height during loading to keep layout stable
	actionsContent := graphResult.ActionsBar
	if actionsContent == "" {
		actionsContent = "Actions:"
	}
	actionsHeight := strings.Count(actionsContent, "\n") + 1

	// Calculate available height for the two scrollable panes
	availableHeight := max(m.height-headerHeight-statusHeight-actionsHeight-separatorLines-paddingLines, 6)

	// Split height: 60% for graph, 40% for files
	graphHeight := (availableHeight * 60) / 100
	filesHeight := availableHeight - graphHeight
	graphHeight = max(graphHeight, 3)
	filesHeight = max(filesHeight, 3)

	// Set up graph viewport
	m.viewport.Height = graphHeight

	// Save scroll position before SetContent (which resets YOffset)
	savedGraphOffset := m.viewport.YOffset

	// Always set content if we have valid graph content (even during loading, to avoid stale content from other views)
	if graphResult.GraphContent != "" {
		m.viewport.SetContent(graphResult.GraphContent)
	}

	// Restore scroll position and clamp to valid range
	m.viewport.YOffset = savedGraphOffset
	maxGraphOffset := max(m.viewport.TotalLineCount()-graphHeight, 0)
	m.viewport.YOffset = max(min(m.viewport.YOffset, maxGraphOffset), 0)

	// Set up files viewport - show placeholder if no files yet
	m.filesViewport.Height = filesHeight
	filesContent := graphResult.FilesContent
	if filesContent == "" {
		filesContent = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("  Loading changed files...")
	}

	// Save scroll position before SetContent
	savedFilesOffset := m.filesViewport.YOffset
	m.filesViewport.SetContent(filesContent)

	// Restore scroll position and clamp to valid range
	m.filesViewport.YOffset = savedFilesOffset
	maxFilesOffset := max(m.filesViewport.TotalLineCount()-filesHeight, 0)
	m.filesViewport.YOffset = max(min(m.filesViewport.YOffset, maxFilesOffset), 0)

	// Simple separator line
	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#444444")).
		Render(strings.Repeat("─", m.width-2))

	// Wrap viewports in zones for click-to-focus
	graphPane := m.zoneManager.Mark(mouse.ZoneGraphPane, m.viewport.View())
	filesPane := m.zoneManager.Mark(mouse.ZoneFilesPane, m.filesViewport.View())

	v := lipgloss.JoinVertical(
		lipgloss.Left,
		graphPane,
		separator,
		actionsContent,
		separator,
		filesPane,
	)

	return m.zoneManager.Scan(v)
}

// getGraphResult returns the GraphResult for the commit graph view
func (m *graphModel) getGraphResult() GraphResult {
	return m.Graph(m.buildGraphData())
}

// buildGraphData builds the GraphData for the commit graph
func (m *graphModel) buildGraphData() GraphData {
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
