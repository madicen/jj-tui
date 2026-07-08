package advanced

import (
	"testing"

	"github.com/madicen/jj-tui/internal/config"
)

// This file characterizes the Advanced settings sub-tab's current behavior
// BEFORE the P2.3 Tab-interface refactor. The parent settings model reads the
// graph revset, sanitize flag, external-editor preset, and cleanup-confirm state
// on save, and drives the editor-preset dropdown overlay. The tests lock those
// accessors plus the preset<->dropdown index sync and SavedExternalEditor output.
//
// PLAN(P2.3): keep this green across the interface migration.

// TestNewModelDefaults locks the documented defaults (sanitize on, no cleanup,
// preset 0 = None).
func TestNewModelDefaults(t *testing.T) {
	m := NewModel()
	if !m.GetSanitizeBookmarks() {
		t.Fatal("default GetSanitizeBookmarks() should be true")
	}
	if m.GetConfirmingCleanup() != "" {
		t.Fatalf("default GetConfirmingCleanup() = %q, want empty", m.GetConfirmingCleanup())
	}
	if m.GetExternalEditorPreset() != 0 {
		t.Fatalf("default preset = %d, want 0", m.GetExternalEditorPreset())
	}
}

// TestGraphRevsetAndCustomEditorAreSeparateFields locks that the two form fields
// (revset=0, custom editor=1) don't cross-contaminate.
func TestGraphRevsetAndCustomEditorAreSeparateFields(t *testing.T) {
	m := NewModel()
	m.SetGraphRevset("mine()")
	if m.GetGraphRevset() != "mine()" {
		t.Fatalf("GetGraphRevset = %q, want mine()", m.GetGraphRevset())
	}
	// Field count must stay 2 (parent uses global indices 14-15).
	if got := len(m.GetInputViews()); got != 2 {
		t.Fatalf("GetInputViews() length = %d, want 2", got)
	}
}

// TestSetExternalEditorPresetSyncsDropdownAndClamps locks that a valid preset
// updates the dropdown and an out-of-range preset is ignored.
func TestSetExternalEditorPresetSyncsDropdownAndClamps(t *testing.T) {
	m := NewModel()
	m.SetExternalEditorPreset(2) // VS Code
	if m.GetExternalEditorPreset() != 2 {
		t.Fatalf("GetExternalEditorPreset = %d, want 2", m.GetExternalEditorPreset())
	}
	if got := m.EditorDropdown().SelectedIndex(); got != 2 {
		t.Fatalf("dropdown index = %d, want 2", got)
	}
	m.SetExternalEditorPreset(999) // ignored
	if m.GetExternalEditorPreset() != 2 {
		t.Fatalf("out-of-range preset should be ignored, got %d", m.GetExternalEditorPreset())
	}
}

// TestSavedExternalEditorMapsPresetToConfig locks the preset->config-string
// mapping and that the custom-editor field is trimmed.
func TestSavedExternalEditorMapsPresetToConfig(t *testing.T) {
	m := NewModel()
	m.SetExternalEditorPreset(0) // None
	m.form.SetValue(fieldCustomEditor, "  cursor -g {path}  ")
	preset, custom := m.SavedExternalEditor()
	if preset != config.ExternalEditorNone {
		t.Fatalf("preset for index 0 = %q, want %q", preset, config.ExternalEditorNone)
	}
	if custom != "cursor -g {path}" {
		t.Fatalf("custom editor should be trimmed, got %q", custom)
	}
}

// TestConfirmingCleanupRoundTrips locks the cleanup-confirm accessor used by the
// destructive-cleanup flow.
func TestConfirmingCleanupRoundTrips(t *testing.T) {
	m := NewModel()
	m.SetConfirmingCleanup("delete_bookmarks")
	if m.GetConfirmingCleanup() != "delete_bookmarks" {
		t.Fatalf("GetConfirmingCleanup = %q, want delete_bookmarks", m.GetConfirmingCleanup())
	}
}

// TestSetInputWidthEnforcesMinimum locks the documented 40-cell minimum so the
// revset field and cursor stay visible.
func TestSetInputWidthEnforcesMinimum(t *testing.T) {
	m := NewModel()
	m.SetInputWidth(10)
	if got := m.form.Input(fieldGraphRevset).Width; got != 40 {
		t.Fatalf("SetInputWidth(10) should clamp to 40, got %d", got)
	}
}
