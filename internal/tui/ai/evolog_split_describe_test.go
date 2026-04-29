package ai

import "testing"

func TestParseEvologDescribeSplitJSONDual(t *testing.T) {
	raw := `{"parent_description":"P line","child_description":"C line"}`
	pd, cd, err := parseEvologDescribeSplitJSON(raw, true)
	if err != nil {
		t.Fatal(err)
	}
	if pd != "P line" || cd != "C line" {
		t.Fatalf("got %q %q", pd, cd)
	}
}

func TestParseEvologDescribeSplitJSONDualRequiresParent(t *testing.T) {
	raw := `{"child_description":"only child"}`
	_, _, err := parseEvologDescribeSplitJSON(raw, true)
	if err == nil {
		t.Fatal("expected error when parent required")
	}
}

func TestParseEvologDescribeSplitJSONChildOnly(t *testing.T) {
	raw := `{"child_description":"only child"}`
	pd, cd, err := parseEvologDescribeSplitJSON(raw, false)
	if err != nil {
		t.Fatal(err)
	}
	if pd != "" || cd != "only child" {
		t.Fatalf("got %q %q", pd, cd)
	}
}
