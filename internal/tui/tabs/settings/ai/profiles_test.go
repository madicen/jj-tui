package ai

import (
	"strings"
	"testing"

	"github.com/madicen/jj-tui/internal/config"
)

// TestProfilesAddSelectDeleteCycle covers the full profile-management flow used
// by the Settings → AI tab UI: add a new profile inherits the current edit,
// switching profiles preserves edits, delete-not-last works, delete-last fails.
func TestProfilesAddSelectDeleteCycle(t *testing.T) {
	m := NewModel()
	if got := m.SelectedName(); got != config.DefaultAIProfileName {
		t.Fatalf("default selected: %q", got)
	}

	m.SetAIProvider("openai_compatible")
	m.aiBaseURLInput.SetValue("https://api.openai.com/v1")
	m.aiModelInput.SetValue("gpt-4o-mini")
	m.aiAPIKeyInput.SetValue("sk-fast")
	m.aiProfileNameInput.SetValue("fast")
	m.CommitInputs()
	if got := m.profiles[0].Name; got != "fast" {
		t.Fatalf("commit rename: got %q", got)
	}

	m.AddProfile()
	if len(m.profiles) != 2 {
		t.Fatalf("add: profiles=%d", len(m.profiles))
	}
	if m.SelectedIndex() != 1 {
		t.Fatalf("add should select new row, got idx=%d", m.SelectedIndex())
	}
	if !strings.HasPrefix(m.SelectedName(), "fast copy") {
		t.Fatalf("derived name: got %q", m.SelectedName())
	}

	m.aiModelInput.SetValue("gpt-4o")
	m.aiProfileNameInput.SetValue("smart")
	m.SelectProfile(0)
	if m.SelectedName() != "fast" {
		t.Fatalf("SelectProfile back: got %q", m.SelectedName())
	}
	if m.aiModelInput.Value() != "gpt-4o-mini" {
		t.Fatalf("inputs should reload from row 0; got %q", m.aiModelInput.Value())
	}
	m.SelectProfile(1)
	if m.aiModelInput.Value() != "gpt-4o" {
		t.Fatalf("inputs should reload from row 1; got %q", m.aiModelInput.Value())
	}

	m.SetActiveByIndex(1)
	if m.ActiveProfileName() != "smart" {
		t.Fatalf("active: got %q", m.ActiveProfileName())
	}

	if got := m.DeleteSelectedProfile(); got != "" {
		t.Fatalf("delete non-last unexpected error: %q", got)
	}
	if len(m.profiles) != 1 {
		t.Fatalf("after delete: profiles=%d", len(m.profiles))
	}
	if m.ActiveProfileName() != "fast" {
		t.Fatalf("active should fall back to first; got %q", m.ActiveProfileName())
	}
	if got := m.DeleteSelectedProfile(); !strings.Contains(strings.ToLower(got), "last") {
		t.Fatalf("delete last should refuse with message; got %q", got)
	}
}

// TestProfilesCycleSelectedWraps confirms CycleSelected wraps at both ends.
func TestProfilesCycleSelectedWraps(t *testing.T) {
	m := NewModel()
	m.aiProfileNameInput.SetValue("a")
	m.CommitInputs()
	m.AddProfile()
	m.aiProfileNameInput.SetValue("b")
	m.CommitInputs()
	m.AddProfile()
	m.aiProfileNameInput.SetValue("c")
	m.CommitInputs()

	m.SelectProfile(0)
	m.CycleSelected(-1)
	if m.SelectedName() != "c" {
		t.Fatalf("wrap left: got %q", m.SelectedName())
	}
	m.CycleSelected(1)
	if m.SelectedName() != "a" {
		t.Fatalf("wrap right: got %q", m.SelectedName())
	}
}

// TestNewModelFromConfig_PopulatesProfileList verifies a multi-profile config
// fills the editor with the active profile's fields.
func TestNewModelFromConfig_PopulatesProfileList(t *testing.T) {
	cfg := &config.Config{
		AIProfiles: []config.AIProfile{
			{Name: "openai", Provider: "openai_compatible", Model: "gpt-4o-mini", APIKey: "sk-a"},
			{Name: "local", Provider: "ollama", Model: "qwen2.5:1.5b", BaseURL: "http://127.0.0.1:11434/v1"},
		},
		AIActiveProfile: "local",
	}
	// The settings sub-model doesn't run normalizeAIProfiles itself — it expects
	// to be handed an already-loaded config. Mimic Load's normalization here so
	// the test is faithful to runtime.
	cfg.AIProvider = "ollama"
	cfg.AIBaseURL = "http://127.0.0.1:11434/v1"
	cfg.AIModel = "qwen2.5:1.5b"

	m := NewModelFromConfig(cfg)
	if got := m.ActiveProfileName(); got != "local" {
		t.Fatalf("active: got %q", got)
	}
	if got := m.SelectedName(); got != "local" {
		t.Fatalf("selected should match active; got %q", got)
	}
	if got := m.GetAIProvider(); got != "ollama" {
		t.Fatalf("provider: got %q", got)
	}
	if got := m.GetAIModel(); got != "qwen2.5:1.5b" {
		t.Fatalf("model: got %q", got)
	}
}
