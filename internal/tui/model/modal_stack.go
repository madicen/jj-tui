package model

import "github.com/madicen/jj-tui/internal/tui/state"

// modal_stack.go introduces the ModalStack (P2.4). It is a DERIVED READ-MODEL:
// modalStack() computes the current stack of active modals — in bottom-to-top
// z-order — from the EXISTING source of truth (m.appState.ViewMode plus the
// three sub-model presence checks: initRepoModel.Path(), errorModal.GetError(),
// warningModal.IsShown()). Nothing here stores new modal-presence state.
//
// chromedSlot() and applyFormModalsOverlay() consume Top()/Has() instead of
// re-testing ViewMode/presence in priority order, so the modal z-order lives in
// exactly one place (buildModalStack). The z-order snapshot
// (chromed_slot_zorder_test.go) proves this is byte-for-byte behavior-preserving.
//
// PLAN(P2.4): the plan sketch proposed a Modal{Kind; Model tea.Model} whose
// storage would replace scattered booleans. A prior worker proved the boolean
// premise wrong: modal *presence* is genuinely driven by ViewMode + the three
// sub-model presence checks (not by removable booleans), and the sub-model
// modals don't share a tea.Model shape (their Update signatures differ). So the
// stack is a read-model over that real state rather than a new storage layer,
// and the remaining Model booleans are NOT modal-presence flags: they are
// underlay/guard/workflow state (see model_state.go) — modalUnderlayValid,
// bookmarkConflictReturnValid (underlay restore), evologDescribePreviewActive,
// absorbPreviewActive (in-graph confirm prompts, not chromed modals),
// silentReloadInFlight (background-refresh guard), chromeConsumedPress
// (mouse press/release pairing), pendingAIRetry*/aiGenReqID/aiGenOverlayActive
// (AI workflow) — and are intentionally left in place.

// ModalKind identifies a modal that can wear the window chrome / be composited
// as an overlay. The zero value ModalNone means "no modal".
type ModalKind int

const (
	ModalNone ModalKind = iota
	// Blocking overlays (highest priority; driven by sub-model presence).
	ModalInitRepo
	ModalError
	ModalWarning
	// Centered form modals (driven by ViewMode).
	ModalEditDescription
	ModalCreatePR
	ModalCreateTicket
	ModalCreateBookmark
	ModalGitHubLogin
	// Graph-overlay modals (driven by ViewMode).
	ModalBookmarkConflict
	ModalDivergent
	ModalWorkspaces
	ModalOperations
	ModalEvologSplit
	ModalFileDiff
)

// Modal is one entry in the ModalStack. Kind is the source of truth for
// priority, presence, and presentation lookups.
type Modal struct {
	Kind ModalKind
}

// ModalStack is an ordered set of active modals, bottom-to-top in z-order.
// Top() is the modal that owns the window chrome on the current frame.
type ModalStack struct {
	stack []Modal
}

// Push appends a modal to the top of the stack.
func (s *ModalStack) Push(m Modal) { s.stack = append(s.stack, m) }

// Pop removes and returns the top modal; ok is false when the stack is empty.
func (s *ModalStack) Pop() (Modal, bool) {
	if len(s.stack) == 0 {
		return Modal{}, false
	}
	top := s.stack[len(s.stack)-1]
	s.stack = s.stack[:len(s.stack)-1]
	return top, true
}

// Top returns the top modal without removing it; ok is false when empty.
func (s *ModalStack) Top() (Modal, bool) {
	if len(s.stack) == 0 {
		return Modal{}, false
	}
	return s.stack[len(s.stack)-1], true
}

// Has reports whether a modal of the given kind is anywhere in the stack.
func (s *ModalStack) Has(kind ModalKind) bool {
	for _, m := range s.stack {
		if m.Kind == kind {
			return true
		}
	}
	return false
}

// Len returns the number of modals in the stack.
func (s *ModalStack) Len() int { return len(s.stack) }

// viewModeModalKind maps the current ViewMode to the modal it selects, or
// ModalNone for the primary content views (graph/prs/tickets/branches/
// settings/help), which are not modals.
func viewModeModalKind(vm state.ViewMode) ModalKind {
	switch vm {
	case state.ViewEditDescription:
		return ModalEditDescription
	case state.ViewCreatePR:
		return ModalCreatePR
	case state.ViewCreateTicket:
		return ModalCreateTicket
	case state.ViewCreateBookmark:
		return ModalCreateBookmark
	case state.ViewGitHubLogin:
		return ModalGitHubLogin
	case state.ViewBookmarkConflict:
		return ModalBookmarkConflict
	case state.ViewDivergentCommit:
		return ModalDivergent
	case state.ViewWorkspaces:
		return ModalWorkspaces
	case state.ViewOperations:
		return ModalOperations
	case state.ViewEvologSplit:
		return ModalEvologSplit
	case state.ViewFileDiff:
		return ModalFileDiff
	default:
		return ModalNone
	}
}

// modalStack derives the current modal stack from the live source of truth.
// Bottom-to-top order encodes the historical chromedSlot() priority:
//
//	(ViewMode-selected modal) < warning < error < initrepo
//
// so Top() resolves the same winner the old priority-ordered if-chain did:
// initrepo > error > warning > ViewMode-selected overlay. The ViewMode modal is
// pushed even when a blocking overlay also shows, so Has(kind) still reports the
// underlying form modal (matching applyFormModalsOverlay, which paints a
// non-chromed form modal behind an active error/warning chrome).
func (m *Model) modalStack() ModalStack {
	var s ModalStack
	if k := viewModeModalKind(m.appState.ViewMode); k != ModalNone {
		s.Push(Modal{Kind: k})
	}
	if m.warningModal.IsShown() {
		s.Push(Modal{Kind: ModalWarning})
	}
	if m.errorModal.GetError() != nil {
		s.Push(Modal{Kind: ModalError})
	}
	if m.initRepoModel.Path() != "" {
		s.Push(Modal{Kind: ModalInitRepo})
	}
	return s
}

// topModalKind returns the chrome-owning modal kind for this frame (ModalNone
// when nothing is chromed).
func (m *Model) topModalKind() ModalKind {
	s := m.modalStack()
	if top, ok := s.Top(); ok {
		return top.Kind
	}
	return ModalNone
}
