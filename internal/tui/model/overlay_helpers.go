package model

import (
	tea "github.com/charmbracelet/bubbletea"
	overlay "github.com/madicen/bubble-overlay"
)

// applyBubbleOverlayCentered composites modalView over fullView at the center (full terminal size).
func applyBubbleOverlayCentered(fullView, modalView string, viewW, viewH int) string {
	if modalView == "" || viewW <= 0 || viewH <= 0 {
		return fullView
	}
	return overlay.OverlayViewInCenter(fullView, modalView, viewW, viewH)
}

// wrapFirstPRLoadCmd shows the busy overlay until the first PR list load finishes.
func (m *Model) wrapFirstPRLoadCmd(prCmd tea.Cmd) tea.Cmd {
	if prCmd == nil || m.appState.PRsLoadedOnce || !m.isGitHubAvailable() {
		return prCmd
	}
	m.appState.Loading = true
	m.appState.StatusMessage = "Loading pull requests…"
	return tea.Batch(prCmd, m.startBusySpinnerCmd())
}

// wrapSpinnerStart starts the busy spinner alongside cmd when a submodel just kicked off a slow
// remote/network op and set Loading=true + SpinnerStartPending (e.g. branch fetch-all/push, PR
// merge/close). The pending flag makes this idempotent: submodels can't start the spinner
// themselves, so main attaches the tick here exactly once and clears the flag.
func (m *Model) wrapSpinnerStart(cmd tea.Cmd) tea.Cmd {
	if cmd == nil || !m.appState.SpinnerStartPending {
		return cmd
	}
	m.appState.SpinnerStartPending = false
	return tea.Batch(cmd, m.startBusySpinnerCmd())
}
