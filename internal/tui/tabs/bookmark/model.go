package bookmark

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/genmenu"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/styles"
	"github.com/mattn/go-runewidth"
)

// Model represents the bookmark creation dialog
type Model struct {
	shown                     bool
	nameInput                 textinput.Model
	commitIdx                 int               // Index of commit to create bookmark on
	existingBookmarks         []string          // List of existing bookmarks
	selectedBookmarkIdx       int               // Index of selected existing bookmark (-1 for new)
	fromJira                  bool              // True if creating bookmark from Jira ticket
	jiraTicketKey             string            // Jira ticket key if creating from Jira
	jiraTicketTitle           string            // Jira ticket summary if creating from Jira
	ticketDisplayKey          string            // Short display key (e.g., "$12u" for Codecks)
	bookmarkNameExists        bool              // True if entered name matches an existing bookmark
	jiraBookmarkTitles        map[string]string // Maps bookmark names to formatted PR titles ("KEY - Title")
	ticketBookmarkDisplayKeys map[string]string // Maps bookmark names to ticket short IDs for commit messages
	repository                *internal.Repository
	nameConflictSources       []string // Branch names + commit branch names (set by main); used for "name exists" check
	zoneManager               *zone.Manager
	// contentWidth is the available width inside the wrapping FrameFormModal (set by main on
	// tea.WindowSizeMsg). The inner "Target:" / "Jira Ticket:" rounded boxes pin their Width to
	// this so a long single-line commit description (which is what jj returns for Summary)
	// can be word-wrapped instead of producing a giant border that the outer frame's Width
	// then chops into stacked horizontal segments.
	contentWidth int
	// Long-press AI profile picker over the Generate chip. Same pattern as the
	// other three generate-bearing modals (descedit, prform, ticketform).
	genMenu       genmenu.State
	profiles      []config.AIProfile
	activeProfile string
}

// NewModel creates a new Bookmark model. zoneManager may be nil.
func NewModel(zoneManager *zone.Manager) Model {
	nameInput := textinput.New()
	nameInput.Placeholder = "bookmark-name"
	// CharLimit is the first-line defense against pathologically long names; the actual
	// operational cap is jj.MaxBookmarkNameLen (50) enforced in bookmark.SubmitCmd. We
	// leave a bit of headroom (80) so users can paste a slightly-too-long name, sanitize
	// can compress it, and the backstop trims the rest — instead of textinput refusing
	// the paste outright.
	nameInput.CharLimit = 80
	nameInput.Width = 50
	nameInput.Focus()

	return Model{
		shown:               false,
		nameInput:           nameInput,
		commitIdx:           -1,
		selectedBookmarkIdx: -1,
		fromJira:            false,
		zoneManager:         zoneManager,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the Bookmark creation view
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.shown {
		return m, nil
	}
	// Handle request messages (main forwards these to us)
	switch msg.(type) {
	case CancelRequestedMsg:
		m.shown = false
		m.nameInput.SetValue("")
		m.genMenu.Reset()
		return m, state.NavigateTarget{Kind: state.NavigateBackToGraph, StatusMessage: "Bookmark creation cancelled"}.Cmd()
	case SubmitRequestedMsg:
		return m, state.NavigateTarget{Kind: state.NavigateSubmitBookmark}.Cmd()
	}
	switch msg := msg.(type) {
	case genmenu.TickMsg:
		m.genMenu.OpenIfMatches(msg)
		return m, nil
	case tea.MouseMsg:
		return m.handleMouseForMenu(msg)
	case zone.MsgZoneInBounds:
		if m.genMenu.IsShown() {
			return m, nil
		}
		m.genMenu.CancelPress()
		if m.zoneManager != nil {
			if zoneID := m.resolveClickedZone(msg); zoneID != "" {
				return m.handleZoneClick(zoneID)
			}
		}
		return m, nil
	case tea.KeyMsg:
		if m.genMenu.IsShown() && msg.String() == "esc" {
			m.genMenu.Close()
			return m, nil
		}
		return m.handleKeyMsg(msg)
	}

	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

// handleMouseForMenu drives the long-press AI profile picker for the Generate chip.
// Mirrors descedit.handleMouseForMenu so the gesture is consistent across modals.
// Long-press is only armed when the bookmark form is in "new name" mode (selectedBookmarkIdx
// is -1) since the Generate chip is otherwise inactive on this form.
func (m Model) handleMouseForMenu(msg tea.MouseMsg) (Model, tea.Cmd) {
	if m.zoneManager == nil {
		return m, nil
	}
	if m.genMenu.IsShown() {
		switch msg.Action {
		case tea.MouseActionMotion, tea.MouseActionPress:
			m.genMenu.UpdateHover(m.zoneManager, msg, len(m.profiles))
			return m, nil
		case tea.MouseActionRelease:
			if msg.Button != tea.MouseButtonLeft {
				return m, nil
			}
			idx := m.genMenu.HitTestRelease(m.zoneManager, msg, len(m.profiles))
			if idx >= 0 && idx < len(m.profiles) {
				return m, state.NavigateTarget{
					Kind:              state.NavigateGenerateBookmarkName,
					AIOverrideProfile: m.profiles[idx].Name,
				}.Cmd()
			}
			return m, nil
		}
		return m, nil
	}
	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button != tea.MouseButtonLeft {
			return m, nil
		}
		if m.selectedBookmarkIdx != -1 {
			return m, nil
		}
		z := m.zoneManager.Get(mouse.ZoneBookmarkGenerate)
		if z != nil && z.InBounds(msg) && len(m.profiles) > 0 {
			return m, m.genMenu.BeginPress(mouse.ZoneBookmarkGenerate, msg)
		}
	case tea.MouseActionMotion:
		m.genMenu.OnMotion(m.zoneManager, msg)
	}
	return m, nil
}

// View renders the Bookmark creation dialog
func (m Model) View() string {
	if !m.shown {
		return ""
	}
	return m.renderBookmark()
}

// handleKeyMsg handles keyboard input; returns request cmds for main to handle cancel/submit.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, CancelRequestedCmd()
	case "ctrl+g":
		if m.selectedBookmarkIdx == -1 {
			return m, state.NavigateTarget{Kind: state.NavigateGenerateBookmarkName}.Cmd()
		}
		return m, nil
	case "enter", "ctrl+s":
		return m, SubmitRequestedCmd()
	case "tab":
		existing := m.existingBookmarks
		sel := m.selectedBookmarkIdx
		if sel == -1 && len(existing) > 0 {
			m.selectedBookmarkIdx = 0
			m.nameInput.Blur()
		} else {
			m.selectedBookmarkIdx = -1
			m.nameInput.Focus()
		}
		return m, nil
	}

	if m.selectedBookmarkIdx == -1 {
		var cmd tea.Cmd
		m.nameInput, cmd = m.nameInput.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "j", "down":
		if len(m.existingBookmarks) > 0 {
			if m.selectedBookmarkIdx < len(m.existingBookmarks)-1 {
				m.selectedBookmarkIdx++
			}
		}
		return m, nil
	case "k", "up":
		if m.selectedBookmarkIdx > -1 {
			m.selectedBookmarkIdx--
			if m.selectedBookmarkIdx < 0 {
				m.nameInput.Focus()
			}
		}
		return m, nil
	}

	return m, nil
}

// ZoneIDs returns the zone IDs this modal uses when rendering (same IDs passed to Mark). Used to resolve clicks.
func (m Model) ZoneIDs() []string {
	ids := []string{mouse.ZoneBookmarkName, mouse.ZoneBookmarkSubmit, mouse.ZoneBookmarkGenerate, mouse.ZoneBookmarkCancel}
	for i := range m.existingBookmarks {
		ids = append(ids, mouse.ZoneExistingBookmark(i))
	}
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

// handleZoneClick handles a zone click by zone ID (called from Update after resolve). Returns (updated model, cmd).
func (m Model) handleZoneClick(zoneID string) (Model, tea.Cmd) {
	if zoneID == mouse.ZoneBookmarkSubmit {
		return m, SubmitRequestedCmd()
	}
	if zoneID == mouse.ZoneBookmarkCancel {
		return m, CancelRequestedCmd()
	}
	if zoneID == mouse.ZoneBookmarkGenerate && m.selectedBookmarkIdx == -1 {
		return m, state.NavigateTarget{Kind: state.NavigateGenerateBookmarkName}.Cmd()
	}
	if zoneID == mouse.ZoneBookmarkName {
		m.selectedBookmarkIdx = -1
		m.nameInput.Focus()
		return m, nil
	}
	const prefix = "zone:bookmark:existing:"
	if strings.HasPrefix(zoneID, prefix) {
		s := strings.TrimPrefix(zoneID, prefix)
		i, err := strconv.Atoi(s)
		if err == nil && i >= 0 && i < len(m.existingBookmarks) {
			m.selectedBookmarkIdx = i
			m.nameInput.Blur()
			return m, nil
		}
	}
	return m, nil
}

// Accessors

// IsShown returns whether the dialog is displayed
func (m *Model) IsShown() bool {
	return m.shown
}

// Show displays the bookmark creation dialog
func (m *Model) Show(commitIdx int, existingBookmarks []string) {
	m.shown = true
	m.commitIdx = commitIdx
	m.existingBookmarks = existingBookmarks
	m.selectedBookmarkIdx = -1
	m.nameInput.SetValue("")
	m.nameInput.Focus()
	m.bookmarkNameExists = false
}

// Hide hides the dialog
func (m *Model) Hide() {
	m.shown = false
	m.nameInput.SetValue("")
}

// GetBookmarkName returns the entered bookmark name
func (m *Model) GetBookmarkName() string {
	return m.nameInput.Value()
}

// SetBookmarkName sets the bookmark name
func (m *Model) SetBookmarkName(name string) {
	m.nameInput.SetValue(name)
}

// GetCommitIdx returns the commit index
func (m *Model) GetCommitIdx() int {
	return m.commitIdx
}

// GetExistingBookmarks returns the list of existing bookmarks (for move)
func (m *Model) GetExistingBookmarks() []string {
	return m.existingBookmarks
}

// GetSelectedBookmarkIdx returns the selected existing bookmark index (-1 for new)
func (m *Model) GetSelectedBookmarkIdx() int {
	return m.selectedBookmarkIdx
}

// SetFromJira sets the jira context
func (m *Model) SetFromJira(ticketKey, ticketTitle, displayKey string) {
	m.fromJira = true
	m.jiraTicketKey = ticketKey
	m.jiraTicketTitle = ticketTitle
	m.ticketDisplayKey = displayKey
}

// ClearJiraContext clears the jira context
func (m *Model) ClearJiraContext() {
	m.fromJira = false
	m.jiraTicketKey = ""
	m.jiraTicketTitle = ""
	m.ticketDisplayKey = ""
}

// IsFromJira returns whether creating from Jira ticket
func (m *Model) IsFromJira() bool {
	return m.fromJira
}

// GetJiraKey returns the Jira ticket key
func (m *Model) GetJiraKey() string {
	return m.jiraTicketKey
}

// GetJiraTicketTitle returns the Jira ticket summary
func (m *Model) GetJiraTicketTitle() string {
	return m.jiraTicketTitle
}

// GetTicketDisplayKey returns the short display key for commit messages
func (m *Model) GetTicketDisplayKey() string {
	return m.ticketDisplayKey
}

// SetNameExists sets whether the name already exists
func (m *Model) SetNameExists(exists bool) {
	m.bookmarkNameExists = exists
}

// SetExistingBookmarks sets the list of existing bookmarks (synced from main model)
func (m *Model) SetExistingBookmarks(bookmarks []string) {
	m.existingBookmarks = bookmarks
}

// SetCommitIdx sets the commit index for the bookmark target
func (m *Model) SetCommitIdx(idx int) {
	m.commitIdx = idx
}

// SetSelectedBookmarkIdx sets the selected existing bookmark index
func (m *Model) SetSelectedBookmarkIdx(idx int) {
	m.selectedBookmarkIdx = idx
}

// NameExists returns whether the name already exists
func (m *Model) NameExists() bool {
	return m.bookmarkNameExists
}

// GetNameInput returns the name input field
func (m *Model) GetNameInput() *textinput.Model {
	return &m.nameInput
}

// UpdateRepository updates the repository (for rendering commit target)
func (m *Model) UpdateRepository(repo *internal.Repository) {
	m.repository = repo
}

// SetNameConflictSources sets the list of names that would conflict (branch names + commit branch names). Main calls this when showing the modal and when branches/repo change.
func (m *Model) SetNameConflictSources(names []string) {
	m.nameConflictSources = names
}

// UpdateNameExistsFromInput checks the current name input against conflict sources and existing bookmarks, optionally sanitizing; sets the NameExists flag.
func (m *Model) UpdateNameExistsFromInput(sanitize bool) {
	name := strings.TrimSpace(m.nameInput.Value())
	if name == "" {
		m.bookmarkNameExists = false
		return
	}
	if sanitize {
		name = jj.SanitizeBookmarkName(name)
	}
	m.bookmarkNameExists = nameExists(name, m.nameConflictSources, m.existingBookmarks)
}

// nameExists returns true if name is in branchNamesOrCommitBranches or existingBookmarks.
func nameExists(name string, branchNamesOrCommitBranches, existingBookmarks []string) bool {
	if name == "" {
		return false
	}
	if slices.Contains(branchNamesOrCommitBranches, name) {
		return true
	}
	return slices.Contains(existingBookmarks, name)
}

// JiraBookmarkTitles / TicketBookmarkDisplayKeys (for PR title formatting from bookmarks)
func (m *Model) GetJiraBookmarkTitles() map[string]string { return m.jiraBookmarkTitles }
func (m *Model) SetJiraBookmarkTitles(mp map[string]string) {
	if mp != nil {
		m.jiraBookmarkTitles = mp
	} else {
		m.jiraBookmarkTitles = make(map[string]string)
	}
}
func (m *Model) GetTicketBookmarkDisplayKeys() map[string]string { return m.ticketBookmarkDisplayKeys }
func (m *Model) SetTicketBookmarkDisplayKeys(mp map[string]string) {
	if mp != nil {
		m.ticketBookmarkDisplayKeys = mp
	} else {
		m.ticketBookmarkDisplayKeys = make(map[string]string)
	}
}

// SetZoneManager sets the zone manager for clickable elements
func (m *Model) SetZoneManager(z *zone.Manager) {
	m.zoneManager = z
}

// SetContentWidth records the modal's inner content width (cols available inside the
// wrapping FrameFormModal). renderBookmark uses it to pin the inner rounded boxes so
// long commit descriptions wrap into the box instead of stretching its border off-frame.
func (m *Model) SetContentWidth(w int) {
	m.contentWidth = w
}

// SetAIProfiles updates the profile list shown by the long-press menu and the
// active profile mark.
func (m *Model) SetAIProfiles(profiles []config.AIProfile, activeProfile string) {
	m.profiles = profiles
	m.activeProfile = activeProfile
}

// MenuState returns a pointer to the long-press menu state.
func (m *Model) MenuState() *genmenu.State {
	return &m.genMenu
}

// MenuOverlay returns the rendered popover (empty when hidden) and its (x, y) anchor.
func (m *Model) MenuOverlay() (string, int, int) {
	if !m.genMenu.IsShown() {
		return "", 0, 0
	}
	view := genmenu.Render(m.zoneManager, m.profiles, m.activeProfile, m.genMenu.HoverIndex())
	x, y := m.genMenu.MouseAnchor()
	return view, x, y
}

func mark(z *zone.Manager, id, content string) string {
	if z == nil {
		return content
	}
	return z.Mark(id, content)
}

// boxWidth returns the Width to set on the inner rounded boxes (Target / Jira Ticket).
// Falls back to a sensible default when contentWidth hasn't been propagated yet (e.g. the
// very first render before tea.WindowSizeMsg arrives), so we never call Width(0) which
// would collapse the boxes to a single column.
func (m Model) boxWidth() int {
	if m.contentWidth >= 24 {
		return m.contentWidth
	}
	return 50
}

func (m Model) renderBookmark() string {
	var lines []string
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorPrimary).
		Padding(0, 1)
	// Pin the inner box to the modal's content width so a multi-paragraph or
	// long-single-line commit description (jj's Summary is the full description) wraps
	// inside the border instead of stretching the border well past the outer frame's
	// Width — which would then chop the border into the stacked horizontal segments.
	if w := m.boxWidth(); w > 0 {
		boxStyle = boxStyle.Width(w)
	}
	if m.fromJira {
		jiraBox := boxStyle.Render(fmt.Sprintf("Jira Ticket: %s\n\nThis will create a new branch from main with the bookmark name below.",
			lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render(m.jiraTicketKey),
		))
		lines = append(lines, jiraBox)
		lines = append(lines, "")
	} else {
		if m.repository != nil && m.commitIdx >= 0 && m.commitIdx < len(m.repository.Graph.Commits) {
			commit := m.repository.Graph.Commits[m.commitIdx]
			commitBox := boxStyle.Render(fmt.Sprintf("Target: %s\n%s",
				lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render(commit.ShortID),
				strings.TrimRight(commit.Summary, "\n"),
			))
			lines = append(lines, commitBox)
			lines = append(lines, "")
		}
		if len(m.existingBookmarks) > 0 {
			lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Move Existing Bookmark:"))
			lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Click or use j/k to select, Enter to move"))
			lines = append(lines, "")
			// Display-only truncation: the actual bookmark name in m.existingBookmarks
			// is what gets resolved for move/click. New names are capped by
			// jj.TruncateBookmarkName before creation, but historical repos may
			// still carry long bookmark names from before that cap existed — those
			// would otherwise be wider than the modal's content area on narrow
			// terminals and force the outer frame to wrap the row.
			nameW := m.boxWidth() - 2 // "  " or "► " prefix; "► " is 1 rune + space
			for i, bookmark := range m.existingBookmarks {
				prefix := "  "
				style := styles.CommitStyle
				if i == m.selectedBookmarkIdx {
					prefix = "► "
					style = styles.CommitSelectedStyle
				}
				display := bookmark
				if nameW > 0 {
					display = runewidth.Truncate(display, nameW, "…")
				}
				bookmarkLine := fmt.Sprintf("%s%s", prefix, display)
				lines = append(lines, mark(m.zoneManager, mouse.ZoneExistingBookmark(i), style.Render(bookmarkLine)))
			}
			lines = append(lines, "")
			lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("─────────────────────────────────"))
			lines = append(lines, "")
		}
	}
	newStyle := lipgloss.NewStyle().Bold(true)
	if m.selectedBookmarkIdx == -1 || m.fromJira {
		newStyle = newStyle.Foreground(styles.ColorPrimary)
	}
	if m.fromJira {
		lines = append(lines, newStyle.Render("Branch/Bookmark Name:"))
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Edit if needed, then press Enter to create"))
	} else {
		lines = append(lines, newStyle.Render("Create New Bookmark:"))
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Type a name and press Enter"))
	}
	lines = append(lines, "")
	inputStyle := lipgloss.NewStyle()
	if m.selectedBookmarkIdx == -1 || m.fromJira {
		inputStyle = inputStyle.Foreground(styles.ColorPrimary)
	}
	nameW := m.nameInput.Width
	if nameW < 8 {
		nameW = 50
	}
	// Align the ✧ ^g chip with the indented input row ("  " + field, width 2+nameW).
	nameRowW := 2 + nameW
	if m.selectedBookmarkIdx == -1 {
		genChip := mark(m.zoneManager, mouse.ZoneBookmarkGenerate, styles.AIGenerateChip())
		lines = append(lines, styles.SpreadRow(nameRowW, inputStyle.Render("  Name:"), genChip))
	} else {
		lines = append(lines, inputStyle.Render("Name:"))
	}
	lines = append(lines, mark(m.zoneManager, mouse.ZoneBookmarkName, "  "+m.nameInput.View()))
	if m.bookmarkNameExists {
		warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E3B341")).Bold(true)
		lines = append(lines, "")
		lines = append(lines, warningStyle.Render("⚠ A bookmark with this name already exists"))
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("  Creating will move the existing bookmark to this commit"))
	}
	lines = append(lines, "")
	var submitLabel string
	if m.fromJira {
		submitLabel = "Create Branch (Enter)"
	} else if m.selectedBookmarkIdx >= 0 && m.selectedBookmarkIdx < len(m.existingBookmarks) {
		submitLabel = fmt.Sprintf("Move '%s' (Enter)", m.existingBookmarks[m.selectedBookmarkIdx])
	} else {
		submitLabel = "Create (Enter)"
	}
	submitButton := mark(m.zoneManager, mouse.ZoneBookmarkSubmit, styles.ButtonStyle.Render(submitLabel))
	cancelButton := mark(m.zoneManager, mouse.ZoneBookmarkCancel, styles.ButtonStyle.Render("Cancel (Esc)"))
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, submitButton, " ", cancelButton))
	return strings.Join(lines, "\n")
}
