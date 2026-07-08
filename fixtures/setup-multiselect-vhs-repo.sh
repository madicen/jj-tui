#!/usr/bin/env bash
# Repo for vhs/multiselect.tape: a short chain of mutable commits for Space batch selection.
# Run: make multiselect-gif
# Requires: jj, git
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$SCRIPT_DIR/multiselect-vhs-repo"

if [ -d "$REPO" ]; then
	chmod -R u+w "$REPO" 2>/dev/null || true
fi
rm -rf "$REPO"
mkdir -p "$REPO"
cd "$REPO"

git init --initial-branch=main >/dev/null
jj git init --colocate >/dev/null

jj config set --repo user.name "Demo User"
jj config set --repo user.email "demo@example.com"

echo "# Multi-select demo" > README.md
jj describe -m "Initial import"
jj bookmark create main
jj new

echo "one" > a.txt
jj describe -m "vhs batch alpha"

jj new
echo "two" > b.txt
jj describe -m "vhs batch beta"

jj new
echo "three" > c.txt
jj describe -m "vhs batch gamma"

echo "Multi-select VHS repo ready: $REPO"
