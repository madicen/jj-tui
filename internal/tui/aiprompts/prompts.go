package aiprompts

import (
	"encoding/json"
	"fmt"
	"strings"
)

const maxPromptDiffRunes = 100_000

// maxChainCommitsInPrompt caps how many commit summaries from the chain we
// inline into a prompt. Beyond this we collapse into a "+N more commits …"
// line so the prompt stays roughly bounded for long stacks; the cumulative
// diff still carries the full code context.
const maxChainCommitsInPrompt = 50

// maxChainCommitDescRunes truncates each commit's full description in the
// chain summary so a single very-chatty commit can't blow past the prompt
// budget. The first-line subject is always included verbatim.
const maxChainCommitDescRunes = 800

func truncateRunes(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "\n\n[truncated]\n"
}

// ChainCommitSummary is the per-commit context the prompt builders inline
// before the cumulative diff. Keeping it as plain strings (no jj-package
// dependency) lets the aiprompts package stay free of the integration layer.
type ChainCommitSummary struct {
	ChangeIDShort string
	Subject       string
	Description   string
}

// FormatChainSummary renders a list of chain commits as a compact bulletted
// block suitable for inlining in a user message. Returns "" when commits is
// empty so callers can drop the whole block from the prompt.
//
// Format:
//
//	1. abc12345  Short subject of commit one
//	   <indented full description>
//	2. def67890  Subject of commit two
//	   ...
func FormatChainSummary(commits []ChainCommitSummary) string {
	if len(commits) == 0 {
		return ""
	}
	var b strings.Builder
	limit := len(commits)
	overflow := 0
	if limit > maxChainCommitsInPrompt {
		overflow = limit - maxChainCommitsInPrompt
		limit = maxChainCommitsInPrompt
	}
	for i := 0; i < limit; i++ {
		c := commits[i]
		subject := strings.TrimSpace(c.Subject)
		if subject == "" {
			subject = "(no description)"
		}
		fmt.Fprintf(&b, "%d. %s  %s\n", i+1, c.ChangeIDShort, subject)
		desc := strings.TrimSpace(c.Description)
		// Skip the body when it's identical to (or empty beyond) the subject
		// — most commits in a stack are subject-only, and repeating it just
		// wastes tokens. We compare on the description's first line so a
		// commit with subject + extra body still gets the body included.
		if desc == "" {
			continue
		}
		firstLine := desc
		if nl := strings.IndexByte(desc, '\n'); nl >= 0 {
			firstLine = desc[:nl]
		}
		if strings.TrimSpace(firstLine) == subject && strings.TrimSpace(desc[len(firstLine):]) == "" {
			continue
		}
		body := truncateRunes(desc, maxChainCommitDescRunes)
		for _, line := range strings.Split(body, "\n") {
			b.WriteString("   ")
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	if overflow > 0 {
		fmt.Fprintf(&b, "… (+%d more commits in this chain not shown)\n", overflow)
	}
	return strings.TrimRight(b.String(), "\n")
}

// CommitDescriptionSystem is the system prompt for commit messages / jj descriptions.
const CommitDescriptionSystem = `You write concise version control commit descriptions for Jujutsu (jj) / Git.
Output plain text only: a short title line (optional) and a blank line and then bullet details if useful.
Do not wrap output in markdown code fences. No preamble.`

// CommitDescriptionUser builds the user message for describing a single change.
func CommitDescriptionUser(changeIDShort, currentDescription, diff string) string {
	var b strings.Builder
	b.WriteString("Change id (short): ")
	b.WriteString(changeIDShort)
	b.WriteString("\n\nCurrent description (may be empty):\n")
	b.WriteString(strings.TrimSpace(currentDescription))
	b.WriteString("\n\nUnified diff (vs parents):\n")
	b.WriteString(truncateRunes(diff, maxPromptDiffRunes))
	b.WriteString("\n\nWrite an improved commit description for this change.")
	return b.String()
}

// PRSystem is the system prompt for pull request title + body generation.
const PRSystem = `You write pull request titles and bodies for GitHub.
Respond with a single JSON object only, no markdown fences, using exactly these keys:
{"title":"...","body":"..."}
The response must be valid JSON: newlines inside body must be \n escapes inside the quoted string, not literal line breaks.
Title: at most 200 characters. Body: markdown allowed, be clear and concise.
If—and only if—the "Suggested title hint" in the user message already begins with a real ticket/issue id from that hint (e.g. a short alphanumeric key with hyphens, sometimes in brackets), copy that exact prefix into the JSON title and only improve the rest. If the hint has no such prefix, the JSON title must not add one: do not invent keys, placeholders, or example ids from these instructions.`

// PRUser builds the user message for PR title+body. chainSummary is an
// optional pre-formatted block listing the commits in baseBranch..headBranch
// (oldest → newest); pass "" to omit. The diff should be the cumulative
// baseBranch → head diff (everything the PR introduces).
func PRUser(baseBranch, headBranch, hintTitle, chainSummary, diff string) string {
	var b strings.Builder
	b.WriteString("Base branch: ")
	b.WriteString(baseBranch)
	b.WriteString("\nHead branch: ")
	b.WriteString(headBranch)
	b.WriteString("\n\nSuggested title hint (from branch or ticket; may be empty):\n")
	b.WriteString(strings.TrimSpace(hintTitle))
	if s := strings.TrimSpace(chainSummary); s != "" {
		b.WriteString("\n\nCommits in this PR (oldest → newest, the chain from base to head):\n")
		b.WriteString(s)
	}
	b.WriteString("\n\nCumulative unified diff (base → head, covering every commit above):\n")
	b.WriteString(truncateRunes(diff, maxPromptDiffRunes))
	b.WriteString("\n\nWrite JSON with title and body for this pull request that reflects the whole chain of commits, not just the tip. Preserve a leading ticket/issue prefix only when it is already present in the hint above; never invent one.")
	return b.String()
}

// TicketSystem is the system prompt for tracker ticket drafts from a diff.
const TicketSystem = `You draft issue-tracker tickets (title + description) that a code change would address or close.
The user supplies a unified diff of a proposed fix. Infer what problem or task the change solves, and write the ticket as if filed before the fix: clear title, markdown body with symptoms, expected vs actual where inferable, and scope. Stay conservative—only state what the diff reasonably supports; say what is unknown rather than guessing.
Respond with a single JSON object only, no markdown fences, using exactly these keys:
{"title":"...","body":"..."}
The response must be valid JSON: any newlines inside body must be written as \n inside the quoted string, not as literal line breaks. Prefer one line for the whole object if unsure.
Title: at most 300 characters. Body: markdown allowed. If—and only if—the draft title in the user message already starts with a real ticket/issue id from that text, keep that exact prefix in the JSON title; otherwise do not add any key or placeholder.`

// TicketUser builds the user message for AI-filled create-ticket fields.
// chainSummary is an optional pre-formatted block listing the commits in
// trunk..changeID (oldest → newest); pass "" to omit. When provided, diff
// should be the cumulative trunk → changeID diff so the AI sees the whole
// stack of work the ticket should describe.
func TicketUser(changeIDShort, hintSummary, hintDescription, chainSummary, diff string) string {
	var b strings.Builder
	b.WriteString("Revision (short id or @): ")
	b.WriteString(changeIDShort)
	b.WriteString("\n\nDraft title already in form (may be empty):\n")
	b.WriteString(strings.TrimSpace(hintSummary))
	b.WriteString("\n\nDraft description already in form (may be empty):\n")
	b.WriteString(strings.TrimSpace(hintDescription))
	if s := strings.TrimSpace(chainSummary); s != "" {
		b.WriteString("\n\nCommits in this stack (oldest → newest, from trunk up to the selected change):\n")
		b.WriteString(s)
		b.WriteString("\n\nCumulative unified diff (trunk → selected change, covering every commit above):\n")
	} else {
		b.WriteString("\n\nUnified diff (vs parents):\n")
	}
	b.WriteString(truncateRunes(diff, maxPromptDiffRunes))
	b.WriteString("\n\nWrite JSON with title and body for the tracker ticket this work would fix or implement, reflecting the whole chain of commits when more than one is shown.")
	return b.String()
}

func stripMarkdownLineBlockquote(line string) string {
	s := line
	for {
		t := strings.TrimLeft(s, " \t")
		if !strings.HasPrefix(t, ">") {
			return t
		}
		s = strings.TrimLeft(t[1:], " \t")
	}
}

func extractMarkdownFenceContent(s string) (string, bool) {
	start := strings.Index(s, "```")
	if start < 0 {
		return "", false
	}
	rest := s[start+3:]
	rest = strings.TrimLeft(rest, "\r\n")
	if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
		first := strings.TrimSpace(rest[:nl])
		if strings.EqualFold(first, "json") {
			rest = rest[nl+1:]
		}
	}
	end := strings.LastIndex(rest, "```")
	if end < 0 {
		return "", false
	}
	return strings.TrimSpace(rest[:end]), true
}

func normalizeStructuredLLMOutput(raw string) string {
	raw = strings.TrimSpace(raw)
	if inner, ok := extractMarkdownFenceContent(raw); ok {
		raw = inner
	}
	lines := strings.Split(raw, "\n")
	for i := range lines {
		lines[i] = stripMarkdownLineBlockquote(lines[i])
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// decodeTitleBodyJSON parses the first JSON object from raw, which may span multiple lines
// (pretty-printed models). json.Decoder stops after one value, so braces inside strings do
// not corrupt the slice the way strings.LastIndex(raw, "}") can when the object is multiline.
func decodeTitleBodyJSON(raw string) (title, body string, ok bool) {
	raw = strings.TrimSpace(raw)
	i := strings.Index(raw, "{")
	if i < 0 {
		return "", "", false
	}
	dec := json.NewDecoder(strings.NewReader(raw[i:]))
	dec.UseNumber()
	var v struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if err := dec.Decode(&v); err != nil {
		return "", "", false
	}
	return strings.TrimSpace(v.Title), strings.TrimSpace(v.Body), true
}

// relaxLiteralWhitespaceInJSONStrings escapes literal control whitespace that appears
// inside JSON string values, producing a payload json.Decoder will accept.
//
// Several local models (including ollama's smaller coder variants) ignore the
// "newlines inside body must be \n escapes" instruction and emit something like:
//
//	{
//	  "title": "Implement hybrid backend",
//	  "body": "Symptoms:
//	- bullet one
//	- bullet two"
//	}
//
// which is invalid JSON. Walking the buffer with a tiny string-tracking state machine
// is enough to rewrite the embedded \n/\r/\t/\b/\f into their escape forms while
// leaving anything outside string values alone. We don't try to handle unescaped
// quotes inside strings because there is no reliable way to recover those without
// re-running the LLM.
func relaxLiteralWhitespaceInJSONStrings(raw string) string {
	var b strings.Builder
	b.Grow(len(raw) + len(raw)/8)
	inString := false
	escaped := false
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if escaped {
			b.WriteByte(c)
			escaped = false
			continue
		}
		if inString && c == '\\' {
			b.WriteByte(c)
			escaped = true
			continue
		}
		if c == '"' {
			inString = !inString
			b.WriteByte(c)
			continue
		}
		if !inString {
			b.WriteByte(c)
			continue
		}
		switch c {
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// decodeTitleBodyJSONLenient is decodeTitleBodyJSON plus a single retry on a
// whitespace-relaxed copy. We keep the strict attempt first so well-formed JSON
// is never reinterpreted (e.g. a body that legitimately contains the two-character
// sequence backslash+n stays intact).
func decodeTitleBodyJSONLenient(raw string) (title, body string, ok bool) {
	if t, b, ok := decodeTitleBodyJSON(raw); ok {
		return t, b, true
	}
	if relaxed := relaxLiteralWhitespaceInJSONStrings(raw); relaxed != raw {
		if t, b, ok := decodeTitleBodyJSON(relaxed); ok {
			return t, b, true
		}
	}
	return "", "", false
}

// ParsePRTitleBody extracts title and body from model output (JSON preferred).
func ParsePRTitleBody(raw string) (title, body string) {
	raw = normalizeStructuredLLMOutput(raw)
	raw = strings.TrimSpace(raw)
	if t, b, ok := decodeTitleBodyJSONLenient(raw); ok {
		return t, b
	}
	// Last-ditch slice from first { to last }: handles cases where the model wraps
	// the JSON with trailing prose that begins with a stray opening brace inside it.
	// json.Decoder above already tolerates plain trailing prose on its own.
	if i := strings.Index(raw, "{"); i >= 0 {
		if j := strings.LastIndex(raw, "}"); j > i {
			if t, b, ok := decodeTitleBodyJSONLenient(raw[i : j+1]); ok {
				return t, b
			}
		}
	}
	lines := strings.Split(raw, "\n")
	if len(lines) == 1 {
		return lines[0], ""
	}
	title = strings.TrimSpace(lines[0])
	body = strings.TrimSpace(strings.Join(lines[1:], "\n"))
	return title, body
}

// BookmarkSystem is the system prompt for bookmark / branch name suggestions.
const BookmarkSystem = `You suggest a short git branch / bookmark name: lowercase, use hyphens between words, only letters, digits, hyphens, underscores.
Output a single line: the name only, no quotes, no explanation. Max 60 characters.`

// BookmarkUser builds context for bookmark name suggestion. chainSummary is
// an optional pre-formatted block listing the commits in trunk..selected
// (oldest → newest); pass "" to omit. When provided, diff should be the
// cumulative trunk → selected diff so the suggested name reflects the whole
// stack of work, not just the tip commit's local changes.
func BookmarkUser(ticketHint, chainSummary, diff string) string {
	var b strings.Builder
	if strings.TrimSpace(ticketHint) != "" {
		b.WriteString("Ticket / context hint: ")
		b.WriteString(strings.TrimSpace(ticketHint))
		b.WriteString("\n\n")
	}
	if s := strings.TrimSpace(chainSummary); s != "" {
		b.WriteString("Commits in this stack (oldest → newest, from trunk up to the selected change):\n")
		b.WriteString(s)
		b.WriteString("\n\nCumulative unified diff (trunk → selected change, covering every commit above):\n")
	} else {
		b.WriteString("Unified diff:\n")
	}
	b.WriteString(truncateRunes(diff, maxPromptDiffRunes))
	b.WriteString("\n\nSuggest one bookmark name that covers the whole chain of work above, not just the tip commit.")
	return b.String()
}
