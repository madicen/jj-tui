// Package jjout holds small, pure helpers for parsing `jj` CLI output. These
// were previously duplicated inline across the jj Service's many parsers.
package jjout

import "strings"

// SplitLines splits raw command output on newlines, trims surrounding
// whitespace from each line, and drops empty lines. This matches the extremely
// common "split, TrimSpace, skip empties" loop used throughout jj output
// parsing.
func SplitLines(out string) []string {
	raw := strings.Split(out, "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// ParseCommitInfo extracts the change id and short commit id from a whitespace
// separated "change_id commit_id description" line. When only one field is
// present both returned values equal that field.
func ParseCommitInfo(info string) (changeID, shortID string) {
	parts := strings.Fields(info)
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	if len(parts) == 1 {
		return parts[0], parts[0]
	}
	return "", ""
}

// ExtractErrorMessage pulls the main error message out of `jj` output. It
// prefers the first line beginning with "Error:" or "Internal error:", then
// appends following Caused by / numbered cause / gpg: lines until a blank,
// Hint:, or Warning: line. Otherwise it returns the first non-empty line that
// isn't a warning or hint.
func ExtractErrorMessage(output string) string {
	lines := strings.Split(output, "\n")
	start := -1
	var head string
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(line, "Error:"):
			head = strings.TrimPrefix(line, "Error: ")
			start = i
		case strings.HasPrefix(line, "Internal error:"):
			head = line
			start = i
		default:
			continue
		}
		break
	}
	if start < 0 {
		for _, raw := range lines {
			line := strings.TrimSpace(raw)
			if line != "" && !strings.HasPrefix(line, "Warning:") && !strings.HasPrefix(line, "Hint:") {
				return line
			}
		}
		return ""
	}

	var b strings.Builder
	b.WriteString(head)
	for _, raw := range lines[start+1:] {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "Hint:") || strings.HasPrefix(line, "Warning:") {
			break
		}
		if !isJJCauseLine(line) {
			break
		}
		b.WriteByte('\n')
		b.WriteString(line)
	}
	return b.String()
}

// isJJCauseLine reports whether line continues a jj "Caused by" chain.
func isJJCauseLine(line string) bool {
	if strings.HasPrefix(line, "Caused by:") || line == "Caused by:" {
		return true
	}
	if strings.HasPrefix(line, "gpg:") {
		return true
	}
	// "1: …" / "12: …"
	digits := 0
	for _, c := range line {
		if c >= '0' && c <= '9' {
			digits++
			continue
		}
		return c == ':' && digits > 0
	}
	return false
}

// IsSigningPinentryFailure reports whether err text looks like a GPG signing /
// pinentry failure (jj "Signing error", "No pinentry", etc.).
func IsSigningPinentryFailure(text string) bool {
	s := strings.ToLower(text)
	return strings.Contains(s, "signing error") ||
		strings.Contains(s, "no pinentry") ||
		strings.Contains(s, "gpg: signing failed")
}
