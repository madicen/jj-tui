#!/usr/bin/env bash
# Repo for vhs/bookmark-conflict.tape: colocated jj + git; feature bookmark pushed to origin, then amended locally.
# After this, `jj bookmark list --all-remotes` shows the usual colocated pattern:
#   vhs/conflict-feature: <local tip>
#     @git:   same as local (underlying git branch)
#     @origin (ahead by … behind by …): remembered remote tip ≠ local  ← jj-tui uses this for Resolve (c) / graph (C).
# That is the same class of situation as “@git vs @origin” confusion when @git tracks your working tree and @origin lags.
# Run: make bookmark-conflict-gif
# Requires: jj, git
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$SCRIPT_DIR/bookmark-conflict-vhs-repo"
ORIG="$SCRIPT_DIR/bookmark-conflict-fake-origin.git"

rm -rf "$REPO" "$ORIG"

git init --bare "$ORIG" >/dev/null
mkdir -p "$REPO"
cd "$REPO"

git init --initial-branch=main >/dev/null
jj git init --colocate >/dev/null
git remote add origin "$ORIG"

jj config set --repo user.name "Demo User"
jj config set --repo user.email "demo@example.com"

echo "# Bookmark conflict demo" > README.md
jj describe -m "Initial commit"
jj bookmark create main
jj new

mkdir -p src
echo 'package demo' > src/app.go
jj describe -m "Add feature"
jj bookmark create vhs/conflict-feature

jj git push --bookmark main --remote origin --allow-new
jj git push --bookmark vhs/conflict-feature --remote origin --allow-new
jj git fetch >/dev/null

jj describe -m "Add feature (amended after push)"
jj git fetch >/dev/null

echo "Bookmark conflict VHS repo ready: $REPO"
