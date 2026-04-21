package divergent

import (
	"testing"

	"github.com/madicen/jj-tui/internal/integrations/jj"
)

func TestViableDivergentKeepIndices(t *testing.T) {
	a := jj.DivergentVersion{CommitID: "a", Immutable: false}
	b := jj.DivergentVersion{CommitID: "b", Immutable: false}
	im := jj.DivergentVersion{CommitID: "i", Immutable: true}

	got := viableDivergentKeepIndices([]jj.DivergentVersion{a, b})
	if len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("both mutable heads: got %v want [0 1]", got)
	}

	got = viableDivergentKeepIndices([]jj.DivergentVersion{im, b})
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("keep immutable only: got %v want [0]", got)
	}

	got = viableDivergentKeepIndices([]jj.DivergentVersion{b, im})
	if len(got) != 1 || got[0] != 1 {
		t.Fatalf("keep immutable second: got %v want [1]", got)
	}

	got = viableDivergentKeepIndices([]jj.DivergentVersion{im, im})
	if len(got) != 0 {
		t.Fatalf("both immutable: got %v want []", got)
	}
}
