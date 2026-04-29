package jj

import (
	"cmp"
	"slices"
	"strings"
)

// SortEvologMultiSplitBasesDeepestFirst returns bases reordered for EvologMultiSplit: largest evolog
// row index first (entries are newest-first; index 0 is the tip). Shallow-first lists make successive
// FAQ steps share the same jj parent and produce sibling commits instead of a linear stack.
// Duplicate commit ids are dropped (first occurrence wins). Ids not found in entries are sorted last.
func SortEvologMultiSplitBasesDeepestFirst(entries []EvologEntry, bases []string) []string {
	if len(bases) <= 1 {
		return append([]string(nil), bases...)
	}
	type row struct {
		rowIdx int
		id     string
	}
	rows := make([]row, 0, len(bases))
	seen := map[string]struct{}{}
	for _, id := range bases {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ix := -1
		for i, e := range entries {
			if strings.TrimSpace(e.CommitID) == id {
				ix = i
				break
			}
		}
		rows = append(rows, row{rowIdx: ix, id: id})
	}
	slices.SortFunc(rows, func(a, b row) int {
		if a.rowIdx < 0 && b.rowIdx < 0 {
			return 0
		}
		if a.rowIdx < 0 {
			return 1
		}
		if b.rowIdx < 0 {
			return -1
		}
		return cmp.Compare(b.rowIdx, a.rowIdx)
	})
	out := make([]string, len(rows))
	for i := range rows {
		out[i] = rows[i].id
	}
	return out
}
