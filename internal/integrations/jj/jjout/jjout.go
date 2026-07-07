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
// prefers the first line beginning with "Error:", otherwise the first
// non-empty line that isn't a warning or hint.
func ExtractErrorMessage(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Error:") {
			return strings.TrimPrefix(line, "Error: ")
		}
	}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "Warning:") && !strings.HasPrefix(line, "Hint:") {
			return line
		}
	}
	return ""
}
