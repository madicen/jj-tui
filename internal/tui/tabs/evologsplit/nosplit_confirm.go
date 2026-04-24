package evologsplit

// noSplitConfirm tracks when AI recommended no split on the current row: splitting that same row
// requires Enter twice (or picking a different row first).
type noSplitConfirm struct {
	idxAtSuggest int  // evolog row index when suggestion arrived
	armed        bool // user pressed Enter once to confirm override
}

func (n *noSplitConfirm) reset() {
	n.idxAtSuggest = -1
	n.armed = false
}

func (n *noSplitConfirm) onAISuggestNoSplit(selectedIdx int) {
	n.idxAtSuggest = selectedIdx
	n.armed = false
}

func (n *noSplitConfirm) onSelectedIdxChange(selectedIdx int) {
	if n.idxAtSuggest < 0 {
		return
	}
	if selectedIdx != n.idxAtSuggest {
		n.armed = false
	}
}

// noSplitFirstEnterOnlyArms returns true if Enter should arm a second confirm instead of running split.
func noSplitFirstEnterOnlyArms(suggestNoSplit bool, selectedIdx int, c noSplitConfirm) bool {
	if !suggestNoSplit || c.idxAtSuggest < 0 {
		return false
	}
	if selectedIdx != c.idxAtSuggest {
		return false
	}
	return !c.armed
}
