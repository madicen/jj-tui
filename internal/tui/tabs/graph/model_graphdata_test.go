package graph

import (
	"testing"

	"github.com/madicen/jj-tui/internal"
)

// DemoPullRequests includes an open PR on vhs/feature for the after-origin VHS tape. The graph must
// not propagate the parent's "main" bookmark into CreatePRBranch for the feature row — that used
// to set ctx.CreatePRBranch to main and block Create PR with "not available for main/master".
func TestBuildGraphData_commitBookmarkDoesNotInheritMainWhenFeatureHasOpenPR(t *testing.T) {
	m := GraphModel{
		repository: &internal.Repository{
			Graph: internal.CommitGraph{
				Commits: []internal.Commit{
					{ID: "root", ChangeID: "root", Parents: nil, Branches: []string{"main"}},
					{ID: "feat", ChangeID: "feat", Parents: []string{"root"}, Branches: []string{"vhs/feature"}},
				},
			},
			PRs: []internal.GitHubPR{
				{State: "open", HeadBranch: "vhs/feature", Number: 901},
			},
		},
		selectedCommit: 1,
	}
	data := m.buildGraphData()
	if got := data.CommitBookmark[1]; got == "main" || isDefaultBranch(got) {
		t.Fatalf("CommitBookmark[1] = %q; must not inherit default branch for create-PR context", got)
	}
	if got := data.CommitBookmark[1]; got != "" {
		t.Fatalf("CommitBookmark[1] = %q; want empty when feature bookmark already has an open PR", got)
	}
}

// After "Merge from main", the working copy is a merge commit carrying the local bookmark (ahead of
// @origin). As long as the bookmark's open PR is present in Repository.PRs, the graph must offer
// "Update PR" (CommitPRBranch set) and not "Create PR" (CommitBookmark empty). This is the scenario
// where the open PR was missing from the bulk list; the targeted lookup adds it back so this holds.
func TestBuildGraphData_mergeFromMainOffersUpdatePRWhenOpenPRPresent(t *testing.T) {
	const branch = "Add_Ivy_product_position_to_tech_platform"
	m := GraphModel{
		repository: &internal.Repository{
			Graph: internal.CommitGraph{
				Commits: []internal.Commit{
					{ID: "merge", ChangeID: "merge", Parents: []string{"mainc", "origintip"}, Branches: []string{branch}, IsWorking: true},
					{ID: "mainc", ChangeID: "mainc", Parents: []string{"origintip"}, Branches: []string{"main"}, Immutable: true},
					{ID: "origintip", ChangeID: "origintip", Parents: nil, Branches: []string{branch + "@origin"}},
				},
			},
			PRs: []internal.GitHubPR{
				{State: "open", HeadBranch: branch, Number: 6358},
			},
		},
		selectedCommit: 0,
	}
	data := m.buildGraphData()
	if got := data.CommitPRBranch[0]; got != branch {
		t.Fatalf("CommitPRBranch[0] = %q; want %q so the merge commit offers Update PR", got, branch)
	}
	if got := data.CommitBookmark[0]; got != "" {
		t.Fatalf("CommitBookmark[0] = %q; want empty (Update PR, not Create PR) when an open PR exists", got)
	}
}

// When the open PR is absent from Repository.PRs (the reported bug: a busy repo's bulk fetch omitted
// it), the graph falls back to "Create PR". This is what the targeted per-branch lookup repairs by
// adding the open PR to the list.
func TestBuildGraphData_mergeFromMainFallsBackToCreatePRWhenPRMissing(t *testing.T) {
	const branch = "Add_Ivy_product_position_to_tech_platform"
	m := GraphModel{
		repository: &internal.Repository{
			Graph: internal.CommitGraph{
				Commits: []internal.Commit{
					{ID: "merge", ChangeID: "merge", Parents: []string{"mainc", "origintip"}, Branches: []string{branch}, IsWorking: true},
					{ID: "mainc", ChangeID: "mainc", Parents: []string{"origintip"}, Branches: []string{"main"}, Immutable: true},
					{ID: "origintip", ChangeID: "origintip", Parents: nil, Branches: []string{branch + "@origin"}},
				},
			},
			PRs: nil,
		},
		selectedCommit: 0,
	}
	data := m.buildGraphData()
	if got := data.CommitPRBranch[0]; got != "" {
		t.Fatalf("CommitPRBranch[0] = %q; want empty when no open PR is loaded", got)
	}
	if got := data.CommitBookmark[0]; got != branch {
		t.Fatalf("CommitBookmark[0] = %q; want %q (Create PR) when the open PR is missing", got, branch)
	}
}
