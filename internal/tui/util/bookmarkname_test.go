package util

import "testing"

func TestNormalizeBookmarkListToken(t *testing.T) {
	tests := []struct {
		token       string
		wantName    string
		wantQMarker bool
	}{
		{"", "", false},
		{"  ", "", false},
		{"feature", "feature", false},
		{"feature*", "feature", false},
		{"feature?", "feature", true},
		{"feature?*", "feature", true},
		{"feature*?", "feature", true},
		{"feature??", "feature", true},
		{"my/feature-1", "my/feature-1", false},
	}
	for _, tt := range tests {
		name, q := NormalizeBookmarkListToken(tt.token)
		if name != tt.wantName || q != tt.wantQMarker {
			t.Errorf("NormalizeBookmarkListToken(%q) = (%q, %v); want (%q, %v)", tt.token, name, q, tt.wantName, tt.wantQMarker)
		}
	}
}

func TestFirstOperableBookmarkName_stripsConflictMarker(t *testing.T) {
	got := FirstOperableBookmarkName([]string{"wip?"})
	if got != "wip" {
		t.Fatalf("FirstOperableBookmarkName = %q; want wip", got)
	}
}
