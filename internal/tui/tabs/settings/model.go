package settings

import (
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/tabs/settings/advanced"
	"github.com/madicen/jj-tui/internal/tui/tabs/settings/branches"
	"github.com/madicen/jj-tui/internal/tui/tabs/settings/codecks"
	"github.com/madicen/jj-tui/internal/tui/tabs/settings/github"
	"github.com/madicen/jj-tui/internal/tui/tabs/settings/jira"
	"github.com/madicen/bubble-color-picker"
	"github.com/madicen/jj-tui/internal/tui/tabs/settings/theme"
	"github.com/madicen/jj-tui/internal/tui/tabs/settings/tickets"
)

// Model represents the state of the Settings tab (routing-only; state lives in sub-models).
type Model struct {
	settingsTab   int
	zoneManager   *zone.Manager
	panelYOffset  [7]int
	width         int
	height        int
	viewOpts      *ViewOpts

	githubModel   github.Model
	jiraModel     jira.Model
	codecksModel  codecks.Model
	ticketsModel  tickets.Model
	branchesModel branches.Model
	themeModel    theme.Model
	advancedModel advanced.Model
}

// NewModel creates a new Settings tab model with default sub-models.
func NewModel() Model {
	return Model{
		settingsTab:   0,
		githubModel:   github.NewModel(),
		jiraModel:     jira.NewModel(),
		codecksModel:  codecks.NewModel(),
		ticketsModel:  tickets.NewModel(),
		branchesModel: branches.NewModel(),
		themeModel:    theme.NewModel(),
		advancedModel: advanced.NewModel(),
	}
}

// NewModelWithConfig creates a Settings tab model with all sub-models initialized from config and env.
func NewModelWithConfig(cfg *config.Config) Model {
	return Model{
		settingsTab:   0,
		githubModel:   github.NewModelFromConfig(cfg),
		jiraModel:     jira.NewModelFromConfig(cfg),
		codecksModel:  codecks.NewModelFromConfig(cfg),
		ticketsModel:  tickets.NewModelFromConfig(cfg),
		branchesModel: branches.NewModelFromConfig(cfg),
		themeModel:    theme.NewModelFromConfig(cfg),
		advancedModel: advanced.NewModelFromConfig(cfg),
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the Settings tab
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}
	// When Theme tab is active and a color picker is open, forward all input to the theme model.
	if m.settingsTab == 5 && m.themeModel.AnyOpen() {
		updated, cmd := m.themeModel.Update(msg)
		m.themeModel = updated
		return m, cmd
	}
	// When Theme tab is active and we get picker result messages, forward to theme model so the swatch closes
	if m.settingsTab == 5 {
		switch msg.(type) {
		case bubblepicker.ColorChosenMsg, bubblepicker.ColorCanceledMsg:
			updated, cmd := m.themeModel.Update(msg)
			m.themeModel = updated
			return m, cmd
		}
	}
	// When Theme tab is active, forward left mouse press to theme swatch if click is in a swatch zone
	// (bubblezone sends MsgZoneInBounds on release, but the swatch opens on press—so we must handle press here)
	if m.settingsTab == 5 && m.zoneManager != nil {
		if mouseMsg, ok := msg.(tea.MouseMsg); ok && mouseMsg.Action == tea.MouseActionPress && mouseMsg.Button == tea.MouseButtonLeft {
			for _, zoneID := range []string{mouse.ZoneSettingsThemePrimary, mouse.ZoneSettingsThemeSecondary, mouse.ZoneSettingsThemeMuted} {
				z := m.zoneManager.Get(zoneID)
				if z != nil && z.InBounds(mouseMsg) {
					idx := ThemeSwatchIndex(zoneID)
					if idx >= 0 {
						updated, cmd := m.themeModel.UpdateSwatch(idx, mouseMsg)
						m.themeModel = updated
						return m, cmd
					}
				}
			}
		}
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case zone.MsgZoneInBounds:
		if m.zoneManager != nil {
			zoneID := m.resolveClickedZone(msg)
			if zoneID != "" {
				return m.routeZoneToPanel(zoneID, msg.Event)
			}
		}
		return m, nil
	case tea.MouseMsg:
		if tea.MouseEvent(msg).IsWheel() {
			delta := 3
			isUp := msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelLeft
			idx := m.settingsTab
			if isUp {
				m.panelYOffset[idx] -= delta
			} else {
				m.panelYOffset[idx] += delta
			}
			if m.panelYOffset[idx] < 0 {
				m.panelYOffset[idx] = 0
			}
			return m, nil
		}
		return m, nil
	}
	// Forward other messages (e.g. textinput.Blink for cursor) to the active submodel
	// so the focused input receives them and can show the cursor / update correctly.
	return m.forwardToActiveSubmodel(msg)
}

// View renders the Settings tab using stored viewOpts (set by main when entering tab or on resize).
func (m Model) View() string {
	if m.viewOpts == nil {
		return ""
	}
	out := RenderWithState(m.zoneManager, &m, *m.viewOpts)
	if m.settingsTab == 5 && m.themeModel.AnyOpen() {
		out = m.themeModel.ViewWithOverlay(out, m.width, m.height)
	}
	return out
}

// SetViewOpts sets the options used by View() to render (called by main when entering settings or on resize).
func (m *Model) SetViewOpts(opts ViewOpts) {
	m.viewOpts = &opts
}

// EscHandledInsideSettings is true when Esc should be handled by the settings tab (theme color
// picker, advanced cleanup confirm) instead of closing settings and returning to the graph.
func (m Model) EscHandledInsideSettings() bool {
	if m.advancedModel.GetConfirmingCleanup() != "" {
		return true
	}
	return m.settingsTab == 5 && m.themeModel.AnyOpen()
}

// handleKeyMsg handles all keyboard input for the Settings tab (cleanup confirm, nav, focus, save, inputs).
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.advancedModel.GetConfirmingCleanup() != "" {
		switch msg.String() {
		case "y", "Y":
			m.advancedModel.SetConfirmingCleanup("")
			return m, RequestConfirmCleanupCmd()
		case "n", "N", "esc":
			m.advancedModel.SetConfirmingCleanup("")
			return m, RequestCancelCleanupCmd()
		}
		return m, nil
	}

	switch msg.String() {
	case "esc":
		return m, PerformCancelCmd()
	case "ctrl+j":
		tab := m.settingsTab - 1
		if tab < 0 {
			tab = 6
		}
		m.settingsTab = tab % 7
		if m.settingsTab == 6 {
			return m, m.advancedModel.SetFocusedField(0)
		}
		return m, nil
	case "ctrl+k":
		m.settingsTab = (m.settingsTab + 1) % 7
		if m.settingsTab == 6 {
			return m, m.advancedModel.SetFocusedField(0)
		}
		return m, nil
	case "ctrl+s", "enter":
		if m.settingsTab == 6 {
			// Advanced tab: forward enter to revset input
			return m.forwardKeyToActiveSubmodelReturn(msg)
		}
		if msg.String() == "enter" {
			// Advance focus within panel if not on last field; otherwise save
			lastField := false
			switch m.settingsTab {
			case 0:
				lastField = m.githubModel.GetFocusedField() >= 0
			case 1:
				lastField = m.jiraModel.GetFocusedField() >= 7
			case 2:
				lastField = m.codecksModel.GetFocusedField() >= 3
			case 3:
				lastField = m.ticketsModel.GetTicketProvider() != "github_issues" || m.ticketsModel.GetFocusedField() >= 0
			case 5:
				lastField = true // Theme tab: Enter saves
			}
			if !lastField {
				var cmd tea.Cmd
				m, cmd = m.forwardKeyToActiveSubmodelReturn(msg)
				return m, cmd
			}
		}
		return m, Request{SaveSettings: true}.Cmd()
	case "ctrl+l":
		return m, Request{SaveSettingsLocal: true}.Cmd()
	case "tab", "down":
		if m.settingsTab != 6 {
			m.forwardKeyToActiveSubmodel(msg)
			return m, nil
		}
	case "shift+tab", "up":
		if m.settingsTab != 6 {
			m.forwardKeyToActiveSubmodel(msg)
			return m, nil
		}
	}

	// Forward all other keys (including letters like j/k) to the focused input (Theme tab has no inputs)
	return m.forwardKeyToActiveSubmodelReturn(msg)
}

// ZoneIDs returns the zone IDs this tab uses when rendering (same IDs used in settingstab.Render). Used to resolve clicks.
func (m *Model) ZoneIDs() []string {
	return []string{
		mouse.ZoneSettingsTabGitHub, mouse.ZoneSettingsTabJira, mouse.ZoneSettingsTabCodecks,
		mouse.ZoneSettingsTabTickets, mouse.ZoneSettingsTabBranches, mouse.ZoneSettingsTabTheme, mouse.ZoneSettingsTabAdvanced,
		mouse.ZoneSettingsThemePrimary, mouse.ZoneSettingsThemeSecondary, mouse.ZoneSettingsThemeMuted,
		mouse.ZoneSettingsThemePrimaryDefault, mouse.ZoneSettingsThemeSecondaryDefault, mouse.ZoneSettingsThemeMutedDefault,
		mouse.ZoneSettingsTicketProviderNone, mouse.ZoneSettingsTicketProviderJira,
		mouse.ZoneSettingsTicketProviderCodecks, mouse.ZoneSettingsTicketProviderGitHubIssues,
		mouse.ZoneSettingsAutoInProgress,
		mouse.ZoneSettingsAdvancedConfirmYes, mouse.ZoneSettingsAdvancedConfirmNo,
		mouse.ZoneSettingsAdvancedDeleteBookmarks, mouse.ZoneSettingsAdvancedAbandonOldCommits,
		mouse.ZoneSettingsGraphRevset, mouse.ZoneSettingsGraphRevsetClear,
		mouse.ZoneSettingsSanitizeBookmarks,
		mouse.ZoneSettingsGitHubLogin,
		mouse.ZoneSettingsGitHubOnlyMine, mouse.ZoneSettingsGitHubShowMerged, mouse.ZoneSettingsGitHubShowClosed,
		mouse.ZoneSettingsGitHubPRLimitDecrease, mouse.ZoneSettingsGitHubPRLimitIncrease,
		mouse.ZoneSettingsGitHubRefreshDecrease, mouse.ZoneSettingsGitHubRefreshIncrease, mouse.ZoneSettingsGitHubRefreshToggle,
		mouse.ZoneSettingsBranchLimitDecrease, mouse.ZoneSettingsBranchLimitIncrease,
		mouse.ZoneSettingsGitHubTokenClear, mouse.ZoneSettingsJiraURLClear, mouse.ZoneSettingsJiraUserClear,
		mouse.ZoneSettingsJiraTokenClear, mouse.ZoneSettingsJiraProjectClear, mouse.ZoneSettingsJiraProjectFilterClear, mouse.ZoneSettingsJiraIssueTypeClear, mouse.ZoneSettingsJiraJQLClear,
		mouse.ZoneSettingsJiraExcludedClear, mouse.ZoneSettingsCodecksSubdomainClear, mouse.ZoneSettingsCodecksTokenClear,
		mouse.ZoneSettingsCodecksProjectClear, mouse.ZoneSettingsCodecksExcludedClear, mouse.ZoneSettingsGitHubIssuesExcludedClear,
		mouse.ZoneSettingsGitHubToken, mouse.ZoneSettingsJiraURL, mouse.ZoneSettingsJiraUser,
		mouse.ZoneSettingsJiraToken, mouse.ZoneSettingsJiraProject, mouse.ZoneSettingsJiraProjectFilter, mouse.ZoneSettingsJiraIssueType, mouse.ZoneSettingsJiraJQL,
		mouse.ZoneSettingsJiraExcluded, mouse.ZoneSettingsCodecksSubdomain, mouse.ZoneSettingsCodecksToken,
		mouse.ZoneSettingsCodecksProject, mouse.ZoneSettingsCodecksExcluded, mouse.ZoneSettingsGitHubIssuesExcluded,
		mouse.ZoneSettingsSave, mouse.ZoneSettingsSaveLocal, mouse.ZoneSettingsCancel,
	}
}

func (m *Model) resolveClickedZone(msg zone.MsgZoneInBounds) string {
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

// SetZoneManager sets the zone manager used to resolve clicks (main's manager; zones are created in settingstab.Render).
func (m *Model) SetZoneManager(zm *zone.Manager) {
	m.zoneManager = zm
	m.themeModel.SetZoneManager(zm)
}

// Sub-model getters (return pointers so zone handlers and BuildSettingsParams can mutate)

func (m *Model) GetGitHubModel() *github.Model    { return &m.githubModel }
func (m *Model) GetJiraModel() *jira.Model        { return &m.jiraModel }
func (m *Model) GetCodecksModel() *codecks.Model  { return &m.codecksModel }
func (m *Model) GetTicketsModel() *tickets.Model  { return &m.ticketsModel }
func (m *Model) GetBranchesModel() *branches.Model { return &m.branchesModel }
func (m *Model) GetThemeModel() *theme.Model      { return &m.themeModel }
func (m *Model) GetAdvancedModel() *advanced.Model { return &m.advancedModel }

// forwardKeyToActiveSubmodel updates focus/state for tab/up/down within the active panel (mutates m in place).
func (m *Model) forwardKeyToActiveSubmodel(msg tea.KeyMsg) {
	switch m.settingsTab {
	case 0:
		gh := m.GetGitHubModel()
		switch msg.String() {
		case "tab", "down", "j":
			if gh.GetFocusedField() < 4 {
				gh.SetFocusedField(gh.GetFocusedField() + 1)
			}
		case "shift+tab", "up", "k":
			if gh.GetFocusedField() > 0 {
				gh.SetFocusedField(gh.GetFocusedField() - 1)
			}
		}
	case 1:
		jr := m.GetJiraModel()
		switch msg.String() {
		case "tab", "down", "j":
			if jr.GetFocusedField() < 7 {
				jr.SetFocusedField(jr.GetFocusedField() + 1)
			}
		case "shift+tab", "up", "k":
			if jr.GetFocusedField() > 0 {
				jr.SetFocusedField(jr.GetFocusedField() - 1)
			}
		}
	case 2:
		cc := m.GetCodecksModel()
		switch msg.String() {
		case "tab", "down", "j":
			if cc.GetFocusedField() < 3 {
				cc.SetFocusedField(cc.GetFocusedField() + 1)
			}
		case "shift+tab", "up", "k":
			if cc.GetFocusedField() > 0 {
				cc.SetFocusedField(cc.GetFocusedField() - 1)
			}
		}
	case 3:
		tk := m.GetTicketsModel()
		if tk.GetTicketProvider() == "github_issues" {
			switch msg.String() {
			case "tab", "down", "j":
				tk.SetFocusedField(0)
			case "shift+tab", "up", "k":
				tk.SetFocusedField(0)
			}
		}
	case 5:
		// Theme tab: no fields to focus
	case 6:
		adv := m.GetAdvancedModel()
		switch msg.String() {
		case "tab", "down", "j":
			if adv.GetFocusedField() < 0 {
				adv.SetFocusedField(0)
			}
		case "shift+tab", "up", "k":
			if adv.GetFocusedField() > 0 {
				adv.SetFocusedField(0)
			}
		}
	}
}

// forwardKeyToActiveSubmodelReturn forwards the key to the active sub-model and returns updated model and cmd.
func (m Model) forwardKeyToActiveSubmodelReturn(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch m.settingsTab {
	case 0:
		updated, cmd := m.githubModel.Update(msg)
		m.githubModel = updated
		return m, cmd
	case 1:
		updated, cmd := m.jiraModel.Update(msg)
		m.jiraModel = updated
		return m, cmd
	case 2:
		updated, cmd := m.codecksModel.Update(msg)
		m.codecksModel = updated
		return m, cmd
	case 3:
		updated, cmd := m.ticketsModel.Update(msg)
		m.ticketsModel = updated
		return m, cmd
	case 5:
		// Theme tab has no text inputs
		return m, nil
	case 6:
		updated, cmd := m.advancedModel.Update(msg)
		m.advancedModel = updated
		return m, cmd
	}
	return m, nil
}

// forwardToActiveSubmodel forwards any message to the active sub-model (e.g. textinput.Blink for cursor).
// Panels with inputs (GitHub, Jira, Codecks, Tickets, Advanced) need to receive these so the cursor blinks.
func (m Model) forwardToActiveSubmodel(msg tea.Msg) (Model, tea.Cmd) {
	switch m.settingsTab {
	case 0:
		updated, cmd := m.githubModel.Update(msg)
		m.githubModel = updated
		return m, cmd
	case 1:
		updated, cmd := m.jiraModel.Update(msg)
		m.jiraModel = updated
		return m, cmd
	case 2:
		updated, cmd := m.codecksModel.Update(msg)
		m.codecksModel = updated
		return m, cmd
	case 3:
		updated, cmd := m.ticketsModel.Update(msg)
		m.ticketsModel = updated
		return m, cmd
	case 5:
		// Theme tab: no inputs
		return m, nil
	case 6:
		updated, cmd := m.advancedModel.Update(msg)
		m.advancedModel = updated
		return m, cmd
	}
	return m, nil
}

// Accessors

// GetSettingsTab returns the currently selected settings tab
func (m *Model) GetSettingsTab() int {
	return m.settingsTab
}

// SetSettingsTab sets the settings sub-tab (0=GitHub, 1=Jira, 2=Codecks, 3=Tickets, 4=Branches, 5=Theme, 6=Advanced)
func (m *Model) SetSettingsTab(tab int) {
	if tab < 0 {
		tab = 0
	}
	m.settingsTab = tab % 7
}

// GetFocusedField returns the currently focused input field (global index 0-14 for BuildRenderData).
func (m *Model) GetFocusedField() int {
	switch m.settingsTab {
	case 0:
		return m.githubModel.GetFocusedField()
	case 1:
		return 1 + m.jiraModel.GetFocusedField()
	case 2:
		return 9 + m.codecksModel.GetFocusedField()
	case 3:
		if m.ticketsModel.GetTicketProvider() == "github_issues" {
			return 13
		}
		return 0
	case 4:
		return 0 // Branches settings: no text inputs
	case 5:
		return 0 // Theme tab has no inputs
	case 6:
		return 14 + m.advancedModel.GetFocusedField()
	}
	return 0
}

// SetFocusedField sets the focused input field (global index); used by zone handlers to focus an input.
// Returns a tea.Cmd when focusing the Advanced revset input (index 14) so the cursor is shown.
func (m *Model) SetFocusedField(idx int) tea.Cmd {
	if idx < 1 {
		m.githubModel.SetFocusedField(0)
		return nil
	}
	if idx < 9 {
		m.jiraModel.SetFocusedField(idx - 1)
		return nil
	}
	if idx < 13 {
		m.codecksModel.SetFocusedField(idx - 9)
		return nil
	}
	if idx < 14 {
		m.ticketsModel.SetFocusedField(0)
		return nil
	}
	return m.advancedModel.SetFocusedField(idx - 14)
}

// ThemeSwatchIndex returns the theme swatch index (0–2) for the given zone ID, or -1.
func ThemeSwatchIndex(zoneID string) int {
	switch zoneID {
	case mouse.ZoneSettingsThemePrimary:
		return 0
	case mouse.ZoneSettingsThemeSecondary:
		return 1
	case mouse.ZoneSettingsThemeMuted:
		return 2
	}
	return -1
}

// ThemeDefaultZoneIndex returns the theme swatch index (0–2) for a [Default] button zone ID, or -1.
func ThemeDefaultZoneIndex(zoneID string) int {
	switch zoneID {
	case mouse.ZoneSettingsThemePrimaryDefault:
		return 0
	case mouse.ZoneSettingsThemeSecondaryDefault:
		return 1
	case mouse.ZoneSettingsThemeMutedDefault:
		return 2
	}
	return -1
}

// EnterTab prepares the tab when main navigates to Settings (focus first field of active panel).
// Returns a tea.Cmd when focusing the Advanced revset input so the cursor is shown.
func (m *Model) EnterTab() tea.Cmd {
	switch m.settingsTab {
	case 0:
		m.githubModel.SetFocusedField(0)
	case 1:
		m.jiraModel.SetFocusedField(0)
	case 2:
		m.codecksModel.SetFocusedField(0)
	case 3:
		m.ticketsModel.SetFocusedField(0)
	case 5:
		// Theme tab: nothing to focus
	case 6:
		return m.advancedModel.SetFocusedField(0)
	}
	return nil
}

// GetSettingsInputs returns a slice of textinput views for BuildRenderData (built from sub-models).
// The layout is fixed so that the Advanced revset input is always at index 14 (needed for renderAdvanced).
func (m *Model) GetSettingsInputs() []struct{ View string } {
	var out []struct{ View string }
	for _, v := range m.githubModel.GetInputViews() {
		out = append(out, struct{ View string }{v})
	}
	for _, v := range m.jiraModel.GetInputViews() {
		out = append(out, struct{ View string }{v})
	}
	for _, v := range m.codecksModel.GetInputViews() {
		out = append(out, struct{ View string }{v})
	}
	for _, v := range m.ticketsModel.GetInputViews() {
		out = append(out, struct{ View string }{v})
	}
	// Pad so Advanced revset is always at index 14 (Tickets returns 0 when provider != github_issues).
	for len(out) < 14 {
		out = append(out, struct{ View string }{""})
	}
	for _, v := range m.advancedModel.GetInputViews() {
		out = append(out, struct{ View string }{v})
	}
	// Ensure exactly 15 elements so views can safely use data.Inputs[0..14].
	for len(out) < 15 {
		out = append(out, struct{ View string }{""})
	}
	return out
}

// SetInputs is a no-op when using sub-models (inputs live in sub-models).
func (m *Model) SetInputs(_ interface{}) {}

// SetInputWidths sets the width of all inputs in sub-models (called on window resize).
func (m *Model) SetInputWidths(width int) {
	m.githubModel.SetInputWidth(width)
	m.jiraModel.SetInputWidth(width)
	m.codecksModel.SetInputWidth(width)
	m.ticketsModel.SetInputWidth(width)
	m.advancedModel.SetInputWidth(width)
}

// SetDimensions sets the content area dimensions (used for scroll viewport).
func (m *Model) SetDimensions(width, height int) {
	m.width = width
	m.height = height
}

// GetSettingsYOffset returns the current scroll offset for the active settings panel.
func (m *Model) GetSettingsYOffset() int {
	if m.settingsTab < 0 || m.settingsTab >= 7 {
		return 0
	}
	return m.panelYOffset[m.settingsTab]
}

// SetSettingInputValue sets the value of the settings input at index (e.g. after GitHub login; index 0 = GitHub token).
func (m *Model) SetSettingInputValue(index int, value string) {
	if index == 0 {
		m.githubModel.SetToken(value)
	}
	// Other indices (jira/codecks/tickets) could be set here if needed
}
func (m *Model) UpdateRepository(repo *internal.Repository) {}

// Getters for toggle/state (delegate to sub-models)
func (m *Model) GetSettingsShowMerged() bool        { return m.githubModel.GetShowMerged() }
func (m *Model) GetSettingsShowClosed() bool       { return m.githubModel.GetShowClosed() }
func (m *Model) GetSettingsOnlyMine() bool         { return m.githubModel.GetOnlyMine() }
func (m *Model) GetSettingsPRLimit() int           { return m.githubModel.GetPRLimit() }
func (m *Model) GetSettingsPRRefreshInterval() int { return m.githubModel.GetRefreshInterval() }
func (m *Model) GetSettingsAutoInProgress() bool   { return m.ticketsModel.GetAutoInProgress() }
func (m *Model) GetSettingsBranchLimit() int       { return m.branchesModel.GetBranchLimit() }
func (m *Model) GetSettingsSanitizeBookmarks() bool { return m.advancedModel.GetSanitizeBookmarks() }
func (m *Model) GetSettingsTicketProvider() string { return m.ticketsModel.GetTicketProvider() }
func (m *Model) GetConfirmingCleanup() string     { return m.advancedModel.GetConfirmingCleanup() }

// Setters for init/zone handlers (delegate to sub-models)
func (m *Model) SetSettingsShowMerged(v bool)        { m.githubModel.SetShowMerged(v) }
func (m *Model) SetSettingsShowClosed(v bool)       { m.githubModel.SetShowClosed(v) }
func (m *Model) SetSettingsOnlyMine(v bool)         { m.githubModel.SetOnlyMine(v) }
func (m *Model) SetSettingsPRLimit(v int)           { m.githubModel.SetPRLimit(v) }
func (m *Model) SetSettingsPRRefreshInterval(v int) { m.githubModel.SetRefreshInterval(v) }
func (m *Model) SetSettingsAutoInProgress(v bool)   { m.ticketsModel.SetAutoInProgress(v) }
func (m *Model) SetSettingsBranchLimit(v int)      { m.branchesModel.SetBranchLimit(v) }
func (m *Model) SetSettingsSanitizeBookmarks(v bool) { m.advancedModel.SetSanitizeBookmarks(v) }
func (m *Model) SetSettingsTicketProvider(s string)  { m.ticketsModel.SetTicketProvider(s) }
func (m *Model) SetConfirmingCleanup(s string)       { m.advancedModel.SetConfirmingCleanup(s) }
