package aiprompts

import (
	"strings"
	"testing"
)

func TestTicketUser_includesRevisionAndHints(t *testing.T) {
	u := TicketUser("abc123", "Draft title", "Draft body", "", "diff here")
	if !strings.Contains(u, "abc123") || !strings.Contains(u, "Draft title") || !strings.Contains(u, "Draft body") || !strings.Contains(u, "diff here") {
		t.Fatalf("unexpected TicketUser output: %q", u)
	}
	// No chain provided → prompt should keep the legacy "vs parents" wording so
	// the AI knows it's looking at a single commit's diff, not a stack.
	if !strings.Contains(u, "vs parents") {
		t.Fatalf("TicketUser without chain should describe the diff as 'vs parents'; got:\n%s", u)
	}
	if strings.Contains(u, "Cumulative unified diff") {
		t.Fatalf("TicketUser without chain must not advertise a cumulative diff; got:\n%s", u)
	}
}

func TestTicketUser_includesChainContextWhenProvided(t *testing.T) {
	chain := FormatChainSummary([]ChainCommitSummary{
		{ChangeIDShort: "aaaa1111", Subject: "Add login form"},
		{ChangeIDShort: "bbbb2222", Subject: "Wire login form to auth API", Description: "Wire login form to auth API\n\nIncludes retry on 5xx and basic telemetry."},
	})
	if chain == "" {
		t.Fatal("FormatChainSummary returned empty for non-empty input")
	}
	u := TicketUser("xyz789", "", "", chain, "cumulative diff body")
	for _, want := range []string{
		"Commits in this stack",
		"aaaa1111",
		"Add login form",
		"bbbb2222",
		"Wire login form to auth API",
		"Includes retry on 5xx",
		"Cumulative unified diff",
		"cumulative diff body",
		"whole chain of commits",
	} {
		if !strings.Contains(u, want) {
			t.Fatalf("TicketUser with chain missing %q; got:\n%s", want, u)
		}
	}
}

func TestBookmarkUser_chainSummaryReplacesUnifiedDiffHeader(t *testing.T) {
	chain := FormatChainSummary([]ChainCommitSummary{
		{ChangeIDShort: "11112222", Subject: "Refactor parser"},
		{ChangeIDShort: "33334444", Subject: "Add error recovery to parser"},
	})
	withChain := BookmarkUser("PROJ-1 stub", chain, "DIFF")
	if !strings.Contains(withChain, "Commits in this stack") || !strings.Contains(withChain, "Refactor parser") || !strings.Contains(withChain, "33334444") {
		t.Fatalf("BookmarkUser with chain missing chain block; got:\n%s", withChain)
	}
	if !strings.Contains(withChain, "Cumulative unified diff") {
		t.Fatalf("BookmarkUser with chain should label the diff as cumulative; got:\n%s", withChain)
	}
	if !strings.Contains(withChain, "whole chain of work") {
		t.Fatalf("BookmarkUser with chain should ask the AI to cover the whole chain; got:\n%s", withChain)
	}

	withoutChain := BookmarkUser("", "", "DIFF")
	if !strings.Contains(withoutChain, "Unified diff:") {
		t.Fatalf("BookmarkUser without chain should keep plain 'Unified diff:' header; got:\n%s", withoutChain)
	}
	if strings.Contains(withoutChain, "Commits in this stack") {
		t.Fatalf("BookmarkUser without chain must not include the chain header; got:\n%s", withoutChain)
	}
}

func TestPRUser_chainSummaryAddedBetweenHintAndDiff(t *testing.T) {
	chain := FormatChainSummary([]ChainCommitSummary{
		{ChangeIDShort: "cc11dd22", Subject: "Bump dependency X"},
	})
	u := PRUser("main", "feature/login", "PROJ-7 hint", chain, "diff body")
	for _, want := range []string{
		"Base branch: main",
		"Head branch: feature/login",
		"Commits in this PR",
		"cc11dd22",
		"Bump dependency X",
		"Cumulative unified diff",
		"diff body",
		"whole chain of commits",
	} {
		if !strings.Contains(u, want) {
			t.Fatalf("PRUser with chain missing %q; got:\n%s", want, u)
		}
	}
}

func TestFormatChainSummary_emptyReturnsEmpty(t *testing.T) {
	if got := FormatChainSummary(nil); got != "" {
		t.Fatalf("FormatChainSummary(nil) = %q, want empty string", got)
	}
	if got := FormatChainSummary([]ChainCommitSummary{}); got != "" {
		t.Fatalf("FormatChainSummary(empty) = %q, want empty string", got)
	}
}

// FormatChainSummary should skip a per-commit body that is identical to the
// subject — most commits in a stack are subject-only and repeating the same
// line just wastes tokens.
func TestFormatChainSummary_skipsBodyEqualToSubject(t *testing.T) {
	out := FormatChainSummary([]ChainCommitSummary{
		{ChangeIDShort: "abcd1234", Subject: "Same subject", Description: "Same subject"},
	})
	// Subject must appear exactly once — once in the bullet line, never as an
	// additional indented body line.
	if got := strings.Count(out, "Same subject"); got != 1 {
		t.Fatalf("subject count = %d, want 1; output:\n%s", got, out)
	}
	if strings.Contains(out, "\n   Same subject") {
		t.Fatalf("body identical to subject should be skipped; output:\n%s", out)
	}
}

// When the body extends the subject (subject + extra paragraphs) we keep the
// whole description so context isn't lost.
func TestFormatChainSummary_includesBodyWhenItExtendsSubject(t *testing.T) {
	out := FormatChainSummary([]ChainCommitSummary{
		{ChangeIDShort: "abcd1234", Subject: "Subject line", Description: "Subject line\n\nExtra body text"},
	})
	if !strings.Contains(out, "Extra body text") {
		t.Fatalf("body extension dropped; output:\n%s", out)
	}
}

func TestFormatChainSummary_overflowAddsTrailer(t *testing.T) {
	commits := make([]ChainCommitSummary, maxChainCommitsInPrompt+3)
	for i := range commits {
		commits[i] = ChainCommitSummary{
			ChangeIDShort: "deadbeef",
			Subject:       "Commit number " + string(rune('a'+i%26)),
		}
	}
	out := FormatChainSummary(commits)
	if !strings.Contains(out, "+3 more commits") {
		t.Fatalf("overflow trailer missing for %d commits; output:\n%s", len(commits), out)
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
