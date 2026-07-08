package graph

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/jj"
)

func annotateTestModel() GraphModel {
	m := NewGraphModel(nil)
	m.SetDimensions(100, 40)
	m.OnRepositoryLoaded(&internal.Repository{
		Graph: internal.CommitGraph{Commits: []internal.Commit{
			{ID: "aaaa1111", ShortID: "aaaa", ChangeID: "zkztqwxr", Summary: "c0"},
			{ID: "bbbb2222", ShortID: "bbbb", ChangeID: "nmqrstuv", Summary: "c1"},
			{ID: "cccc3333", ShortID: "cccc", ChangeID: "wpqxlmno", Summary: "c2"},
		}},
	})
	m.selectedCommit = 0
	m.graphFocused = false
	return m
}

// TestAnnotate_BKeyOpensRequest verifies B in the files pane emits an Annotate request.
func TestAnnotate_BKeyOpensRequest(t *testing.T) {
	m := annotateTestModel()
	m.changedFiles = []jj.ChangedFile{{Path: "a.go", Status: "M"}}
	m.selectedFile = 0

	_, req, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("B")})
	if req == nil || !req.Annotate {
		t.Fatalf("expected Annotate request, got %+v", req)
	}
}

// TestAnnotate_EnterJumpsToChange verifies Enter on a blame line jumps the graph
// selection to the change that introduced it and requests its changed files.
func TestAnnotate_EnterJumpsToChange(t *testing.T) {
	m := annotateTestModel()
	// Line 0 (selected by default) is blamed on commit index 2 (wpqxlmno).
	m.ShowAnnotate("aaaa", "a.go", []jj.AnnotationLine{
		{ChangeID: "wpqxlmno", Author: "alan", Age: "1 week ago", LineNumber: 1, Content: "x"},
		{ChangeID: "nmqrstuv", Author: "grace", Age: "2 days ago", LineNumber: 2, Content: "y"},
	})
	if !m.AnnotateShown() {
		t.Fatal("overlay should be shown")
	}

	updated, req, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEnter})
	if updated.AnnotateShown() {
		t.Error("overlay should be closed after Enter")
	}
	if updated.selectedCommit != 2 {
		t.Errorf("selectedCommit = %d, want 2 (jumped to blamed change)", updated.selectedCommit)
	}
	if req == nil || req.LoadChangedFiles == nil || *req.LoadChangedFiles != "wpqxlmno" {
		t.Fatalf("expected LoadChangedFiles request for wpqxlmno, got %+v", req)
	}
}

// TestAnnotate_MoveThenJump verifies moving the highlight then Enter jumps to the
// newly-highlighted line's change.
func TestAnnotate_MoveThenJump(t *testing.T) {
	m := annotateTestModel()
	m.ShowAnnotate("aaaa", "a.go", []jj.AnnotationLine{
		{ChangeID: "wpqxlmno", Author: "alan", Age: "1 week ago", LineNumber: 1, Content: "x"},
		{ChangeID: "nmqrstuv", Author: "grace", Age: "2 days ago", LineNumber: 2, Content: "y"},
	})
	// Move down to line 1 (nmqrstuv -> commit index 1).
	m, _, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if got := m.SelectedAnnotateChangeID(); got != "nmqrstuv" {
		t.Fatalf("after j, selected change = %q, want nmqrstuv", got)
	}
	updated, req, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEnter})
	if updated.selectedCommit != 1 {
		t.Errorf("selectedCommit = %d, want 1", updated.selectedCommit)
	}
	if req == nil || req.LoadChangedFiles == nil || *req.LoadChangedFiles != "nmqrstuv" {
		t.Fatalf("expected LoadChangedFiles for nmqrstuv, got %+v", req)
	}
}

// TestAnnotate_EscCloses verifies Esc closes the overlay without changing selection.
func TestAnnotate_EscCloses(t *testing.T) {
	m := annotateTestModel()
	m.selectedCommit = 0
	m.ShowAnnotate("aaaa", "a.go", []jj.AnnotationLine{
		{ChangeID: "wpqxlmno", Author: "alan", Age: "1 week ago", LineNumber: 1, Content: "x"},
	})
	updated, req, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if updated.AnnotateShown() {
		t.Error("overlay should be closed after Esc")
	}
	if req != nil {
		t.Errorf("Esc should not emit a request, got %+v", req)
	}
	if updated.selectedCommit != 0 {
		t.Errorf("selectedCommit changed on Esc: %d", updated.selectedCommit)
	}
}
