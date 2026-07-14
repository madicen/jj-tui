package model

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	bubbledropdown "github.com/madicen/bubble-dropdown"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/tui/data"
	"github.com/madicen/jj-tui/internal/tui/genmenu"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/util"
)

// estimatedContentHeight returns height available for tab content (excluding header/status).
// Used in Update() when delegating to tabs so viewport/list dimensions are correct for scroll handling.
func (m *Model) estimatedContentHeight() int {
	return max(m.height-4, 1)
}

func (m *Model) beginModalUnderlay() {
	m.modalUnderlayView = m.appState.ViewMode
	m.modalUnderlayValid = true
}

func (m *Model) clearModalUnderlay() {
	m.modalUnderlayValid = false
}

// restoreModalUnderlayOrGraph restores the tab from beginModalUnderlay, or the graph if none.
func (m *Model) restoreModalUnderlayOrGraph() {
	if m.modalUnderlayValid {
		m.appState.ViewMode = m.modalUnderlayView
		m.modalUnderlayValid = false
		return
	}
	m.appState.ViewMode = state.ViewCommitGraph
}

func (m *Model) clearAIGenOverlay() {
	m.aiGenOverlayActive = false
}

// clearPendingAIRetry forgets any saved AI replay target. Called on successful generation, on
// error dismissal, and whenever we transition back to the main views without a form modal open.
func (m *Model) clearPendingAIRetry() {
	m.pendingAIRetryActive = false
	m.pendingAIRetryKind = 0
	m.pendingAIRetryOverrideProfile = ""
}

// resolveAIOverride looks up the optional one-shot profile override on t against the
// active config. Returns nil when the override is empty or the profile is not found.
// A missing profile name is treated as "use active" (the menu can only be populated
// from the live profile list, so this is defensive only).
func (m *Model) resolveAIOverride(t state.NavigateTarget) *config.AIProfile {
	name := strings.TrimSpace(t.AIOverrideProfile)
	if name == "" || m.appState.Config == nil {
		return nil
	}
	if p, ok := m.appState.Config.FindAIProfile(name); ok {
		return &p
	}
	return nil
}

// aiGenStatusMessage decorates a default generating-message with the profile name
// when an override is in use so the user can see which model is actually running.
func aiGenStatusMessage(base string, override *config.AIProfile) string {
	if override == nil {
		return base
	}
	return fmt.Sprintf("%s (using %s)", base, override.Name)
}

// pushAIProfilesToFormModals propagates the current config's AI profile list and
// active profile name to every form modal that has a long-press generate menu.
// Called whenever a generate-bearing modal opens and after settings changes
// alter the saved profile list. When cfg is nil we still push an empty list so
// the modals don't carry stale state from a previous repo.
func (m *Model) pushAIProfilesToFormModals() {
	var profiles []config.AIProfile
	active := ""
	if m.appState.Config != nil {
		profiles = m.appState.Config.AIProfileList()
		active = m.appState.Config.ActiveAIProfile().Name
	}
	m.desceditModal.SetAIProfiles(profiles, active)
	m.prFormModal.SetAIProfiles(profiles, active)
	m.bookmarkModal.SetAIProfiles(profiles, active)
	m.ticketFormModal.SetAIProfiles(profiles, active)
}

// buildSettingsViewOpts builds ViewOpts for the settings tab (used when entering settings or on resize).
func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(autoRefreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// isGitHubAvailable returns true if GitHub functionality is available (real service or demo mode).
func (m *Model) isGitHubAvailable() bool {
	return m.appState.GitHubService != nil || m.appState.DemoMode
}

// isSelectedCommitValid returns true if selected commit index points to a valid commit.
func (m *Model) isSelectedCommitValid() bool {
	return m.appState.Repository != nil &&
		m.GetSelectedCommit() >= 0 &&
		m.GetSelectedCommit() < len(m.appState.Repository.Graph.Commits)
}

// applyRepositoryLoaded applies a loaded repository from data or actions package (shared logic).
func (m *Model) bookmarksNeedingPRLookup() []string {
	if m.appState.Repository == nil {
		return nil
	}
	openPRBranches := make(map[string]bool)
	for _, pr := range m.appState.Repository.PRs {
		if pr.State == "open" {
			openPRBranches[pr.HeadBranch] = true
		}
	}
	const maxLookups = 25
	seen := make(map[string]bool)
	var names []string
	for _, commit := range m.appState.Repository.Graph.Commits {
		for _, branch := range commit.Branches {
			local := util.LocalBookmarkName(branch)
			if local == "" || seen[local] {
				continue
			}
			if m.appState.DefaultBranch != "" && local == m.appState.DefaultBranch {
				continue
			}
			if local == "main" || local == "master" || local == "trunk" {
				continue
			}
			if openPRBranches[local] || openPRBranches[branch] {
				continue
			}
			seen[local] = true
			names = append(names, local)
			if len(names) >= maxLookups {
				return names
			}
		}
	}
	return names
}

// refreshRepository starts a refresh of the repository data.
func (m *Model) createIsZoneClickedFuncWithEvent(event tea.MouseMsg) func(string) bool {
	return func(zoneID string) bool {
		z := m.zoneManager.Get(zoneID)
		return z != nil && z.InBounds(event)
	}
}

// --- Handlers: main routes to tabs; tabs own context (BuildRequestContextFrom) and execution (ExecuteRequest / EnterTab). ---

// processGraphRequest runs a graph request via the graph tab; ApplyResult mutates app and returns cmd.
func (m *Model) handleNavigateToGraphTab() (tea.Model, tea.Cmd) {
	m.appState.ViewMode = state.ViewCommitGraph
	m.appState.StatusMessage = "Loading commit graph"
	return m, m.refreshRepository()
}

func (m *Model) handleNavigateToSettingsTab() (tea.Model, tea.Cmd) {
	m.appState.ViewMode = state.ViewSettings
	m.settingsTabModel.SetViewOpts(m.buildSettingsViewOpts())
	m.refreshSettingsOriginURL()
	return m, m.settingsTabModel.EnterTab()
}

// refreshSettingsOriginURL queries the jj service for the current `origin` URL and caches it on
// the GitHub settings sub-model so renderRepositoryRemote can display "Current origin: …" /
// "(none configured)". Called on Settings open and after every successful Apply / Create /
// Remove operation so the cached value reflects reality without a refresh shortcut.
func (m *Model) refreshSettingsOriginURL() {
	gh := m.settingsTabModel.GetGitHubModel()
	if m.appState.JJService == nil {
		gh.SetCurrentOrigin("")
		return
	}
	url, err := m.appState.JJService.GetGitRemoteURL(context.Background())
	if err != nil || strings.TrimSpace(url) == "" {
		gh.SetCurrentOrigin("")
		return
	}
	gh.SetCurrentOrigin(strings.TrimSpace(url))
	// Pre-fill the input with the existing URL so users see what's there and can edit/replace it
	// rather than retyping from scratch. Empty input is left empty.
	if gh.GetOriginURL() == "" {
		gh.SetOriginURL(strings.TrimSpace(url))
	}
}

func (m *Model) handleNavigateToHelpTab() (tea.Model, tea.Cmd) {
	m.appState.ViewMode = state.ViewHelp
	m.refreshHelpCommandHistory()
	m.helpTabModel.SetSelectedCommand(0)
	m.appState.StatusMessage = "Loaded Help"
	return m, nil
}

func (m *Model) handleClipboardCopiedMsg(msg util.ClipboardCopiedMsg) (tea.Model, tea.Cmd) {
	if msg.Success {
		if m.appState.ViewMode == state.ViewGitHubLogin {
			m.appState.StatusMessage = "Code copied to clipboard! Paste it in your browser."
		} else if m.errorModal.GetError() != nil {
			m.errorModal.SetCopied(true)
			m.appState.StatusMessage = "Error copied to clipboard!"
		} else {
			m.appState.StatusMessage = "Copied to clipboard!"
		}
	} else {
		m.appState.StatusMessage = fmt.Sprintf("Failed to copy: %v", msg.Err)
	}
	return m, nil
}

// SetRepository sets the repository data and syncs to tab models (e.g. for tests)
func (m *Model) SetRepository(repo *internal.Repository) {
	m.appState.UpdateRepository(repo)
	m.propagateRepository()
	m.prsTabModel.SetGithubService(m.isGitHubAvailable())
}

// Init implements tea.Model
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		data.InitializeServices(m.appState.DemoMode),
		m.tickCmd(),
	)
}

// Update implements tea.Model.
// Message responsibility: see internal/tui/model/RESPONSIBILITY.md.
// Flow: globals (SetStatus, WindowSize) → state.NavigateMsg (from submodels) →
// modal request messages (descedit/bookmark/prform/warning forward to modals) →
// async result messages (data.*, graphtab.*, prstab.*, etc.) → zone/key routing.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Lock the spinner label across the rest of this Update invocation. Submodels
	// throughout the tree set m.appState.Loading directly and write progress text to
	// StatusMessage, but StatusMessage is *also* the footer; any later footer update
	// (background PR polls, "Loaded N tickets", "Ready") would otherwise overwrite the
	// spinner's caption mid-operation. By snapshotting at the false→true edge here we
	// keep the spinner text stable for the entire duration of the loading operation
	// without having to plumb a separate setter through every call site.
	defer m.snapshotSpinnerMessage()

	switch msg := msg.(type) {
	case SetStatusMsg:
		m.appState.StatusMessage = msg.Status
		return m, nil

	case spinner.TickMsg:
		if !m.appState.Loading && !m.aiGenOverlayActive {
			return m, nil
		}
		var spinCmd tea.Cmd
		m.busySpinner, spinCmd = m.busySpinner.Update(msg)
		return m, spinCmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.errorModal.SetWidth(m.width)
		m.errorModal.SetHeight(m.height)

		inputWidth := ModalInnerWidth(m.width)

		m.desceditModal.SetDimensions(inputWidth, max(m.height-24, 3))
		m.prFormModal.GetBodyInput().SetWidth(inputWidth)
		m.prFormModal.GetTitleInput().Width = inputWidth
		m.ticketFormModal.GetBodyInput().SetWidth(inputWidth)
		m.ticketFormModal.GetTitleInput().Width = inputWidth
		// PR form body uses full content height when in create-PR view
		contentHeight := m.estimatedContentHeight()
		if m.appState.ViewMode == state.ViewCreatePR {
			const fixedFormLines = 11
			bodyH := contentHeight - fixedFormLines
			if bodyH < 3 {
				bodyH = 3
			}
			m.prFormModal.GetBodyInput().SetHeight(bodyH)
		}
		if m.appState.ViewMode == state.ViewCreateTicket {
			const fixedFormLines = 12
			bodyH := contentHeight - fixedFormLines
			if bodyH < 3 {
				bodyH = 3
			}
			m.ticketFormModal.GetBodyInput().SetHeight(bodyH)
		}
		m.bookmarkModal.GetNameInput().Width = inputWidth
		m.bookmarkModal.SetContentWidth(inputWidth)

		m.settingsTabModel.SetInputWidths(min(max(m.width-24, 36), 76))

		if m.appState.ViewMode == state.ViewSettings {
			m.settingsTabModel.SetViewOpts(m.buildSettingsViewOpts())
		}

		// Propagate dimensions to tab models so they can render
		cmds := util.PropagateUpdate(msg, &m.graphTabModel, &m.prsTabModel, &m.branchesTabModel, &m.ticketsTabModel, &m.settingsTabModel, &m.helpTabModel)
		// Set content-area height on tabs so graph/files split fills the content area (not full window)
		for _, vm := range m.tabOrder {
			m.tabRegistry[vm].SetDimensions(m.width, contentHeight)
		}
		m.evologSplitModal = m.evologSplitModal.SetDimensions(m.width, m.height).WithSuggestConfig(m.appState.Config)
		m.fileDiffModal = m.fileDiffModal.SetDimensions(m.width, m.height)
		m.divergentModal = m.divergentModal.SetDimensions(m.width, m.height)
		m.conflictModal = m.conflictModal.SetDimensions(m.width, m.height)
		m.workspacesModal = m.workspacesModal.SetDimensions(m.width, m.height)
		m.operationsModal = m.operationsModal.SetDimensions(m.width, m.height)
		if len(cmds) > 0 {
			return m, tea.Batch(cmds...)
		}
		return m, nil

	case tea.KeyMsg:
		// Window chrome keyboard nudge (Alt+arrow to move, Alt+Shift+arrow
		// to resize) is consumed before any modal/tab handling so the
		// keystroke can never collide with a textinput's own bindings —
		// HandleChromeKey only matches the alt/alt-shift arrow combos and
		// returns Consumed=false for everything else.
		key, content, title, closeCmd := m.chromedSlot()
		if consumed, cmd := m.chrome.Update(msg, content, title, key, m.width, m.height, closeCmd); consumed {
			return m, cmd
		}
		if m.evologDescribePreviewActive {
			switch msg.String() {
			case "y", "Y":
				pd, cd := m.evologDescribeParent, m.evologDescribeChild
				m.evologDescribePreviewActive = false
				m.evologDescribePreviewFromPlan = false
				skipP := m.evologDescribeSkipParent
				m.evologDescribeSkipParent = false
				m.evologDescribeParent, m.evologDescribeChild = "", ""
				m.appState.Loading = true
				m.appState.StatusMessage = "Applying descriptions…"
				return m, tea.Batch(
					m.applyEvologSplitDescriptionsCmd(pd, cd, skipP),
					m.startBusySpinnerCmd(),
				)
			case "n", "N", "esc":
				m.evologDescribePreviewActive = false
				m.evologDescribePreviewFromPlan = false
				m.evologDescribeSkipParent = false
				m.evologDescribeParent, m.evologDescribeChild = "", ""
				m.appState.StatusMessage = "Descriptions preview dismissed"
				return m, nil
			default:
				return m, nil
			}
		}
		if m.absorbPreviewActive {
			switch msg.String() {
			case "y", "Y":
				m.absorbPreviewActive = false
				m.absorbPreviewSummary = ""
				m.redoOperationID = ""
				m.pendingUndoHint = true
				m.appState.Loading = true
				m.appState.StatusMessage = "Absorbing…"
				return m, tea.Batch(
					m.absorbApplyCmd(),
					m.startBusySpinnerCmd(),
				)
			case "n", "N", "esc":
				m.absorbPreviewActive = false
				m.absorbPreviewSummary = ""
				m.appState.StatusMessage = "Absorb cancelled"
				return m, nil
			default:
				return m, nil
			}
		}
		// When an overlay or blocking modal is showing, route keys to handleKeyMsg (init, error, warning) or view modals.
		if m.initRepoModel.Path() != "" || m.errorModal.GetError() != nil || m.warningModal.IsShown() {
			return m.handleKeyMsg(msg)
		}
		// View-specific modals (divergent, bookmark conflict): route keys to handleKeyMsg so the modal gets them.
		if m.appState.ViewMode == state.ViewDivergentCommit || m.appState.ViewMode == state.ViewBookmarkConflict || m.appState.ViewMode == state.ViewEvologSplit || m.appState.ViewMode == state.ViewFileDiff || m.appState.ViewMode == state.ViewWorkspaces || m.appState.ViewMode == state.ViewOperations {
			return m.handleKeyMsg(msg)
		}
		// Esc in Settings: close in-tab overlays (theme picker, cleanup confirm) first; otherwise leave settings.
		if m.appState.ViewMode == state.ViewSettings && msg.String() == "esc" {
			if !m.escHandledInsideSettings() {
				return m.handleNavigate(state.NavigateTarget{Kind: state.NavigateBackToGraph, StatusMessage: "Settings cancelled"})
			}
		}
		// Delegate to the active tab behind the registry (tabs own selection
		// state). The adapters apply the per-tab command wrapper the inline
		// paths used (graph → wrapGraphTabCmd; prs/branches → wrapSpinnerStart)
		// so the returned cmd is already wrapped. Per-view control flow (Settings
		// always consumes; Help swallows tab/shift+tab; Tickets swallows the
		// esc that closed status-change mode) is preserved below; every other
		// view falls through to handleKeyMsg when the tab didn't consume the key.
		if t, ok := m.tabRegistry[m.appState.ViewMode]; ok {
			switch m.appState.ViewMode {
			case state.ViewTickets:
				wasStatusChange := m.isTicketsStatusChangeMode()
				if _, cmd := t.Update(msg, &m.appState); cmd != nil {
					return m, cmd
				}
				if msg.String() == "esc" && wasStatusChange && !m.isTicketsStatusChangeMode() {
					return m, nil
				}
			case state.ViewSettings:
				_, cmd := t.Update(msg, &m.appState)
				if cmd != nil {
					return m, cmd
				}
				return m, nil
			case state.ViewHelp:
				if _, cmd := t.Update(msg, &m.appState); cmd != nil {
					return m, cmd
				}
				// Tab/shift+tab switch help sub-tab; don't fall through to handleKeyMsg (which would switch to graph)
				if msg.String() == "tab" || msg.String() == "shift+tab" {
					return m, nil
				}
			default: // graph, prs, branches
				if _, cmd := t.Update(msg, &m.appState); cmd != nil {
					return m, cmd
				}
			}
		}
		return m.handleKeyMsg(msg)

	case tea.MouseMsg:
		// Window chrome (title-bar drag, [x] close, edge resize) gets first
		// look so a drag started on the tab keeps consuming subsequent
		// motion / release events even if they cross over an underlying
		// zone. We only swallow the event here when the chrome actually
		// engaged — otherwise (Consumed=false, Pop=false) we fall through
		// so the rest of the modal still receives normal clicks. Pop fires
		// when the user clicked [x]; chromedSlot supplies the modal-specific
		// Cancel / Dismiss navigation so close-via-tab and close-via-Esc go
		// through the same teardown path.
		key, content, title, closeCmd := m.chromedSlot()
		if consumed, cmd := m.chrome.Update(msg, content, title, key, m.width, m.height, closeCmd); consumed {
			// Track press/release pairing so a [x] close (which only consumes the
			// press) doesn't leak its release into underlay zones — see the
			// chromeConsumedPress field comment.
			switch msg.Action {
			case tea.MouseActionPress:
				m.chromeConsumedPress = true
			case tea.MouseActionRelease:
				m.chromeConsumedPress = false
			}
			return m, cmd
		}
		// Chrome did NOT consume this event. If it's the release half of a click
		// whose press chrome already claimed (typically the [x] close button),
		// drop it here so bubblezone doesn't dispatch it to whatever underlay
		// zone (e.g. the graph "split" button) happens to sit beneath the modal.
		if msg.Action == tea.MouseActionRelease && m.chromeConsumedPress {
			m.chromeConsumedPress = false
			return m, nil
		}
		// Minimized-chrome pass-through: chromed modal is collapsed to its
		// tab strip and the user clicked outside that strip — they're asking
		// to interact with the underlying view, not the dormant modal. Send
		// the raw MouseMsg to the underlay tab so wheel scrolls and hover
		// updates land, then on release also fan out to bubblezone so
		// underlay zones (commits / PR rows / etc) fire. The form-modal /
		// genmenu interceptions below are skipped because the modal body
		// isn't painted right now — its zones would either be stale or
		// missing from the latest Scan.
		if m.mouseTransparent(msg.X, msg.Y) {
			cmd, _ := m.routeMouseToUnderlay(msg)
			if msg.Action == tea.MouseActionRelease {
				_, zoneCmd := m.zoneManager.AnyInBoundsAndUpdate(m, msg)
				return m, tea.Batch(cmd, zoneCmd)
			}
			return m, cmd
		}
		// Generate-bearing form modals: forward the raw MouseMsg first so the
		// long-press AI profile picker can arm/cancel/hover/resolve. The modal
		// returns a non-nil cmd only when it actually captured the event (the
		// BeginPress tick, a menu-row release that fires NavigateGenerate*, etc.).
		if genState := m.activeFormModalGenMenu(); genState != nil {
			wasShown := genState.IsShown()
			if cmd := m.forwardMouseToActiveFormModal(msg); cmd != nil {
				return m, cmd
			}
			// When the popover was visible at the start of this event and is now
			// closed (a release that landed off the menu), suppress the normal
			// release-zone dispatch so the underlying generate-chip click does
			// not also fire after the user dismissed the popover.
			if wasShown && !genState.IsShown() && msg.Action == tea.MouseActionRelease {
				return m, nil
			}
			// When the popover is currently shown, swallow non-release mouse
			// events that didn't produce a cmd so they don't leak to the tab
			// layer (hover updates already happened inside the modal Update).
			if genState.IsShown() && msg.Action != tea.MouseActionRelease {
				return m, nil
			}
		}
		// Blocking overlays and modal views: run zone check on release first so clicks reach the modal, not the tab.
		if msg.Action == tea.MouseActionRelease &&
			(m.initRepoModel.Path() != "" || m.errorModal.GetError() != nil || m.warningModal.IsShown() ||
				m.appState.ViewMode == state.ViewCreatePR || m.appState.ViewMode == state.ViewCreateTicket || m.appState.ViewMode == state.ViewEditDescription || m.appState.ViewMode == state.ViewCreateBookmark || m.appState.ViewMode == state.ViewDivergentCommit || m.appState.ViewMode == state.ViewBookmarkConflict || m.appState.ViewMode == state.ViewEvologSplit || m.appState.ViewMode == state.ViewFileDiff) {
			return m.zoneManager.AnyInBoundsAndUpdate(m, msg)
		}
		// Handle wheel: IsWheel() covers standard encodings; also accept raw X11 4/5
		isWheel := tea.MouseEvent(msg).IsWheel() || msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown
		if isWheel {
			if m.appState.ViewMode == state.ViewEvologSplit {
				updated, cmd := m.evologSplitModal.Update(msg)
				m.evologSplitModal = updated
				return m, cmd
			}
			if m.appState.ViewMode == state.ViewFileDiff {
				updated, cmd := m.fileDiffModal.Update(msg)
				m.fileDiffModal = updated
				return m, cmd
			}
			// Wheel over a primary tab: every view sized its tab before
			// delegating, so do that generically through the registry (the
			// adapters apply the same command wrappers the inline paths used).
			contentHeight := m.estimatedContentHeight()
			if t, ok := m.tabRegistry[m.appState.ViewMode]; ok {
				t.SetDimensions(m.width, contentHeight)
				if _, cmd := t.Update(msg, &m.appState); cmd != nil {
					return m, cmd
				}
			}
			return m, nil
		}
		// Delegate other mouse to active tab (same as KeyMsg) for any other scroll/click handling.
		// The list tabs (prs/branches/tickets) size themselves first so scroll works even when the
		// wheel encoding wasn't recognized above; graph/settings/help delegate without a resize,
		// matching the pre-registry behavior exactly.
		contentHeight := m.estimatedContentHeight()
		if t, ok := m.tabRegistry[m.appState.ViewMode]; ok {
			switch m.appState.ViewMode {
			case state.ViewPullRequests, state.ViewBranches, state.ViewTickets:
				t.SetDimensions(m.width, contentHeight)
			}
			if _, cmd := t.Update(msg, &m.appState); cmd != nil {
				return m, cmd
			}
		}
		if msg.Action == tea.MouseActionRelease {
			return m.zoneManager.AnyInBoundsAndUpdate(m, msg)
		}
		return m, nil

	case genmenu.TickMsg:
		// Long-press tick for the AI profile picker on the active form modal.
		// The modal gates this internally (Owner + PressID checks) so we can
		// route it unconditionally; stale ticks become no-ops.
		if cmd := m.forwardGenMenuTickToActiveFormModal(msg); cmd != nil {
			return m, cmd
		}
		return m, nil

	case zone.MsgZoneInBounds:
		// Minimized-chrome pass-through: route the zone directly to the
		// underlay tab so commits / PR rows / branches / tickets clicks
		// reach the view the user can actually see behind the collapsed
		// modal. Without this branch the existing routing keys off
		// m.appState.ViewMode and forwards every zone to the dormant form
		// modal, which silently drops it.
		if m.mouseTransparent(msg.Event.X, msg.Event.Y) {
			if cmd, handled := m.routeMouseToUnderlay(msg); handled {
				return m, cmd
			}
			return m, nil
		}
		// Blocking overlays (init, error, warning) get zone clicks first so tabs don't consume them
		if m.initRepoModel.Path() != "" || m.errorModal.GetError() != nil || m.warningModal.IsShown() {
			return m.handleZoneClick(msg)
		}
		// View modals (divergent, conflict) get zone clicks so they're not consumed by the tab
		if m.appState.ViewMode == state.ViewDivergentCommit {
			updated, cmd := m.divergentModal.Update(msg)
			m.divergentModal = updated
			return m, cmd
		}
		if m.appState.ViewMode == state.ViewBookmarkConflict {
			updated, cmd := m.conflictModal.Update(msg)
			m.conflictModal = updated
			return m, cmd
		}
		if m.appState.ViewMode == state.ViewEvologSplit {
			updated, cmd := m.evologSplitModal.Update(msg)
			m.evologSplitModal = updated
			return m, cmd
		}
		if m.appState.ViewMode == state.ViewFileDiff {
			updated, cmd := m.fileDiffModal.Update(msg)
			m.fileDiffModal = updated
			return m, cmd
		}
		// Delegate to the active content tab (graph/prs/branches/tickets) so it
		// can return requests; settings/help are handled inside handleZoneClick.
		// Graph, PRs, Branches, and Tickets already receive zone.MsgZoneInBounds
		// here before handleZoneClick runs, so handleZoneClick must not re-Update
		// them (double-processing the release).
		switch m.appState.ViewMode {
		case state.ViewCommitGraph, state.ViewPullRequests, state.ViewBranches, state.ViewTickets:
			if t, ok := m.tabRegistry[m.appState.ViewMode]; ok {
				if _, cmd := t.Update(msg, &m.appState); cmd != nil {
					return m, cmd
				}
			}
		}
		return m.handleZoneClick(msg)

	case bubbledropdown.ItemChosenMsg, bubbledropdown.ItemCanceledMsg:
		// A settings dropdown emitted its selection/cancel via a deferred cmd.
		// Route it to the settings tab so the active sub-model applies and closes.
		if m.appState.ViewMode == state.ViewSettings {
			updated, cmd := m.settingsTabModel.Update(msg)
			m.settingsTabModel = updated
			return m, cmd
		}
		return m, nil

	case state.NavigateMsg:
		return m.handleNavigate(msg.Target)

	default:
		return m.dispatchAsyncMsg(msg)
	}
}

// isStaleFileDiffGlobalStatus reports status strings tied to the file-diff overlay that should not
// linger on the status line (or a concurrent loading overlay) after the modal closes.
func isStaleFileDiffGlobalStatus(msg string) bool {
	s := strings.TrimSpace(msg)
	if s == "File diff failed" {
		return true
	}
	return strings.HasPrefix(s, "File diff —")
}
