package settings

// Golden View() coverage for every Settings sub-tab. Settings is golden-locked
// here (rather than through the root Model.View() composite) because its
// root-model render depends on exec.LookPath("gh") and reads integration env
// vars — non-deterministic across machines. Here we build the tab directly from
// a fixed config, clear the integration env vars, and pin GhAvailable=false so
// output is byte-stable.
//
// Update goldens with: go test ./internal/tui/tabs/settings -run TestGoldenSettings -update
//
// PLAN(P2.3): keep these green across the tab-interface migration.

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/testutil"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

// clearIntegrationEnv removes env vars that the Jira/Codecks/GitHub sub-models
// read at construction, so goldens do not depend on the developer's shell.
func clearIntegrationEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"GITHUB_TOKEN",
		"JIRA_URL", "JIRA_USER", "JIRA_TOKEN", "JIRA_PROJECT",
		"JIRA_PROJECT_FILTER", "JIRA_ISSUE_TYPE", "JIRA_JQL",
		"CODECKS_SUBDOMAIN", "CODECKS_TOKEN", "CODECKS_PROJECT",
	} {
		t.Setenv(k, "")
	}
}

// goldenSettingsConfig returns a deterministic config covering every settings pane.
func goldenSettingsConfig() *config.Config {
	boolp := func(b bool) *bool { return &b }
	intp := func(i int) *int { return &i }
	return &config.Config{
		GitHubConfig: config.GitHubConfig{
			GitHubShowMerged: boolp(true),
			GitHubOnlyMine:   boolp(true),
			GitHubPRLimit:    intp(25),
		},
		JiraConfig: config.JiraConfig{
			JiraProject:       "APP",
			JiraProjectFilter: "APP,TEAM",
			JiraIssueType:     "Task",
			JiraJQL:           "sprint in openSprints()",
		},
		CodecksConfig: config.CodecksConfig{
			CodecksProject: "Personal Projects",
		},
		TicketsConfig: config.TicketsConfig{
			TicketProvider: "jira",
		},
		UIConfig: config.UIConfig{
			BranchStatsLimit:   intp(20),
			GraphRevset:        "trunk() | ancestors(@)",
			ConfirmDestructive: boolp(true),
		},
		ThemeConfig: config.ThemeConfig{
			ThemePrimary:   "#7E00AF",
			ThemeSecondary: "#50FA7B",
			ThemeMuted:     "#6272A4",
		},
		AIConfig: config.AIConfig{
			AIEnabled:        boolp(true),
			AIProvider:       "ollama",
			AIBaseURL:        "http://127.0.0.1:11434/v1",
			AIModel:          "qwen2.5-coder:7b",
			AITimeoutSeconds: intp(150),
			AIProfiles: []config.AIProfile{
				{Name: "default", Provider: "ollama", BaseURL: "http://127.0.0.1:11434/v1", Model: "qwen2.5-coder:7b", TimeoutSeconds: 150},
			},
			AIActiveProfile: "default",
		},
		AdvancedConfig: config.AdvancedConfig{
			ExternalFileEditor: "cursor",
		},
	}
}

// newGoldenSettings builds a deterministic settings model with the given active
// sub-tab, pinned theme, and fixed ViewOpts (GhAvailable=false).
func newGoldenSettings(t *testing.T, tabIndex int) Model {
	t.Helper()
	clearIntegrationEnv(t)
	testutil.ForceDeterministicRendering()
	styles.SetTheme("#7E00AF", "#50FA7B", "#6272A4")

	cfg := goldenSettingsConfig()
	m := NewModelWithConfig(cfg)
	m.SetZoneManager(zone.New())
	m.SetActiveSettingsTabIndex(tabIndex)
	m.SetDimensions(100, 40)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m.SetInputWidths(60)
	m.SetViewOpts(ViewOpts{
		GitHubAvailable:   true,
		TicketServiceName: "Jira",
		Config:            cfg,
		ContentHeight:     0, // 0 = no scroll clipping; render the full pane
		GhAvailable:       false,
	})
	return m
}

func TestGoldenSettings(t *testing.T) {
	panes := []struct {
		name  string
		index int
	}{
		{"github", 0},
		{"jira", 1},
		{"codecks", 2},
		{"tickets", 3},
		{"branches", 4},
		{"theme", 5},
		{"ai", 6},
		{"advanced", 7},
	}
	for _, p := range panes {
		t.Run(p.name, func(t *testing.T) {
			m := newGoldenSettings(t, p.index)
			testutil.AssertGolden(t, "settings/"+p.name, m.View())
		})
	}
}

// TestGoldenSettingsFocusedField locks the rendering of a non-default focused
// field, so the migration can't silently change which input shows the cursor.
func TestGoldenSettingsFocusedField(t *testing.T) {
	t.Run("jira_field_3_focused", func(t *testing.T) {
		m := newGoldenSettings(t, 1) // Jira
		m.GetJiraModel().SetFocusedField(3)
		testutil.AssertGolden(t, "settings/jira_field_3_focused", m.View())
	})

	t.Run("ai_field_1_focused", func(t *testing.T) {
		m := newGoldenSettings(t, 6) // AI
		m.GetAIModel().SetFocusedField(1)
		testutil.AssertGolden(t, "settings/ai_field_1_focused", m.View())
	})
}
