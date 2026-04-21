package ai

import "testing"

func TestParsePRTitleBody_JSON(t *testing.T) {
	title, body := ParsePRTitleBody(`Here is JSON:
{"title":"Fix bug","body":"## Summary\n- a\n- b"}`)
	if title != "Fix bug" || body != "## Summary\n- a\n- b" {
		t.Fatalf("got title=%q body=%q", title, body)
	}
}

func TestParsePRTitleBody_FallbackLines(t *testing.T) {
	title, body := ParsePRTitleBody("One line title\n\nBody para")
	if title != "One line title" || body != "Body para" {
		t.Fatalf("got title=%q body=%q", title, body)
	}
}
