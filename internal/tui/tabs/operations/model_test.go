package operations

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/state"
)

func sampleOps() []jj.Operation {
	return []jj.Operation{
		{ID: "aaaa1111", Description: "describe commit", Time: "t0", IsCurrent: true},
		{ID: "bbbb2222", Description: "new empty commit", Time: "t1"},
		{ID: "cccc3333", Description: "snapshot working copy", Time: "t2"},
	}
}

func key(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestOperations_NavigationClamps(t *testing.T) {
	m := NewModel().SetDimensions(100, 40)
	m.Show(sampleOps())

	if m.selectedIdx != 0 {
		t.Fatalf("expected initial selection 0, got %d", m.selectedIdx)
	}
	// Down past the end should clamp at the last index.
	for i := 0; i < 5; i++ {
		m, _ = m.Update(key("j"))
	}
	if m.selectedIdx != 2 {
		t.Fatalf("expected selection clamped at 2, got %d", m.selectedIdx)
	}
	// Up past the top should clamp at 0.
	for i := 0; i < 5; i++ {
		m, _ = m.Update(key("k"))
	}
	if m.selectedIdx != 0 {
		t.Fatalf("expected selection clamped at 0, got %d", m.selectedIdx)
	}
	// g/G jump to top/bottom.
	m, _ = m.Update(key("G"))
	if m.selectedIdx != 2 {
		t.Fatalf("expected G to jump to bottom (2), got %d", m.selectedIdx)
	}
	m, _ = m.Update(key("g"))
	if m.selectedIdx != 0 {
		t.Fatalf("expected g to jump to top (0), got %d", m.selectedIdx)
	}
}

func TestOperations_EnterOnCurrentIsNoop(t *testing.T) {
	m := NewModel().SetDimensions(100, 40)
	m.Show(sampleOps()) // index 0 is current
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeNormal {
		t.Fatalf("Enter on current op must not enter confirm mode, mode=%v", m.mode)
	}
	if cmd != nil {
		t.Fatalf("Enter on current op must not emit a command")
	}
}

func TestOperations_RestoreConfirmFlow(t *testing.T) {
	m := NewModel().SetDimensions(100, 40)
	m.Show(sampleOps())
	m, _ = m.Update(key("j")) // select index 1 (not current)

	// Enter opens the inline confirm.
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeConfirmingRestore {
		t.Fatalf("Enter on a non-current op should open confirm, mode=%v", m.mode)
	}
	if cmd != nil {
		t.Fatalf("opening confirm should not emit a command yet")
	}
	if m.restoreOpID != "bbbb2222" {
		t.Fatalf("expected pending restore id bbbb2222, got %q", m.restoreOpID)
	}

	// 'n' cancels back to normal without a command.
	mc := m
	mc, cmd = mc.Update(key("n"))
	if mc.mode != modeNormal || cmd != nil {
		t.Fatalf("'n' should cancel confirm without command; mode=%v cmd!=nil=%v", mc.mode, cmd != nil)
	}

	// 'y' confirms: emits NavigateRestoreOperation with the id and hides the modal.
	m, cmd = m.Update(key("y"))
	if cmd == nil {
		t.Fatal("'y' should emit a restore navigation command")
	}
	if m.IsShown() {
		t.Fatal("modal should hide after confirming restore")
	}
	navMsg, ok := cmd().(state.NavigateMsg)
	if !ok {
		t.Fatalf("expected NavigateMsg, got %T", cmd())
	}
	if navMsg.Target.Kind != state.NavigateRestoreOperation {
		t.Fatalf("expected NavigateRestoreOperation, got %v", navMsg.Target.Kind)
	}
	if navMsg.Target.OperationID != "bbbb2222" {
		t.Fatalf("expected OperationID bbbb2222, got %q", navMsg.Target.OperationID)
	}
}

func TestOperations_EscCloses(t *testing.T) {
	m := NewModel().SetDimensions(100, 40)
	m.Show(sampleOps())
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.IsShown() {
		t.Fatal("Esc should hide the modal")
	}
	if cmd == nil {
		t.Fatal("Esc should emit a close navigation command")
	}
	navMsg, ok := cmd().(state.NavigateMsg)
	if !ok {
		t.Fatalf("expected NavigateMsg, got %T", cmd())
	}
	if navMsg.Target.Kind != state.NavigateCloseOperations {
		t.Fatalf("expected NavigateCloseOperations, got %v", navMsg.Target.Kind)
	}
}
