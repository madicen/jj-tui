package model

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/tui/state"
	aitab "github.com/madicen/jj-tui/internal/tui/ai"
)

// navigate_ai.go holds the AI-generation NavigateKind handler (commit
// description, PR form, bookmark name, ticket form). Each case shares the same
// bookkeeping — config gate, monotonic request id, spinner overlay, and retry
// target — factored into aiNotConfigured / beginAIGen so the individual cases
// stay small and identical in behavior to the pre-split switch.

// aiNotConfigured reports whether AI generation is unavailable (no config or AI
// disabled / missing key).
func (m *Model) aiNotConfigured() bool {
	return m.appState.Config == nil || !m.appState.Config.AIConfiguredForGeneration()
}

// aiNotConfiguredResult sets the "enable AI" hint and returns a handled result.
func (m *Model) aiNotConfiguredResult() (tea.Model, tea.Cmd, bool) {
	m.appState.StatusMessage = fmt.Sprintf("Enable AI in Settings → AI and set an API key (or %s)", config.EnvAIAPIKey)
	return m, nil, true
}

// beginAIGen advances the request id, sets the generating status + overlay, and
// arms the retry target. It returns the new request id and the spinner-start cmd
// to batch with the provider command.
func (m *Model) beginAIGen(kind state.NavigateKind, t state.NavigateTarget, override *config.AIProfile, statusBase string) (int, tea.Cmd) {
	m.aiGenReqID++
	rid := m.aiGenReqID
	m.appState.StatusMessage = aiGenStatusMessage(statusBase, override)
	m.aiGenOverlayActive = true
	m.pendingAIRetryKind = kind
	m.pendingAIRetryActive = true
	m.pendingAIRetryOverrideProfile = t.AIOverrideProfile
	return rid, m.startBusySpinnerCmd()
}

// handleNavigateAI covers the four AI-generation navigations.
func (m *Model) handleNavigateAI(t state.NavigateTarget) (tea.Model, tea.Cmd, bool) {
	switch t.Kind {
	case state.NavigateGenerateCommitDescription:
		if m.aiNotConfigured() {
			return m.aiNotConfiguredResult()
		}
		changeID := m.desceditModal.GetEditingCommitID()
		if changeID == "" {
			return m, nil, true
		}
		override := m.resolveAIOverride(t)
		rid, spin := m.beginAIGen(state.NavigateGenerateCommitDescription, t, override, "Generating description…")
		return m, tea.Batch(
			aitab.GenerateCommitDescriptionCmd(rid, m.appState.JJService, m.appState.Config, changeID, m.desceditModal.GetCommitShortID(), m.desceditModal.GetDescriptionValue(), override),
			spin,
		), true
	case state.NavigateGeneratePRForm:
		if m.aiNotConfigured() {
			return m.aiNotConfiguredResult()
		}
		repo := m.appState.Repository
		idx := m.prFormModal.GetCommitIndex()
		if repo == nil || idx < 0 || idx >= len(repo.Graph.Commits) {
			return m, nil, true
		}
		changeID := repo.Graph.Commits[idx].ChangeID
		override := m.resolveAIOverride(t)
		rid, spin := m.beginAIGen(state.NavigateGeneratePRForm, t, override, "Generating PR title and body…")
		return m, tea.Batch(
			aitab.GeneratePRFormCmd(rid, m.appState.JJService, m.appState.Config, changeID, m.prFormModal.GetBaseBranch(), m.prFormModal.GetHeadBranch(), m.prFormModal.GetTitle(), override),
			spin,
		), true
	case state.NavigateGenerateBookmarkName:
		if m.aiNotConfigured() {
			return m.aiNotConfiguredResult()
		}
		repo := m.appState.Repository
		idx := m.bookmarkModal.GetCommitIdx()
		rev := "@"
		if repo != nil && idx >= 0 && idx < len(repo.Graph.Commits) {
			rev = repo.Graph.Commits[idx].ChangeID
		}
		hint := ""
		if m.bookmarkModal.IsFromJira() {
			hint = strings.TrimSpace(m.bookmarkModal.GetJiraKey() + " " + m.bookmarkModal.GetJiraTicketTitle())
		}
		override := m.resolveAIOverride(t)
		rid, spin := m.beginAIGen(state.NavigateGenerateBookmarkName, t, override, "Generating bookmark name…")
		return m, tea.Batch(
			aitab.GenerateBookmarkNameCmd(rid, m.appState.JJService, m.appState.Config, rev, hint, override),
			spin,
		), true
	case state.NavigateGenerateTicketForm:
		if m.aiNotConfigured() {
			return m.aiNotConfiguredResult()
		}
		repo := m.appState.Repository
		idx := m.GetSelectedCommit()
		changeID := "@"
		changeShort := "@"
		if repo != nil && idx >= 0 && idx < len(repo.Graph.Commits) {
			c := repo.Graph.Commits[idx]
			changeID = c.ChangeID
			if strings.TrimSpace(c.ShortID) != "" {
				changeShort = strings.TrimSpace(c.ShortID)
			} else {
				changeShort = changeID
			}
		}
		override := m.resolveAIOverride(t)
		rid, spin := m.beginAIGen(state.NavigateGenerateTicketForm, t, override, "Generating ticket title and description…")
		return m, tea.Batch(
			aitab.GenerateTicketFormCmd(rid, m.appState.JJService, m.appState.Config, changeID, changeShort, m.ticketFormModal.GetSummary(), m.ticketFormModal.GetDescription(), override),
			spin,
		), true
	default:
		return m, nil, false
	}
}
