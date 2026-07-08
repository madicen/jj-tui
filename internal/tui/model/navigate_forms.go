package model

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/state"
	bookmarktab "github.com/madicen/jj-tui/internal/tui/tabs/bookmark"
	descedittab "github.com/madicen/jj-tui/internal/tui/tabs/descedit"
	prstab "github.com/madicen/jj-tui/internal/tui/tabs/prs"
)

// navigate_forms.go holds the per-domain NavigateKind handlers for the centered
// form modals: edit-description, bookmark, PR, and ticket. Each returns
// handled=false for kinds it doesn't own (see navigate.go).

// handleNavigateDescription covers the edit-description modal and the shared
// back-to-graph teardown used by several graph-anchored modals.
func (m *Model) handleNavigateDescription(t state.NavigateTarget) (tea.Model, tea.Cmd, bool) {
	switch t.Kind {
	case state.NavigateEditDescription:
		// If we're entering edit-description from the empty-description warning, ensure the warning is closed.
		m.warningModal.Hide()
		if m.appState.Repository != nil {
			for i, c := range m.appState.Repository.Graph.Commits {
				if c.ChangeID == t.Commit.ChangeID {
					m.graphTabModel.SelectCommit(i)
					break
				}
			}
		}
		model, cmd := m.startEditingDescription(t.Commit)
		return model, cmd, true
	case state.NavigateBackToGraph:
		m.clearAIGenOverlay()
		m.clearPendingAIRetry()
		m.evologSplitModal.Hide()
		m.evologStepwiseRemainderAfterSplit = nil
		m.evologStepwiseBookmarkName = ""
		m.fileDiffModal.Hide()
		m.restoreModalUnderlayOrGraph()
		m.appState.Loading = false
		if t.StatusMessage != "" {
			m.appState.StatusMessage = t.StatusMessage
		}
		return m, nil, true
	case state.NavigateSaveDescription:
		// A second save while the first describe is still running causes parallel jj operations on the
		// same revision → divergent commits (same message, sibling children of one parent).
		if m.appState.Loading || m.aiGenOverlayActive {
			return m, nil, true
		}
		if t.SaveCommitID != "" && m.appState.JJService != nil {
			m.appState.Loading = true
			m.appState.StatusMessage = "Saving description…"
			cmd := descedittab.SaveDescriptionCmd(m.appState.JJService, t.SaveCommitID, strings.TrimSpace(t.SaveDescription))
			return m, tea.Batch(cmd, m.startBusySpinnerCmd()), true
		}
		return m, nil, true
	default:
		return m, nil, false
	}
}

// handleNavigateBookmark covers the create-bookmark modal (from graph and from a
// ticket) plus its submit.
func (m *Model) handleNavigateBookmark(t state.NavigateTarget) (tea.Model, tea.Cmd, bool) {
	switch t.Kind {
	case state.NavigateCreateBookmark:
		m.startCreateBookmark()
		return m, m.applyEffects(effLoadBranches{}), true
	case state.NavigateCreateBookmarkFromTicket:
		m.beginModalUnderlay()
		m.appState.ViewMode = state.ViewCreateBookmark
		m.appState.StatusMessage = bookmarktab.OpenCreateBookmarkFromTicket(&m.bookmarkModal, m.appState.Repository, t.TicketKey, t.TicketTitle, t.TicketDisplayKey, m.branchesTabModel.BuildBookmarkNameConflictSources(), m.appState.Config != nil && m.appState.Config.ShouldSanitizeBookmarkNames(), ModalInnerWidth(m.width))
		m.pushAIProfilesToFormModals()
		return m, nil, true
	case state.NavigateDeleteBookmark:
		// Graph tab requested a bookmark delete (P2.5); it emits the resolved name
		// and main constructs the command. Status/Loading were set by the graph
		// ApplyResult follow-up on the same frame.
		return m, bookmarktab.DeleteBookmarkCmd(m.appState.JJService, t.DeleteBookmarkName), true
	case state.NavigateSubmitBookmark:
		if m.appState.JJService != nil {
			cmd, status := bookmarktab.SubmitBookmark(&m.bookmarkModal, m.appState.Repository, m.appState.Config, m.appState.JJService)
			m.appState.StatusMessage = status
			if cmd == nil {
				return m, nil, true
			}
			m.appState.Loading = true
			// Batch the spinner tick so the busy overlay animates while jj creates the bookmark
			// and the repo reloads (cleared by applyRepositoryLoaded). Test harnesses that drain
			// cmds one message per step must expand the resulting tea.BatchMsg.
			return m, tea.Batch(cmd, m.startBusySpinnerCmd()), true
		}
		return m, nil, true
	default:
		return m, nil, false
	}
}

// handleNavigatePR covers the create-PR modal open/submit/close.
func (m *Model) handleNavigatePR(t state.NavigateTarget) (tea.Model, tea.Cmd, bool) {
	switch t.Kind {
	case state.NavigateCreatePR:
		m.startCreatePR()
		return m, nil, true
	case state.NavigateSubmitPR:
		if m.isGitHubAvailable() && m.appState.JJService != nil {
			return m, m.submitPR(), true
		}
		return m, nil, true
	case state.NavigateUpdatePR:
		// Graph tab requested pushing the selected commit's branch to its open PR
		// (P2.5); main constructs prstab.PushToPRCmd. Status/Loading were set by the
		// graph ApplyResult follow-up on the same frame.
		return m, prstab.PushToPRCmd(m.appState.JJService, t.UpdatePRBranch, t.UpdatePRCommitID, t.UpdatePRNeedsMoveBookmark, m.appState.DemoMode), true
	case state.NavigateBackFromPRForm:
		m.clearAIGenOverlay()
		m.clearPendingAIRetry()
		m.prFormModal.Hide()
		m.restoreModalUnderlayOrGraph()
		if t.StatusMessage != "" {
			m.appState.StatusMessage = t.StatusMessage
		}
		return m, nil, true
	default:
		return m, nil, false
	}
}

// handleNavigateTicket covers the create-ticket modal open/submit/close.
func (m *Model) handleNavigateTicket(t state.NavigateTarget) (tea.Model, tea.Cmd, bool) {
	switch t.Kind {
	case state.NavigateCreateTicket:
		m.startCreateTicket()
		return m, nil, true
	case state.NavigateSubmitTicket:
		return m, m.submitTicket(), true
	case state.NavigateBackFromTicketForm:
		m.clearAIGenOverlay()
		m.clearPendingAIRetry()
		m.ticketFormModal.Hide()
		if m.modalUnderlayValid {
			m.appState.ViewMode = m.modalUnderlayView
			m.modalUnderlayValid = false
		} else {
			m.appState.ViewMode = state.ViewTickets
		}
		if t.StatusMessage != "" {
			m.appState.StatusMessage = t.StatusMessage
		}
		return m, nil, true
	default:
		return m, nil, false
	}
}
