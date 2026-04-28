package jj

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const evologHunkSplitDiffMaxBytes = 4 << 20

// SplitRevisionByHunkPrefix runs non-interactive `jj split` using a one-shot ui.diff-editor that
// rewrites the output tree to parent + the first k hunks per path (see prefixByPath).
// revision is typically "@"; diff is always read from @- → @.
func (s *Service) SplitRevisionByHunkPrefix(ctx context.Context, revision, message string, prefixByPath map[string]int) error {
	if len(prefixByPath) == 0 {
		return nil
	}
	diff, err := s.GitFormatDiffFromTo(ctx, "@-", "@", evologHunkSplitDiffMaxBytes)
	if err != nil {
		return fmt.Errorf("git diff @- → @: %w", err)
	}
	if err := ValidateHunkPrefixPlan(diff, prefixByPath); err != nil {
		return err
	}
	spec := EvologHunkSplitSpec{GitDiff: diff, PrefixByPath: prefixByPath}
	specFile, err := os.CreateTemp("", "jj-tui-hunk-spec-*.json")
	if err != nil {
		return err
	}
	specPath := specFile.Name()
	defer func() { _ = os.Remove(specPath) }()
	enc, err := json.Marshal(&spec)
	if err != nil {
		_ = specFile.Close()
		return err
	}
	if _, err := specFile.Write(enc); err != nil {
		_ = specFile.Close()
		return err
	}
	if err := specFile.Close(); err != nil {
		return err
	}
	cfgFile, err := os.CreateTemp("", "jj-tui-hunk-cfg-*.toml")
	if err != nil {
		return err
	}
	cfgPath := cfgFile.Name()
	defer func() { _ = os.Remove(cfgPath) }()
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("executable path: %w", err)
	}
	exeAbs, err := filepath.Abs(exe)
	if err != nil {
		return err
	}
	cfgBody := buildEvologHunkSplitMergeToolToml(exeAbs)
	if _, err := cfgFile.WriteString(cfgBody); err != nil {
		_ = cfgFile.Close()
		return err
	}
	if err := cfgFile.Close(); err != nil {
		return err
	}
	rev := strings.TrimSpace(revision)
	if rev == "" {
		rev = "@"
	}
	msg := strings.TrimSpace(message)
	if msg == "" {
		msg = EvologSplitDefaultMessage
	}
	args := []string{
		"--config-file", cfgPath,
		"split", "-r", rev, "-m", msg,
		"--tool", "jj-tui-hunk-split",
	}
	// When @ already has one direct child, default jj split rebases that child onto the new
	// remainder commit; successive peels pile into the same descendant. --insert-before inserts
	// the peeled commit between @ and that child (omit if 0 or 2+ children).
	if kids, err := s.directRevisionChildrenCommitIDs(ctx, rev); err == nil && len(kids) == 1 {
		args = append(args, "--insert-before", kids[0])
	}
	env := EvologHunkSplitSpecEnv + "=" + specPath
	return s.runJJWithExtraEnv(ctx, []string{env}, args)
}

// directRevisionChildrenCommitIDs lists direct children (commits whose parent is rev) as full ids.
func (s *Service) directRevisionChildrenCommitIDs(ctx context.Context, rev string) ([]string, error) {
	rev = strings.TrimSpace(rev)
	if rev == "" {
		rev = "@"
	}
	rs := fmt.Sprintf("children(%s)", rev)
	out, err := s.runJJOutputNoHistory(ctx, "log", "-r", rs, "--no-graph", "-T", "commit_id ++ \"\\n\"", "--limit", "16")
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			ids = append(ids, line)
		}
	}
	return ids, nil
}

func (s *Service) runJJWithExtraEnv(ctx context.Context, extraEnv []string, args []string) error {
	cmdStr := "jj " + strings.Join(args, " ")
	startTime := time.Now()
	cmd := exec.CommandContext(ctx, "jj", args...)
	cmd.Dir = s.RepoPath
	cmd.Env = append(append([]string{}, os.Environ()...), extraEnv...)
	out, err := cmd.CombinedOutput()
	duration := time.Since(startTime)
	entry := CommandHistoryEntry{
		Command:   cmdStr,
		Timestamp: startTime,
		Duration:  duration,
		Success:   err == nil,
	}
	if err != nil {
		errMsg := extractErrorMessage(string(out))
		if errMsg != "" {
			entry.Error = errMsg
			s.addToHistory(entry)
			return fmt.Errorf("%s", errMsg)
		}
		entry.Error = err.Error()
		s.addToHistory(entry)
		return fmt.Errorf("command failed: %w", err)
	}
	s.addToHistory(entry)
	return nil
}

func buildEvologHunkSplitMergeToolToml(exe string) string {
	var b strings.Builder
	b.WriteString("[merge-tools.jj-tui-hunk-split]\n")
	b.WriteString("program = ")
	b.WriteString(strconv.Quote(exe))
	b.WriteString("\n")
	b.WriteString("edit-args = [\"diff-editor-evolog-hunk-split\", \"$left\", \"$right\", \"$output\"]\n\n")
	b.WriteString("[ui]\n")
	b.WriteString("diff-editor = \"jj-tui-hunk-split\"\n")
	return b.String()
}
