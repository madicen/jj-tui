package evologsplit

import "testing"

func TestNoSplitFirstEnterOnlyArms(t *testing.T) {
	var c noSplitConfirm
	c.onAISuggestNoSplit(1)
	if !noSplitFirstEnterOnlyArms(true, 1, c) {
		t.Fatal("expected arm on first enter same row")
	}
	c.armed = true
	if noSplitFirstEnterOnlyArms(true, 1, c) {
		t.Fatal("second enter should not arm-only")
	}
	if noSplitFirstEnterOnlyArms(true, 2, c) {
		t.Fatal("different row: split immediately, no arm")
	}
	if noSplitFirstEnterOnlyArms(false, 1, c) {
		t.Fatal("no suggest no split")
	}
}

func TestNoSplitConfirmRowChangeClearsArm(t *testing.T) {
	var c noSplitConfirm
	c.onAISuggestNoSplit(1)
	c.armed = true
	c.onSelectedIdxChange(2)
	if c.armed {
		t.Fatal("moving row should clear armed")
	}
}
