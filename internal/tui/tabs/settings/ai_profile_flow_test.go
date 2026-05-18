package settings

import (
	"strings"
	"testing"

	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/tui/mouse"
)

// TestZoneIDs_IncludesAIProfileManagement guards against the "+ new does
// nothing" bug: resolveClickedZone only matches against ZoneIDs(), so any
// AI profile zone missing here is silently dropped on click.
func TestZoneIDs_IncludesAIProfileManagement(t *testing.T) {
	cfg := &config.Config{
		AIProfiles: []config.AIProfile{
			{Name: "a", Provider: "openai_compatible"},
			{Name: "b", Provider: "openai_compatible"},
		},
		AIActiveProfile: "a",
	}
	m := NewModelWithConfig(cfg)
	ids := m.ZoneIDs()
	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}
	wants := []string{
		mouse.ZoneSettingsAIProfileNew,
		mouse.ZoneSettingsAIProfileDelete,
		mouse.ZoneSettingsAIProfileCyclePrev,
		mouse.ZoneSettingsAIProfileCycleNext,
		mouse.ZoneSettingsAIProfileName,
		mouse.ZoneSettingsAIProfileRow(0),
		mouse.ZoneSettingsAIProfileRow(1),
	}
	for _, want := range wants {
		if _, ok := idSet[want]; !ok {
			t.Errorf("ZoneIDs() missing %q — clicks on this zone will be dropped by resolveClickedZone", want)
		}
	}
}

// TestBuildSettingsParams_PropagatesProfiles confirms that the Settings → AI
// tab's profile list flows through BuildSettingsParams: the active profile's
// fields mirror onto the flat AI* params (back-compat), and the full list
// rides alongside.
func TestBuildSettingsParams_PropagatesProfiles(t *testing.T) {
	cfg := &config.Config{
		AIProfiles: []config.AIProfile{
			{Name: "openai", Provider: "openai_compatible", Model: "gpt-4o-mini", APIKey: "sk-a"},
			{Name: "local", Provider: "ollama", Model: "qwen2.5:1.5b", BaseURL: "http://127.0.0.1:11434/v1"},
		},
		AIActiveProfile: "local",
		AIProvider:      "ollama",
		AIBaseURL:       "http://127.0.0.1:11434/v1",
		AIModel:         "qwen2.5:1.5b",
	}
	m := NewModelWithConfig(cfg)
	params := BuildSettingsParams(&m, "", "")

	if len(params.AIProfiles) != 2 {
		t.Fatalf("AIProfiles count: %d", len(params.AIProfiles))
	}
	if params.AIActiveProfile != "local" {
		t.Fatalf("AIActiveProfile: %q", params.AIActiveProfile)
	}
	if params.AIProvider != "ollama" {
		t.Fatalf("flat provider mirror: %q", params.AIProvider)
	}
	if params.AIModel != "qwen2.5:1.5b" {
		t.Fatalf("flat model mirror: %q", params.AIModel)
	}
	if params.AIBaseURL != "http://127.0.0.1:11434/v1" {
		t.Fatalf("flat base url mirror: %q", params.AIBaseURL)
	}
}

// TestBuildSettingsParams_AddProfileThenSave confirms that adding a profile in
// the AI sub-model and changing the active one propagates to params.
func TestBuildSettingsParams_AddProfileThenSave(t *testing.T) {
	cfg := &config.Config{
		AIProfiles: []config.AIProfile{
			{Name: "default", Provider: "openai_compatible", Model: "gpt-4o-mini"},
		},
		AIActiveProfile: "default",
	}
	m := NewModelWithConfig(cfg)
	aim := m.GetAIModel()
	aim.AddProfile()
	idx := aim.SelectedIndex()
	// Rename the new profile by writing into the name input then committing.
	aim.SetFocusedField(3)
	aim.SetProfileNameInputValue("smart")
	aim.CommitInputs()
	aim.SetActiveByIndex(idx)

	params := BuildSettingsParams(&m, "", "")
	if params.AIActiveProfile != "smart" {
		t.Fatalf("AIActiveProfile: got %q", params.AIActiveProfile)
	}
	hasSmart := false
	for _, p := range params.AIProfiles {
		if strings.EqualFold(p.Name, "smart") {
			hasSmart = true
			break
		}
	}
	if !hasSmart {
		t.Fatalf("expected 'smart' profile in params: %+v", params.AIProfiles)
	}
}
