package aiprompts

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var leadingIssueKeyPrefix = regexp.MustCompile(
	`(?i)^[ \t]*(?:` +
		`\[[A-Z][A-Z0-9]{1,15}-\d+\](?:\s*[-:–|]\s*|\s+)|` +
		`[A-Z][A-Z0-9]{1,15}-\d+(?:\s*-\s*|\s*[-:–|]\s*|\s+)` +
		`)`,
)

var issueKeyOnlyLine = regexp.MustCompile(
	`(?i)^[ \t]*(?:\[[A-Z][A-Z0-9]{1,15}-\d+\]|[A-Z][A-Z0-9]{1,15}-\d+)\s*$`,
)

var bareIssueKeyFromText = regexp.MustCompile(`(?i)\[([A-Z][A-Z0-9]{1,15}-\d+)\]|([A-Z][A-Z0-9]{1,15}-\d+)`)

func extractLeadingIssuePrefix(hint string) string {
	h := strings.TrimSpace(hint)
	if h == "" {
		return ""
	}
	if loc := leadingIssueKeyPrefix.FindStringIndex(h); loc != nil {
		return h[:loc[1]]
	}
	if issueKeyOnlyLine.MatchString(h) {
		return strings.TrimSpace(h) + " - "
	}
	return ""
}

func bareIssueKeyUpper(s string) string {
	subs := bareIssueKeyFromText.FindStringSubmatch(s)
	if subs == nil {
		return ""
	}
	if subs[1] != "" {
		return strings.ToUpper(subs[1])
	}
	return strings.ToUpper(subs[2])
}

func titleStartsWithIssueKey(title, bareKeyUpper string) bool {
	if bareKeyUpper == "" {
		return false
	}
	u := strings.ToUpper(strings.TrimSpace(title))
	k := bareKeyUpper
	if strings.HasPrefix(u, "["+k+"]") {
		return true
	}
	if !strings.HasPrefix(u, k) {
		return false
	}
	if len(u) == len(k) {
		return true
	}
	rest := u[len(k):]
	r, _ := utf8.DecodeRuneInString(rest)
	if r == utf8.RuneError {
		return false
	}
	if unicode.IsSpace(r) {
		return true
	}
	return strings.ContainsRune("-:|–—", r)
}

// MergeGeneratedPRTitle prepends the leading issue key (and delimiter) from hintTitle when the
// model dropped it, so CI that requires e.g. Jira keys in PR titles still passes.
func MergeGeneratedPRTitle(hintTitle, generatedTitle string) string {
	gen := strings.TrimSpace(generatedTitle)
	if gen == "" {
		return gen
	}
	prefix := extractLeadingIssuePrefix(hintTitle)
	if prefix == "" {
		return gen
	}
	key := bareIssueKeyUpper(prefix)
	if titleStartsWithIssueKey(gen, key) {
		return gen
	}
	return prefix + gen
}
