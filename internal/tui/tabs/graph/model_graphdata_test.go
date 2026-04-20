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
