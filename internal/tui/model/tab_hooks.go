package model

import "github.com/madicen/jj-tui/internal/tui/state"

// tab_hooks.go declares the optional, tab-specific query interfaces the root
// model needs during dispatch. They are deliberately kept OUT of the base
// tab.Tab contract (per P2.3) and are satisfied by the concrete-tab adapters in
// tab_adapters.go. The model type-asserts the registry entry against a hook and
// falls back to a zero value when the active tab doesn't implement it, so the
// dispatch code never names a concrete tab package.

// statusChangeModeReporter is implemented by the tickets tab: it reports
// whether the tab is currently in its inline status-change mode. The root uses
// this to swallow the esc that closed status-change mode (so esc doesn't also
// leave the tab).
type statusChangeModeReporter interface {
	IsStatusChangeMode() bool
}

// escConsumer is implemented by the settings tab: it reports whether an esc was
// handled by an in-tab overlay (theme picker / cleanup confirm) rather than
// leaving the tab.
type escConsumer interface {
	EscHandledInsideSettings() bool
}

func (m *Model) isTicketsStatusChangeMode() bool {
	if t, ok := m.tabRegistry[state.ViewTickets].(statusChangeModeReporter); ok {
		return t.IsStatusChangeMode()
	}
	return false
}

func (m *Model) escHandledInsideSettings() bool {
	if t, ok := m.tabRegistry[state.ViewSettings].(escConsumer); ok {
		return t.EscHandledInsideSettings()
	}
	return false
}
