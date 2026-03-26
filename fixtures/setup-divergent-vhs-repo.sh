#!/usr/bin/env bash
# Small jj repo for vhs/divergent.tape: one change ID with two visible revisions (divergent).
# Pattern: create commit, record id, squash-amend via child+--into @-, then `jj new <old id>`
# so the pre-amend revision is visible again. Bookmark the other head so it appears in the
# default graph revset (sibling is not in ancestors(@)).
# Tape: j → d → (optional j in modal) → Enter.
# Run: make divergent-gif
# Requires: jj, git
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$SCRIPT_DIR/divergent-vhs-repo"

rm -rf "$REPO"
mkdir -p "$REPO"
cd "$REPO"

git init --initial-branch=main >/dev/null
jj git init --colocate >/dev/null

jj config set --repo user.name "Demo User"
jj config set --repo user.email "demo@example.com"

mkdir -p src docs
echo "# Divergent resolution demo" > README.md
jj describe -m "Initial import"
jj bookmark create main
jj new

echo 'package widget

// APIVersion is bumped when the public shape changes.
const APIVersion = 1
' > src/widget.go
jj describe -m "Add widget API"
OLD=$(jj log -r @ -T commit_id --no-graph | tr -d '\n')

jj new
echo "rollout=phased" > docs/plan.md
jj squash --into @- -m "Add widget API + rollout plan"

jj new "$OLD" --ignore-immutable

CID=$(jj log -r @- -T change_id --no-graph | tr -d '\n')
PARENT_ID=$(jj log -r @- -T commit_id --no-graph | tr -d '\n')
jj bookmark create amended-tip -r "change_id($CID) & ~commit_id($PARENT_ID)"

echo "Divergent VHS repo ready: $REPO"
