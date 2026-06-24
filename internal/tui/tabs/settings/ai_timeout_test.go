package settings

import (
	"strings"
	"testing"
)

// TestRenderAI_TimeoutStepperVisible verifies the AI tab renders the new
// generation-timeout stepper with [-] / [+] affordances and the unit-suffixed
// numeric label. Keeps the README/help copy honest: if the row disappears,
// users have no in-UI way to bump the timeout, and we want to know.
func TestRenderAI_TimeoutStepperVisible(t *testing.T) {
	t.Parallel()

	r := renderCtx{} // no zone manager: mark() returns content unchanged.
	data := RenderData{
		AIEnabled:        true,
		AIProviderID:     "openai_compatible",
		AITimeoutSeconds: 120,
		// Inputs slots 16..18 are required by renderAI but we don't assert on them.
		Inputs:         make([]struct{ View string }, 19),
		EvologMultiMax: 1,
	}
	out := strings.Join(r.renderAI(data, 0), "\n")

	for _, want := range []string{
		"Generation timeout:",
		"[-]",
		"120s",
		"[+]",
		// Help copy that points at the stepper rather than the JSON config.
		"raise the generation timeout below if needed",
		// The old "Local models may need 120s+" guidance should still be present so
		// users have a concrete recommendation next to the control.
		"Local models (Ollama) may need 120s+",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("renderAI output missing %q\n--- output ---\n%s\n--- end ---", want, out)
		}
	}
}
