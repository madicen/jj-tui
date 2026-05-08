package initrepo

import (
	"strings"
	"testing"
)

// TestViewRendersWelcomeScreenContent confirms the welcome screen's three sections (init,
// optional remote URL, optional gh repo create) all render the strings the README documents.
// If the in-app text drifts, the README's "Welcome screen" section needs to be updated to match.
func TestViewRendersWelcomeScreenContent(t *testing.T) {
	t.Parallel()

	m := NewModel()
	m.SetPath("/tmp/example-project")
	view := m.View()

	want := []string{
		"Welcome to jj-tui",
		"This directory is not yet a Jujutsu repository.",
		"Initialize Repository (i)",
		"Runs `jj git init --colocate` in this directory.",
		"Optional: connect a remote",
		"Remote URL",
		"Tab/u to focus, paste a URL, then press Enter or i to initialize with origin set.",
		"Or create a brand-new GitHub repo",
		"Press Esc to dismiss",
	}
	for _, s := range want {
		if !strings.Contains(view, s) {
			t.Errorf("welcome screen view missing expected fragment %q", s)
		}
	}
}
