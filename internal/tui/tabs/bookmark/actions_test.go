package bookmark

import (
	"reflect"
	"testing"

	"github.com/madicen/jj-tui/internal"
)

// TestGetExistingBookmarks_RemoteTrackingDoesNotFilterLocal is the regression for the
// "move bookmark to parent commit" UX bug: the target commit only carried the
// remote-tracking entry (e.g. "feature@origin"), and that incorrectly filtered the
// local bookmark of the same name out of the move-popup list — leaving the user
// unable to move the local bookmark back onto its remote-tracking position from
// the modal. The fix only filters when the LOCAL ref is on the target commit.
func TestGetExistingBookmarks_RemoteTrackingDoesNotFilterLocal(t *testing.T) {
	repo := &internal.Repository{
		Graph: internal.CommitGraph{
			Commits: []internal.Commit{
				{ID: "c63d7fd8", ChangeID: "c63d7fd8"},
				{ID: "390d877f", ChangeID: "390d877f", Branches: []string{"product-crawler-staged-flow-kestra"}},
				{ID: "dac102d1", ChangeID: "dac102d1", Branches: []string{"product-crawler-staged-flow-kestra@origin"}},
				{ID: "f2fb36e4", ChangeID: "f2fb36e4", Branches: []string{"main"}},
			},
		},
	}
	// Target the commit that only has the remote-tracking entry; the local bookmark
	// (which is on a sibling commit) must still appear so the user can move it here.
	got := GetExistingBookmarks(repo, 2)
	want := []string{"main", "product-crawler-staged-flow-kestra"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetExistingBookmarks(@origin-only target) = %v, want %v", got, want)
	}
}

// TestGetExistingBookmarks_LocalOnCommitIsFiltered keeps the original guarantee: if
// the LOCAL ref is on the target commit, it's already there and shouldn't be offered
// as a move target.
func TestGetExistingBookmarks_LocalOnCommitIsFiltered(t *testing.T) {
	repo := &internal.Repository{
		Graph: internal.CommitGraph{
			Commits: []internal.Commit{
				{ID: "a", ChangeID: "a", Branches: []string{"feature"}},
				{ID: "b", ChangeID: "b", Branches: []string{"main"}},
			},
		},
	}
	got := GetExistingBookmarks(repo, 0)
	want := []string{"main"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetExistingBookmarks(local on target) = %v, want %v", got, want)
	}
}

// TestGetExistingBookmarks_BothLocalAndRemoteOnTarget covers the case where the target
// commit carries both the local and remote-tracking refs — the local IS here, so the
// bookmark should still be filtered out of the move list.
func TestGetExistingBookmarks_BothLocalAndRemoteOnTarget(t *testing.T) {
	repo := &internal.Repository{
		Graph: internal.CommitGraph{
			Commits: []internal.Commit{
				{ID: "a", ChangeID: "a", Branches: []string{"feature", "feature@origin"}},
				{ID: "b", ChangeID: "b", Branches: []string{"main"}},
			},
		},
	}
	got := GetExistingBookmarks(repo, 0)
	want := []string{"main"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetExistingBookmarks(local+remote on target) = %v, want %v", got, want)
	}
}
