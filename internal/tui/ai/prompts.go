package ai

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

const prSystem = `You write pull request titles and bodies for GitHub.
Respond with a single JSON object only, no markdown fences, using exactly these keys:
{"title":"...","body":"..."}
Title: at most 200 characters. Body: markdown allowed, be clear and concise.
If the suggested title hint starts with an issue tracker key (e.g. Jira ABC-123, or [ABC-123]), keep that exact key and the same spacing or punctuation after it at the beginning of the JSON title; only improve the human-readable part after the key.`

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
	b.WriteString("\n\nWrite JSON with title and body for this pull request. Preserve any leading issue key from the hint in the title exactly as required for branch/CI conventions.")
	return b.String()
}

const ticketSystem = `You draft issue-tracker tickets (title + description) that a code change would address or close.
The user supplies a unified diff of a proposed fix. Infer what problem or task the change solves, and write the ticket as if filed before the fix: clear title, markdown body with symptoms, expected vs actual where inferable, and scope. Stay conservative—only state what the diff reasonably supports; say what is unknown rather than guessing.
Respond with a single JSON object only, no markdown fences, using exactly these keys:
{"title":"...","body":"..."}
Title: at most 300 characters. Body: markdown allowed. If the draft title hint starts with an issue tracker key (e.g. Jira ABC-123), keep that exact key and spacing at the start of the JSON title; only improve the rest.`

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

// ParsePRTitleBody extracts title and body from model output (JSON preferred).
func ParsePRTitleBody(raw string) (title, body string) {
	raw = strings.TrimSpace(raw)
	if i := strings.Index(raw, "{"); i >= 0 {
		if j := strings.LastIndex(raw, "}"); j > i {
			var v struct {
				Title string `json:"title"`
				Body  string `json:"body"`
			}
			if err := json.Unmarshal([]byte(raw[i:j+1]), &v); err == nil {
				return strings.TrimSpace(v.Title), strings.TrimSpace(v.Body)
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

const bookmarkSystem = `You suggest a short git branch / bookmark name: lowercase, use hyphens between words, only letters, digits, hyphens, underscores.
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
