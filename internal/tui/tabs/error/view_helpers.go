package error

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/util"
)

// GetErrorMessage returns the message to use for copy/display from err or status.
// Prefers err.Error(); if err is nil, checks statusMessage for "error"/"failed".
func GetErrorMessage(err error, statusMessage string) string {
	if err != nil {
		return err.Error()
	}
	statusLower := strings.ToLower(statusMessage)
	if strings.Contains(statusLower, "error") || strings.Contains(statusLower, "failed") {
		return statusMessage
	}
	return ""
}

// CopyErrorCmd returns a command that copies the given message to the clipboard.
func CopyErrorCmd(errMsg string) tea.Cmd {
	return util.CopyToClipboard(errMsg)
}

// fixedChromeLines is a conservative count of non-body lines in the bordered modal (title, gaps,
// separator, button row, padding, and border). Used to cap the error body so the dialog fits the terminal.
const fixedChromeLines = 15

// renderModal renders the error dialog (title, message, dismiss/copy/retry/quit buttons).
// Content is intended to be centered by the caller.
func renderModal(zm *zone.Manager, width, height int, errStr string, copied bool) string {
	modalWidth := min(max(width-8, 50), 80)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF5555")).
		MarginBottom(1)

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Width(modalWidth - 4)

	mutedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8B949E"))

	buttonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#30363d")).
		Padding(0, 1).
		Bold(true)

	maxModalTotal := height - 2
	if height <= 0 {
		maxModalTotal = 256
	}
	if maxModalTotal < 10 {
		maxModalTotal = 10
	}
	maxBodyLines := max(maxModalTotal-fixedChromeLines, 3)

	wrapped := errorStyle.Render(errStr)
	bodyLines := strings.Split(wrapped, "\n")
	truncated := len(bodyLines) > maxBodyLines
	if truncated {
		avail := maxBodyLines - 1
		if avail < 1 {
			avail = 1
		}
		bodyLines = bodyLines[:min(avail, len(bodyLines))]
		bodyLines = append(bodyLines, mutedStyle.Render("… truncated — press c to copy the full message"))
	}
	errBody := strings.Join(bodyLines, "\n")

	var content strings.Builder
	content.WriteString(titleStyle.Render("⚠ Error"))
	content.WriteString("\n\n")
	content.WriteString(errBody)
	content.WriteString("\n\n")
	content.WriteString(mutedStyle.Render("─────────────────────────────────────"))
	content.WriteString("\n\n")

	mark := func(id, s string) string {
		if zm != nil {
			return zm.Mark(id, s)
		}
		return s
	}

	dismissBtn := mark(mouse.ZoneActionDismissError, buttonStyle.Render("Dismiss (Esc)"))

	var copyBtn string
	if copied {
		copiedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#2ea44f")).
			Bold(true)
		copyBtn = copiedStyle.Render("✓ Copied!")
	} else {
		copyBtn = mark(mouse.ZoneActionCopyError, buttonStyle.Render("Copy (c)"))
	}

	retryBtn := mark(mouse.ZoneActionRetry, buttonStyle.Render("Retry (^r)"))
	quitBtn := mark(mouse.ZoneActionQuit, buttonStyle.Background(lipgloss.Color("#c9302c")).Render("Quit (^q)"))

	content.WriteString(dismissBtn + "  " + copyBtn + "  " + retryBtn + "  " + quitBtn)

	modalBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF5555")).
		Padding(1, 2).
		Width(modalWidth).
		Render(content.String())

	return modalBox
}
