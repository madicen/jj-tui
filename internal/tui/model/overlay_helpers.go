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

// wrapBranchFetchCmd starts the spinner when the branches tab kicked off fetch-all remotes.
func (m *Model) wrapBranchFetchCmd(cmd tea.Cmd) tea.Cmd {
	if cmd == nil || !m.appState.BranchRemoteFetchPending {
		return cmd
	}
	m.appState.BranchRemoteFetchPending = false
	return tea.Batch(cmd, m.startBusySpinnerCmd())
}
