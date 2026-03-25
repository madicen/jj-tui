#!/usr/bin/env bash
# Small jj repo for vhs/evolog-split.tape: feature bookmark whose change has multiple jj evolog
# revisions after squashing a file edit from a child WC into the feature commit.
# Tape: z → j (pick an older row with a non-empty diff vs tip) → Enter.
# Run: make evolog-split-gif
# Requires: jj, git
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$SCRIPT_DIR/evolog-split-vhs-repo"

rm -rf "$REPO"
mkdir -p "$REPO"
cd "$REPO"

git init --initial-branch=main >/dev/null
jj git init --colocate >/dev/null

jj config set --repo user.name "Demo User"
jj config set --repo user.email "demo@example.com"

echo a > f
jj describe -m "Initial import"
jj bookmark create main
jj new

echo b > f
jj describe -m "Add feature flag"
jj bookmark create demo/feature

# Empty WC on top of feature so `jj squash` folds the edit into the feature commit (evolution), not into main.
jj new
echo c >> f
jj squash -m "Add feature flag"

jj edit demo/feature >/dev/null

echo "Evolog split VHS repo ready: $REPO"
