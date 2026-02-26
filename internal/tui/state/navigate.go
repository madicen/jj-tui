package state

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
)

// NavigateKind is the discriminant for a navigation request from a submodel.
type NavigateKind int

const (
	NavigateEditDescription NavigateKind = iota
	NavigateCreateBookmark
	NavigateCreateBookmarkFromTicket
	NavigateWarning
	NavigateCreatePR
	NavigateBackToGraph
	NavigateBackToBranches
	NavigateBackToSettings
	NavigateDismissError
	NavigateDismissInit
	NavigateGitHubLoginCancel
	// Modal "perform" actions: main runs the cmd or applies the effect.
	NavigateSaveDescription
	NavigateSubmitBookmark
	NavigateSubmitPR
	NavigateResolveConflict
	NavigateResolveDivergent
	NavigateWarningCancel
	NavigateRunInit
	NavigateDismissErrorAndRefresh
	NavigateBackFromPRForm // back to graph and hide PR form modal
)

// NavigateTarget describes a navigation request. Only main can perform these
// (it owns modals and cross-tab state). Submodels set AppState.ViewMode/StatusMessage
// and return tea.Cmd for everything else.
type NavigateTarget struct {
	Kind NavigateKind

	// Payloads for specific kinds (only one set per kind).
	Commit            internal.Commit
	WarningTitle      string
	WarningMessage    string
	WarningCommits    []internal.Commit
	TicketKey         string
	TicketTitle       string
	TicketDisplayKey  string
	StatusMessage     string
	ClearError        bool
	ClearInit         bool
	// Save description
	SaveCommitID   string
	SaveDescription string
	// Resolve conflict
	ConflictBookmarkName string
	ConflictResolution  string
	// Resolve divergent
	DivergentChangeID   string
	DivergentKeepCommitID string
	// Dismiss error and then run refresh
	RefreshAfterDismiss bool
}

// NavigateMsg is the only callback from submodels to main: they request a view change or
// modal open that only main can perform. Submodels set AppState (ViewMode, StatusMessage)
// and return tea.Cmd for everything else.
type NavigateMsg struct {
	Target NavigateTarget
}

// Cmd returns a tea.Cmd that sends this navigation to the main model.
func (t NavigateTarget) Cmd() tea.Cmd {
	return func() tea.Msg { return NavigateMsg{Target: t} }
}
