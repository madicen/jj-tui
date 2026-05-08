package initrepo

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/styles"
	"github.com/madicen/jj-tui/internal/tui/util"
)

// Model is the "not a jj repo" welcome screen. Path non-empty means the screen is active.
//
// The screen offers three onboarding paths so a user lands in a useful state without leaving the
// TUI: bare init (just `jj git init --colocate`), init plus an existing remote URL, and init plus
// `gh repo create` for a brand-new GitHub repo. We default to colocate because every subsequent
// flow (git remote add, gh repo create --source=., bookmark/PR pushes) assumes a colocated
// `.git/` exists; running plain `jj git init` consistently produced the "No git remote named
// 'origin'" error users were hitting on first push.
type Model struct {
	path        string
	urlInput    textinput.Model
	ghAvailable bool // cached `exec.LookPath("gh")` result; refreshed on SetPath
	ghPrivate   bool // visibility for the `gh repo create` button (default: true => --private)
	zoneManager *zone.Manager
}

// NewModel creates a new init-repo screen model.
func NewModel() Model {
	in := textinput.New()
	in.Placeholder = "git@github.com:owner/repo.git  or  https://github.com/owner/repo.git"
	in.CharLimit = 512
	in.Width = 60
	return Model{
		ghPrivate: true,
	}
}

// Init is required by tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles keys and zone clicks; sends NavigateRunInit / NavigateDismissInit to main.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if m.path == "" {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case zone.MsgZoneInBounds:
		if m.zoneManager != nil {
			if zoneID := m.resolveClickedZone(msg); zoneID != "" {
				return m.handleZoneClick(zoneID)
			}
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	// When the URL field has focus most keys must reach the textinput so the user can paste/type
	// freely. Only Tab/Enter (commit and run) and Esc (blur) are intercepted; everything else is
	// forwarded.
	if m.urlInput.Focused() {
		switch msg.String() {
		case "esc":
			m.urlInput.Blur()
			return m, nil
		case "tab", "shift+tab":
			m.urlInput.Blur()
			return m, nil
		case "enter":
			// Pressing Enter inside the URL field commits the URL and triggers init. This is the
			// path users naturally take after pasting a URL.
			m.urlInput.Blur()
			return m, m.runInitCmd()
		case "ctrl+q", "ctrl+c":
			util.FlushMouse()
			return m, tea.Quit
		}
		var cmd tea.Cmd
		m.urlInput, cmd = m.urlInput.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "esc":
		return m, state.NavigateTarget{Kind: state.NavigateDismissInit, StatusMessage: "Dismissed"}.Cmd()
	case "i", "enter":
		return m, m.runInitCmd()
	case "g":
		if m.ghAvailable {
			return m, m.runGhCreateCmd()
		}
		return m, nil
	case "ctrl+v":
		m.ghPrivate = !m.ghPrivate
		return m, nil
	case "tab", "u":
		m.urlInput.Focus()
		return m, nil
	case "ctrl+q", "ctrl+c":
		util.FlushMouse()
		return m, tea.Quit
	}
	return m, nil
}

// runInitCmd builds the NavigateRunInit target for "Initialize Repository" using the optional URL.
func (m Model) runInitCmd() tea.Cmd {
	url := strings.TrimSpace(m.urlInput.Value())
	return state.NavigateTarget{
		Kind:          state.NavigateRunInit,
		InitColocate:  true,
		InitRemoteURL: url,
	}.Cmd()
}

// runGhCreateCmd builds the NavigateRunInit target for "Create new GitHub repo".
func (m Model) runGhCreateCmd() tea.Cmd {
	name := filepath.Base(m.path)
	return state.NavigateTarget{
		Kind:              state.NavigateRunInit,
		InitColocate:      true,
		InitGhCreateRepo:  true,
		InitGhRepoName:    name,
		InitGhRepoPrivate: m.ghPrivate,
	}.Cmd()
}

func (m Model) resolveClickedZone(msg zone.MsgZoneInBounds) string {
	if msg.Zone == nil {
		return ""
	}
	for _, id := range m.zoneIDs() {
		z := m.zoneManager.Get(id)
		if z != nil && z.InBounds(msg.Event) {
			return id
		}
	}
	return ""
}

func (m Model) zoneIDs() []string {
	ids := []string{
		mouse.ZoneActionJJInit,
		mouse.ZoneActionInitURLInput,
		mouse.ZoneActionInitVisibilityToggle,
	}
	if m.ghAvailable {
		ids = append(ids, mouse.ZoneActionInitGhRepoCreate)
	}
	return ids
}

func (m Model) handleZoneClick(zoneID string) (Model, tea.Cmd) {
	switch zoneID {
	case mouse.ZoneActionJJInit:
		m.urlInput.Blur()
		return m, m.runInitCmd()
	case mouse.ZoneActionInitURLInput:
		m.urlInput.Focus()
		return m, nil
	case mouse.ZoneActionInitGhRepoCreate:
		if !m.ghAvailable {
			return m, nil
		}
		m.urlInput.Blur()
		return m, m.runGhCreateCmd()
	case mouse.ZoneActionInitVisibilityToggle:
		m.ghPrivate = !m.ghPrivate
		return m, nil
	}
	return m, nil
}

// View renders the welcome screen (path must be non-empty). Main applies modal centering.
func (m Model) View() string {
	if m.path == "" {
		return ""
	}
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E"))
	pathStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#58A6FF"))
	mark := func(id, s string) string {
		if m.zoneManager == nil {
			return s
		}
		return m.zoneManager.Mark(id, s)
	}

	repoName := filepath.Base(m.path)
	var lines []string
	lines = append(lines, styles.TitleStyle.Render("Welcome to jj-tui"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Directory: %s", pathStyle.Render(m.path)))
	lines = append(lines, "")
	lines = append(lines, "This directory is not yet a Jujutsu repository.")
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Initialize"))
	lines = append(lines, "")

	initButton := styles.ButtonStyle.Background(lipgloss.Color("#238636")).Render("Initialize Repository (i)")
	lines = append(lines, mark(mouse.ZoneActionJJInit, initButton))
	lines = append(lines, "")
	lines = append(lines, mutedStyle.Render("Runs `jj git init --colocate` in this directory."))
	lines = append(lines, "")
	lines = append(lines, mutedStyle.Render(strings.Repeat("─", 60)))
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Optional: connect a remote"))
	lines = append(lines, "")

	urlLabel := "Remote URL"
	if m.urlInput.Focused() {
		urlLabel = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true).Render("Remote URL")
	}
	urlBox := mark(mouse.ZoneActionInitURLInput, m.urlInput.View())
	lines = append(lines, urlLabel+":  "+urlBox)
	lines = append(lines, "")
	lines = append(lines, mutedStyle.Render("Tab/u to focus, paste a URL, then press Enter or i to initialize with origin set."))
	lines = append(lines, "")
	lines = append(lines, mutedStyle.Render(strings.Repeat("─", 60)))
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Or create a brand-new GitHub repo"))
	lines = append(lines, "")

	if m.ghAvailable {
		ghButton := styles.ButtonStyle.Background(lipgloss.Color("#1f6feb")).Render(fmt.Sprintf("Create new GitHub repo (%s) (g)", repoName))
		visLabel := "Public"
		if m.ghPrivate {
			visLabel = "Private"
		}
		visButton := styles.ButtonStyle.Render(fmt.Sprintf("Visibility: %s  (^v)", visLabel))
		lines = append(lines,
			mark(mouse.ZoneActionInitGhRepoCreate, ghButton)+"  "+mark(mouse.ZoneActionInitVisibilityToggle, visButton),
		)
		lines = append(lines, "")
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("Runs `gh repo create %s --%s --source=. --remote=origin`. Requires gh CLI authentication.", repoName, strings.ToLower(visLabel))))
	} else {
		lines = append(lines, mutedStyle.Render("`gh` CLI not found in PATH. Install GitHub CLI and run `gh auth login` to enable this option."))
	}
	lines = append(lines, "")
	lines = append(lines, mutedStyle.Render("Press Esc to dismiss · Ctrl+q to quit"))

	return strings.Join(lines, "\n")
}

// SetPath sets the directory path and activates the screen; empty path hides it. Refreshes the
// gh-availability cache so we can render the GitHub option only when it would actually work.
func (m *Model) SetPath(path string) {
	prev := m.path
	m.path = path
	if path != "" && path != prev {
		_, err := exec.LookPath("gh")
		m.ghAvailable = err == nil
		m.urlInput.SetValue("")
		m.urlInput.Blur()
	}
	if path == "" {
		m.urlInput.Blur()
	}
}

// Path returns the current path (empty if screen is not active).
func (m *Model) Path() string {
	return m.path
}

// SetZoneManager sets the zone manager for clickable elements.
func (m *Model) SetZoneManager(zm *zone.Manager) {
	m.zoneManager = zm
}
