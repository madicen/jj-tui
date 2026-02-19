package help

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/tabs/help/commandhistory"
)

// renderTabBar renders the Shortcuts | History tab bar.
func (m Model) renderTabBar() string {
	shortcutsStyle := helpTabStyle
	commandsStyle := helpTabStyle
	if m.activeTab == 0 {
		shortcutsStyle = helpTabActiveStyle
	} else {
		commandsStyle = helpTabActiveStyle
	}
	shortcutsTab := mark(m.zoneManager, mouse.ZoneHelpTabShortcuts, shortcutsStyle.Render("Shortcuts"))
	commandsTab := mark(m.zoneManager, mouse.ZoneHelpTabCommands, commandsStyle.Render("History"))
	return lipgloss.JoinHorizontal(lipgloss.Left, shortcutsTab, " │ ", commandsTab)
}

// mark wraps content in a zone for click detection. Returns content unchanged if zoneManager is nil.
func mark(z *zone.Manager, id, content string) string {
	if z == nil {
		return content
	}
	return z.Mark(id, content)
}

// isAutoRefreshCommand returns true if the command is part of auto-refresh (filtered from history).
func isAutoRefreshCommand(cmd string) bool {
	autoRefreshPatterns := []string{
		"jj log -r mutable()",
		"jj log -r empty()",
	}
	for _, pattern := range autoRefreshPatterns {
		if strings.HasPrefix(cmd, pattern) {
			return true
		}
	}
	return false
}

// formatDuration formats a duration for display (e.g. "<1ms", "42ms", "1.2s").
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return "<1ms"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

// BuildCommandHistoryEntries builds display entries from the jj service, filtering auto-refresh and formatting.
// Returns nil if svc is nil.
func BuildCommandHistoryEntries(svc *jj.Service) []commandhistory.Entry {
	if svc == nil {
		return nil
	}
	var out []commandhistory.Entry
	for _, e := range svc.GetCommandHistory() {
		if isAutoRefreshCommand(e.Command) {
			continue
		}
		out = append(out, commandhistory.Entry{
			Command:   e.Command,
			Timestamp: e.Timestamp.Format("15:04:05"),
			Duration:  formatDuration(e.Duration),
			Success:   e.Success,
			Error:     e.Error,
		})
	}
	return out
}
