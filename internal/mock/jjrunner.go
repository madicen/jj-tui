package mock

import (
	"context"
	"strings"

	"github.com/madicen/jj-tui/internal/integrations/jj"
)

// FakeRunnerCall records a single jj invocation routed through a FakeRunner.
type FakeRunnerCall struct {
	Opts   jj.RunOpts
	Args   []string
	Output bool // true if it came through RunOutput, false for Run
}

// FakeRunner is a jj.Runner that returns canned output instead of shelling out
// to a real jj binary. It lets jj service methods be unit-tested for parsing and
// control-flow without a fixture repo.
//
// Provide RunOutputFn / RunFn to control responses. Both may inspect the args to
// return different output per subcommand. Every call is appended to Calls.
type FakeRunner struct {
	// RunOutputFn handles RunOutput calls. If nil, RunOutput returns "" and nil.
	RunOutputFn func(ctx context.Context, opts jj.RunOpts, args ...string) (string, error)
	// RunFn handles Run calls. If nil, Run returns nil.
	RunFn func(ctx context.Context, opts jj.RunOpts, args ...string) error

	Calls []FakeRunnerCall
}

var _ jj.Runner = (*FakeRunner)(nil)

// Run implements jj.Runner.
func (f *FakeRunner) Run(ctx context.Context, opts jj.RunOpts, args ...string) error {
	f.Calls = append(f.Calls, FakeRunnerCall{Opts: opts, Args: append([]string(nil), args...)})
	if f.RunFn != nil {
		return f.RunFn(ctx, opts, args...)
	}
	return nil
}

// RunOutput implements jj.Runner.
func (f *FakeRunner) RunOutput(ctx context.Context, opts jj.RunOpts, args ...string) (string, error) {
	f.Calls = append(f.Calls, FakeRunnerCall{Opts: opts, Args: append([]string(nil), args...), Output: true})
	if f.RunOutputFn != nil {
		return f.RunOutputFn(ctx, opts, args...)
	}
	return "", nil
}

// LastArgs returns the args of the most recent call, or nil if none.
func (f *FakeRunner) LastArgs() []string {
	if len(f.Calls) == 0 {
		return nil
	}
	return f.Calls[len(f.Calls)-1].Args
}

// ArgsContain reports whether any recorded call's args joined by spaces contains sub.
func (f *FakeRunner) ArgsContain(sub string) bool {
	for _, c := range f.Calls {
		if strings.Contains(strings.Join(c.Args, " "), sub) {
			return true
		}
	}
	return false
}
