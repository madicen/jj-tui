package model

// model_state.go holds the root Model struct definition. It is kept separate
// from model.go so the concrete-tab field types (which force the tab-package
// imports) live here rather than in model.go. model.go's Update/View dispatch
// routes the primary tabs generically through the tab.Tab registry and the
// hook interfaces, so model.go itself no longer needs to import the concrete
// tab packages; construction (New) and the field types stay confined to this
// file and init.go.

import (
	"context"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	overlay "github.com/madicen/bubble-overlay"
	"github.com/madicen/jj-tui/internal/tui/keys"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/tab"
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
	helptab "github.com/madicen/jj-tui/internal/tui/tabs/help"
	initrepotab "github.com/madicen/jj-tui/internal/tui/tabs/initrepo"
	operationstab "github.com/madicen/jj-tui/internal/tui/tabs/operations"
	prformtab "github.com/madicen/jj-tui/internal/tui/tabs/prform"
	prstab "github.com/madicen/jj-tui/internal/tui/tabs/prs"
	settingstab "github.com/madicen/jj-tui/internal/tui/tabs/settings"
	ticketformtab "github.com/madicen/jj-tui/internal/tui/tabs/ticketform"
	ticketstab "github.com/madicen/jj-tui/internal/tui/tabs/tickets"
	warningtab "github.com/madicen/jj-tui/internal/tui/tabs/warning"
	workspacestab "github.com/madicen/jj-tui/internal/tui/tabs/workspaces"
)

// Model is the main TUI model using bubblezone for mouse handling.
// All clickable elements are wrapped with zone.Mark() in the View.
// Mouse events are handled via zone.MsgZoneInBounds messages.
type Model struct {
	ctx         context.Context
	zoneManager *zone.Manager
	appState    state.AppState // Shared state and services; submodels receive &appState

	// keys holds the global (always-available) keybindings. Tabs hold their own
	// scoped KeyMaps; see internal/tui/keys.
	keys keys.GlobalKeyMap

	// Dimensions (main only)
	width  int
	height int
	// When ViewBookmarkConflict is open: tab to show under the overlay and restore on close/resolve.
	bookmarkConflictReturnValid bool
	bookmarkConflictReturnView  state.ViewMode
	// When a centered form modal is open (edit description, PR/ticket forms, bookmark, GitHub login): tab
	// content and tab bar highlight use this; ViewMode stays the modal for input routing.
	modalUnderlayValid bool
	modalUnderlayView  state.ViewMode
	// Selection state lives in tab models: graph (commit/file), prs, tickets, branches
	redoOperationID string
	// Undo-hint state (P5.4): after a mutating jj command, pendingUndoHint drives a
	// cheap `jj op log --limit 1` fetch once the repo reloads; the result is shown
	// as "Ctrl+z undoes: <op>" in the status bar for undoHintDuration. undoHintSeq
	// invalidates stale expiry ticks so a newer hint isn't cleared early.
	pendingUndoHint bool
	undoHint        string
	undoHintSeq     int
	// Silent background graph refresh (handleTickMsg) runs concurrently per Bubble Tea Batch;
	// without this guard, overlapping GetRepository calls can retain multi-copy graphs and spike RSS.
	silentReloadInFlight bool
	// Monotonic id for optional LLM requests; stale responses are ignored.
	aiGenReqID int
	// aiGenOverlayActive shows the centered spinner while Generate*Cmd runs (form modals + description editor).
	aiGenOverlayActive bool
	// pendingAIRetryKind is the most recently dispatched AI generation kind (NavigateGenerateCommitDescription,
	// NavigateGeneratePRForm, NavigateGenerateBookmarkName, NavigateGenerateTicketForm). When the cmd reports an
	// error, the error modal shows Retry (^r) and clicking it replays this kind via handleNavigate so the user
	// keeps the open form modal and any text they had typed. Cleared on success or dismiss.
	pendingAIRetryKind   state.NavigateKind
	pendingAIRetryActive bool
	// pendingAIRetryOverrideProfile preserves the long-press menu's one-shot profile selection so a retry
	// after a transient error uses the same model the user picked. Empty = retry with active profile.
	pendingAIRetryOverrideProfile string
	// pendingRetryCmd is the generic (non-AI) replay target for the error modal's Retry (^r) button.
	// P5.5: any retryable operation that surfaces a failure via effShowRetryableError stashes the exact
	// command that reproduces the attempt (e.g. re-load PRs, re-load tickets, re-push bookmarks) here so
	// NavigateRetryError can re-run it verbatim. Cleared on retry, on dismiss, and whenever a plain
	// (non-retryable) effShowError fires so a stale command can never replay against an unrelated error.
	pendingRetryCmd tea.Cmd

	// Tab-specific models (own all tab/modal state; main model does not duplicate)
	graphTabModel    graphtab.GraphModel
	prsTabModel      prstab.Model
	branchesTabModel branchestab.Model
	ticketsTabModel  ticketstab.Model
	settingsTabModel settingstab.Model
	helpTabModel     helptab.Model

	// tabRegistry / tabOrder hold the six primary content tabs behind the
	// tab.Tab interface (via the adapters in tab_adapters.go). The root's
	// Update/View dispatch iterates the registry and forwards messages
	// generically instead of naming each concrete field, so model.go no longer
	// imports the concrete tab packages. The concrete fields above remain the
	// source of truth for the accessors and effect dispatcher (which live in
	// sibling files that may import the concrete packages). Populated once in
	// New(); safe because *Model is never copied.
	tabRegistry map[state.ViewMode]tab.Tab
	tabOrder    []state.ViewMode

	// Modal models (dialogs and modals)
	initRepoModel    initrepotab.Model
	errorModal       errortab.Model
	warningModal     warningtab.Model
	conflictModal    conflicttab.Model
	divergentModal   divergenttab.Model
	workspacesModal  workspacestab.Model
	operationsModal  operationstab.Model
	evologSplitModal evologsplittab.Model
	// evologPostSplitDescribe is set when the user confirms split with “AI describe after split”; cleared after describe runs or on graph return.
	evologPostSplitDescribe bool
	// evologStepwiseRemainderAfterSplit: after an intermediate stepwise FAQ split, reload evolog without closing the modal.
	evologStepwiseRemainderAfterSplit []string
	evologStepwiseBookmarkName        string
	// Post-split AI describe preview (y apply / n discard).
	evologDescribePreviewActive   bool
	evologDescribePreviewFromPlan bool // true when text came from suggest-phase chain preview (not post-split LLM)
	evologDescribeSkipParent      bool // @- immutable: apply only describes @
	evologDescribeParent          string
	evologDescribeChild           string
	// evologPrecomputedDescribe* come from AI suggest (chain preview); used when describe-after-split runs without a second LLM.
	evologPrecomputedDescribeParent string
	evologPrecomputedDescribeChild  string
	// Absorb dry-run preview (y confirm / n discard). Populated from a
	// non-mutating `jj absorb --no-integrate-operation` before the real absorb.
	absorbPreviewActive  bool
	absorbPreviewSummary string
	fileDiffModal        filedifftab.Model
	bookmarkModal        bookmarktab.Model
	prFormModal          prformtab.Model
	ticketFormModal      ticketformtab.Model
	desceditModal        descedittab.Model
	githubLoginModel     githublogintab.Model

	busySpinner spinner.Model

	// chrome routes draggable window chrome for the active modal (see window_chrome.go).
	chrome overlay.Window
	// chromeConsumedPress is set when window chrome consumed a mouse press (e.g. the
	// [x] close button, the title-bar drag handle, or a resize edge). The MouseMsg
	// handler uses it to swallow the matching release when chrome doesn't claim it
	// itself — without this, a [x] click closes the modal on press, then the release
	// leaks through to the underlay zones (e.g. the graph tab's "split" button) and
	// fires whatever happens to sit beneath the close button. Drag/resize releases
	// stay handled by chrome.Update, so the flag only kicks in when chrome's release
	// path didn't engage (i.e. the press triggered a Pop/close).
	chromeConsumedPress bool
	// lastFileDiffDimsSeq snapshots filediff.Model.DimensionsSeq() from the last
	// frame. The file diff modal auto-sizes to its current patch, but bubble-
	// overlay's chrome locks ContentWidth/ContentHeight after the first frame
	// (see InitLayerContentSize). Comparing the sequence per frame lets View()
	// nudge the chrome to re-seed only when our natural size actually changed
	// (load complete, terminal resize), so user drag/resize/minimize state on
	// quiet frames isn't clobbered by an unconditional reset.
	lastFileDiffDimsSeq int
	// lastEvologContentSig is the "WxH" signature of the evolog split modal's
	// rendered content on the last frame. The modal opens on a tiny "Loading
	// jj evolog…" placeholder and only grows to full size once the evolog
	// loads (and again when an AI plan adds lines), but chrome locks its
	// content size on the first frame — so without re-seeding it stays stuck
	// at the placeholder's height ("mostly minimized"). Comparing the signature
	// per frame lets View() re-seed only when the natural size actually changed.
	lastEvologContentSig string
}

// doPollMsg is a message used to trigger a GitHub token poll.
type doPollMsg struct{}
