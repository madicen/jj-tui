package model

import (
	"fmt"
	"testing"

	"github.com/madicen/jj-tui/internal/tui/state"
)

// TestModalStackPushPopTopHas exercises the ModalStack primitives directly.
func TestModalStackPushPopTopHas(t *testing.T) {
	var s ModalStack
	if _, ok := s.Top(); ok {
		t.Fatal("empty stack Top() ok = true, want false")
	}
	if _, ok := s.Pop(); ok {
		t.Fatal("empty stack Pop() ok = true, want false")
	}
	if s.Len() != 0 {
		t.Fatalf("empty stack Len() = %d, want 0", s.Len())
	}

	s.Push(Modal{Kind: ModalEditDescription})
	s.Push(Modal{Kind: ModalWarning})
	s.Push(Modal{Kind: ModalError})

	if s.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", s.Len())
	}
	if top, ok := s.Top(); !ok || top.Kind != ModalError {
		t.Fatalf("Top() = %v ok=%v, want ModalError", top.Kind, ok)
	}
	if !s.Has(ModalEditDescription) || !s.Has(ModalWarning) || !s.Has(ModalError) {
		t.Fatal("Has() missed a pushed kind")
	}
	if s.Has(ModalInitRepo) {
		t.Fatal("Has(ModalInitRepo) = true, want false")
	}

	popped, ok := s.Pop()
	if !ok || popped.Kind != ModalError {
		t.Fatalf("Pop() = %v ok=%v, want ModalError", popped.Kind, ok)
	}
	if top, _ := s.Top(); top.Kind != ModalWarning {
		t.Fatalf("after Pop Top() = %v, want ModalWarning", top.Kind)
	}
	if s.Has(ModalError) {
		t.Fatal("Has(ModalError) = true after Pop, want false")
	}
}

// TestModalStackDerivedPriority verifies modalStack() encodes the same z-order
// priority chromedSlot() historically used: initrepo > error > warning >
// ViewMode-selected overlay. This mirrors the chromed-slot z-order snapshot but
// asserts directly on the derived stack Top().
func TestModalStackDerivedPriority(t *testing.T) {
	cases := []struct {
		name  string
		apply func(m *Model)
		want  ModalKind
	}{
		{"graph_none", func(m *Model) { m.appState.ViewMode = state.ViewCommitGraph }, ModalNone},
		{"viewmode_pr", func(m *Model) { m.appState.ViewMode = state.ViewCreatePR }, ModalCreatePR},
		{"warning_over_viewmode", func(m *Model) {
			m.appState.ViewMode = state.ViewCreatePR
			m.warningModal.Show("t", "msg", nil)
		}, ModalWarning},
		{"error_over_warning", func(m *Model) {
			m.appState.ViewMode = state.ViewEditDescription
			m.warningModal.Show("t", "msg", nil)
			m.errorModal.SetError(fmt.Errorf("boom"), false, "")
		}, ModalError},
		{"initrepo_over_all", func(m *Model) {
			m.appState.ViewMode = state.ViewEditDescription
			m.warningModal.Show("t", "msg", nil)
			m.errorModal.SetError(fmt.Errorf("boom"), false, "")
			m.initRepoModel.SetPath("/tmp/x")
		}, ModalInitRepo},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel()
			defer m.Close()
			tc.apply(m)
			if got := m.topModalKind(); got != tc.want {
				t.Fatalf("topModalKind() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestModalStackHasUnderlyingFormModal confirms a form modal remains reported by
// Has() even when a blocking overlay (error/warning) sits on top — matching
// applyFormModalsOverlay, which paints the non-chromed form modal behind it.
func TestModalStackHasUnderlyingFormModal(t *testing.T) {
	m := newTestModel()
	defer m.Close()
	m.appState.ViewMode = state.ViewCreatePR
	m.errorModal.SetError(fmt.Errorf("boom"), false, "")
	s := m.modalStack()
	if top, _ := s.Top(); top.Kind != ModalError {
		t.Fatalf("Top() = %v, want ModalError", top.Kind)
	}
	if !s.Has(ModalCreatePR) {
		t.Fatal("Has(ModalCreatePR) = false, want true (form modal under error chrome)")
	}
}
