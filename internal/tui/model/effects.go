package model

import (
	tea "github.com/charmbracelet/bubbletea"
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

// effShowError routes an error into the shared error modal.
type effShowError struct{ err error }

func (effShowError) isEffect() {}

// effClearError dismisses whatever is currently in the error modal.
type effClearError struct{}

func (effClearError) isEffect() {}

// effResolveOpenPRs kicks off targeted per-bookmark open-PR lookups for local
// bookmarks that the bulk PR list did not match.
type effResolveOpenPRs struct{}

func (effResolveOpenPRs) isEffect() {}

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
		m.errorModal.SetError(ef.err, false, "")
		return nil
	case effClearError:
		m.errorModal.SetError(nil, false, "")
		return nil
	case effResolveOpenPRs:
		return prstab.ResolveOpenPRsForBookmarksCmd(m.appState.GitHubService, m.bookmarksNeedingPRLookup(), m.appState.DemoMode)
	default:
		return nil
	}
}
