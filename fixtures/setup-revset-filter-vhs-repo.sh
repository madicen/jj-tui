#!/usr/bin/env bash
# Repo for vhs/revset-filter.tape: commits with distinct descriptions for `/` text filter.
# Run: make revset-filter-gif
# Requires: jj, git
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$SCRIPT_DIR/revset-filter-vhs-repo"

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

echo "# Revset filter demo" > README.md
jj describe -m "Initial import"
jj bookmark create main
jj new

echo 'package golden' > widget.go
jj describe -m "Add golden widget API"

jj new
echo "fix readme typo" >> README.md
jj describe -m "Fix typo in readme"

jj new
echo "refactor" > handlers.go
jj describe -m "Refactor handlers"

echo "Revset filter VHS repo ready: $REPO"
