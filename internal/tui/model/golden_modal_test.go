package model

// Golden View() coverage for the overlay/modal layer rendered through the root
// Model.View() composite: error, warning, bookmark-conflict, github-login,
// workspaces, the four form modals (describe / create-PR / create-bookmark /
// create-ticket), graph rebase/merge modes, and a list-tab context menu.
//
// These lock the FULL composited output (chrome titlebar + modal body +
// underlay tab) so the tab-interface migration can't change modal stacking,
// z-order, titles, or bodies without a loud failure.
//
// Update goldens with: go test ./internal/tui/model -run TestGoldenModal -update
//
// PLAN(P2.3): keep these green across the tab-interface migration.

import (
	"fmt"
	"testing"

	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/mock"
	"github.com/madicen/jj-tui/internal/testutil"
	"github.com/madicen/jj-tui/internal/tui/state"
	bookmarktab "github.com/madicen/jj-tui/internal/tui/tabs/bookmark"
	branchestab "github.com/madicen/jj-tui/internal/tui/tabs/branches"
	operationstab "github.com/madicen/jj-tui/internal/tui/tabs/operations"
	prformtab "github.com/madicen/jj-tui/internal/tui/tabs/prform"
	prstab "github.com/madicen/jj-tui/internal/tui/tabs/prs"
	ticketformtab "github.com/madicen/jj-tui/internal/tui/tabs/ticketform"
	workspacestab "github.com/madicen/jj-tui/internal/tui/tabs/workspaces"
)

func TestGoldenModalError(t *testing.T) {
	t.Run("no_retry", func(t *testing.T) {
		m := newGoldenModel(t)
		defer m.Close()
		m.errorModal.SetError(fmt.Errorf("failed to push bookmark: remote rejected update"), false, "")
		m.errorModal.SetWidth(m.width)
		m.errorModal.SetHeight(m.height)
		testutil.AssertGolden(t, "modal/error_no_retry", m.View())
	})

	t.Run("with_retry", func(t *testing.T) {
		m := newGoldenModel(t)
		defer m.Close()
		m.errorModal.SetError(fmt.Errorf("failed to load pull requests: network timeout"), true, "")
		m.errorModal.SetHasRetry(true)
		m.errorModal.SetWidth(m.width)
		m.errorModal.SetHeight(m.height)
		testutil.AssertGolden(t, "modal/error_with_retry", m.View())
	})
}

func TestGoldenModalWarning(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()
	m.warningModal.Show(
		"Commits Need Descriptions",
		"The following commits have no description. GitHub requires one before creating a PR.",
		[]internal.Commit{
			{Summary: "Refactor tab dispatch", ChangeID: "nmqrstuv"},
			{Summary: "Resolve merge conflict", ChangeID: "wpqxlmno"},
		},
	)
	testutil.AssertGolden(t, "modal/warning", m.View())
}

func TestGoldenModalBookmarkConflict(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()
	m.SetViewMode(state.ViewBranches)
	m.branchesTabModel.SetSelectedBranch(3)
	newModel, _ := m.Update(branchestab.BookmarkConflictInfoMsg{
		BookmarkName:  "hotfix/urgent",
		LocalID:       "cccc3333",
		RemoteID:      "eeee5555",
		LocalSummary:  "local hotfix tip",
		RemoteSummary: "origin hotfix tip",
		LocalWhen:     "2 hours ago",
		RemoteWhen:    "1 day ago",
	})
	m = newModel.(*Model)
	testutil.AssertGolden(t, "modal/bookmark_conflict", m.View())
}

func TestGoldenModalGitHubLogin(t *testing.T) {
	t.Run("device_flow", func(t *testing.T) {
		m := newGoldenModel(t)
		defer m.Close()
		m.beginModalUnderlay()
		m.githubLoginModel.SetDeviceFlow("device-code-xyz", "ABCD-1234", "https://github.com/login/device", 5)
		m.appState.ViewMode = state.ViewGitHubLogin
		testutil.AssertGolden(t, "modal/githublogin_device", m.View())
	})

	t.Run("gh_cli", func(t *testing.T) {
		m := newGoldenModel(t)
		defer m.Close()
		m.beginModalUnderlay()
		m.githubLoginModel.SetGhCLILoginMode()
		m.appState.ViewMode = state.ViewGitHubLogin
		testutil.AssertGolden(t, "modal/githublogin_ghcli", m.View())
	})
}

func TestGoldenModalWorkspaces(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()
	newModel, _ := m.Update(workspacestab.WorkspacesLoadedMsg{
		Workspaces: []jj.Workspace{
			{Name: "default", ChangeID: "aaaa1111", Description: "Add golden test safety net", Current: true},
			{Name: "feature-ws", ChangeID: "bbbb2222", Description: "Refactor tab dispatch"},
		},
	})
	m = newModel.(*Model)
	testutil.AssertGolden(t, "modal/workspaces", m.View())
}

func TestGoldenModalOperations(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()
	newModel, _ := m.Update(operationstab.OperationsLoadedMsg{
		Operations: []jj.Operation{
			{ID: "aaaa11112222", Description: "describe commit abcd1234", Time: "2026-07-07 22:56:26", User: "alice@host", IsCurrent: true},
			{ID: "bbbb33334444", Description: "new empty commit", Time: "2026-07-07 22:56:15", User: "alice@host"},
			{ID: "cccc55556666", Description: "snapshot working copy", Time: "2026-07-07 22:56:00", User: "alice@host"},
		},
	})
	m = newModel.(*Model)
	testutil.AssertGolden(t, "modal/operations", m.View())
}

func TestGoldenModalDescribe(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()
	commit := m.appState.Repository.Graph.Commits[1]
	m.beginModalUnderlay()
	m.appState.ViewMode = state.ViewEditDescription
	m.desceditModal.Show(commit.ChangeID, commit.ShortID)
	m.desceditModal.SetDescription("Refactor tab dispatch\n\nRoute background messages through the registry.")
	m.desceditModal.SetDimensions(ModalInnerWidth(m.width), maxInt(m.height-24, 3))
	testutil.AssertGolden(t, "modal/describe", m.View())
}

func TestGoldenModalCreatePR(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()
	m.graphTabModel.SelectCommit(0) // has bookmark "feature/goldens"
	m.beginModalUnderlay()
	res := prformtab.OpenCreatePR(&m.prFormModal, m.appState.Repository, 0, nil, "main",
		ModalInnerWidth(m.width), m.estimatedContentHeight())
	if !res.Ok {
		t.Fatalf("OpenCreatePR did not open a modal: %q", res.StatusMessage)
	}
	m.appState.ViewMode = state.ViewCreatePR
	testutil.AssertGolden(t, "modal/create_pr", m.View())
}

func TestGoldenModalCreateBookmark(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()
	m.graphTabModel.SelectCommit(1)
	m.beginModalUnderlay()
	m.appState.ViewMode = state.ViewCreateBookmark
	m.appState.StatusMessage = bookmarktab.OpenCreateBookmark(
		&m.bookmarkModal, m.appState.Repository, 1, nil, true, ModalInnerWidth(m.width))
	testutil.AssertGolden(t, "modal/create_bookmark", m.View())
}

func TestGoldenModalCreateTicket(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()
	svc := mock.NewTicketService("jira")
	m.appState.TicketService = svc
	m.beginModalUnderlay()
	res := ticketformtab.OpenCreateTicket(&m.ticketFormModal, svc,
		ModalInnerWidth(m.width), m.estimatedContentHeight())
	if !res.Ok {
		t.Fatalf("OpenCreateTicket did not open a modal: %q", res.StatusMessage)
	}
	m.appState.ViewMode = state.ViewCreateTicket
	testutil.AssertGolden(t, "modal/create_ticket", m.View())
}

func TestGoldenModalGraphModes(t *testing.T) {
	t.Run("rebase_mode", func(t *testing.T) {
		m := newGoldenModel(t)
		defer m.Close()
		m.SetViewMode(state.ViewCommitGraph)
		m.graphTabModel.SelectCommit(1)
		m.graphTabModel.StartRebaseMode(0)
		testutil.AssertGolden(t, "modal/graph_rebase_mode", m.View())
	})

	t.Run("merge_mode", func(t *testing.T) {
		m := newGoldenModel(t)
		defer m.Close()
		m.SetViewMode(state.ViewCommitGraph)
		m.graphTabModel.SelectCommit(1)
		m.graphTabModel.StartMergeMode(0)
		testutil.AssertGolden(t, "modal/graph_merge_mode", m.View())
	})
}

// TestGoldenModalPRContextMenu locks the list-tab long-press context menu, which
// the migration must preserve. The menu opens via a LongPressTickMsg matching the
// armed press id (listnav fields are set directly to avoid mouse-timing flakiness).
func TestGoldenModalPRContextMenu(t *testing.T) {
	m := newGoldenModel(t)
	defer m.Close()
	m.appState.GitHubService = nil
	m.prsTabModel.SetGithubService(true)
	m.SetViewMode(state.ViewPullRequests)
	m.SetSelectedPR(0)
	m.View() // lay out zones
	m.prsTabModel.LongPressItemIndex = 0
	m.prsTabModel.LongPressPressID = 1
	m.prsTabModel.LongPressMouseX = 10
	m.prsTabModel.LongPressMouseY = 3
	newModel, _ := m.Update(prstab.LongPressTickMsg{PressID: 1})
	m = newModel.(*Model)
	testutil.AssertGolden(t, "modal/pr_context_menu", m.View())
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
