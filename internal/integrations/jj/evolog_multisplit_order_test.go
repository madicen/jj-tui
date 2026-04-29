package jj

import (
	"testing"
)

func TestSortEvologMultiSplitBasesDeepestFirst(t *testing.T) {
	entries := []EvologEntry{
		{CommitID: "tip11111111111111111111111111111111"},
		{CommitID: "mid22222222222222222222222222222222"},
		{CommitID: "old33333333333333333333333333333333"},
	}
	shallowFirst := []string{
		"mid22222222222222222222222222222222",
		"old33333333333333333333333333333333",
	}
	got := SortEvologMultiSplitBasesDeepestFirst(entries, shallowFirst)
	if len(got) != 2 || got[0] != entries[2].CommitID || got[1] != entries[1].CommitID {
		t.Fatalf("got %#v want [%s %s]", got, entries[2].CommitID, entries[1].CommitID)
	}
	if s := SortEvologMultiSplitBasesDeepestFirst(entries, []string{"mid22222222222222222222222222222222"}); len(s) != 1 || s[0] != entries[1].CommitID {
		t.Fatalf("single: %#v", s)
	}
	dup := []string{
		"old33333333333333333333333333333333",
		"mid22222222222222222222222222222222",
		"old33333333333333333333333333333333",
	}
	got2 := SortEvologMultiSplitBasesDeepestFirst(entries, dup)
	if len(got2) != 2 {
		t.Fatalf("dedupe: %#v", got2)
	}
}
