package theme

import (
	"testing"

	"github.com/madicen/jj-tui/internal/config"
)

// This file characterizes the Theme settings sub-tab's current behavior BEFORE
// the P2.3 Tab-interface refactor. The parent settings model reads Primary/
// Secondary/Muted and drives swatch overlays; the tests lock the accessor
// surface and the default-reset behavior.
//
// PLAN(P2.3): keep this green across the interface migration.

// TestNewModelDefaults locks that a fresh model returns the documented default
// palette (used when no config is present).
func TestNewModelDefaults(t *testing.T) {
	m := NewModel()
	if m.Primary() != DefaultPrimary {
		t.Fatalf("Primary() = %q, want %q", m.Primary(), DefaultPrimary)
	}
	if m.Secondary() != DefaultSecondary {
		t.Fatalf("Secondary() = %q, want %q", m.Secondary(), DefaultSecondary)
	}
	if m.Muted() != DefaultMuted {
		t.Fatalf("Muted() = %q, want %q", m.Muted(), DefaultMuted)
	}
}

// TestNewModelFromConfigOverrides locks that config colors override the defaults.
func TestNewModelFromConfigOverrides(t *testing.T) {
	cfg := &config.Config{}
	cfg.ThemePrimary = "#111111"
	cfg.ThemeSecondary = "#222222"
	cfg.ThemeMuted = "#333333"
	m := NewModelFromConfig(cfg)
	if m.Primary() != "#111111" || m.Secondary() != "#222222" || m.Muted() != "#333333" {
		t.Fatalf("config colors not applied: got %q/%q/%q", m.Primary(), m.Secondary(), m.Muted())
	}
}

// TestSetSwatchToDefaultResetsOnlyThatColor locks that resetting one swatch does
// not disturb the others.
func TestSetSwatchToDefaultResetsOnlyThatColor(t *testing.T) {
	cfg := &config.Config{}
	cfg.ThemePrimary = "#111111"
	cfg.ThemeSecondary = "#222222"
	cfg.ThemeMuted = "#333333"
	m := NewModelFromConfig(cfg)
	m.SetSwatchToDefault(0)
	if m.Primary() != DefaultPrimary {
		t.Fatalf("Primary should reset to default, got %q", m.Primary())
	}
	if m.Secondary() != "#222222" || m.Muted() != "#333333" {
		t.Fatalf("resetting Primary disturbed other swatches: %q/%q", m.Secondary(), m.Muted())
	}
}

// TestSetSwatchToDefaultIgnoresOutOfRange locks that an out-of-range index is a
// no-op rather than a panic.
func TestSetSwatchToDefaultIgnoresOutOfRange(t *testing.T) {
	m := NewModel()
	m.SetSwatchToDefault(99) // must not panic
	if m.Primary() != DefaultPrimary {
		t.Fatalf("out-of-range reset should not change palette, got %q", m.Primary())
	}
}

// TestSwatchIndexBounds locks that Swatch returns nil for out-of-range indices
// and a non-nil picker for 0..2.
func TestSwatchIndexBounds(t *testing.T) {
	m := NewModel()
	for i := 0; i < 3; i++ {
		if m.Swatch(i) == nil {
			t.Fatalf("Swatch(%d) should be non-nil", i)
		}
	}
	if m.Swatch(-1) != nil || m.Swatch(3) != nil {
		t.Fatal("out-of-range Swatch should be nil")
	}
}

// TestAnyOpenFalseByDefault locks that no picker is open on a fresh model (so
// the parent doesn't render a stray overlay).
func TestAnyOpenFalseByDefault(t *testing.T) {
	m := NewModel()
	if m.AnyOpen() {
		t.Fatal("no swatch picker should be open by default")
	}
}
