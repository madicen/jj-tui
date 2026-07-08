package jj

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// AnnotationLine is one line of `jj file annotate` output: the source change
// that last touched the line, plus the line's 1-based number and text content.
type AnnotationLine struct {
	ChangeID   string // shortest unambiguous change-id prefix (see annotateTemplate)
	Author     string // author email local-part (e.g. "alice")
	Age        string // human-relative timestamp, e.g. "2 hours ago"
	LineNumber int    // 1-based line number within the file
	Content    string // line text, without the trailing newline
}

// annotateTemplate renders each AnnotationLine as tab-separated fields with the
// raw line content LAST so embedded tabs in source code stay inside Content.
// content already includes the trailing newline, so no extra "\n" is appended.
const annotateTemplate = `commit.change_id().shortest(8) ++ "\t" ++ commit.author().email().local() ++ "\t" ++ commit.author().timestamp().ago() ++ "\t" ++ line_number ++ "\t" ++ content`

// AnnotateFile runs `jj file annotate <path> [-r <revision>]` and parses the
// per-line blame into AnnotationLine records. revision may be empty to annotate
// the working-copy view of the file.
func (s *Service) AnnotateFile(ctx context.Context, revision, path string) ([]AnnotationLine, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}
	args := []string{"file", "annotate", "-T", annotateTemplate}
	if rev := strings.TrimSpace(revision); rev != "" {
		args = append(args, "-r", rev)
	}
	args = append(args, path)
	out, err := s.runJJOutput(ctx, args...)
	if err != nil {
		return nil, err
	}
	return parseAnnotateOutput(out)
}

// parseAnnotateOutput parses the tab-separated annotateTemplate output into
// AnnotationLine records. Content is preserved verbatim (leading whitespace and
// embedded tabs intact); only the line-splitting newline is stripped.
func parseAnnotateOutput(out string) ([]AnnotationLine, error) {
	var lines []AnnotationLine
	for _, raw := range strings.Split(out, "\n") {
		raw = strings.TrimSuffix(raw, "\r")
		if raw == "" {
			continue
		}
		parts := strings.SplitN(raw, "\t", 5)
		if len(parts) < 5 {
			return nil, fmt.Errorf("expected 5 tab fields, got %d in %q", len(parts), raw)
		}
		num, err := strconv.Atoi(strings.TrimSpace(parts[3]))
		if err != nil {
			return nil, fmt.Errorf("invalid line number %q: %w", parts[3], err)
		}
		lines = append(lines, AnnotationLine{
			ChangeID:   strings.TrimSpace(parts[0]),
			Author:     strings.TrimSpace(parts[1]),
			Age:        strings.TrimSpace(parts[2]),
			LineNumber: num,
			Content:    parts[4],
		})
	}
	return lines, nil
}
