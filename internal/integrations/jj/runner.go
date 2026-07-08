package jj

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// RunOpts configures a single jj invocation. It collapses what used to be six
// near-identical run* helpers (global-args and history-logging variants, plus a
// with-extra-env variant) into orthogonal options on a single Runner.
type RunOpts struct {
	// Global holds jj global flags placed before the subcommand
	// (e.g. --ignore-working-copy). Merged via jjMergeGlobalArgs.
	Global []string
	// Env, when non-empty, is appended to the process environment (os.Environ)
	// for this invocation. Empty means inherit the current environment.
	Env []string
	// NoHistory, when true, skips command-history logging for this invocation.
	NoHistory bool
}

// Runner executes jj commands. Production uses execRunner; tests can inject a
// fake (see internal/mock.FakeRunner) to feed canned output without a real repo.
type Runner interface {
	// Run executes jj and returns a cleaned error if it fails. It uses
	// CombinedOutput (stdout+stderr merged) for error extraction.
	Run(ctx context.Context, opts RunOpts, args ...string) error
	// RunOutput executes jj and returns its stdout only; stderr is captured
	// separately so jj hints/warnings do not contaminate parsed output.
	RunOutput(ctx context.Context, opts RunOpts, args ...string) (string, error)
	// RunCombined executes jj and returns merged stdout+stderr on both success
	// and failure. PLAN(P4.2): needed by absorb, whose human-readable summary
	// ("Absorbed changes into N revisions: …") is written to stderr; RunOutput
	// would drop it. Kept as a distinct method so RunOutput's stdout-only
	// contract (relied on by parsers) is unaffected.
	RunCombined(ctx context.Context, opts RunOpts, args ...string) (string, error)
}

// execRunner is the production Runner: it shells out to the jj binary and
// records command history through the record callback.
type execRunner struct {
	repoPath string
	record   func(CommandHistoryEntry)
}

func newExecRunner(repoPath string, record func(CommandHistoryEntry)) *execRunner {
	return &execRunner{repoPath: repoPath, record: record}
}

func (r *execRunner) log(opts RunOpts, entry CommandHistoryEntry) {
	if opts.NoHistory || r.record == nil {
		return
	}
	r.record(entry)
}

// Run mirrors the legacy runJJ / runJJWithGlobal / runJJWithExtraEnv behavior.
func (r *execRunner) Run(ctx context.Context, opts RunOpts, args ...string) error {
	merged := jjMergeGlobalArgs(opts.Global, args)
	cmdStr := "jj " + strings.Join(merged, " ")
	startTime := time.Now()

	cmd := exec.CommandContext(ctx, "jj", merged...)
	cmd.Dir = r.repoPath
	if len(opts.Env) > 0 {
		cmd.Env = append(append([]string{}, os.Environ()...), opts.Env...)
	}
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
			r.log(opts, entry)
			return fmt.Errorf("%s", errMsg)
		}
		entry.Error = err.Error()
		r.log(opts, entry)
		return fmt.Errorf("command failed: %w", err)
	}

	r.log(opts, entry)
	return nil
}

// RunOutput mirrors the legacy runJJOutput / *WithGlobal / *NoHistory* behavior.
// The NoHistory branch preserves the historically-different error format (no
// extractErrorMessage, simple "%w: %s") that its callers relied on.
func (r *execRunner) RunOutput(ctx context.Context, opts RunOpts, args ...string) (string, error) {
	merged := jjMergeGlobalArgs(opts.Global, args)
	cmdStr := "jj " + strings.Join(merged, " ")
	startTime := time.Now()

	cmd := exec.CommandContext(ctx, "jj", merged...)
	cmd.Dir = r.repoPath
	if len(opts.Env) > 0 {
		cmd.Env = append(append([]string{}, os.Environ()...), opts.Env...)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(startTime)

	if err != nil {
		errOutput := stderr.String()
		if errOutput == "" {
			errOutput = stdout.String()
		}
		// PLAN(P2.1): the historical no-history variant used a distinct, terser
		// error format and skipped extractErrorMessage; preserved verbatim so
		// error text shown to users does not change.
		if opts.NoHistory {
			return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(errOutput))
		}
		entry := CommandHistoryEntry{
			Command:   cmdStr,
			Timestamp: startTime,
			Duration:  duration,
			Success:   false,
		}
		entry.Error = extractErrorMessage(errOutput)
		if entry.Error == "" {
			entry.Error = err.Error()
		}
		r.log(opts, entry)
		return "", fmt.Errorf("jj command '%s' failed: %w\nOutput: %s",
			cmdStr, err, errOutput)
	}

	r.log(opts, CommandHistoryEntry{
		Command:   cmdStr,
		Timestamp: startTime,
		Duration:  duration,
		Success:   true,
	})
	return stdout.String(), nil
}

// RunCombined executes jj and returns merged stdout+stderr regardless of exit
// status, so callers can display or parse jj's status output (which mostly goes
// to stderr). History logging matches Run's behavior.
func (r *execRunner) RunCombined(ctx context.Context, opts RunOpts, args ...string) (string, error) {
	merged := jjMergeGlobalArgs(opts.Global, args)
	cmdStr := "jj " + strings.Join(merged, " ")
	startTime := time.Now()

	cmd := exec.CommandContext(ctx, "jj", merged...)
	cmd.Dir = r.repoPath
	if len(opts.Env) > 0 {
		cmd.Env = append(append([]string{}, os.Environ()...), opts.Env...)
	}
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
		if errMsg == "" {
			errMsg = err.Error()
		}
		entry.Error = errMsg
		r.log(opts, entry)
		return string(out), fmt.Errorf("%s", errMsg)
	}
	r.log(opts, entry)
	return string(out), nil
}

// jjMergeGlobalArgs prepends global jj flags (placed before the subcommand)
// to args, returning a fresh slice.
func jjMergeGlobalArgs(global, args []string) []string {
	if len(global) == 0 {
		out := make([]string, len(args))
		copy(out, args)
		return out
	}
	out := make([]string, 0, len(global)+len(args))
	out = append(out, global...)
	out = append(out, args...)
	return out
}
