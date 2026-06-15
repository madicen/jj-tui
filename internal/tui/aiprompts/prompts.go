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

// maxFilesInStat caps how many files we list in the "Files changed" summary so
// a sprawling change can't blow the prompt budget; the rest collapse into a
// "+N more files" trailer. The raw diff still carries full per-line context.
const maxFilesInStat = 50

type fileStat struct {
	path           string
	added, removed int
}

// diffFileStats walks a git-format unified diff and returns per-file add/remove
// counts in first-seen order. File header lines ("+++ ", "--- ") and hunk
// markers are not counted as content changes.
func diffFileStats(diff string) []fileStat {
	var stats []fileStat
	idx := map[string]int{}
	cur := -1
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			path := parseDiffGitPath(line)
			if path == "" {
				cur = -1
				continue
			}
			if i, ok := idx[path]; ok {
				cur = i
				continue
			}
			idx[path] = len(stats)
			cur = len(stats)
			stats = append(stats, fileStat{path: path})
			continue
		}
		if cur < 0 {
			continue
		}
		switch {
		case strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- "):
			// file header, not a content change
		case strings.HasPrefix(line, "+"):
			stats[cur].added++
		case strings.HasPrefix(line, "-"):
			stats[cur].removed++
		}
	}
	return stats
}

// parseDiffGitPath extracts the post-image path from a "diff --git a/x b/y"
// header, returning the "b/" side with its prefix stripped. Returns "" when the
// line doesn't parse (paths with embedded spaces are uncommon and fall back to
// the empty string, which callers treat as "no file").
func parseDiffGitPath(line string) string {
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return ""
	}
	return strings.TrimPrefix(fields[len(fields)-1], "b/")
}

// FormatDiffFileStat renders a compact "Files changed" block from a git-format
// diff, e.g.:
//
//	Files changed (2):
//	  internal/tui/aiprompts/prompts.go  (+62 -8)
//	  internal/tui/aiprompts/prompts_test.go  (+57 -0)
//
// Returns "" when no files are detected so callers can omit the block.
func FormatDiffFileStat(diff string) string {
	stats := diffFileStats(diff)
	if len(stats) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Files changed (%d):\n", len(stats))
	limit := len(stats)
	overflow := 0
	if limit > maxFilesInStat {
		overflow = limit - maxFilesInStat
		limit = maxFilesInStat
	}
	for i := 0; i < limit; i++ {
		s := stats[i]
		fmt.Fprintf(&b, "  %s  (+%d -%d)\n", s.path, s.added, s.removed)
	}
	if overflow > 0 {
		fmt.Fprintf(&b, "  … (+%d more files)\n", overflow)
	}
	return strings.TrimRight(b.String(), "\n")
}

// CommitDescriptionSystem is the system prompt for commit messages / jj descriptions.
const CommitDescriptionSystem = `You write commit subject lines for Jujutsu (jj) / Git.
Output exactly one line: a short imperative summary of the change's primary purpose (aim for under ~72 characters).
Use the "Files changed" summary to judge scope: lead with the change that dominates the diff. If the diff touches clearly unrelated areas, describe the overarching intent rather than a single file. Do not infer purpose from one incidental hunk.
No body, no bullets, no blank lines, no trailing period, no markdown code fences, no preamble.`

// CommitDescriptionUser builds the user message for describing a single change.
func CommitDescriptionUser(changeIDShort, currentDescription, diff string) string {
	var b strings.Builder
	b.WriteString("Change id (short): ")
	b.WriteString(changeIDShort)
	b.WriteString("\n\nCurrent description (may be empty):\n")
	b.WriteString(strings.TrimSpace(currentDescription))
	if stat := FormatDiffFileStat(diff); stat != "" {
		b.WriteString("\n\n")
		b.WriteString(stat)
	}
	b.WriteString("\n\nUnified diff (vs parents):\n")
	b.WriteString(truncateRunes(diff, maxPromptDiffRunes))
	b.WriteString("\n\nWrite a single concise subject line capturing the main purpose of this change.")
	return b.String()
}

// Change-size tiers, keyed off the size of the cumulative diff. Thresholds are
// deliberately generous so a one-line fix and a large refactor get visibly
// different detail budgets; tune here if generated bodies feel too terse or too
// chatty. Shared by PR and ticket generation.
const (
	smallChangeMaxLines = 30
	largeChangeMaxLines = 300
	smallChangeMaxFiles = 1
)

type changeTier int

const (
	tierSmall changeTier = iota
	tierMedium
	tierLarge
)

// diffChangeTier classifies a cumulative diff into a size tier used to scale how
// much detail we ask the model for.
func diffChangeTier(diff string) changeTier {
	changedLines, files := countDiffChange(diff)
	switch {
	case changedLines < smallChangeMaxLines && files <= smallChangeMaxFiles:
		return tierSmall
	case changedLines < largeChangeMaxLines:
		return tierMedium
	default:
		return tierLarge
	}
}

// countDiffChange returns the number of changed (added/removed) lines and the
// number of files touched in a unified diff. File header lines ("+++ ", "--- ",
// "diff --git ") and hunk markers ("@@") are excluded from the changed-line
// count so the measure reflects real edits, not diff scaffolding.
func countDiffChange(diff string) (changedLines, files int) {
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			files++
		case strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- "):
			// file header, not a content change
		case strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-"):
			changedLines++
		}
	}
	// Fall back to +++ markers when the diff has no "diff --git" lines (e.g.
	// some plain `diff -u` outputs) so single-file diffs still register a file.
	if files == 0 {
		for _, line := range strings.Split(diff, "\n") {
			if strings.HasPrefix(line, "+++ ") {
				files++
			}
		}
	}
	return changedLines, files
}

// prDetailGuidance returns a one-line length budget for the PR body, scaled to
// the size of the change. The diff is the cumulative base → head diff; callers
// inline the result into the PR user message so the model targets a length that
// matches the change instead of a fixed verbosity.
func prDetailGuidance(diff string) string {
	switch diffChangeTier(diff) {
	case tierSmall:
		return "This is a small change: keep the body to a single sentence and use no bullets."
	case tierMedium:
		return "This is a medium change: write a 1-2 sentence summary, then up to ~4 short bullets only if they add detail the summary can't convey."
	default:
		return "This is a large change: write a short summary paragraph, then concise bullets grouped by area or file-set. Stay skimmable and avoid a blow-by-blow of every edit."
	}
}

// ticketDetailGuidance is prDetailGuidance's sibling for tracker tickets: same
// size tiers, wording aimed at a ticket description rather than a PR body.
func ticketDetailGuidance(diff string) string {
	switch diffChangeTier(diff) {
	case tierSmall:
		return "This is a small change: keep the description to a sentence or two; no bullets."
	case tierMedium:
		return "This is a medium change: a short summary, then a few bullets only if they add detail the summary can't convey."
	default:
		return "This is a large change: a short summary paragraph, then concise bullets grouped by area. Stay skimmable; don't enumerate every edit."
	}
}

// PRSystem is the system prompt for pull request title + body generation.
const PRSystem = `You write pull request titles and bodies for GitHub.
Respond with a single JSON object only, no markdown fences, using exactly these keys:
{"title":"...","body":"..."}
The response must be valid JSON: newlines inside body must be \n escapes inside the quoted string, not literal line breaks.
Title: at most 200 characters, capturing the change's primary purpose. Body: a plain summary of what changed and why. Markdown allowed; use bullets only when they add detail the summary can't. No boilerplate headings (no "## Summary", "## Test plan"), do not restate the diff line by line, and keep it skimmable. Match the length to the change: the user message states a length budget for this PR—follow it.
Use the "Files changed" summary to judge scope and lead with the change that dominates. If the commits span clearly unrelated areas, describe the overarching intent rather than one file; never infer the whole PR's purpose from a single incidental hunk.
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
	if stat := FormatDiffFileStat(diff); stat != "" {
		b.WriteString("\n\n")
		b.WriteString(stat)
	}
	b.WriteString("\n\nCumulative unified diff (base → head, covering every commit above):\n")
	b.WriteString(truncateRunes(diff, maxPromptDiffRunes))
	b.WriteString("\n\nLength budget for the body: ")
	b.WriteString(prDetailGuidance(diff))
	b.WriteString("\n\nWrite JSON with title and body for this pull request that reflects the whole chain of commits, not just the tip. Preserve a leading ticket/issue prefix only when it is already present in the hint above; never invent one.")
	return b.String()
}

// TicketSystem is the system prompt for tracker ticket drafts from a diff.
const TicketSystem = `You draft issue-tracker tickets (title + description) that a code change would address or close.
The user supplies a unified diff of a proposed fix. Infer what problem or task the change solves, and write the ticket as if filed before the fix. Stay conservative—only state what the diff reasonably supports; say what is unknown rather than guessing.
Respond with a single JSON object only, no markdown fences, using exactly these keys:
{"title":"...","body":"..."}
The response must be valid JSON: any newlines inside body must be written as \n inside the quoted string, not as literal line breaks. Prefer one line for the whole object if unsure.
Title: at most 300 characters, capturing the core problem or task. Body: a concise plain summary of the problem; markdown allowed but no boilerplate headings, and don't restate the diff. Match the length to the change: the user message states a length budget—follow it. Use the "Files changed" summary to judge scope and focus on the dominant change; if the diff spans unrelated areas, describe the overarching intent rather than one file. If—and only if—the draft title in the user message already starts with a real ticket/issue id from that text, keep that exact prefix in the JSON title; otherwise do not add any key or placeholder.`

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
	if stat := FormatDiffFileStat(diff); stat != "" {
		b.WriteString("\n\n")
		b.WriteString(stat)
	}
	if s := strings.TrimSpace(chainSummary); s != "" {
		b.WriteString("\n\nCommits in this stack (oldest → newest, from trunk up to the selected change):\n")
		b.WriteString(s)
		b.WriteString("\n\nCumulative unified diff (trunk → selected change, covering every commit above):\n")
	} else {
		b.WriteString("\n\nUnified diff (vs parents):\n")
	}
	b.WriteString(truncateRunes(diff, maxPromptDiffRunes))
	b.WriteString("\n\nLength budget for the description: ")
	b.WriteString(ticketDetailGuidance(diff))
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
	if stat := FormatDiffFileStat(diff); stat != "" {
		b.WriteString(stat)
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
