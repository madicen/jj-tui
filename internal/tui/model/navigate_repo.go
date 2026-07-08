package model

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/data"
	"github.com/madicen/jj-tui/internal/tui/state"
)

// navigate_repo.go holds the per-domain NavigateKind handlers for repository /
// remote setup (init, origin remote, push) and the error-modal dismiss/retry
// paths, plus the GitHub-login cancel teardown.

// handleNavigateInit covers running and dismissing the init-repository modal.
func (m *Model) handleNavigateInit(t state.NavigateTarget) (tea.Model, tea.Cmd, bool) {
	switch t.Kind {
	case state.NavigateRunInit:
		m.appState.Loading = true
		switch {
		case t.InitGhCreateRepo:
			m.appState.StatusMessage = "Initializing repository and creating GitHub repo…"
		case strings.TrimSpace(t.InitRemoteURL) != "":
			m.appState.StatusMessage = "Initializing repository and adding remote…"
		default:
			m.appState.StatusMessage = "Initializing repository…"
		}
		opts := data.InitOptions{
			Colocate:      t.InitColocate,
			RemoteURL:     t.InitRemoteURL,
			GhCreateRepo:  t.InitGhCreateRepo,
			GhRepoName:    t.InitGhRepoName,
			GhRepoPrivate: t.InitGhRepoPrivate,
		}
		return m, tea.Batch(data.RunJJInit(opts), m.startBusySpinnerCmd()), true
	case state.NavigateDismissInit:
		m.initRepoModel.SetPath("")
		m.appState.ViewMode = state.ViewCommitGraph
		if t.StatusMessage != "" {
			m.appState.StatusMessage = t.StatusMessage
		}
		return m, m.tickCmd(), true
	default:
		return m, nil, false
	}
}

// handleNavigateRemote covers origin-remote configuration, GitHub repo creation,
// bookmark push, and the GitHub-login cancel teardown (all reached from
// Settings → GitHub).
func (m *Model) handleNavigateRemote(t state.NavigateTarget) (tea.Model, tea.Cmd, bool) {
	switch t.Kind {
	case state.NavigateGitHubLoginCancel:
		m.githubLoginModel.ClearFlow()
		m.clearModalUnderlay()
		m.appState.ViewMode = state.ViewSettings
		if t.StatusMessage != "" {
			m.appState.StatusMessage = t.StatusMessage
		}
		return m, nil, true
	case state.NavigateRemoteApply:
		url := strings.TrimSpace(t.RemoteURL)
		if url == "" {
			// Empty URL on a tab where origin is already configured is a "I cleared the field
			// to remove origin" intent; route to remove instead so the user doesn't have to
			// remember the Ctrl+x shortcut.
			gh := m.settingsTabModel.GetGitHubModel()
			if gh.GetCurrentOrigin() != "" {
				return m, data.RemoveOriginCmd(m.appState.JJService), true
			}
			m.appState.StatusMessage = "Enter a remote URL first"
			return m, nil, true
		}
		m.appState.Loading = true
		m.appState.StatusMessage = "Configuring origin remote…"
		return m, tea.Batch(data.ApplyOriginCmd(m.appState.JJService, url), m.startBusySpinnerCmd()), true
	case state.NavigateRemoteCreateGh:
		m.appState.Loading = true
		m.appState.StatusMessage = "Creating GitHub repository…"
		// Repo name is implicitly the current working directory; the data layer derives it from
		// filepath.Base when name is empty so we don't need to plumb it through here.
		return m, tea.Batch(data.CreateGhRepoCmd(m.appState.JJService, "", t.RemoteRepoPrivate), m.startBusySpinnerCmd()), true
	case state.NavigateRemoteRemove:
		m.appState.Loading = true
		m.appState.StatusMessage = "Removing origin remote…"
		return m, tea.Batch(data.RemoveOriginCmd(m.appState.JJService), m.startBusySpinnerCmd()), true
	case state.NavigatePushBookmarks:
		m.appState.Loading = true
		if t.PushAll {
			m.appState.StatusMessage = "Pushing all bookmarks to origin…"
		} else {
			m.appState.StatusMessage = "Pushing current bookmark to origin…"
		}
		return m, tea.Batch(data.PushBookmarksCmd(m.appState.JJService, t.PushAll), m.startBusySpinnerCmd()), true
	default:
		return m, nil, false
	}
}

// handleNavigateError covers dismissing the error modal and the AI-replay retry.
func (m *Model) handleNavigateError(t state.NavigateTarget) (tea.Model, tea.Cmd, bool) {
	switch t.Kind {
	case state.NavigateDismissError:
		m.errorModal.ClearError()
		m.clearPendingAIRetry()
		// If a form modal (Edit Description, PR/Ticket/Bookmark, GitHub login) is open, keep it
		// open after dismissing the error. Previously we forced ViewMode back to the graph,
		// which silently discarded whatever the user had typed. Errors triggered from these
		// views are typically AI/network failures the user wants to recover from inline.
		if !m.isFormModalView() {
			m.appState.ViewMode = state.ViewCommitGraph
		}
		if t.StatusMessage != "" {
			m.appState.StatusMessage = t.StatusMessage
		}
		if t.RefreshAfterDismiss {
			return m, m.refreshRepository(), true
		}
		return m, m.tickCmd(), true
	case state.NavigateRetryError:
		// If we have a saved AI replay target, clear the modal and re-dispatch the same
		// NavigateGenerate* request via handleNavigate. The form modal underneath stays open
		// so the user keeps any text they typed, and the spinner overlay flips back on.
		if m.pendingAIRetryActive {
			m.errorModal.ClearError()
			retryKind := m.pendingAIRetryKind
			retryOverride := m.pendingAIRetryOverrideProfile
			// pendingAIRetryActive will be set again by the NavigateGenerate* handler.
			m.pendingAIRetryActive = false
			model, cmd := m.handleNavigate(state.NavigateTarget{Kind: retryKind, AIOverrideProfile: retryOverride})
			return model, cmd, true
		}
		// No replayable action: fall back to the legacy behavior of dismissing and refreshing
		// the repository. Today the Retry button is hidden in this case (errortab.HasRetry is
		// false), so this branch is only reached if the user binds ctrl+r elsewhere.
		m.errorModal.ClearError()
		if !m.isFormModalView() {
			m.appState.ViewMode = state.ViewCommitGraph
		}
		return m, m.refreshRepository(), true
	default:
		return m, nil, false
	}
}
