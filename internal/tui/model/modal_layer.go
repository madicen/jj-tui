package model

import (
	tea "github.com/charmbracelet/bubbletea"
	overlay "github.com/madicen/bubble-overlay"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/tabs/githublogin"
)

// chromedSlot returns the modal that should wear bubble-overlay window chrome
// on this frame: a stable key (drives Window's auto-recenter on switch), the
// rendered modal view, the tab title, and the close command fired on [x] or
// chrome-Esc. Empty key = nothing chromed; key wins by priority (init >
// error > warning > active form/view-mode modal). Each branch reuses the
// modal's existing Navigate* close path so close-via-tab and close-via-Esc
// converge on the same teardown.
func (m *Model) chromedSlot() (key, content, title string, closeCmd tea.Cmd) {
	if m.initRepoModel.Path() != "" {
		return "initrepo", m.initRepoModel.View(), "Initialize repository",
			state.NavigateTarget{Kind: state.NavigateDismissInit, StatusMessage: "Init cancelled"}.Cmd()
	}
	if m.errorModal.GetError() != nil {
		return "error", m.errorModal.View(), "Error",
			state.NavigateTarget{Kind: state.NavigateDismissError}.Cmd()
	}
	if m.warningModal.IsShown() {
		// Warning's title is set per-warning (e.g. "Empty commit description"),
		// so surface it in the chrome tab instead of the generic "Warning".
		title := m.warningModal.GetTitle()
		if title == "" {
			title = "Warning"
		}
		return "warning", m.warningModal.View(), title,
			state.NavigateTarget{Kind: state.NavigateWarningCancel, StatusMessage: "Warning dismissed"}.Cmd()
	}
	switch m.appState.ViewMode {
	case state.ViewEditDescription:
		return "descedit", m.desceditModal.View(), "Edit description",
			state.NavigateTarget{Kind: state.NavigateBackToGraph, StatusMessage: "Edit description cancelled"}.Cmd()
	case state.ViewCreatePR:
		return "pr", m.prFormModal.View(), "Create pull request",
			state.NavigateTarget{Kind: state.NavigateBackFromPRForm, StatusMessage: "Create PR cancelled"}.Cmd()
	case state.ViewCreateTicket:
		return "ticket", m.ticketFormModal.View(), "Create ticket",
			state.NavigateTarget{Kind: state.NavigateBackFromTicketForm, StatusMessage: "Create ticket cancelled"}.Cmd()
	case state.ViewCreateBookmark:
		return "bookmark", m.bookmarkModal.View(), "Create bookmark",
			state.NavigateTarget{Kind: state.NavigateBackToGraph, StatusMessage: "Create bookmark cancelled"}.Cmd()
	case state.ViewGitHubLogin:
		// Differentiate device flow from gh-cli flow in the titlebar so the
		// user can tell at a glance which side of the login they're on.
		title := "GitHub login"
		if m.githubLoginModel.Mode() == githublogin.LoginModeGhCLI {
			title = "GitHub CLI login"
		}
		return "githublogin", m.githubLoginModel.View(), title,
			state.NavigateTarget{Kind: state.NavigateGitHubLoginCancel, StatusMessage: "GitHub login cancelled"}.Cmd()
	case state.ViewBookmarkConflict:
		// Include the diverged bookmark's name in the title so consumers can
		// tell which bookmark the modal is resolving without an in-content
		// header line.
		title := "Bookmark conflict"
		if name := m.conflictModal.GetBookmarkName(); name != "" {
			title = "Bookmark conflict: " + name
		}
		return "conflict", m.conflictModal.View(), title,
			state.NavigateTarget{Kind: state.NavigateCloseBookmarkConflict, StatusMessage: "Bookmark conflict review cancelled"}.Cmd()
	case state.ViewDivergentCommit:
		// No NavigateCloseDivergent today; the modal already handles Esc, so
		// synthesise it for chrome [x].
		return "divergent", m.divergentModal.View(), "Divergent commit",
			func() tea.Msg { return tea.KeyMsg{Type: tea.KeyEsc} }
	case state.ViewEvologSplit:
		return "evolog", m.evologSplitModal.View(), "Evolog split",
			state.NavigateTarget{Kind: state.NavigateBackToGraph, StatusMessage: "Evolog split cancelled"}.Cmd()
	case state.ViewFileDiff:
		// Callers (e.g. evolog split) can set a context-specific overlay
		// title; the chrome tab mirrors it, falling back to "File diff".
		title := m.fileDiffModal.OverlayTitle()
		if title == "" {
			title = "File diff"
		}
		return "filediff", m.fileDiffModal.View(), title,
			state.NavigateTarget{Kind: state.NavigateCloseFileDiff}.Cmd()
	}
	return "", "", "", nil
}

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

// applyFormModalsOverlay composites the form modals (init repo, descedit /
// PR / ticket / bookmark forms, GitHub login) over the main layout. The
// caller passes the chromedSlot key so the modal that owns the WindowChrome
// is skipped here — View() composites that one last, after the chrome wrap,
// so the tab/drag handle sits above every other layer. A non-chromed form
// modal (e.g. PR form sitting under an active error chrome) still gets a
// centered render here so it remains visible behind the chromed slot.
//
// The long-press AI profile popover is intentionally NOT painted here:
// View() composites it after chrome.View so it lands on top of the
// dragged window (see applyGenMenuOverlay).
func (m *Model) applyFormModalsOverlay(fullView, skipKey string) string {
	if skipKey != "initrepo" && m.initRepoModel.Path() != "" {
		if content := m.initRepoModel.View(); content != "" {
			fullView = applyBubbleOverlayCentered(fullView, FrameFormModal(content, m.width), m.width, m.height)
		}
	}
	if skipKey != "githublogin" && m.appState.ViewMode == state.ViewGitHubLogin {
		if content := m.githubLoginModel.View(); content != "" {
			fullView = applyBubbleOverlayCentered(fullView, FrameFormModal(content, m.width), m.width, m.height)
		}
	}
	if skipKey != "descedit" && m.appState.ViewMode == state.ViewEditDescription {
		if content := m.desceditModal.View(); content != "" {
			fullView = applyBubbleOverlayCentered(fullView, FrameFormModal(content, m.width), m.width, m.height)
		}
	}
	if skipKey != "pr" && m.appState.ViewMode == state.ViewCreatePR {
		if content := m.prFormModal.View(); content != "" {
			fullView = applyBubbleOverlayCentered(fullView, FrameFormModal(content, m.width), m.width, m.height)
		}
	}
	if skipKey != "ticket" && m.appState.ViewMode == state.ViewCreateTicket {
		if content := m.ticketFormModal.View(); content != "" {
			fullView = applyBubbleOverlayCentered(fullView, FrameFormModal(content, m.width), m.width, m.height)
		}
	}
	if skipKey != "bookmark" && m.appState.ViewMode == state.ViewCreateBookmark {
		if content := m.bookmarkModal.View(); content != "" {
			fullView = applyBubbleOverlayCentered(fullView, FrameFormModal(content, m.width), m.width, m.height)
		}
	}
	return fullView
}

// applyGenMenuOverlay composites the long-press AI profile popover anchored
// at the press point on the originating generate chip. Painted by View()
// after chrome.View so the menu sits on top of the (possibly dragged)
// window — without this ordering the popover renders underneath the chrome
// body and becomes unreachable.
func (m *Model) applyGenMenuOverlay(fullView string) string {
	menuView, mx, my := m.activeFormModalGenMenuOverlay()
	if menuView == "" || m.width <= 0 || m.height <= 0 {
		return fullView
	}
	return overlay.OverlayViewAtPoint(fullView, menuView, m.width, m.height, my, mx)
}
