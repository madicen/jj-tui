package graph

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/data"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/styles"
	overlay "github.com/madicen/bubble-overlay"
)

// revsetFilterInput holds the `/` search overlay state.
type revsetFilterInput struct {
	open  bool
	input textinput.Model
}

func newRevsetFilterInput() revsetFilterInput {
	ti := textinput.New()
	ti.Placeholder = "text or :revset"
	ti.Prompt = "/ "
	ti.CharLimit = 512
	return revsetFilterInput{input: ti}
}

// FilterInputOpen reports whether the search overlay is active.
func (m GraphModel) FilterInputOpen() bool {
	return m.filterInput.open
}

// beginFilterInput opens the `/` search overlay.
func (m *GraphModel) beginFilterInput(app *state.AppState) tea.Cmd {
	if m.filterInput.input.Prompt == "" {
		m.filterInput = newRevsetFilterInput()
	}
	m.filterInput.open = true
	val := ""
	if app != nil {
		val = app.GraphFilterQuery
	}
	m.filterInput.input.SetValue(val)
	m.filterInput.input.Focus()
	return textinput.Blink
}

func (m *GraphModel) closeFilterInput() {
	m.filterInput.open = false
	m.filterInput.input.Blur()
}

// handleFilterInputKey owns keyboard input while the search overlay is open.
func (m *GraphModel) handleFilterInputKey(msg tea.KeyMsg, app *state.AppState) (GraphModel, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.CancelSelection):
		m.closeFilterInput()
		return *m, nil
	case key.Matches(msg, m.keys.Checkout):
		query := strings.TrimSpace(m.filterInput.input.Value())
		m.closeFilterInput()
		if query == "" {
			return *m, nil
		}
		revset, display, err := jj.CompileGraphFilterRevset(query)
		if err != nil {
			if app != nil {
				app.StatusMessage = err.Error()
			}
			return *m, nil
		}
		if app != nil {
			app.Loading = true
			app.StatusMessage = "Filtering graph…"
		}
		if app == nil || app.JJService == nil {
			return *m, nil
		}
		return *m, tea.Batch(data.ApplyGraphFilterCmd(app.JJService, display, revset))
	}
	var cmd tea.Cmd
	m.filterInput.input, cmd = m.filterInput.input.Update(msg)
	return *m, cmd
}

// clearGraphFilter removes the active filter and reloads the default graph revset.
func clearGraphFilter(app *state.AppState) tea.Cmd {
	if app == nil {
		return nil
	}
	app.GraphFilterQuery = ""
	app.GraphFilterRevset = ""
	app.GraphFilterError = ""
	if app.JJService == nil {
		return nil
	}
	app.Loading = true
	app.StatusMessage = "Clearing filter…"
	return data.LoadRepository(app.JJService, "")
}

// renderFilterInputOverlay renders the centered search input box.
func (m GraphModel) renderFilterInputOverlay() string {
	if !m.filterInput.open {
		return ""
	}
	muted := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	modalW := min(max(48, m.width-12), 72)
	lines := []string{
		muted.Render("Search commits (Enter apply · Esc cancel)"),
		m.filterInput.input.View(),
		muted.Render("Free text matches description or author; prefix : for raw revset"),
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorPrimary).
		Padding(0, 1).
		Width(modalW).
		Render(strings.Join(lines, "\n"))
	return box
}

// overlayFilterInput composites the search overlay onto the graph view.
func (m GraphModel) overlayFilterInput(v string) string {
	if box := m.renderFilterInputOverlay(); box != "" {
		return overlay.OverlayViewInCenterWithOffset(v, box, m.width, m.height, 0, 0)
	}
	return v
}
