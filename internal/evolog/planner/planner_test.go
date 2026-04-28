package planner

import "testing"

// TestEvologSplit_Integrity (design council) — initial checks for proposal invariants.
// Future: mock VCS, assert Initial -> Split -> Apply reconstructs HEAD.
func TestEvologSplit_Integrity(t *testing.T) {
	t.Run("valid_linear_chain", func(t *testing.T) {
		p := []CommitProposal{
			{SequenceID: 0, Message: "add parser", Hunks: []string{"diff --git a/x.go b/x.go"}},
			{SequenceID: 1, Message: "wire CLI", Hunks: []string{"diff --git a/main.go b/main.go"}},
		}
		if err := ValidateProposals(p); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("sequence_gap_rejected", func(t *testing.T) {
		p := []CommitProposal{
			{SequenceID: 0, Message: "a", Hunks: []string{"h1"}},
			{SequenceID: 2, Message: "b", Hunks: []string{"h2"}},
		}
		if err := ValidateProposals(p); err == nil {
			t.Fatal("expected error for non-contiguous sequence_id")
		}
	})

	t.Run("empty_hunks_rejected", func(t *testing.T) {
		p := []CommitProposal{{SequenceID: 0, Message: "x", Hunks: nil}}
		if err := ValidateProposals(p); err == nil {
			t.Fatal("expected error for empty hunks")
		}
	})

	t.Run("stub_analyze_empty_diff", func(t *testing.T) {
		var s StubPlanner
		out, err := s.Analyze("")
		if err != nil || len(out) != 0 {
			t.Fatalf("Analyze(\"\"): out=%v err=%v", out, err)
		}
	})

	t.Run("stub_analyze_non_empty_errors", func(t *testing.T) {
		var s StubPlanner
		_, err := s.Analyze("diff --git")
		if err == nil {
			t.Fatal("expected not implemented")
		}
	})
}
