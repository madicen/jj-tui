package graph

import (
	"sort"
	"strings"
)

// multiSelect holds commit indices toggled with Space for batch operations.
type multiSelect struct {
	indices map[int]struct{}
}

func newMultiSelect() multiSelect {
	return multiSelect{indices: make(map[int]struct{})}
}

func (s *multiSelect) count() int {
	return len(s.indices)
}

func (s *multiSelect) clear() {
	s.indices = make(map[int]struct{})
}

func (s *multiSelect) toggle(idx int) {
	if idx < 0 {
		return
	}
	if _, ok := s.indices[idx]; ok {
		delete(s.indices, idx)
		return
	}
	s.indices[idx] = struct{}{}
}

func (s *multiSelect) sortedIndices() []int {
	out := make([]int, 0, len(s.indices))
	for i := range s.indices {
		out = append(out, i)
	}
	sort.Ints(out)
	return out
}

// MultiSelectCount reports how many commits are selected for batch ops.
func (m GraphModel) MultiSelectCount() int {
	return m.multiSelect.count()
}

// ClearMultiSelect clears all batch-selected commits.
func (m *GraphModel) ClearMultiSelect() {
	m.multiSelect.clear()
}

// ToggleMultiSelect toggles batch selection for a commit index.
func (m *GraphModel) ToggleMultiSelect(idx int) {
	m.multiSelect.toggle(idx)
}

// SetMultiSelect sets batch selection explicitly (for tests/goldens).
func (m *GraphModel) SetMultiSelect(indices ...int) {
	m.multiSelect.clear()
	for _, i := range indices {
		m.multiSelect.toggle(i)
	}
}

func (m GraphModel) multiSelectShortIDs() []string {
	if m.repository == nil {
		return nil
	}
	var ids []string
	for _, idx := range m.multiSelect.sortedIndices() {
		if idx < 0 || idx >= len(m.repository.Graph.Commits) {
			continue
		}
		c := m.repository.Graph.Commits[idx]
		if c.Immutable || c.IsWorking {
			continue
		}
		if c.ShortID != "" {
			ids = append(ids, c.ShortID)
		}
	}
	return ids
}

func (m GraphModel) multiSelectChangeIDs() []string {
	if m.repository == nil {
		return nil
	}
	var ids []string
	for _, idx := range m.multiSelect.sortedIndices() {
		if idx < 0 || idx >= len(m.repository.Graph.Commits) {
			continue
		}
		c := m.repository.Graph.Commits[idx]
		if c.Immutable || c.IsWorking {
			continue
		}
		id := strings.TrimSpace(c.ChangeID)
		if id == "" {
			id = strings.TrimSpace(c.ID)
		}
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func (m GraphModel) multiSelectMap() map[int]bool {
	out := make(map[int]bool, m.multiSelect.count())
	for _, idx := range m.multiSelect.sortedIndices() {
		out[idx] = true
	}
	return out
}
