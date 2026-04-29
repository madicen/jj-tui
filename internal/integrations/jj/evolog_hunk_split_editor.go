package jj

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const EvologHunkSplitSpecEnv = "JJ_TUI_EVOLOG_HUNK_SPEC"

// EvologHunkSplitSpec is written to a temp file; the diff-editor subcommand reads it from EvologHunkSplitSpecEnv.
type EvologHunkSplitSpec struct {
	GitDiff      string         `json:"git_diff"`
	PrefixByPath map[string]int `json:"prefix_by_path"`
}

// RunEvologHunkSplitDiffEditor is the entry point for `jj-tui diff-editor-evolog-hunk-split $left $right $output`.
// It rewrites files under outputDir to match parent + the first k hunks per path (see spec).
func RunEvologHunkSplitDiffEditor(leftDir, rightDir, outputDir string) error {
	specPath := strings.TrimSpace(os.Getenv(EvologHunkSplitSpecEnv))
	if specPath == "" {
		return fmt.Errorf("missing %s", EvologHunkSplitSpecEnv)
	}
	raw, err := os.ReadFile(specPath)
	if err != nil {
		return fmt.Errorf("read hunk split spec: %w", err)
	}
	var spec EvologHunkSplitSpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return fmt.Errorf("parse hunk split spec: %w", err)
	}
	if strings.TrimSpace(spec.GitDiff) == "" {
		return fmt.Errorf("empty git_diff in hunk split spec")
	}
	hunksPerPath, binaryPaths, err := ParseGitUnifiedHunksPerPath(spec.GitDiff)
	if err != nil {
		return fmt.Errorf("parse hunks: %w", err)
	}
	prefix := spec.PrefixByPath
	if prefix == nil {
		prefix = map[string]int{}
	}
	return writeHunkSplitOutputDirs(leftDir, rightDir, outputDir, hunksPerPath, binaryPaths, prefix)
}

// normalizeTreeTextForHunkSplit aligns working-tree bytes with git-unified parsing (CRLF, UTF-8 BOM).
func normalizeTreeTextForHunkSplit(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.TrimPrefix(s, "\ufeff")
	return s
}

func writeHunkSplitOutputDirs(leftDir, rightDir, outputDir string, hunksPerPath map[string][]UnifiedHunk, binaryPaths map[string]struct{}, prefix map[string]int) error {
	seen := make(map[string]struct{})
	process := func(rel string) error {
		rel = filepath.ToSlash(rel)
		if rel == "." || rel == "" {
			return nil
		}
		if _, ok := seen[rel]; ok {
			return nil
		}
		if err := processHunkSplitRel(rel, leftDir, rightDir, outputDir, hunksPerPath, binaryPaths, prefix); err != nil {
			return err
		}
		seen[rel] = struct{}{}
		return nil
	}
	if err := filepath.WalkDir(rightDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(rightDir, path)
		if err != nil {
			return err
		}
		return process(rel)
	}); err != nil {
		return err
	}
	// Paths that appear in the git diff but not under rightDir (e.g. deleted on @ so absent from the
	// right tree) are never visited by the walk; jj then fails the merge with a generic "Failed to edit diff".
	extras := make(map[string]struct{})
	for k := range hunksPerPath {
		extras[filepath.ToSlash(strings.TrimSpace(k))] = struct{}{}
	}
	for k := range binaryPaths {
		extras[filepath.ToSlash(strings.TrimSpace(k))] = struct{}{}
	}
	for k := range prefix {
		k = filepath.ToSlash(strings.TrimSpace(k))
		if k != "" {
			extras[k] = struct{}{}
		}
	}
	keys := make([]string, 0, len(extras))
	for k := range extras {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if _, ok := seen[k]; ok {
			continue
		}
		if err := processHunkSplitRel(k, leftDir, rightDir, outputDir, hunksPerPath, binaryPaths, prefix); err != nil {
			return err
		}
		seen[k] = struct{}{}
	}
	return nil
}

func processHunkSplitRel(rel string, leftDir, rightDir, outputDir string, hunksPerPath map[string][]UnifiedHunk, binaryPaths map[string]struct{}, prefix map[string]int) error {
	rel = filepath.ToSlash(rel)
	if rel == "." || rel == "" {
		return nil
	}
	base := filepath.Base(rel)
	if base == "JJ-INSTRUCTIONS" || strings.HasPrefix(base, "JJ-INSTRUCTIONS") {
		return copyFileToOutput(leftDir, rightDir, outputDir, rel)
	}
	leftPath := filepath.Join(leftDir, filepath.FromSlash(rel))
	rightPath := filepath.Join(rightDir, filepath.FromSlash(rel))
	outPath := filepath.Join(outputDir, filepath.FromSlash(rel))

	leftBytes, lerr := os.ReadFile(leftPath)
	if lerr != nil && !os.IsNotExist(lerr) {
		return fmt.Errorf("read left %s: %w", rel, lerr)
	}
	var leftRaw []byte
	if lerr == nil {
		leftRaw = leftBytes
	}
	leftStr := ""
	if lerr == nil {
		leftStr = normalizeTreeTextForHunkSplit(string(leftRaw))
	}

	rightBytes, rerr := os.ReadFile(rightPath)
	if rerr != nil && !os.IsNotExist(rerr) {
		return fmt.Errorf("read right %s: %w", rel, rerr)
	}
	var rightRaw []byte
	if rerr == nil {
		rightRaw = rightBytes
	}
	rightStr := ""
	if rerr == nil {
		rightStr = normalizeTreeTextForHunkSplit(string(rightRaw))
	}

	if _, isBin := binaryPaths[rel]; isBin {
		k := 0
		if kv, ok := prefix[rel]; ok {
			k = kv
		}
		if k != 0 {
			return fmt.Errorf("%s: binary file cannot use non-zero hunk prefix (use jj split with paths or files_first_commit)", rel)
		}
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		if os.IsNotExist(lerr) {
			if err := os.Remove(outPath); err != nil && !os.IsNotExist(err) {
				return err
			}
			return nil
		}
		return os.WriteFile(outPath, leftRaw, 0o644)
	}
	hunks := hunksPerPath[rel]
	k, ok := prefix[rel]
	if !ok {
		k = 0
	}
	if len(hunks) == 0 {
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		if os.IsNotExist(rerr) {
			if err := os.Remove(outPath); err != nil && !os.IsNotExist(err) {
				return err
			}
			return nil
		}
		return os.WriteFile(outPath, rightRaw, 0o644)
	}
	if k < 0 || k > len(hunks) {
		return fmt.Errorf("%s: hunk prefix k=%d exceeds %d @@ hunks (k is how many leading hunks to peel, not a line number)", rel, k, len(hunks))
	}
	if len(hunks) == 1 && k > 0 {
		return fmt.Errorf("%s: only 1 @@ hunk — cannot split this path by hunk prefix (use file-level jj split for the whole file)", rel)
	}
	if k == len(hunks) {
		return fmt.Errorf("%s: hunk prefix would move every hunk off @ (need strict subset)", rel)
	}
	if err := VerifyUnifiedHunksReconstructRight(leftStr, rightStr, hunks); err != nil {
		return fmt.Errorf("%s: %w", rel, err)
	}
	outStr, err := ApplyUnifiedHunkPrefix(leftStr, hunks, k)
	if err != nil {
		return fmt.Errorf("%s: %w", rel, err)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(outPath, []byte(outStr), 0o644)
}

func copyFileToOutput(leftDir, rightDir, outputDir, rel string) error {
	// Prefer right (matches jj initial output copy); fall back to left.
	for _, dir := range []string{rightDir, leftDir} {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		b, err := os.ReadFile(p)
		if err == nil {
			out := filepath.Join(outputDir, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
				return err
			}
			return os.WriteFile(out, b, 0o644)
		}
	}
	return nil
}
