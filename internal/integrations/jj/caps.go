package jj

import "context"

// jj capability / version drift helpers live here. Centralizing them keeps
// version-specific command names out of the domain methods.

const (
	// verbBackout is the older jj subcommand that applies the reverse of a revision.
	verbBackout = "backout"
	// verbRevert is the newer name for the same operation (jj renamed
	// `backout` to `revert`).
	verbRevert = "revert"
)

// backoutOrRevertArgs builds the argv for applying the reverse of `rev`, using
// the given verb.
//
// `jj revert` requires an explicit destination (--onto / --insert-after /
// --insert-before), so we apply the reverse on top of the working copy (@).
// The historical `jj backout` defaulted onto @ implicitly, so it takes no
// destination flag. Keeping both in one place means callers never encode the
// version difference themselves.
func backoutOrRevertArgs(verb, rev string) []string {
	if verb == verbBackout {
		return []string{verbBackout, "-r", rev}
	}
	return []string{verbRevert, "-r", rev, "--onto", "@"}
}

// backoutVerb reports the reverse-revision subcommand this jj build supports,
// probing once and caching the result on the Service.
//
// It runs `jj backout --help`: exit 0 means this jj still has `backout`; a
// non-zero exit (unrecognized subcommand) means it was renamed to `revert`.
// The probe is history-free so it doesn't pollute the command log.
func (s *Service) backoutVerb(ctx context.Context) string {
	s.backoutVerbOnce.Do(func() {
		if _, err := s.runJJOutputNoHistory(ctx, "backout", "--help"); err == nil {
			s.backoutVerbVal = verbBackout
		} else {
			s.backoutVerbVal = verbRevert
		}
	})
	return s.backoutVerbVal
}
