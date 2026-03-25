#!/usr/bin/env bash
# Small jj repo for vhs/evolog-split.tape: feature bookmark whose change has multiple jj evolog
# revisions after squashing edits from a child WC into the feature commit.
# The main "Add feature flag" commit touches several files; only 1–2 files change in the squash
# so evolog base→tip diff is small and the split (rollout vs rest) reads clearly in the modal.
# Tape: z → j j (pick an older row with a non-empty diff vs tip) → Enter.
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

mkdir -p src docs
cat > src/feature-flag.txt <<'EOF'
mode=light
EOF
cat > src/ui-settings.toml <<'EOF'
theme = "light"
animations = true
EOF
cat > docs/changelog.md <<'EOF'
# Changelog

## 0.1.0
- Initial import
EOF

jj describe -m "Initial import"
jj bookmark create main
jj new

cat > src/feature-flag.txt <<'EOF'
mode=dark
EOF
cat > src/ui-settings.toml <<'EOF'
theme = "dark"
animations = true
dark_mode_preview = true
EOF
cat >> docs/changelog.md <<'EOF'

## Unreleased
- Dark mode and preview toggle
EOF
jj describe -m "Add feature flag"
jj bookmark create demo/feature

# Empty WC on top of feature so `jj squash` folds the edit into the feature commit (evolution), not into main.
# Rollout tweaks only (changelog already updated above): split modal shows a 2-file delta vs older evolog rows.
jj new
cat >> src/feature-flag.txt <<'EOF'
rollout=10pct
EOF
cat >> src/ui-settings.toml <<'EOF'
rollout_percent = 10
EOF
jj squash -m "Add feature flag"

jj edit demo/feature >/dev/null

echo "Evolog split VHS repo ready: $REPO"
