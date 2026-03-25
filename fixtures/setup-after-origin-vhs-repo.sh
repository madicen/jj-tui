#!/bin/bash
# Small jj repo for vhs/after-origin.tape: one feature bookmark pushed to a fake origin,
# clean working copy. The tape creates a PR (demo), edits the working tree without jj commit,
# runs Forgot New Commit? (f), then Update PR (u). Demo mock PR #901 matches vhs/feature for (u) after relaunch.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$SCRIPT_DIR/after-origin-vhs-repo"
ORIG="$SCRIPT_DIR/after-origin-fake-origin.git"

rm -rf "$REPO" "$ORIG"

git init --bare "$ORIG" >/dev/null
mkdir -p "$REPO"
cd "$REPO"

git init --initial-branch=main >/dev/null
jj git init --colocate >/dev/null
git remote add origin "$ORIG"

jj config set --repo user.name "Demo User"
jj config set --repo user.email "demo@example.com"

echo "# After-origin VHS demo" > README.md
jj describe -m "Initial commit"
jj bookmark create main
jj new

mkdir -p src
cat > src/app.go << 'EOF'
package main

import "fmt"

func main() {
	fmt.Println("demo")
}
EOF

jj describe -m "VHS-101 Add application entry"
jj bookmark create vhs/feature

jj git push --bookmark main --remote origin --allow-new
jj git push --bookmark vhs/feature --remote origin --allow-new
jj git fetch --remote origin >/dev/null

jj edit vhs/feature >/dev/null

echo ""
echo "After-origin VHS repo ready: $REPO"
