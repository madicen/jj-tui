package model

import (
	overlay "github.com/madicen/bubble-overlay"
	"github.com/madicen/jj-tui/internal/tui/state"
)

// layoutContentMode selects which tab body to paint under centered form modals and init-repo overlay.
func (m *Model) layoutContentMode() state.ViewMode {
	if m.initRepoModel.Path() != "" {
		return state.ViewCommitGraph
	}
	switch m.appState.ViewMode {
	case state.ViewEditDescription, state.ViewCreatePR, state.ViewCreateTicket, state.ViewCreateBookmark, state.ViewGitHubLogin:
		if m.modalUnderlayValid {
			return m.modalUnderlayView
		}
		return state.ViewCommitGraph
	case state.ViewBookmarkConflict:
		if m.bookmarkConflictReturnValid {
			switch m.bookmarkConflictReturnView {
			case state.ViewBranches:
				return state.ViewBranches
			case state.ViewCommitGraph:
				return state.ViewCommitGraph
			default:
				return state.ViewCommitGraph
			}
		}
		return state.ViewCommitGraph
	case state.ViewDivergentCommit, state.ViewEvologSplit, state.ViewFileDiff:
		return state.ViewCommitGraph
	default:
		return m.appState.ViewMode
	}
}

// tabHighlightMode selects which tab appears active in the header (under form modals, show prior tab).
func (m *Model) tabHighlightMode() state.ViewMode {
	if m.initRepoModel.Path() != "" {
		return state.ViewCommitGraph
	}
	vm := m.appState.ViewMode
	switch vm {
	case state.ViewEditDescription, state.ViewCreatePR, state.ViewCreateTicket, state.ViewCreateBookmark, state.ViewGitHubLogin:
		if m.modalUnderlayValid {
			return m.modalUnderlayView
		}
	case state.ViewBookmarkConflict:
		if m.bookmarkConflictReturnValid {
			return m.bookmarkConflictReturnView
		}
	case state.ViewDivergentCommit, state.ViewEvologSplit, state.ViewFileDiff:
		return state.ViewCommitGraph
	}
	return vm
}

// isFormModalView is true for centered form/login modals (not init-repo path, not graph-only pickers).
func (m *Model) isFormModalView() bool {
	switch m.appState.ViewMode {
	case state.ViewEditDescription, state.ViewCreatePR, state.ViewCreateTicket, state.ViewCreateBookmark, state.ViewGitHubLogin:
		return true
	default:
		return false
	}
}

// applyFormModalsOverlay composites centered dialogs (init repo, forms, GitHub login) over the main layout.
func (m *Model) applyFormModalsOverlay(fullView string) string {
	if m.initRepoModel.Path() != "" {
		if content := m.initRepoModel.View(); content != "" {
			fullView = applyBubbleOverlayCentered(fullView, FrameFormModal(content, m.width), m.width, m.height)
		}
	}
	if m.appState.ViewMode == state.ViewGitHubLogin {
		if content := m.githubLoginModel.View(); content != "" {
			fullView = applyBubbleOverlayCentered(fullView, FrameFormModal(content, m.width), m.width, m.height)
		}
	}
	if m.appState.ViewMode == state.ViewEditDescription {
		if content := m.desceditModal.View(); content != "" {
			fullView = applyBubbleOverlayCentered(fullView, FrameFormModal(content, m.width), m.width, m.height)
		}
	}
	if m.appState.ViewMode == state.ViewCreatePR {
		if content := m.prFormModal.View(); content != "" {
			fullView = applyBubbleOverlayCentered(fullView, FrameFormModal(content, m.width), m.width, m.height)
		}
	}
	if m.appState.ViewMode == state.ViewCreateTicket {
		if content := m.ticketFormModal.View(); content != "" {
			fullView = applyBubbleOverlayCentered(fullView, FrameFormModal(content, m.width), m.width, m.height)
		}
	}
	if m.appState.ViewMode == state.ViewCreateBookmark {
		if content := m.bookmarkModal.View(); content != "" {
			fullView = applyBubbleOverlayCentered(fullView, FrameFormModal(content, m.width), m.width, m.height)
		}
	}
	// Long-press AI profile popover (descedit / PR / bookmark / ticket forms).
	// Drawn after the modal so it overlays the generate chip at the click point.
	if menuView, mx, my := m.activeFormModalGenMenuOverlay(); menuView != "" && m.width > 0 && m.height > 0 {
		fullView = overlay.OverlayViewAtPoint(fullView, menuView, m.width, m.height, my, mx)
	}
	return fullView
}
