package model

import (
	"fmt"
	"testing"

	"github.com/madicen/jj-tui/internal/tui/state"
)

// TestChromedSlotZOrderSnapshot characterizes the current priority / z-order of
// chromedSlot() BEFORE the P2.4 modal-stack refactor. chromedSlot() decides
// which modal wears the bubble-overlay window chrome on a given frame, and it
// resolves ties by a fixed priority:
//
//	initrepo > error > warning > (the ViewMode-selected form/overlay modal)
//
// P2.4 replaces the scattered modal-state booleans with a ModalStack whose Top
// drives this decision. This table locks in the observable outcome (the chrome
// "key" that wins) so the refactor can be proven behavior-preserving: the same
// combination of active modal conditions must resolve to the same key.
//
// PLAN(P2.4): keep this table green across the refactor; if a stack push order
// ever changes the winning key here, that is a user-visible z-order regression.
func TestChromedSlotZOrderSnapshot(t *testing.T) {
	// setup applies a combination of "active modal" conditions to a fresh model.
	type setup struct {
		name    string
		apply   func(m *Model)
		wantKey string
	}

	// Each ViewMode-selected overlay maps to a stable chrome key.
	viewModeKeys := []struct {
		vm  state.ViewMode
		key string
	}{
		{state.ViewEditDescription, "descedit"},
		{state.ViewCreatePR, "pr"},
		{state.ViewCreateTicket, "ticket"},
		{state.ViewCreateBookmark, "bookmark"},
		{state.ViewGitHubLogin, "githublogin"},
		{state.ViewBookmarkConflict, "conflict"},
		{state.ViewDivergentCommit, "divergent"},
		{state.ViewEvologSplit, "evolog"},
		{state.ViewFileDiff, "filediff"},
	}

	cases := make([]setup, 0, 2+2*len(viewModeKeys)+2)

	// 1. No modal active -> nothing is chromed.
	cases = append(cases, setup{
		name:    "no_modal/graph",
		apply:   func(m *Model) { m.appState.ViewMode = state.ViewCommitGraph },
		wantKey: "",
	})
	cases = append(cases, setup{
		name:    "no_modal/pull_requests",
		apply:   func(m *Model) { m.appState.ViewMode = state.ViewPullRequests },
		wantKey: "",
	})

	// 2. Each ViewMode-selected overlay wins on its own.
	for _, vk := range viewModeKeys {
		vk := vk
		cases = append(cases, setup{
			name:    "viewmode_only/" + vk.key,
			apply:   func(m *Model) { m.appState.ViewMode = vk.vm },
			wantKey: vk.key,
		})
	}

	// 3. Warning outranks every ViewMode overlay.
	for _, vk := range viewModeKeys {
		vk := vk
		cases = append(cases, setup{
			name: "warning_over/" + vk.key,
			apply: func(m *Model) {
				m.appState.ViewMode = vk.vm
				m.warningModal.Show("Heads up", "msg", nil)
			},
			wantKey: "warning",
		})
	}

	// 4. Error outranks warning and every ViewMode overlay.
	cases = append(cases, setup{
		name: "error_over_warning_and_viewmode",
		apply: func(m *Model) {
			m.appState.ViewMode = state.ViewEditDescription
			m.warningModal.Show("Heads up", "msg", nil)
			m.errorModal.SetError(fmt.Errorf("boom"), false, "")
		},
		wantKey: "error",
	})

	// 5. Init-repo outranks everything.
	cases = append(cases, setup{
		name: "initrepo_over_all",
		apply: func(m *Model) {
			m.appState.ViewMode = state.ViewEditDescription
			m.warningModal.Show("Heads up", "msg", nil)
			m.errorModal.SetError(fmt.Errorf("boom"), false, "")
			m.initRepoModel.SetPath("/test/path")
		},
		wantKey: "initrepo",
	})

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel()
			defer m.Close()
			tc.apply(m)
			key, _, _, _ := m.chromedSlot()
			if key != tc.wantKey {
				t.Fatalf("chromedSlot key = %q, want %q", key, tc.wantKey)
			}
		})
	}
}
