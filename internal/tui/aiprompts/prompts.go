package aiprompts

import (
	"encoding/json"
	"strings"
)

const maxPromptDiffRunes = 100_000

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

// PRUser builds the user message for PR title+body.
func PRUser(baseBranch, headBranch, hintTitle, diff string) string {
	var b strings.Builder
	b.WriteString("Base branch: ")
	b.WriteString(baseBranch)
	b.WriteString("\nHead branch: ")
	b.WriteString(headBranch)
	b.WriteString("\n\nSuggested title hint (from branch or ticket; may be empty):\n")
	b.WriteString(strings.TrimSpace(hintTitle))
	b.WriteString("\n\nUnified diff:\n")
	b.WriteString(truncateRunes(diff, maxPromptDiffRunes))
	b.WriteString("\n\nWrite JSON with title and body for this pull request. Preserve a leading ticket/issue prefix only when it is already present in the hint above; never invent one.")
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
func TicketUser(changeIDShort, hintSummary, hintDescription, diff string) string {
	var b strings.Builder
	b.WriteString("Revision (short id or @): ")
	b.WriteString(changeIDShort)
	b.WriteString("\n\nDraft title already in form (may be empty):\n")
	b.WriteString(strings.TrimSpace(hintSummary))
	b.WriteString("\n\nDraft description already in form (may be empty):\n")
	b.WriteString(strings.TrimSpace(hintDescription))
	b.WriteString("\n\nUnified diff (vs parents):\n")
	b.WriteString(truncateRunes(diff, maxPromptDiffRunes))
	b.WriteString("\n\nWrite JSON with title and body for the tracker ticket this change would fix or implement.")
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

// BookmarkUser builds context for bookmark name suggestion.
func BookmarkUser(ticketHint, diff string) string {
	var b strings.Builder
	if strings.TrimSpace(ticketHint) != "" {
		b.WriteString("Ticket / context hint: ")
		b.WriteString(strings.TrimSpace(ticketHint))
		b.WriteString("\n\n")
	}
	b.WriteString("Unified diff:\n")
	b.WriteString(truncateRunes(diff, maxPromptDiffRunes))
	b.WriteString("\n\nSuggest one bookmark name for this work.")
	return b.String()
}
