package model

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/data"
	branchestab "github.com/madicen/jj-tui/internal/tui/tabs/branches"
	prstab "github.com/madicen/jj-tui/internal/tui/tabs/prs"
)

// effect is a cross-cutting intention produced by a message handler and applied
// by the Model's single dispatcher (applyEffects). It exists to move the
// orchestration that touches OTHER components' Model-private state (the error
// modal, the bookmark modal, cross-tab reconciliation, follow-up loads) out of
// each inline case and into one place.
//
// PLAN(P2.3): this is the prerequisite for the Tab interface migration. Once a
// cross-cutting operation is expressed as an effect value, a tab's own Update
// can surface it without reaching into the root Model. Migrating handlers to
// return effects is behavior-preserving: applyEffect does exactly what the
// inline code did before.
type effect interface{ isEffect() }

// effShowError routes an error into the shared error modal (no Retry offered).
type effShowError struct{ err error }

func (effShowError) isEffect() {}

// effShowRetryableError routes an error into the shared error modal AND offers a Retry (^r)
// button that replays the exact command that produced the failure. P5.5: use this for any
// transient/network operation the user can simply re-run (PR/ticket API loads, bookmark push).
// retry must be the self-contained command that reproduces the attempt; it is stashed on the
// Model and re-run by NavigateRetryError. A nil retry degrades to a plain error (no Retry).
type effShowRetryableError struct {
	err   error
	retry tea.Cmd
}

func (effShowRetryableError) isEffect() {}

// effClearError dismisses whatever is currently in the error modal.
type effClearError struct{}

func (effClearError) isEffect() {}

// effResolveOpenPRs kicks off targeted per-bookmark open-PR lookups for local
// bookmarks that the bulk PR list did not match.
type effResolveOpenPRs struct{}

func (effResolveOpenPRs) isEffect() {}

// effReloadRepository reloads the repository graph (foreground load).
type effReloadRepository struct{}

func (effReloadRepository) isEffect() {}

// effLoadBranches reloads the branch list using the configured limit.
type effLoadBranches struct{}

func (effLoadBranches) isEffect() {}

// effSetBookmarkConflictSources refreshes the create-bookmark modal's
// name-conflict source list from the branches tab and re-evaluates whether the
// current input collides with an existing name.
type effSetBookmarkConflictSources struct{}

func (effSetBookmarkConflictSources) isEffect() {}

// applyEffects applies every effect in order and batches any resulting commands.
// Effects that only mutate component state contribute no command.
func (m *Model) applyEffects(effs ...effect) tea.Cmd {
	var cmds []tea.Cmd
	for _, e := range effs {
		if c := m.applyEffect(e); c != nil {
			cmds = append(cmds, c)
		}
	}
	switch len(cmds) {
	case 0:
		return nil
	case 1:
		return cmds[0]
	default:
		return tea.Batch(cmds...)
	}
}

// applyEffect performs the single cross-cutting operation described by e,
// returning any command it wants scheduled. It is the one place that reaches
// into cross-cutting component state on behalf of message handlers.
func (m *Model) applyEffect(e effect) tea.Cmd {
	switch ef := e.(type) {
	case effShowError:
		// A plain error clears any stale generic replay target so Retry can never fire against an
		// unrelated failure (SetError already turns hasRetry off).
		m.pendingRetryCmd = nil
		m.errorModal.SetError(ef.err, false, "")
		return nil
	case effShowRetryableError:
		m.errorModal.SetError(ef.err, false, "")
		m.pendingRetryCmd = ef.retry
		m.errorModal.SetHasRetry(ef.retry != nil)
		return nil
	case effClearError:
		m.errorModal.SetError(nil, false, "")
		return nil
	case effResolveOpenPRs:
		return prstab.ResolveOpenPRsForBookmarksCmd(m.appState.GitHubService, m.bookmarksNeedingPRLookup(), m.appState.DemoMode)
	case effReloadRepository:
		return data.LoadRepository(m.appState.JJService)
	case effLoadBranches:
		return branchestab.LoadBranchesCmd(m.appState.JJService, m.settingsTabModel.GetSettingsBranchLimit())
	case effSetBookmarkConflictSources:
		m.bookmarkModal.SetNameConflictSources(m.branchesTabModel.BuildBookmarkNameConflictSources())
		m.bookmarkModal.UpdateNameExistsFromInput(m.appState.Config != nil && m.appState.Config.ShouldSanitizeBookmarkNames())
		return nil
	default:
		return nil
	}
}
