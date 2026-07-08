package jj

import (
	"fmt"
	"strings"
)

// CompileGraphFilterRevset turns graph search input into a jj revset.
// Free text compiles to description(substring-i:"…") | author(substring-i:"…").
// A leading ":" means the remainder is a raw revset expression.
func CompileGraphFilterRevset(input string) (revset, display string, err error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", "", fmt.Errorf("empty search")
	}
	if strings.HasPrefix(input, ":") {
		raw := strings.TrimSpace(input[1:])
		if raw == "" {
			return "", "", fmt.Errorf("empty revset")
		}
		return raw, raw, nil
	}
	pat := revsetDoubleQuotedString(input)
	revset = fmt.Sprintf(`description(substring-i:%s) | author(substring-i:%s)`, pat, pat)
	return revset, input, nil
}

func revsetDoubleQuotedString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
