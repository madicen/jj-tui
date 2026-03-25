#!/usr/bin/env bash
# Used by vhs/after-origin.tape: tweak src/app.go and add src/review_followup.go (same jj revision) then relaunch jj-tui --demo.
# Run from jj-tui repo root (same as other fixture scripts).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT/fixtures/after-origin-vhs-repo"
printf '%s\n' '// Post-PR edit (same jj revision, no jj commit)' >> src/app.go
cat > src/review_followup.go << 'EOF'
package main

// ReviewFollowup holds feedback items from PR review (added without a new jj commit).
type ReviewFollowup struct {
	Notes string
}
EOF
exec "$ROOT/jj-tui" --demo
