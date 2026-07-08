package model

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	bubblepicker "github.com/madicen/bubble-color-picker"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	aitab "github.com/madicen/jj-tui/internal/tui/ai"
	"github.com/madicen/jj-tui/internal/tui/data"
	"github.com/madicen/jj-tui/internal/tui/state"
	bookmarktab "github.com/madicen/jj-tui/internal/tui/tabs/bookmark"
	branchestab "github.com/madicen/jj-tui/internal/tui/tabs/branches"
	conflicttab "github.com/madicen/jj-tui/internal/tui/tabs/conflict"
	descedittab "github.com/madicen/jj-tui/internal/tui/tabs/descedit"
	divergenttab "github.com/madicen/jj-tui/internal/tui/tabs/divergent"
	errortab "github.com/madicen/jj-tui/internal/tui/tabs/error"
	evologsplittab "github.com/madicen/jj-tui/internal/tui/tabs/evologsplit"
	filedifftab "github.com/madicen/jj-tui/internal/tui/tabs/filediff"
	githublogintab "github.com/madicen/jj-tui/internal/tui/tabs/githublogin"
	graphtab "github.com/madicen/jj-tui/internal/tui/tabs/graph"
	"github.com/madicen/jj-tui/internal/tui/tabs/help/commandhistory"
	initrepotab "github.com/madicen/jj-tui/internal/tui/tabs/initrepo"
	operationstab "github.com/madicen/jj-tui/internal/tui/tabs/operations"
	prformtab "github.com/madicen/jj-tui/internal/tui/tabs/prform"
	prstab "github.com/madicen/jj-tui/internal/tui/tabs/prs"
	settingstab "github.com/madicen/jj-tui/internal/tui/tabs/settings"
	ticketformtab "github.com/madicen/jj-tui/internal/tui/tabs/ticketform"
	ticketstab "github.com/madicen/jj-tui/internal/tui/tabs/tickets"
	warningtab "github.com/madicen/jj-tui/internal/tui/tabs/warning"
	workspacestab "github.com/madicen/jj-tui/internal/tui/tabs/workspaces"
	"github.com/madicen/jj-tui/internal/tui/util"
)

// dispatchAsyncMsg handles all asynchronous result / effect messages emitted by
// the tab packages and background services. It is the `default` arm of the
// Update type-switch: any message not consumed by the top-level input/window
// cases in model.go falls through to here. Housing these concrete tab-package
// message cases in this sibling file keeps model.go's own imports of the tab
// packages construction-time-only.
func (m *Model) dispatchAsyncMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case aitab.TextGeneratedMsg:
		if msg.ReqID != m.aiGenReqID {
			return m, nil
		}
		m.clearAIGenOverlay()
		if msg.Err != nil {
			var label string
			switch msg.Kind {
			case aitab.KindCommitDescription:
				label = "Commit description (AI)"
			case aitab.KindPR:
				label = "Pull request (AI)"
			case aitab.KindBookmark:
				label = "Bookmark name (AI)"
			case aitab.KindTicket:
				label = "Create ticket (AI)"
			default:
				label = "AI"
			}
			nm, cmd := m.Update(errorMsg{Err: fmt.Errorf("%s: %w", label, msg.Err)})
			// errorMsg path resets hasRetry to false; turn it back on so the user sees the
			// Retry button. The pending replay target is whatever NavigateGenerate* last set
			// (still valid here because we only got here from one of those code paths).
			m.errorModal.SetHasRetry(m.pendingAIRetryActive)
			return nm, cmd
		}
		// Success: forget the saved retry target so a later non-AI failure doesn't accidentally
		// offer Retry that replays a stale generation.
		m.clearPendingAIRetry()
		switch msg.Kind {
		case aitab.KindCommitDescription:
			if m.appState.ViewMode != state.ViewEditDescription || m.desceditModal.GetEditingCommitID() != msg.CommitID {
				return m, nil
			}
			cur := strings.TrimSpace(m.desceditModal.GetDescriptionValue())
			next := strings.TrimSpace(msg.Text)
			if next == "" {
				return m, nil
			}
			if cur == "" {
				m.desceditModal.SetDescription(next)
			} else {
				m.desceditModal.SetDescription(cur + "\n\n" + next)
			}
			m.appState.StatusMessage = "Description generated (review, then save)"
		case aitab.KindPR:
			if m.appState.ViewMode != state.ViewCreatePR {
				return m, nil
			}
			if t := strings.TrimSpace(msg.Title); t != "" {
				m.prFormModal.SetTitle(t)
			}
			if b := strings.TrimSpace(msg.Body); b != "" {
				m.prFormModal.SetBody(b)
			}
			m.appState.StatusMessage = "PR fields generated (review, then create)"
		case aitab.KindBookmark:
			if m.appState.ViewMode != state.ViewCreateBookmark {
				return m, nil
			}
			name := strings.TrimSpace(msg.Text)
			if m.appState.Config != nil && m.appState.Config.ShouldSanitizeBookmarkNames() {
				name = jj.SanitizeBookmarkName(name)
			}
			name = jj.TruncateBookmarkName(name)
			m.bookmarkModal.SetBookmarkName(name)
			m.bookmarkModal.UpdateNameExistsFromInput(m.appState.Config != nil && m.appState.Config.ShouldSanitizeBookmarkNames())
			m.appState.StatusMessage = "Bookmark name suggested (edit if needed)"
		case aitab.KindTicket:
			if m.appState.ViewMode != state.ViewCreateTicket {
				return m, nil
			}
			if t := strings.TrimSpace(msg.Title); t != "" {
				m.ticketFormModal.SetSummary(t)
			}
			if b := strings.TrimSpace(msg.Body); b != "" {
				m.ticketFormModal.SetDescription(b)
			}
			m.appState.StatusMessage = "Ticket fields generated from graph revision (review, then create)"
		}
		return m, nil

	case commandhistory.Request:
		return m.handleHelpRequest(msg)

	case settingstab.Request:
		return m.handleSettingsRequest(msg)
	case settingstab.SaveSettingsEffect:
		return m, m.saveSettings()
	case settingstab.SaveSettingsLocalEffect:
		return m, m.saveSettingsLocal()
	case settingstab.PerformCancelMsg:
		return m.handleNavigate(state.NavigateTarget{Kind: state.NavigateBackToGraph, StatusMessage: "Settings cancelled"})

	case ticketstab.OpenURLEffect:
		return m, util.OpenURL(msg.URL)
	case ticketstab.ToggleModeEffect:
		mode := !m.ticketsTabModel.IsStatusChangeMode()
		m.ticketsTabModel.SetStatusChangeMode(mode)
		m.appState.StatusMessage = msg.Status
		return m, nil
	case ticketstab.OpenCreateBookmarkFromTicketEffect:
		return m.handleNavigate(state.NavigateTarget{
			Kind:             state.NavigateCreateBookmarkFromTicket,
			TicketKey:        msg.TicketKey,
			TicketTitle:      msg.Title,
			TicketDisplayKey: msg.DisplayKey,
		})

	case descedittab.SaveRequestedMsg, descedittab.CancelRequestedMsg:
		updated, cmd := m.desceditModal.Update(msg)
		m.desceditModal = updated
		return m, cmd

	case bookmarktab.CancelRequestedMsg, bookmarktab.SubmitRequestedMsg:
		updated, cmd := m.bookmarkModal.Update(msg)
		m.bookmarkModal = updated
		m.bookmarkModal.UpdateNameExistsFromInput(m.appState.Config != nil && m.appState.Config.ShouldSanitizeBookmarkNames())
		return m, cmd

	case prformtab.CancelRequestedMsg, prformtab.SubmitRequestedMsg:
		updated, cmd := m.prFormModal.Update(msg)
		m.prFormModal = updated
		return m, cmd

	case ticketformtab.CancelRequestedMsg, ticketformtab.SubmitRequestedMsg:
		updated, cmd := m.ticketFormModal.Update(msg)
		m.ticketFormModal = updated
		return m, cmd

	case settingstab.RequestConfirmCleanupMsg:
		return m, m.confirmCleanup()
	case settingstab.RequestCancelCleanupMsg:
		m.appState.StatusMessage = settingstab.CancelCleanupStatus
		return m, nil
	case settingstab.RequestSetStatusMsg:
		m.appState.StatusMessage = msg.Status
		return m, nil

	case errortab.RequestCopyMsg:
		if m.errorModal.GetError() != nil {
			m.errorModal.SetCopied(true)
			return m, util.CopyToClipboard(m.errorModal.GetError().Error())
		}
		return m, nil

	case warningtab.EditCommitRequestedMsg:
		updated, cmd := m.warningModal.Update(msg)
		m.warningModal = updated
		return m, cmd

	case graphtab.EditCompletedMsg:
		// Preserve PRs from previous repository
		var oldPRs []internal.GitHubPR
		if m.appState.Repository != nil {
			oldPRs = m.appState.Repository.PRs
		}
		m.appState.UpdateRepository(msg.Repository)
		m.appState.Repository.PRs = oldPRs // Restore PRs temporarily
		// Push fresh graph into tab models before clearing loading so the overlay stays up until
		// the UI can render the new @ / tree (appState alone does not update GraphModel).
		m.propagateRepository()
		m.prsTabModel.SetGithubService(m.isGitHubAvailable())
		// Don't clear error modal here - let errors persist until dismissed
		var workingChangeID string
		for i, commit := range msg.Repository.Graph.Commits {
			if commit.IsWorking {
				m.graphTabModel.SelectCommit(i)
				workingChangeID = commit.ChangeID
				break
			}
		}
		m.appState.Loading = false
		m.appState.StatusMessage = "Now editing working copy"

		var cmds []tea.Cmd
		cmds = append(cmds, m.tickCmd())
		if workingChangeID != "" && m.appState.JJService != nil {
			cmds = append(cmds, graphtab.LoadChangedFilesCmd(m.appState.JJService, workingChangeID))
		}

		// Also refresh PRs when GitHub is connected (needed for Update PR button)
		if m.appState.GitHubService != nil {
			existingPRs := 0
			if m.appState.Repository != nil {
				existingPRs = len(m.appState.Repository.PRs)
			}
			cmds = append(cmds, m.wrapFirstPRLoadCmd(prstab.LoadPRsCmd(m.appState.GitHubService, m.appState.GithubInfo, m.appState.DemoMode, existingPRs)))
		}

		return m, tea.Batch(cmds...)

	case errorMsg:
		m.evologDescribePreviewActive = false
		m.evologDescribePreviewFromPlan = false
		m.evologDescribeSkipParent = false
		m.evologDescribeParent = ""
		m.evologDescribeChild = ""
		if m.appState.ViewMode == state.ViewEvologSplit || m.appState.ViewMode == state.ViewFileDiff || m.appState.ViewMode == state.ViewEditDescription {
			m.appState.Loading = false
		}
		cmd, info := errortab.HandleError(errortab.ErrorInput{NotJJRepo: msg.NotJJRepo, CurrentPath: msg.CurrentPath, Err: msg.Err}, &m.appState)
		if info != nil {
			if info.NotJJRepo {
				m.initRepoModel.SetPath(info.CurrentPath)
			} else {
				m.applyEffects(effShowError{info.Err})
			}
		}
		return m, cmd
	case data.InitErrorMsg:
		// Soft-failure path: jj init succeeded but a follow-up step (gh repo create / git remote
		// add) failed. The directory is now a valid jj repo, so dismiss the init screen and load
		// services as if init had fully succeeded. The follow-up error is still surfaced via the
		// error modal so the user knows to set up the remote manually.
		if msg.JJInitialized {
			m.initRepoModel.SetPath("")
			m.applyEffects(effShowError{msg.Err})
			m.appState.StatusMessage = "Repository initialized; remote setup failed"
			return m, data.InitializeServices(m.appState.DemoMode)
		}
		cmd, info := initrepotab.HandleInitError(msg, &m.appState)
		if info != nil {
			if info.NotJJRepo {
				m.initRepoModel.SetPath(info.CurrentPath)
			} else {
				m.applyEffects(effShowError{info.Err})
			}
		}
		return m, cmd
	case data.JJInitSuccessMsg:
		m.initRepoModel.SetPath("")
		m.applyEffects(effClearError{})
		return m, initrepotab.HandleJJInitSuccess(msg, &m.appState)
	case data.RemoteOpResultMsg:
		return m.handleRemoteOpResultMsg(msg)
	case data.PushResultMsg:
		return m.handlePushResultMsg(msg)
	case data.RepoReadyMsg:
		return m.handleRepoReadyMsg(msg)
	case data.AuxServicesReadyMsg:
		return m.handleAuxServicesReadyMsg(msg)
	case data.ServicesInitializedMsg: //nolint:staticcheck // SA1019: transitional case still dispatches the deprecated one-shot message.
		return m.handleDataServicesInitializedMsg(msg)
	case data.RepositoryLoadedMsg:
		return m.handleDataRepositoryLoadedMsg(msg)
	case graphtab.RepositoryLoadedMsg:
		return m.handleActionsRepositoryLoadedMsg(msg)
	case data.SilentRepositoryLoadedMsg:
		return m.handleDataSilentRepositoryLoadedMsg(msg)

	case prstab.PrsLoadedMsg:
		m.appState.PRsLoadedOnce = true
		m.appState.Loading = false
		updated, cmd := m.prsTabModel.UpdateWithApp(msg, &m.appState)
		m.prsTabModel = updated
		m.prsTabModel.OnRepositoryLoaded(m.appState.Repository)
		// The bulk list just replaced Repository.PRs; resolve any still-unmatched local bookmarks to
		// their open PR via targeted lookups so the graph can offer "Update PR" for branches the
		// limited bulk fetch omitted. Run after the bulk load so PrsLoadedMsg can't clobber the result.
		if resolveCmd := m.applyEffects(effResolveOpenPRs{}); resolveCmd != nil {
			cmd = tea.Batch(cmd, resolveCmd)
		}
		return m, cmd
	case prstab.OpenPRsResolvedMsg:
		return m.handleOpenPRsResolvedMsg(msg)
	case prstab.PrMergedMsg, prstab.PrClosedMsg:
		updated, cmd := m.prsTabModel.UpdateWithApp(msg, &m.appState)
		m.prsTabModel = updated
		var err error
		switch mmsg := msg.(type) {
		case prstab.PrMergedMsg:
			err = mmsg.Err
		case prstab.PrClosedMsg:
			err = mmsg.Err
		}
		if err != nil {
			m.appState.Loading = false
			return m, m.applyEffects(effShowError{err})
		}
		return m, cmd
	case prstab.LoadErrorMsg:
		m.appState.PRsLoadedOnce = true
		m.appState.Loading = false
		updated, _ := m.prsTabModel.UpdateWithApp(msg, &m.appState)
		m.prsTabModel = updated
		// P5.5: PR fetch is a transient GitHub API call; offer Retry that re-runs the same load.
		existingPRs := 0
		if m.appState.Repository != nil {
			existingPRs = len(m.appState.Repository.PRs)
		}
		return m, m.applyEffects(effShowRetryableError{
			err:   msg.Err,
			retry: prstab.LoadPRsCmd(m.appState.GitHubService, m.appState.GithubInfo, m.appState.DemoMode, existingPRs),
		})
	case prstab.ReauthNeededMsg:
		updated, _ := m.prsTabModel.UpdateWithApp(msg, &m.appState)
		m.prsTabModel = updated
		return m.handleReauthNeededEffect(prstab.ApplyReauthNeededEffect(msg))
	case prstab.PrTickMsg:
		prInput := prstab.PrTickInput{
			IsPRView:      m.appState.ViewMode == state.ViewPullRequests,
			Loading:       m.appState.Loading,
			HasError:      m.errorModal.GetError() != nil,
			GitHubService: m.appState.GitHubService,
			GithubInfo:    m.appState.GithubInfo,
			DemoMode:      m.appState.DemoMode,
			ExistingCount: 0,
		}
		if m.appState.Repository != nil {
			prInput.ExistingCount = len(m.appState.Repository.PRs)
		}
		updated, cmd := m.prsTabModel.UpdateWithApp(prInput, &m.appState)
		m.prsTabModel = updated
		return m, cmd

	case ticketstab.TicketsLoadedMsg:
		m.appState.TicketsLoadedOnce = true
		m.appState.Loading = false
		input := ticketstab.TicketsLoadedInput{
			Tickets:      msg.Tickets,
			ProviderName: "",
			HasService:   m.appState.TicketService != nil,
			CanCreate:    m.appState.TicketService != nil && m.appState.TicketService.CanCreateTicket(),
		}
		if m.appState.TicketService != nil {
			input.ProviderName = m.appState.TicketService.GetProviderName()
		}
		updated, cmd := m.ticketsTabModel.UpdateWithApp(input, &m.appState)
		m.ticketsTabModel = updated
		return m, cmd
	case ticketstab.TransitionsLoadedMsg:
		updated, cmd := m.ticketsTabModel.UpdateWithApp(msg, &m.appState)
		m.ticketsTabModel = updated
		return m, cmd
	case ticketstab.TransitionCompletedMsg:
		updated, cmd := m.ticketsTabModel.UpdateWithApp(msg, &m.appState)
		m.ticketsTabModel = updated
		if msg.Err != nil {
			return m, m.applyEffects(effShowError{msg.Err})
		}
		return m, cmd
	case ticketstab.LoadErrorMsg:
		m.appState.TicketsLoadedOnce = true
		m.appState.Loading = false
		updated, _ := m.ticketsTabModel.UpdateWithApp(msg, &m.appState)
		m.ticketsTabModel = updated
		// P5.5: ticket fetch is a transient provider API call; offer Retry that re-runs the load.
		m.applyEffects(effShowRetryableError{
			err:   msg.Err,
			retry: ticketstab.LoadTicketsCmd(m.appState.TicketService, m.appState.DemoMode),
		})
		m.appState.StatusMessage = fmt.Sprintf("Error: %v", msg.Err)
		return m, nil

	case branchestab.BranchesLoadedMsg:
		input := branchestab.BranchesLoadedInput{
			BranchesLoadedMsg:    msg,
			InCreateBookmarkView: m.appState.ViewMode == state.ViewCreateBookmark,
			HasError:             m.errorModal.GetError() != nil,
		}
		updated, cmd := m.branchesTabModel.UpdateWithApp(input, &m.appState)
		m.branchesTabModel = updated
		if input.InCreateBookmarkView {
			m.applyEffects(effSetBookmarkConflictSources{})
		}
		return m, cmd
	case branchestab.BranchActionMsg:
		updated, _ := m.branchesTabModel.UpdateWithApp(msg, &m.appState)
		m.branchesTabModel = updated
		if msg.Action == "fetch" {
			m.appState.SpinnerStartPending = false
		}
		if msg.Err != nil {
			// Branches tab already set StatusMessage (e.g. "Failed to push branch: ...").
			m.appState.Loading = false
			return m, nil
		}
		return m, m.applyEffects(effLoadBranches{}, effReloadRepository{})

	case settingstab.SettingsSavedMsg:
		wasSettings := m.appState.ViewMode == state.ViewSettings
		cmd, errInfo := settingstab.HandleSettingsSavedMsg(msg, &m.appState)
		if errInfo != nil {
			return m, m.applyEffects(effShowError{errInfo.Err})
		}
		if wasSettings {
			m.settingsTabModel.SetViewOpts(m.buildSettingsViewOpts())
		}
		// The "show all remote branches" toggle changes how bookmarks are listed; re-apply it
		// to the live service and reload the branch list so the change is reflected immediately.
		if m.appState.JJService != nil && m.appState.Config != nil {
			m.appState.JJService.BookmarkListPreferTracked = m.appState.Config.BranchesFilterToTrackedAndMine()
			cmd = tea.Batch(cmd, m.applyEffects(effLoadBranches{}))
		}
		// Broadcast the config change so config-dependent modals/tabs re-read the
		// new snapshot (handled by the config.ChangedMsg case below).
		cfg := m.appState.Config
		return m, tea.Batch(cmd, func() tea.Msg { return config.ChangedMsg{Config: cfg} })

	case config.ChangedMsg:
		// Config changed (e.g. settings saved): re-sync config-dependent modals so
		// they read the new values instead of a stale snapshot.
		if msg.Config != nil {
			// Keep the evolog split modal in sync (e.g. after saving AI settings).
			m.evologSplitModal = m.evologSplitModal.WithSuggestConfig(msg.Config)
		}
		// Propagate the new AI profile list to any open generate-bearing modal.
		m.pushAIProfilesToFormModals()
		return m, nil

	case settingstab.GitHubDeviceFlowStartedMsg:
		m.beginModalUnderlay()
		m.githubLoginModel.SetDeviceFlow(msg.DeviceCode, msg.UserCode, msg.VerificationURL, msg.Interval)
		m.appState.ViewMode = state.ViewGitHubLogin
		m.appState.StatusMessage = "Waiting for GitHub authorization..."
		// Do not auto-open the browser; user can press Enter or click "Copy Code & Open Browser" on the login screen.
		return m, settingstab.PollGitHubTokenCmd(m.githubLoginModel.GetDeviceCode())

	case settingstab.GitHubCLILoginShowMsg:
		m.beginModalUnderlay()
		m.githubLoginModel.SetGhCLILoginMode()
		m.appState.ViewMode = state.ViewGitHubLogin
		m.appState.StatusMessage = "GitHub CLI: press Enter or click Run to start gh auth login."
		return m, nil

	case githublogintab.GhCLIAuthFinishedMsg:
		if m.appState.ViewMode != state.ViewGitHubLogin {
			return m, nil
		}
		m.githubLoginModel.ClearFlow()
		m.clearModalUnderlay()
		m.appState.ViewMode = state.ViewSettings
		m.settingsTabModel.SetViewOpts(m.buildSettingsViewOpts())
		if msg.Err != nil {
			m.appState.StatusMessage = fmt.Sprintf("gh auth login: %v", msg.Err)
			m.errorModal.SetError(msg.Err, false, "")
			return m, nil
		}
		tok, ok := config.TryGitHubCLIToken()
		if !ok || strings.TrimSpace(tok) == "" {
			err := fmt.Errorf("gh finished but no token was available from gh auth token; try gh auth login again or gh auth status")
			m.appState.StatusMessage = err.Error()
			m.errorModal.SetError(err, false, "")
			return m, nil
		}
		cfg, err := config.Load()
		if err != nil || cfg == nil {
			err := fmt.Errorf("could not load config after gh login: %v", err)
			m.appState.StatusMessage = err.Error()
			m.errorModal.SetError(err, false, "")
			return m, nil
		}
		cfg.GitHubToken = ""
		cfg.GitHubTokenSource = config.GitHubTokenSourceGhCLI
		cfg.GitHubAuthMethod = config.GitHubAuthGhCLI
		if err := cfg.Save(); err != nil {
			m.appState.StatusMessage = fmt.Sprintf("could not save config: %v", err)
			m.errorModal.SetError(err, false, "")
			return m, nil
		}
		_ = os.Unsetenv("GITHUB_TOKEN")
		m.settingsTabModel.GetGitHubModel().SetTokenSource(config.GitHubTokenSourceGhCLI)
		m.settingsTabModel.GetGitHubModel().SetToken("")
		m.settingsTabModel.SetSettingInputValue(0, "")
		m.appState.StatusMessage = "GitHub CLI login successful!"
		return m, data.InitializeServices(m.appState.DemoMode)

	case settingstab.GitHubLoginPollMsg:
		if m.githubLoginModel.GetPolling() {
			if msg.Interval > 0 {
				m.githubLoginModel.SetPollInterval(m.githubLoginModel.GetPollInterval() + msg.Interval)
			}
			return m, tea.Tick(time.Duration(m.githubLoginModel.GetPollInterval())*time.Second, func(t time.Time) tea.Msg {
				return doPollMsg{}
			})
		}
		return m, nil

	case doPollMsg:
		if m.githubLoginModel.GetPolling() {
			return m, settingstab.PollGitHubTokenCmd(m.githubLoginModel.GetDeviceCode())
		}
		return m, nil

	case settingstab.GitHubLoginSuccessMsg:
		m.githubLoginModel.ClearFlow()
		m.clearModalUnderlay()
		m.appState.ViewMode = state.ViewSettings
		m.settingsTabModel.SetViewOpts(m.buildSettingsViewOpts())
		m.appState.StatusMessage = "GitHub login successful!"
		cfg, _ := config.Load()
		cfg.SetGitHubToken(msg.Token, config.GitHubAuthDeviceFlow)
		_ = cfg.Save()
		_ = os.Setenv("GITHUB_TOKEN", msg.Token)
		m.settingsTabModel.SetSettingInputValue(0, msg.Token)
		return m, data.InitializeServices(m.appState.DemoMode)

	case settingstab.GitHubLoginErrorMsg:
		m.githubLoginModel.ClearFlow()
		m.clearModalUnderlay()
		m.appState.ViewMode = state.ViewSettings
		m.appState.StatusMessage = fmt.Sprintf("GitHub login error: %v", msg.Err)
		m.errorModal.SetError(msg.Err, false, "")
		return m, nil

	case prformtab.PRCreatedMsg:
		m.clearAIGenOverlay()
		m.prFormModal.Hide()
		m.clearModalUnderlay()
		// P2.5: the root owns the prs-tab import and builds the reload command, so
		// prform no longer imports the prs tab.
		prCreatedExisting := 0
		if m.appState.Repository != nil {
			prCreatedExisting = len(m.appState.Repository.PRs)
		}
		return m, prformtab.HandlePRCreatedMsg(prformtab.PRCreatedInput{PRCreatedMsg: msg, DemoMode: m.appState.DemoMode}, &m.appState,
			prstab.LoadPRsCmd(m.appState.GitHubService, m.appState.GithubInfo, m.appState.DemoMode, prCreatedExisting))
	case ticketformtab.TicketCreatedMsg:
		m.clearAIGenOverlay()
		m.ticketFormModal.Hide()
		m.clearModalUnderlay()
		m.appState.Loading = false
		m.appState.ViewMode = state.ViewTickets
		if msg.Ticket != nil {
			m.appState.StatusMessage = fmt.Sprintf("Created %s: %s", msg.Ticket.DisplayKey, msg.Ticket.Summary)
			cmd := ticketformtab.HandleTicketCreatedMsg(msg.Ticket, m.appState.TicketService, m.appState.DemoMode)
			if cmd != nil {
				return m, tea.Batch(cmd, ticketstab.LoadTicketsCmd(m.appState.TicketService, m.appState.DemoMode))
			}
			return m, ticketstab.LoadTicketsCmd(m.appState.TicketService, m.appState.DemoMode)
		}
		return m, ticketstab.LoadTicketsCmd(m.appState.TicketService, m.appState.DemoMode)
	case prstab.BranchPushedMsg:
		// P2.5: cross-tab orchestration (reload repo + PRs) lives here in the root
		// instead of the branches tab, so branches no longer imports prs.
		m.appState.Loading = false
		m.appState.StatusMessage = fmt.Sprintf("Pushed %s to remote", msg.Branch)
		existing := 0
		if m.appState.Repository != nil {
			existing = len(m.appState.Repository.PRs)
		}
		return m, tea.Batch(
			data.LoadRepository(m.appState.JJService),
			prstab.LoadPRsCmd(m.appState.GitHubService, m.appState.GithubInfo, m.appState.DemoMode, existing),
		)
	case bookmarktab.BookmarkCreatedMsg:
		m.clearAIGenOverlay()
		m.bookmarkModal.Hide()
		m.clearModalUnderlay()
		m.appState.Loading = false
		// P2.5: the root owns the tickets-tab import and builds the optional
		// ticket-transition command, so the bookmark tab no longer imports tickets.
		var transitionCmd tea.Cmd
		if msg.TicketKey != "" && m.appState.TicketService != nil && m.appState.Config != nil && m.appState.Config.AutoInProgressOnBranch() {
			transitionCmd = ticketstab.TransitionTicketToInProgressCmd(m.appState.TicketService, msg.TicketKey)
		}
		return m, bookmarktab.HandleBookmarkCreatedMsg(msg, &m.appState, transitionCmd)
	case bookmarktab.BookmarkDeletedMsg:
		// P2.5: cross-tab orchestration (reload repo + PRs) lives here in the root
		// instead of the branches tab, so branches no longer imports bookmark.
		m.appState.ViewMode = state.ViewCommitGraph
		m.appState.StatusMessage = fmt.Sprintf("Bookmark '%s' deleted", msg.BookmarkName)
		existing := 0
		if m.appState.Repository != nil {
			existing = len(m.appState.Repository.PRs)
		}
		return m, tea.Batch(
			data.LoadRepository(m.appState.JJService),
			prstab.LoadPRsCmd(m.appState.GitHubService, m.appState.GithubInfo, m.appState.DemoMode, existing),
		)
	case branchestab.BookmarkConflictInfoMsg:
		cmd, info := conflicttab.HandleBookmarkConflictInfoMsg(conflicttab.ConflictInfoInput{
			BookmarkName:  msg.BookmarkName,
			LocalID:       msg.LocalID,
			RemoteID:      msg.RemoteID,
			LocalSummary:  msg.LocalSummary,
			RemoteSummary: msg.RemoteSummary,
			LocalWhen:     msg.LocalWhen,
			RemoteWhen:    msg.RemoteWhen,
			Err:           msg.Err,
		}, &m.appState)
		if msg.Err != nil {
			m.applyEffects(effShowError{msg.Err})
		} else {
			m.applyEffects(effClearError{})
		}
		if info != nil {
			m.bookmarkConflictReturnView = m.appState.ViewMode
			m.bookmarkConflictReturnValid = true
			m.conflictModal = m.conflictModal.SetDimensions(m.width, m.height)
			m.conflictModal.Show(info.BookmarkName, info.LocalID, info.RemoteID, info.LocalSummary, info.RemoteSummary, info.LocalWhen, info.RemoteWhen)
			m.appState.ViewMode = state.ViewBookmarkConflict
		}
		return m, cmd
	case conflicttab.BookmarkConflictResolvedMsg:
		m.conflictModal.Hide()
		restore := state.ViewBranches
		if m.bookmarkConflictReturnValid {
			restore = m.bookmarkConflictReturnView
		}
		m.bookmarkConflictReturnValid = false
		m.appState.ViewMode = restore
		if msg.Err != nil {
			m.applyEffects(effShowError{msg.Err})
		}
		return m, conflicttab.HandleBookmarkConflictResolvedMsg(msg, &m.appState, branchestab.LoadBranchesCmd(m.appState.JJService, m.settingsTabModel.GetSettingsBranchLimit()))
	case workspacestab.WorkspacesLoadedMsg:
		m.appState.Loading = false
		if msg.Err != nil {
			return m, m.applyEffects(effShowError{msg.Err})
		}
		if m.appState.ViewMode == state.ViewWorkspaces && m.workspacesModal.IsShown() {
			// Refresh in place (e.g. after add/forget) without re-opening.
			m.workspacesModal.SetWorkspaces(msg.Workspaces)
		} else {
			m.workspacesModal = m.workspacesModal.SetDimensions(m.width, m.height)
			m.workspacesModal.Show(msg.Workspaces)
			m.appState.ViewMode = state.ViewWorkspaces
		}
		m.appState.StatusMessage = "Workspaces"
		return m, nil
	case workspacestab.WorkspaceChangedMsg:
		m.appState.Loading = false
		if msg.Err != nil {
			return m, m.applyEffects(effShowError{msg.Err})
		}
		if msg.StatusMessage != "" {
			m.appState.StatusMessage = msg.StatusMessage
		}
		// Reload the list so the modal reflects the change.
		return m, workspacestab.LoadWorkspacesCmd(m.appState.JJService)
	case operationstab.OperationsLoadedMsg:
		m.appState.Loading = false
		if msg.Err != nil {
			return m, m.applyEffects(effShowError{msg.Err})
		}
		if m.appState.ViewMode == state.ViewOperations && m.operationsModal.IsShown() {
			// Refresh in place without re-opening (preserves selection/scroll).
			m.operationsModal.SetOperations(msg.Operations)
		} else {
			m.operationsModal = m.operationsModal.SetDimensions(m.width, m.height)
			m.operationsModal.Show(msg.Operations)
			m.appState.ViewMode = state.ViewOperations
		}
		m.appState.StatusMessage = "Operation log"
		return m, nil
	case operationstab.OperationRestoredMsg:
		m.appState.Loading = false
		if msg.Err != nil {
			return m, m.applyEffects(effShowError{msg.Err})
		}
		if msg.StatusMessage != "" {
			m.appState.StatusMessage = msg.StatusMessage
		}
		// Reload the graph so it reflects the restored operation.
		return m, m.applyEffects(effReloadRepository{})
	case graphtab.AbsorbPreviewReadyMsg:
		m.appState.Loading = false
		if msg.Err != nil {
			return m, m.applyEffects(effShowError{msg.Err})
		}
		if msg.Preview == nil || msg.Preview.Nothing {
			m.appState.StatusMessage = "Nothing to absorb"
			return m, nil
		}
		m.absorbPreviewSummary = msg.Preview.Summary
		m.absorbPreviewActive = true
		m.appState.StatusMessage = "Review absorb preview: y confirm · n or Esc cancel"
		return m, nil
	case graphtab.DivergentCommitInfoMsg:
		cmd, info := divergenttab.HandleDivergentCommitInfoMsg(divergenttab.DivergentCommitInfoInput{ChangeID: msg.ChangeID, Versions: msg.Versions, Err: msg.Err}, &m.appState)
		if info != nil {
			m.divergentModal = m.divergentModal.SetDimensions(m.width, m.height)
			m.divergentModal.Show(info.ChangeID, info.Versions)
			m.appState.ViewMode = state.ViewDivergentCommit
		}
		return m, cmd
	case divergenttab.DivergentCommitResolvedMsg:
		m.divergentModal.Hide()
		return m, divergenttab.HandleDivergentCommitResolvedMsg(msg, &m.appState)
	case evologsplittab.EvologLoadedMsg:
		updated, cmd := m.evologSplitModal.Update(msg)
		m.evologSplitModal = updated
		if msg.Err == nil {
			m.appState.StatusMessage = "Pick parent (j/k, Enter); o step diff; s AI suggest; p plan preview (opens after suggest)"
		} else {
			m.appState.StatusMessage = "Evolog load failed"
		}
		return m, cmd
	case evologsplittab.EvologDiffLoadRequestedMsg:
		seq, from, to, prevFrom, prevTo, ok := m.evologSplitModal.DiffSnapshotForLoad()
		if !ok {
			return m, nil
		}
		return m, evologsplittab.LoadEvologSplitDiffCmd(m.appState.JJService, seq, from, to, prevFrom, prevTo)
	case evologsplittab.OverlaySpinTickMsg:
		if m.appState.ViewMode != state.ViewEvologSplit || !m.evologSplitModal.IsShown() {
			return m, nil
		}
		updated, cmd := m.evologSplitModal.Update(msg)
		m.evologSplitModal = updated
		return m, cmd
	case evologsplittab.EvologSplitSuggestRequestedMsg:
		if m.appState.JJService == nil {
			return m, func() tea.Msg {
				return aitab.EvologSplitSuggestMsg{ReqID: msg.ReqID, Err: fmt.Errorf("jj service not available")}
			}
		}
		if m.appState.Config == nil || !m.appState.Config.AIConfiguredForGeneration() {
			return m, func() tea.Msg {
				return aitab.EvologSplitSuggestMsg{ReqID: msg.ReqID, Err: fmt.Errorf("AI is disabled or no API key (Settings → AI, or %s)", config.EnvAIAPIKey)}
			}
		}
		return m, aitab.EvologSuggestPrepChainStartCmd(msg.ReqID, m.appState.JJService, m.appState.Config, m.evologSplitModal.EvologEntries())

	case aitab.EvologSuggestPrepProgressMsg:
		if !m.evologSplitModal.IsShown() || msg.ReqID != m.evologSplitModal.SuggestReqID() {
			return m, nil
		}
		m.evologSplitModal = m.evologSplitModal.WithSuggestPrepProgress(msg.JJDone, msg.JJTotal, "jj")
		return m, nil

	case aitab.EvologSuggestPrepDoneMsg:
		if !m.evologSplitModal.IsShown() || msg.ReqID != m.evologSplitModal.SuggestReqID() {
			return m, nil
		}
		if msg.Err != nil {
			return m, func() tea.Msg {
				return aitab.EvologSplitSuggestMsg{ReqID: msg.ReqID, Err: msg.Err}
			}
		}
		m.evologSplitModal = m.evologSplitModal.WithSuggestPrepProgress(msg.JJTotal, msg.JJTotal, "llm")
		return m, aitab.EvologSuggestLLMCmd(msg.ReqID, m.appState.JJService, m.appState.Config, m.evologSplitModal.EvologEntries(), msg.UserPrompt)
	case evologsplittab.EvologSplitDiffLoadedMsg:
		updated, cmd := m.evologSplitModal.Update(msg)
		m.evologSplitModal = updated
		return m, cmd
	case evologsplittab.EvologOutcomePreviewRequestedMsg:
		if m.appState.JJService == nil {
			return m, nil
		}
		if m.appState.ViewMode != state.ViewEvologSplit || !m.evologSplitModal.IsShown() {
			return m, nil
		}
		return m, evologsplittab.LoadEvologOutcomePreviewCmd(m.appState.JJService, msg.Seq)
	case evologsplittab.EvologOutcomePreviewLoadedMsg:
		if m.appState.ViewMode != state.ViewEvologSplit || !m.evologSplitModal.IsShown() {
			return m, nil
		}
		updated, cmd := m.evologSplitModal.Update(msg)
		m.evologSplitModal = updated
		return m, cmd
	case aitab.EvologSplitSuggestMsg:
		if !m.evologSplitModal.IsShown() {
			return m, nil
		}
		var warnCmd tea.Cmd
		if msg.Err != nil {
			msgText := strings.TrimSpace(msg.Err.Error())
			if r := []rune(msgText); len(r) > 900 {
				msgText = string(r[:900]) + "…"
			}
			warnCmd = state.NavigateTarget{
				Kind:           state.NavigateWarning,
				WarningTitle:   "AI suggest split failed",
				WarningMessage: msgText + "\n\nPress Esc to dismiss.",
				WarningCommits: nil,
			}.Cmd()
		}
		updated, sub := m.evologSplitModal.Update(msg)
		m.evologSplitModal = updated
		if msg.Err == nil && !msg.NoSplit && msg.PickIndex > 0 {
			m.appState.StatusMessage = "AI plan: preview opened — Esc closes overlay, then Enter to split or adjust row"
		}
		if msg.Err == nil && msg.NoSplit {
			m.appState.StatusMessage = "AI: no split — use p for WC files or pick another row"
		}
		if warnCmd != nil && sub != nil {
			return m, tea.Batch(sub, warnCmd)
		}
		if warnCmd != nil {
			return m, warnCmd
		}
		return m, sub
	case evologsplittab.EvologSplitCompletedMsg:
		if len(m.evologStepwiseRemainderAfterSplit) > 0 {
			rem := m.evologStepwiseRemainderAfterSplit
			m.evologStepwiseRemainderAfterSplit = nil
			m.evologSplitModal.SetPendingMultiSplitIDs(rem)
			m2, cmd := m.applyRepositoryLoaded(msg.Repository)
			m2.appState.ViewMode = state.ViewEvologSplit
			m2.appState.StatusMessage = fmt.Sprintf("Stepwise split: %d base(s) left — review evolog, then Enter", len(rem))
			wc := msg.Repository.WorkingCopy
			bn := m2.evologStepwiseBookmarkName
			loadCmd := evologsplittab.LoadEvologCmd(m2.appState.JJService, bn, wc)
			if cmd != nil {
				return m2, tea.Batch(cmd, loadCmd)
			}
			return m2, loadCmd
		}
		m.evologStepwiseBookmarkName = ""
		m.evologSplitModal.Hide()
		m.appState.ViewMode = state.ViewCommitGraph
		m2, cmd := m.applyRepositoryLoaded(msg.Repository)
		m2.appState.StatusMessage = "Split complete — Graph (g) shows what jj did; compare to the plan you saw in Preview (p) before split"
		if m2.evologPostSplitDescribe && m2.appState.JJService != nil && m2.appState.Config != nil && m2.appState.Config.AIConfiguredForGeneration() {
			m2.evologPostSplitDescribe = false
			preChild := strings.TrimSpace(m2.evologPrecomputedDescribeChild)
			preParent := strings.TrimSpace(m2.evologPrecomputedDescribeParent)
			m2.evologPrecomputedDescribeParent = ""
			m2.evologPrecomputedDescribeChild = ""
			if preChild != "" {
				descCtx, descCancel := context.WithTimeout(context.Background(), m2.appState.Config.AITimeout())
				parentOK, perr := aitab.DescribeSplitParentWritable(descCtx, m2.appState.JJService)
				descCancel()
				skipParent := perr != nil || !parentOK
				m2.evologDescribePreviewActive = true
				m2.evologDescribePreviewFromPlan = true
				m2.evologDescribeSkipParent = skipParent
				m2.evologDescribeParent = preParent
				m2.evologDescribeChild = preChild
				m2.appState.StatusMessage = "Split complete — Graph (g) vs plan preview; review AI descriptions (y apply, n discard)"
				if cmd != nil {
					return m2, cmd
				}
				return m2, nil
			}
			m2.appState.StatusMessage = "Split complete — Graph (g) vs plan preview; generating descriptions with AI…"
			m2.appState.Loading = true
			if cmd != nil {
				return m2, tea.Batch(cmd, aitab.SuggestEvologSplitDescriptionsCmd(0, m2.appState.JJService, m2.appState.Config), m2.startBusySpinnerCmd())
			}
			return m2, tea.Batch(aitab.SuggestEvologSplitDescriptionsCmd(0, m2.appState.JJService, m2.appState.Config), m2.startBusySpinnerCmd())
		}
		m2.evologPostSplitDescribe = false
		m2.evologPrecomputedDescribeParent = ""
		m2.evologPrecomputedDescribeChild = ""
		return m2, cmd
	case aitab.EvologDescribeSplitPreviewMsg:
		m.appState.Loading = false
		if msg.Err != nil {
			return m.Update(errorMsg{Err: fmt.Errorf("post-split describe preview: %w", msg.Err)})
		}
		m.evologDescribePreviewActive = true
		m.evologDescribePreviewFromPlan = false
		m.evologDescribeSkipParent = msg.SkipParentDescribe
		m.evologDescribeParent = msg.ParentDescription
		m.evologDescribeChild = msg.ChildDescription
		m.appState.StatusMessage = "AI descriptions ready — y apply, n discard, Esc cancel"
		return m, nil
	case aitab.EvologDescribeSplitDoneMsg:
		m.appState.Loading = false
		m.evologDescribePreviewActive = false
		m.evologDescribePreviewFromPlan = false
		if msg.Err != nil {
			return m.Update(errorMsg{Err: fmt.Errorf("post-split describe: %w", msg.Err)})
		}
		m2, cmd := m.applyRepositoryLoaded(msg.Repository)
		if msg.OnlyChild {
			m2.appState.StatusMessage = "Description updated for @ (parent @- is immutable)"
		} else {
			m2.appState.StatusMessage = "Descriptions updated for @- and @"
		}
		return m2, cmd
	case graphtab.FileMoveCompletedMsg:
		graphtab.HandleFileMoveCompletedMsg(graphtab.FileMoveInput{
			FileMoveCompletedMsg: msg,
			ChangedFilesCommitID: m.graphTabModel.GetChangedFilesCommitID(),
		}, &m.appState)
		m.graphTabModel.OnRepositoryLoaded(m.appState.Repository)
		if m.appState.Repository != nil {
			for i, commit := range m.appState.Repository.Graph.Commits {
				if commit.ChangeID == m.graphTabModel.GetChangedFilesCommitID() {
					m.graphTabModel.SelectCommit(i)
					break
				}
			}
			// Load changed files for the currently selected commit so the files pane updates.
			idx := m.graphTabModel.GetSelectedCommit()
			commits := m.appState.Repository.Graph.Commits
			if idx >= 0 && idx < len(commits) && m.appState.JJService != nil {
				return m, graphtab.LoadChangedFilesCmd(m.appState.JJService, commits[idx].ChangeID)
			}
		}
		return m, nil
	case graphtab.FileRevertedMsg:
		graphtab.HandleFileRevertedMsg(graphtab.FileRevertedInput{
			FileRevertedMsg:      msg,
			ChangedFilesCommitID: m.graphTabModel.GetChangedFilesCommitID(),
		}, &m.appState)
		m.graphTabModel.OnRepositoryLoaded(m.appState.Repository)
		if m.appState.Repository != nil {
			for i, commit := range m.appState.Repository.Graph.Commits {
				if commit.ChangeID == m.graphTabModel.GetChangedFilesCommitID() {
					m.graphTabModel.SelectCommit(i)
					break
				}
			}
			idx := m.graphTabModel.GetSelectedCommit()
			commits := m.appState.Repository.Graph.Commits
			if idx >= 0 && idx < len(commits) && m.appState.JJService != nil {
				return m, graphtab.LoadChangedFilesCmd(m.appState.JJService, commits[idx].ChangeID)
			}
		}
		return m, nil
	case graphtab.LongPressTickMsg:
		updated, cmd := m.graphTabModel.UpdateWithApp(msg, &m.appState)
		m.graphTabModel = updated
		return m, cmd
	case graphtab.CommitLongPressTickMsg:
		updated, cmd := m.graphTabModel.UpdateWithApp(msg, &m.appState)
		m.graphTabModel = updated
		return m, cmd
	case prstab.LongPressTickMsg:
		updated, cmd := m.prsTabModel.UpdateWithApp(msg, &m.appState)
		m.prsTabModel = updated
		return m, cmd
	case ticketstab.LongPressTickMsg:
		updated, cmd := m.ticketsTabModel.UpdateWithApp(msg, &m.appState)
		m.ticketsTabModel = updated
		return m, cmd
	case branchestab.LongPressTickMsg:
		updated, cmd := m.branchesTabModel.UpdateWithApp(msg, &m.appState)
		m.branchesTabModel = updated
		return m, cmd

	case descedittab.DescriptionSavedMsg:
		cmd := descedittab.HandleDescriptionSavedMsg(msg, &m.appState)
		m.clearAIGenOverlay()
		m.desceditModal.Hide()
		m.clearModalUnderlay()
		// Keep Loading true through the LoadRepository reload returned above so the busy
		// overlay stays up (now over the graph) until applyRepositoryLoaded renders the
		// updated description. Re-batch a spinner tick in case clearAIGenOverlay stopped it.
		return m, tea.Batch(cmd, m.startBusySpinnerCmd())
	case descedittab.DescriptionLoadedMsg:
		if m.appState.ViewMode != state.ViewEditDescription || m.desceditModal.GetEditingCommitID() != msg.CommitID {
			return m, nil
		}
		finalDesc := descedittab.SuggestDescriptionForLoad(descedittab.DescriptionLoadedInput{
			CommitID:       msg.CommitID,
			Description:    msg.Description,
			Repository:     m.appState.Repository,
			CommitIdx:      commitIdxForChangeID(m.appState.Repository, msg.CommitID),
			TicketKeys:     m.bookmarkModal.GetTicketBookmarkDisplayKeys(),
			FindBookmarkFn: bookmarktab.FindBookmarkForCommit,
		})
		if finalDesc == "" {
			finalDesc = msg.Description
			if finalDesc == "(no description)" {
				finalDesc = ""
			}
		}
		m.desceditModal.SetDescription(finalDesc)
		m.appState.StatusMessage = "Editing description (Ctrl+S to save, Esc to cancel)"
		return m, nil
	case util.ClipboardCopiedMsg:
		return m.handleClipboardCopiedMsg(msg)
	case settingstab.CleanupCompletedMsg:
		return m, settingstab.HandleCleanupCompletedMsg(msg, &m.appState)

	case graphtab.ChangedFilesLoadedMsg:
		updated, cmd := m.graphTabModel.Update(msg)
		if g, ok := updated.(*graphtab.GraphModel); ok {
			m.graphTabModel = *g
		}
		return m, cmd
	case filedifftab.FileDiffLoadedMsg:
		updated, cmd := m.fileDiffModal.Update(msg)
		m.fileDiffModal = updated
		// Modal header/footer already explain Esc/scroll. Do not set StatusMessage when Loading:
		// another op may own StatusMessage, and the centered overlay would show misleading text after
		// this modal closes (see shouldShowLoadingOverlay / NavigateCloseFileDiff).
		if msg.Err != nil {
			if !m.appState.Loading {
				m.appState.StatusMessage = "File diff failed"
			}
		} else if !m.appState.Loading {
			m.appState.StatusMessage = ""
		}
		return m, cmd
	case loadChangedFilesTriggerMsg:
		if m.appState.JJService != nil && m.appState.Repository != nil {
			commits := m.appState.Repository.Graph.Commits
			idx := m.graphTabModel.GetSelectedCommit()
			if idx >= 0 && idx < len(commits) {
				return m, graphtab.LoadChangedFilesCmd(m.appState.JJService, commits[idx].ChangeID)
			}
		}
		return m, nil
	case tickMsg:
		return m.handleTickMsg()
	case undoHintReadyMsg:
		return m.handleUndoHintReady(msg)
	case undoHintExpiredMsg:
		return m.handleUndoHintExpired(msg)
	case graphtab.UndoCompletedMsg:
		cmd, errInfo := graphtab.HandleUndoCompletedMsg(msg, &m.appState)
		if errInfo != nil {
			m.appState.Loading = false
			return m, m.applyEffects(effShowError{errInfo.Err})
		}
		if msg.Message == "Undo completed" {
			m.redoOperationID = msg.RedoOpID
		} else {
			m.redoOperationID = ""
		}
		// P5.4: undo/redo change the current operation; refresh the hint on reload.
		m.pendingUndoHint = true
		return m, cmd

	// Handle our custom messages
	case TabSelectedMsg:
		m.appState.ViewMode = msg.Tab
		if msg.Tab == state.ViewSettings {
			m.settingsTabModel.SetViewOpts(m.buildSettingsViewOpts())
		}
		if msg.Tab == state.ViewHelp {
			m.refreshHelpCommandHistory()
		}
		return m, nil

	// Theme color picker: close picker and update color when user confirms or cancels
	case bubblepicker.ColorChosenMsg, bubblepicker.ColorCanceledMsg:
		if m.appState.ViewMode == state.ViewSettings {
			cmds := util.PropagateUpdate(msg, &m.settingsTabModel)
			if len(cmds) > 0 && cmds[0] != nil {
				return m, cmds[0]
			}
		}
		return m, nil

	case ActionMsg:
		return m.handleAction(msg.Action)

	// Handle messages from actions package
	case util.ExternalEditorOpenedMsg:
		m.appState.Loading = false
		if strings.TrimSpace(msg.FileBase) != "" {
			m.appState.StatusMessage = fmt.Sprintf("Opened %s", msg.FileBase)
		} else {
			m.appState.StatusMessage = "Opened in external editor"
		}
		return m, nil

	case util.ErrorMsg:
		if msg.StatusOnly {
			m.appState.Loading = false
			m.appState.StatusMessage = util.StatusStringFromError(msg.Err, 220)
			return m, nil
		}
		m.evologPostSplitDescribe = false
		m.evologDescribePreviewActive = false
		m.evologDescribePreviewFromPlan = false
		m.evologDescribeSkipParent = false
		m.evologDescribeParent = ""
		m.evologDescribeChild = ""
		m.evologPrecomputedDescribeParent = ""
		m.evologPrecomputedDescribeChild = ""
		return m.Update(errorMsg{Err: msg.Err})
	}

	return m, nil
}
