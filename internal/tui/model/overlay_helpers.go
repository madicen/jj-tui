package model

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	overlay "github.com/madicen/bubble-overlay"
)

// applyBubbleOverlayCentered composites modalView over fullView at the center (full terminal size).
func applyBubbleOverlayCentered(fullView, modalView string, viewW, viewH int) string {
	if modalView == "" || viewW <= 0 || viewH <= 0 {
		return fullView
	}
	boxLines := strings.Split(modalView, "\n")
	modalH := len(boxLines)
	modalW := 0
	for _, l := range boxLines {
		if w := lipgloss.Width(l); w > modalW {
			modalW = w
		}
	}
	top := max((viewH-modalH)/2, 0)
	left := max((viewW-modalW)/2, 0)
	return overlay.OverlayView(fullView, modalView, viewW, viewH, top, left)
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
