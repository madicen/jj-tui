package settings

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/bubble-color-picker"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/tabs/settings/advanced"
	"github.com/madicen/jj-tui/internal/tui/tabs/settings/ai"
	"github.com/madicen/jj-tui/internal/tui/tabs/settings/branches"
	"github.com/madicen/jj-tui/internal/tui/tabs/settings/codecks"
	"github.com/madicen/jj-tui/internal/tui/tabs/settings/github"
	"github.com/madicen/jj-tui/internal/tui/tabs/settings/jira"
	"github.com/madicen/jj-tui/internal/tui/tabs/settings/theme"
	"github.com/madicen/jj-tui/internal/tui/tabs/settings/tickets"
)

// Model represents the state of the Settings tab (routing-only; state lives in sub-models).
type Model struct {
	// settingsTab selects the visible sub-panel. Indices (UI label):
	// 0 GitHub, 1 Jira, 2 Codecks, 3 Tickets, 4 Branches, 5 Theme, 6 AI, 7 Advanced.
	settingsTab  int
	zoneManager  *zone.Manager
	panelYOffset [8]int // scroll offset per sub-tab; index matches settingsTab order above
	width        int
	height       int
	viewOpts     *ViewOpts

	githubModel   github.Model
	jiraModel     jira.Model
	codecksModel  codecks.Model
	ticketsModel  tickets.Model
	branchesModel branches.Model
	themeModel    theme.Model
	aiModel       ai.Model
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
		aiModel:       ai.NewModel(),
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
		aiModel:       ai.NewModelFromConfig(cfg),
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
	if m.settingsTab == 5 && m.themeModel.AnyOpen() { // Theme
		updated, cmd := m.themeModel.Update(msg)
		m.themeModel = updated
		return m, cmd
	}
	// When Theme tab is active and we get picker result messages, forward to theme model so the swatch closes
	if m.settingsTab == 5 { // Theme
		switch msg.(type) {
		case bubblepicker.ColorChosenMsg, bubblepicker.ColorCanceledMsg:
			updated, cmd := m.themeModel.Update(msg)
			m.themeModel = updated
			return m, cmd
		}
	}
	// When Theme tab is active, forward left mouse press to theme swatch if click is in a swatch zone
	// (bubblezone sends MsgZoneInBounds on release, but the swatch opens on press—so we must handle press here)
	if m.settingsTab == 5 && m.zoneManager != nil { // Theme
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
	if m.settingsTab == 5 && m.themeModel.AnyOpen() { // Theme
		out = m.themeModel.ViewWithOverlay(out, m.width, m.height)
	}
	return out
}

// SetViewOpts sets the options used by View() to render (called by main when entering settings or on resize).
func (m *Model) SetViewOpts(opts ViewOpts) {
	m.viewOpts = &opts
}

// EscHandledInsideSettings is true when Esc should be handled inside Settings (Theme tab / index 5
// color picker, Advanced / index 7 cleanup confirm) instead of closing settings and returning to the graph.
func (m Model) EscHandledInsideSettings() bool {
	if m.advancedModel.GetConfirmingCleanup() != "" {
		return true
	}
	return m.settingsTab == 5 && m.themeModel.AnyOpen() // Theme
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

	// Repository remote shortcuts (Settings → GitHub only). Handled here so they fire from any
	// focusedField on the GitHub panel (including the token input row), and so they don't
	// collide with j/k/space toggle handling further below.
	if m.settingsTab == 0 {
		switch msg.String() {
		case "ctrl+v":
			m.githubModel.ToggleGhPrivate()
			return m, nil
		case "ctrl+x":
			return m, state.NavigateTarget{Kind: state.NavigateRemoteRemove}.Cmd()
		case "ctrl+@", "ctrl+ ": // ctrl+enter on most terminals; fall-through below also handles enter when focused on origin URL.
			return m, state.NavigateTarget{
				Kind:      state.NavigateRemoteApply,
				RemoteURL: strings.TrimSpace(m.githubModel.GetOriginURL()),
			}.Cmd()
		}
		// `g` only fires "Create on GitHub" when the user is NOT typing into a textinput.
		if msg.String() == "g" && m.githubModel.GetFocusedField() != 0 && m.githubModel.GetFocusedField() != 5 {
			return m, state.NavigateTarget{
				Kind:              state.NavigateRemoteCreateGh,
				RemoteRepoPrivate: m.githubModel.GetGhPrivate(),
			}.Cmd()
		}
		// `p` / `P` fire push (current / all) when not typing in an input. Lowercase = current
		// bookmark only (matches `jj git push`'s default scope), uppercase = all bookmarks
		// (matches the auto-push scope after a successful Create).
		if (msg.String() == "p" || msg.String() == "P") && m.githubModel.GetFocusedField() != 0 && m.githubModel.GetFocusedField() != 5 {
			return m, state.NavigateTarget{
				Kind:    state.NavigatePushBookmarks,
				PushAll: msg.String() == "P",
			}.Cmd()
		}
	}

	switch msg.String() {
	case "esc":
		return m, PerformCancelCmd()
	case "ctrl+j":
		tab := m.settingsTab - 1
		if tab < 0 {
			tab = 7
		}
		m.settingsTab = tab % 8
		if m.settingsTab == 6 { // AI
			return m, m.aiModel.SetFocusedField(0)
		}
		if m.settingsTab == 7 { // Advanced
			return m, m.advancedModel.SetFocusedField(0)
		}
		return m, nil
	case "ctrl+k":
		m.settingsTab = (m.settingsTab + 1) % 8
		if m.settingsTab == 6 { // AI
			return m, m.aiModel.SetFocusedField(0)
		}
		if m.settingsTab == 7 { // Advanced
			return m, m.advancedModel.SetFocusedField(0)
		}
		return m, nil
	case "ctrl+s", "enter":
		if m.settingsTab == 6 || m.settingsTab == 7 { // AI or Advanced
			// Forward keys to text inputs
			return m.forwardKeyToActiveSubmodelReturn(msg)
		}
		if msg.String() == "enter" {
			// Apply remote URL when the user presses Enter while typing into the origin URL
			// field. This mirrors the welcome-screen behaviour and is more discoverable than
			// the (terminal-dependent) Ctrl+Enter chord we expose for non-input rows.
			if m.settingsTab == 0 && m.githubModel.GetFocusedField() == 5 {
				return m, state.NavigateTarget{
					Kind:      state.NavigateRemoteApply,
					RemoteURL: strings.TrimSpace(m.githubModel.GetOriginURL()),
				}.Cmd()
			}
			// Advance focus within panel if not on last field; otherwise save
			lastField := false
			switch m.settingsTab {
			case 0: // GitHub
				lastField = m.githubModel.GetFocusedField() >= 0
			case 1: // Jira
				lastField = m.jiraModel.GetFocusedField() >= 7
			case 2: // Codecks
				lastField = m.codecksModel.GetFocusedField() >= 3
			case 3: // Tickets
				lastField = m.ticketsModel.GetTicketProvider() != "github_issues" || m.ticketsModel.GetFocusedField() >= 0
			case 5: // Theme
				lastField = true // Enter saves
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
		if m.settingsTab != 6 && m.settingsTab != 7 { // not AI or Advanced
			m.forwardKeyToActiveSubmodel(msg)
			return m, nil
		}
	case "shift+tab", "up":
		if m.settingsTab != 6 && m.settingsTab != 7 { // not AI or Advanced
			m.forwardKeyToActiveSubmodel(msg)
			return m, nil
		}
	}

	// Forward all other keys (including letters like j/k) to the focused input (Theme tab has no inputs)
	return m.forwardKeyToActiveSubmodelReturn(msg)
}

// ZoneIDs returns the zone IDs this tab uses when rendering (same IDs used in settingstab.Render). Used to resolve clicks.
func (m *Model) ZoneIDs() []string {
	ids := []string{
		mouse.ZoneSettingsTabGitHub, mouse.ZoneSettingsTabJira, mouse.ZoneSettingsTabCodecks,
		mouse.ZoneSettingsTabTickets, mouse.ZoneSettingsTabBranches, mouse.ZoneSettingsTabTheme, mouse.ZoneSettingsTabAI, mouse.ZoneSettingsTabAdvanced,
		mouse.ZoneSettingsThemePrimary, mouse.ZoneSettingsThemeSecondary, mouse.ZoneSettingsThemeMuted,
		mouse.ZoneSettingsThemePrimaryDefault, mouse.ZoneSettingsThemeSecondaryDefault, mouse.ZoneSettingsThemeMutedDefault,
		mouse.ZoneSettingsTicketProviderNone, mouse.ZoneSettingsTicketProviderJira,
		mouse.ZoneSettingsTicketProviderCodecks, mouse.ZoneSettingsTicketProviderGitHubIssues,
		mouse.ZoneSettingsAutoInProgress,
		mouse.ZoneSettingsAdvancedConfirmYes, mouse.ZoneSettingsAdvancedConfirmNo,
		mouse.ZoneSettingsAdvancedDeleteBookmarks, mouse.ZoneSettingsAdvancedAbandonOldCommits,
		mouse.ZoneSettingsGraphRevset, mouse.ZoneSettingsGraphRevsetClear,
		mouse.ZoneSettingsAIEnabled, mouse.ZoneSettingsAIProvider(0), mouse.ZoneSettingsAIProvider(1), mouse.ZoneSettingsAIProvider(2),
		mouse.ZoneSettingsAIBaseURL, mouse.ZoneSettingsAIModel, mouse.ZoneSettingsAIAPIKey,
		mouse.ZoneSettingsAIEvologDescribeDefault, mouse.ZoneSettingsAIEvologFileSplit, mouse.ZoneSettingsAIEvologHunkSplit, mouse.ZoneSettingsAIEvologMultiStepwise,
		mouse.ZoneSettingsAIEvologMultiMaxDecrease, mouse.ZoneSettingsAIEvologMultiMaxIncrease,
		mouse.ZoneSettingsAITimeoutDecrease, mouse.ZoneSettingsAITimeoutIncrease,
		// AI profile management (Settings → AI list + add/delete/cycle + name editor).
		// resolveClickedZone only matches against ids in this list, so missing entries
		// here silently drop clicks (the cause of the "+ new does nothing" bug).
		mouse.ZoneSettingsAIProfileNew, mouse.ZoneSettingsAIProfileDelete,
		mouse.ZoneSettingsAIProfileCyclePrev, mouse.ZoneSettingsAIProfileCycleNext,
		mouse.ZoneSettingsAIProfileName,
	}
	// AI profile rows are indexed; enumerate all currently-visible ones so a
	// click on any row resolves through resolveClickedZone. Use ProfileCount
	// (not Profiles) so this read does not flush input edits.
	for i := 0; i < m.aiModel.ProfileCount(); i++ {
		ids = append(ids, mouse.ZoneSettingsAIProfileRow(i))
	}
	for i := range advanced.ExternalEditorPresetLabels {
		ids = append(ids, mouse.ZoneSettingsExternalEditorPreset(i))
	}
	ids = append(ids,
		mouse.ZoneSettingsExternalEditorCustom,
		mouse.ZoneSettingsSanitizeBookmarks,
		mouse.ZoneSettingsGitHubLogin,
		mouse.ZoneSettingsRemoteOriginInput, mouse.ZoneSettingsRemoteApply,
		mouse.ZoneSettingsRemoteCreateGh, mouse.ZoneSettingsRemoteRemove,
		mouse.ZoneSettingsRemoteVisibilityToggle,
		mouse.ZoneSettingsRemotePushCurrent, mouse.ZoneSettingsRemotePushAll,
		mouse.ZoneSettingsGitHubAuthSaved, mouse.ZoneSettingsGitHubAuthEnv, mouse.ZoneSettingsGitHubAuthGhCLI,
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
	)
	return ids
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

func (m *Model) GetGitHubModel() *github.Model { return &m.githubModel }

// GetGitHubTokenSource returns the selected GitHub API token source (saved | env | gh_cli).
func (m *Model) GetGitHubTokenSource() string      { return m.githubModel.GetTokenSource() }
func (m *Model) GetJiraModel() *jira.Model         { return &m.jiraModel }
func (m *Model) GetCodecksModel() *codecks.Model   { return &m.codecksModel }
func (m *Model) GetTicketsModel() *tickets.Model   { return &m.ticketsModel }
func (m *Model) GetBranchesModel() *branches.Model { return &m.branchesModel }
func (m *Model) GetThemeModel() *theme.Model       { return &m.themeModel }
func (m *Model) GetAIModel() *ai.Model             { return &m.aiModel }
func (m *Model) GetAdvancedModel() *advanced.Model { return &m.advancedModel }

// forwardKeyToActiveSubmodel updates focus/state for tab/up/down within the active panel (mutates m in place).
func (m *Model) forwardKeyToActiveSubmodel(msg tea.KeyMsg) {
	switch m.settingsTab {
	case 0: // GitHub
		gh := m.GetGitHubModel()
		switch msg.String() {
		case "tab", "down", "j":
			if gh.GetFocusedField() < github.MaxFocusedField {
				gh.SetFocusedField(gh.GetFocusedField() + 1)
			}
		case "shift+tab", "up", "k":
			if gh.GetFocusedField() > 0 {
				gh.SetFocusedField(gh.GetFocusedField() - 1)
			}
		}
	case 1: // Jira
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
	case 2: // Codecks
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
	case 3: // Tickets
		tk := m.GetTicketsModel()
		if tk.GetTicketProvider() == "github_issues" {
			switch msg.String() {
			case "tab", "down", "j":
				tk.SetFocusedField(0)
			case "shift+tab", "up", "k":
				tk.SetFocusedField(0)
			}
		}
	case 5: // Theme
		// No fields to focus
	case 6: // AI
		aim := m.GetAIModel()
		switch msg.String() {
		case "tab", "down", "j":
			if aim.GetFocusedField() < 2 {
				aim.SetFocusedField(aim.GetFocusedField() + 1)
			}
		case "shift+tab", "up", "k":
			if aim.GetFocusedField() > 0 {
				aim.SetFocusedField(aim.GetFocusedField() - 1)
			}
		}
	case 7: // Advanced
		adv := m.GetAdvancedModel()
		switch msg.String() {
		case "tab", "down", "j":
			if adv.GetFocusedField() < 1 {
				adv.SetFocusedField(adv.GetFocusedField() + 1)
			}
		case "shift+tab", "up", "k":
			if adv.GetFocusedField() > 0 {
				adv.SetFocusedField(adv.GetFocusedField() - 1)
			}
		}
	}
}

// forwardKeyToActiveSubmodelReturn forwards the key to the active sub-model and returns updated model and cmd.
func (m Model) forwardKeyToActiveSubmodelReturn(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch m.settingsTab {
	case 0: // GitHub
		updated, cmd := m.githubModel.Update(msg)
		m.githubModel = updated
		return m, cmd
	case 1: // Jira
		updated, cmd := m.jiraModel.Update(msg)
		m.jiraModel = updated
		return m, cmd
	case 2: // Codecks
		updated, cmd := m.codecksModel.Update(msg)
		m.codecksModel = updated
		return m, cmd
	case 3: // Tickets
		updated, cmd := m.ticketsModel.Update(msg)
		m.ticketsModel = updated
		return m, cmd
	case 5: // Theme
		// No text inputs
		return m, nil
	case 6: // AI
		updated, cmd := m.aiModel.Update(msg)
		m.aiModel = updated
		return m, cmd
	case 7: // Advanced
		updated, cmd := m.advancedModel.Update(msg)
		m.advancedModel = updated
		return m, cmd
	}
	return m, nil
}

// forwardToActiveSubmodel forwards any message to the active sub-model (e.g. textinput.Blink for cursor).
// Panels with inputs (GitHub, Jira, Codecks, Tickets, AI, Advanced) need to receive these so the cursor blinks.
func (m Model) forwardToActiveSubmodel(msg tea.Msg) (Model, tea.Cmd) {
	switch m.settingsTab {
	case 0: // GitHub
		updated, cmd := m.githubModel.Update(msg)
		m.githubModel = updated
		return m, cmd
	case 1: // Jira
		updated, cmd := m.jiraModel.Update(msg)
		m.jiraModel = updated
		return m, cmd
	case 2: // Codecks
		updated, cmd := m.codecksModel.Update(msg)
		m.codecksModel = updated
		return m, cmd
	case 3: // Tickets
		updated, cmd := m.ticketsModel.Update(msg)
		m.ticketsModel = updated
		return m, cmd
	case 5: // Theme
		// No inputs
		return m, nil
	case 6: // AI
		updated, cmd := m.aiModel.Update(msg)
		m.aiModel = updated
		return m, cmd
	case 7: // Advanced
		updated, cmd := m.advancedModel.Update(msg)
		m.advancedModel = updated
		return m, cmd
	}
	return m, nil
}

// Accessors

// GetActiveSettingsTabIndex returns the selected sub-tab index (see Model.settingsTab).
func (m *Model) GetActiveSettingsTabIndex() int {
	return m.settingsTab
}

// SetActiveSettingsTabIndex sets the visible sub-tab (indices: GitHub, Jira, Codecks, Tickets, Branches, Theme, AI, Advanced).
func (m *Model) SetActiveSettingsTabIndex(tab int) {
	if tab < 0 {
		tab = 0
	}
	m.settingsTab = tab % 8
}

// GetFocusedField returns the focused field’s global input index. Advanced tab uses 14–15 (revset, custom editor); AI tab uses 16–18 (API URL, model, key).
func (m *Model) GetFocusedField() int {
	switch m.settingsTab {
	case 0: // GitHub
		return m.githubModel.GetFocusedField()
	case 1: // Jira
		return 1 + m.jiraModel.GetFocusedField()
	case 2: // Codecks
		return 9 + m.codecksModel.GetFocusedField()
	case 3: // Tickets
		if m.ticketsModel.GetTicketProvider() == "github_issues" {
			return 13
		}
		return 0
	case 4: // Branches
		return 0 // no text inputs
	case 5: // Theme
		return 0 // no inputs
	case 6: // AI
		return 16 + m.aiModel.GetFocusedField() // 16..19 (16=base URL, 17=model, 18=API key, 19=profile name)
	case 7: // Advanced
		return 14 + m.advancedModel.GetFocusedField() // 14..15
	}
	return 0
}

// SetFocusedField sets the focused input field (global index); used by zone handlers to focus an input.
// Returns a tea.Cmd when focusing Advanced or AI text inputs so the cursor is shown.
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
	if idx < 16 {
		return m.advancedModel.SetFocusedField(idx - 14)
	}
	return m.aiModel.SetFocusedField(idx - 16)
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
// Returns a tea.Cmd when focusing AI or Advanced text inputs so the cursor is shown.
func (m *Model) EnterTab() tea.Cmd {
	switch m.settingsTab {
	case 0: // GitHub
		m.githubModel.SetFocusedField(0)
	case 1: // Jira
		m.jiraModel.SetFocusedField(0)
	case 2: // Codecks
		m.codecksModel.SetFocusedField(0)
	case 3: // Tickets
		m.ticketsModel.SetFocusedField(0)
	case 5: // Theme
		// Nothing to focus
	case 6: // AI
		return m.aiModel.SetFocusedField(0)
	case 7: // Advanced
		return m.advancedModel.SetFocusedField(0)
	}
	return nil
}

// GetSettingsInputs returns textinput views for BuildRenderData (built from sub-models).
// Global indices 14–15 are the Advanced tab (revset, custom editor); 16–18 are the AI tab (URL, model, key).
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
	for len(out) < 14 {
		out = append(out, struct{ View string }{""})
	}
	for _, v := range m.advancedModel.GetInputViews() {
		out = append(out, struct{ View string }{v})
	}
	for _, v := range m.aiModel.GetInputViews() {
		out = append(out, struct{ View string }{v})
	}
	for len(out) < 19 {
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
	m.aiModel.SetInputWidth(width)
}

// SetDimensions sets the content area dimensions (used for scroll viewport).
func (m *Model) SetDimensions(width, height int) {
	m.width = width
	m.height = height
}

// GetSettingsYOffset returns the scroll offset for the active sub-tab (indices match Model.settingsTab).
func (m *Model) GetSettingsYOffset() int {
	if m.settingsTab < 0 || m.settingsTab >= 8 {
		return 0
	}
	return m.panelYOffset[m.settingsTab]
}

// SetSettingInputValue sets the value of the settings input at index (e.g. after GitHub login; index 0 = GitHub token).
func (m *Model) SetSettingInputValue(index int, value string) {
	if index == 0 {
		m.githubModel.SetTokenSource(config.GitHubTokenSourceSaved)
		m.githubModel.SetToken(value)
	}
	// Other indices (jira/codecks/tickets) could be set here if needed
}
func (m *Model) UpdateRepository(repo *internal.Repository) {}

// Getters for toggle/state (delegate to sub-models)
func (m *Model) GetSettingsShowMerged() bool        { return m.githubModel.GetShowMerged() }
func (m *Model) GetSettingsShowClosed() bool        { return m.githubModel.GetShowClosed() }
func (m *Model) GetSettingsOnlyMine() bool          { return m.githubModel.GetOnlyMine() }
func (m *Model) GetSettingsPRLimit() int            { return m.githubModel.GetPRLimit() }
func (m *Model) GetSettingsPRRefreshInterval() int  { return m.githubModel.GetRefreshInterval() }
func (m *Model) GetSettingsAutoInProgress() bool    { return m.ticketsModel.GetAutoInProgress() }
func (m *Model) GetSettingsBranchLimit() int        { return m.branchesModel.GetBranchLimit() }
func (m *Model) GetSettingsSanitizeBookmarks() bool { return m.advancedModel.GetSanitizeBookmarks() }
func (m *Model) GetSettingsTicketProvider() string  { return m.ticketsModel.GetTicketProvider() }
func (m *Model) GetConfirmingCleanup() string       { return m.advancedModel.GetConfirmingCleanup() }

// Setters for init/zone handlers (delegate to sub-models)
func (m *Model) SetSettingsShowMerged(v bool)        { m.githubModel.SetShowMerged(v) }
func (m *Model) SetSettingsShowClosed(v bool)        { m.githubModel.SetShowClosed(v) }
func (m *Model) SetSettingsOnlyMine(v bool)          { m.githubModel.SetOnlyMine(v) }
func (m *Model) SetSettingsPRLimit(v int)            { m.githubModel.SetPRLimit(v) }
func (m *Model) SetSettingsPRRefreshInterval(v int)  { m.githubModel.SetRefreshInterval(v) }
func (m *Model) SetSettingsAutoInProgress(v bool)    { m.ticketsModel.SetAutoInProgress(v) }
func (m *Model) SetSettingsBranchLimit(v int)        { m.branchesModel.SetBranchLimit(v) }
func (m *Model) SetSettingsSanitizeBookmarks(v bool) { m.advancedModel.SetSanitizeBookmarks(v) }
func (m *Model) SetSettingsTicketProvider(s string)  { m.ticketsModel.SetTicketProvider(s) }
func (m *Model) SetConfirmingCleanup(s string)       { m.advancedModel.SetConfirmingCleanup(s) }
