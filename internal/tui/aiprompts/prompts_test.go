package aiprompts

import (
	"strings"
	"testing"
)

func TestTicketUser_includesRevisionAndHints(t *testing.T) {
	u := TicketUser("abc123", "Draft title", "Draft body", "diff here")
	if !strings.Contains(u, "abc123") || !strings.Contains(u, "Draft title") || !strings.Contains(u, "Draft body") || !strings.Contains(u, "diff here") {
		t.Fatalf("unexpected TicketUser output: %q", u)
	}
}

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

func TestParsePRTitleBody_MarkdownFence(t *testing.T) {
	title, body := ParsePRTitleBody("```json\n{\"title\":\"T\",\"body\":\"B\"}\n```")
	if title != "T" || body != "B" {
		t.Fatalf("got title=%q body=%q", title, body)
	}
}

func TestParsePRTitleBody_BlockquotedFence(t *testing.T) {
	in := "> ```json\n" +
		"> {\"title\":\"Enhance AI\",\"body\":\"## Symptoms\\n\\nDetails\"}\n" +
		"> ```"
	title, body := ParsePRTitleBody(in)
	if title != "Enhance AI" || body != "## Symptoms\n\nDetails" {
		t.Fatalf("got title=%q body=%q", title, body)
	}
}

func TestParsePRTitleBody_PrettyPrintedMultilineJSON(t *testing.T) {
	// Models often emit pretty-printed objects; line-based fallback used to treat "{" alone as title.
	in := "{\n  \"title\": \"Refactor settings tab\",\n  \"body\": \"## Symptoms\\n\\nExpected vs actual.\"\n}"
	title, body := ParsePRTitleBody(in)
	if title != "Refactor settings tab" || body != "## Symptoms\n\nExpected vs actual." {
		t.Fatalf("got title=%q body=%q", title, body)
	}
}

func TestParsePRTitleBody_PreambleThenPrettyJSON(t *testing.T) {
	in := "Here is the ticket draft:\n\n{\n  \"title\": \"Fix parse\",\n  \"body\": \"Details\"\n}\n"
	title, body := ParsePRTitleBody(in)
	if title != "Fix parse" || body != "Details" {
		t.Fatalf("got title=%q body=%q", title, body)
	}
}

func TestParsePRTitleBody_BodyWithBracesInsideString(t *testing.T) {
	// Closing brace appears inside the JSON string; slice [first { : last }] would mis-parse.
	in := `{"title":"T","body":"See } and { in markdown or pseudo-code"}`
	title, body := ParsePRTitleBody(in)
	if title != "T" || body != "See } and { in markdown or pseudo-code" {
		t.Fatalf("got title=%q body=%q", title, body)
	}
}

// Reproduces the failure mode where a local model emits pretty-printed JSON but
// uses literal line breaks inside the body string instead of \n escapes. The old
// parser fell back to line splitting and yielded title="{" with the entire JSON
// in the body.
func TestParsePRTitleBody_LiteralNewlinesInBody(t *testing.T) {
	in := "{\n" +
		"  \"title\": \"Implement hybrid backend for product crawler\",\n" +
		"  \"body\": \"The proposed changes introduce a new `hybrid` backend.\n" +
		"\n" +
		"**Symptoms:**\n" +
		"- Existing AI-only pipeline is too expensive.\n" +
		"- Operators need to balance cost and accuracy.\"\n" +
		"}"
	title, body := ParsePRTitleBody(in)
	if title != "Implement hybrid backend for product crawler" {
		t.Fatalf("title = %q", title)
	}
	wantBody := "The proposed changes introduce a new `hybrid` backend.\n\n**Symptoms:**\n- Existing AI-only pipeline is too expensive.\n- Operators need to balance cost and accuracy."
	if body != wantBody {
		t.Fatalf("body mismatch\n got: %q\nwant: %q", body, wantBody)
	}
}

// Mixed: title is on its own line, body contains both real \n escapes (already
// correct) and unrelated literal whitespace around the value. The relaxed pass
// must not double-escape the proper \n sequences.
func TestParsePRTitleBody_LiteralAndEscapedNewlinesMixed(t *testing.T) {
	in := "{\n" +
		"  \"title\": \"Title\",\n" +
		"  \"body\": \"line one\\nline two\nline three\"\n" +
		"}"
	title, body := ParsePRTitleBody(in)
	if title != "Title" {
		t.Fatalf("title = %q", title)
	}
	if body != "line one\nline two\nline three" {
		t.Fatalf("body = %q", body)
	}
}

// Tabs inside the body value also have to be relaxed; otherwise json.Decoder
// rejects them with "invalid character '\\t' in string literal".
func TestParsePRTitleBody_LiteralTabInBody(t *testing.T) {
	in := "{\"title\":\"T\",\"body\":\"col1\tcol2\"}"
	_, body := ParsePRTitleBody(in)
	if body != "col1\tcol2" {
		t.Fatalf("body = %q", body)
	}
}

func TestMergeGeneratedPRTitle_prependsDroppedJiraKey(t *testing.T) {
	hint := "PROJ-456 - Original Jira summary"
	gen := "Add retry logic for checkout"
	want := "PROJ-456 - Add retry logic for checkout"
	if got := MergeGeneratedPRTitle(hint, gen); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestMergeGeneratedPRTitle_bracketedHint(t *testing.T) {
	hint := "[TEAM-12] Some ticket title"
	gen := "Fix flaky integration test"
	want := "[TEAM-12] Fix flaky integration test"
	if got := MergeGeneratedPRTitle(hint, gen); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestMergeGeneratedPRTitle_noopWhenModelKeptKey(t *testing.T) {
	hint := "ABC-1 - Old"
	gen := "ABC-1 - New improved title"
	if got := MergeGeneratedPRTitle(hint, gen); got != gen {
		t.Fatalf("got %q want %q", got, gen)
	}
}

func TestMergeGeneratedPRTitle_keyOnlyHint(t *testing.T) {
	hint := "FOO-99"
	gen := "Deploy config split"
	want := "FOO-99 - Deploy config split"
	if got := MergeGeneratedPRTitle(hint, gen); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestMergeGeneratedPRTitle_noHintKey(t *testing.T) {
	gen := "Plain title from model"
	if got := MergeGeneratedPRTitle("Just words", gen); got != gen {
		t.Fatalf("got %q want %q", got, gen)
	}
}
