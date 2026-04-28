package planner

import (
	"fmt"
	"sort"
	"strings"
)

// ProposalValidator is one step in a chain-of-responsibility style validation pipeline.
type ProposalValidator interface {
	Validate(proposals []CommitProposal) error
}

// ValidateProposals runs the default validation chain (sequence IDs, messages, hunks).
func ValidateProposals(proposals []CommitProposal) error {
	chain := []ProposalValidator{
		sequenceValidator{},
		nonEmptyHunkListValidator{},
	}
	for _, v := range chain {
		if err := v.Validate(proposals); err != nil {
			return err
		}
	}
	return nil
}

type sequenceValidator struct{}

func (sequenceValidator) Validate(proposals []CommitProposal) error {
	if len(proposals) == 0 {
		return nil
	}
	bySeq := append([]CommitProposal(nil), proposals...)
	sort.Slice(bySeq, func(i, j int) bool { return bySeq[i].SequenceID < bySeq[j].SequenceID })
	for i := range bySeq {
		if bySeq[i].SequenceID != i {
			return fmt.Errorf("proposal sequence_id: want contiguous 0..%d, got id %d at sorted index %d", len(bySeq)-1, bySeq[i].SequenceID, i)
		}
	}
	return nil
}

type nonEmptyHunkListValidator struct{}

func (nonEmptyHunkListValidator) Validate(proposals []CommitProposal) error {
	for i, p := range proposals {
		if len(p.Hunks) == 0 {
			return fmt.Errorf("proposal %d (sequence_id=%d): at least one hunk entry is required", i, p.SequenceID)
		}
		for hi, h := range p.Hunks {
			if strings.TrimSpace(h) == "" {
				return fmt.Errorf("proposal %d (sequence_id=%d): hunk %d is empty", i, p.SequenceID, hi)
			}
		}
	}
	return nil
}
