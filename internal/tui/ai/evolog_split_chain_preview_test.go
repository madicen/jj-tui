package ai

import (
	"strings"
	"testing"
)

func TestParseEvologChainPreviewJSON(t *testing.T) {
	raw := `{"steps":[{"label":"FAQ","description":"Rewire parent"},{"label":"Hunk peel","description":"API in first commit"}],"final_parent_description":"Parent title\n","final_child_description":"Tip title\nbody"}`
	steps, pd, cd, err := parseEvologChainPreviewJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 2 {
		t.Fatalf("steps=%d", len(steps))
	}
	if steps[0].Label != "FAQ" || !strings.Contains(steps[0].Description, "Rewire") {
		t.Fatalf("step0=%+v", steps[0])
	}
	if pd == "" || cd == "" {
		t.Fatalf("pd=%q cd=%q", pd, cd)
	}
}
