package ai

import (
	"testing"

	"github.com/madicen/jj-tui/internal/config"
)

// TestAITimeoutStepper_BoundsAndStep exercises the +/- stepper to confirm the
// clamp constants are wired correctly: Inc/Dec walks by AITimeoutStepSeconds
// and refuses to cross AITimeoutMinSeconds / AITimeoutMaxSeconds.
func TestAITimeoutStepper_BoundsAndStep(t *testing.T) {
	t.Parallel()

	m := NewModel()
	if got := m.GetAITimeoutSeconds(); got != AITimeoutDefaultSeconds {
		t.Fatalf("fresh model timeout = %d, want %d (default)", got, AITimeoutDefaultSeconds)
	}

	m.IncAITimeout()
	if got, want := m.GetAITimeoutSeconds(), AITimeoutDefaultSeconds+AITimeoutStepSeconds; got != want {
		t.Fatalf("after Inc: got %d, want %d", got, want)
	}
	m.DecAITimeout()
	m.DecAITimeout()
	if got, want := m.GetAITimeoutSeconds(), AITimeoutDefaultSeconds-AITimeoutStepSeconds; got != want {
		t.Fatalf("after Dec*2: got %d, want %d", got, want)
	}

	// Saturate at the upper bound.
	for range AITimeoutMaxSeconds {
		m.IncAITimeout()
	}
	if got := m.GetAITimeoutSeconds(); got != AITimeoutMaxSeconds {
		t.Fatalf("after many Inc: got %d, want %d (clamped to max)", got, AITimeoutMaxSeconds)
	}

	// Saturate at the lower bound.
	for range AITimeoutMaxSeconds {
		m.DecAITimeout()
	}
	if got := m.GetAITimeoutSeconds(); got != AITimeoutMinSeconds {
		t.Fatalf("after many Dec: got %d, want %d (clamped to min)", got, AITimeoutMinSeconds)
	}
}

// TestAITimeoutFromConfig_NilFallsBackToDefault verifies that a config without
// the field set surfaces in the UI as the effective 60s default (not as 0),
// because the stepper derives from cfg.AITimeout() which has that fallback baked in.
func TestAITimeoutFromConfig_NilFallsBackToDefault(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	m := NewModelFromConfig(cfg)
	if got := m.GetAITimeoutSeconds(); got != AITimeoutDefaultSeconds {
		t.Fatalf("AITimeoutSeconds with nil config field = %d, want %d", got, AITimeoutDefaultSeconds)
	}
}

// TestAITimeoutFromConfig_OutOfBoundsIsClamped ensures a hand-edited config that
// stored e.g. 1 second or 99999 seconds gets snapped back into the stepper's range
// when the UI loads, so the user can't observe a value the stepper can't reproduce.
func TestAITimeoutFromConfig_OutOfBoundsIsClamped(t *testing.T) {
	t.Parallel()

	tiny := 1
	huge := 99_999
	for _, tc := range []struct {
		name      string
		stored    int
		wantFloor int
		wantCeil  int
	}{
		{"tiny", tiny, AITimeoutMinSeconds, 0},
		{"huge", huge, 0, AITimeoutMaxSeconds},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := tc.stored
			cfg := &config.Config{AITimeoutSeconds: &v}
			m := NewModelFromConfig(cfg)
			got := m.GetAITimeoutSeconds()
			if tc.wantFloor > 0 && got != tc.wantFloor {
				t.Fatalf("stored=%d -> UI=%d, want %d (min clamp)", tc.stored, got, tc.wantFloor)
			}
			if tc.wantCeil > 0 && got != tc.wantCeil {
				t.Fatalf("stored=%d -> UI=%d, want %d (max clamp)", tc.stored, got, tc.wantCeil)
			}
		})
	}
}
