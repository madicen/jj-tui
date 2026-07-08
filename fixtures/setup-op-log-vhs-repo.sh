#!/usr/bin/env bash
# Repo for vhs/op-log.tape: three mutating jj operations so the operation-log browser
# has entries to list and restore.
# Run: make op-log-gif
# Requires: jj, git
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$SCRIPT_DIR/op-log-vhs-repo"

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

echo "# Operation log demo" > README.md
jj describe -m "vhs op-one seed"
jj bookmark create main

echo "first change" > a.txt
jj describe -m "vhs op-one describe"

jj new -m "vhs op-two new"

echo "second change" > b.txt
jj describe -m "vhs op-three describe"

echo "Operation log VHS repo ready: $REPO"
