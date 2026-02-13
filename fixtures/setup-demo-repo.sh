#!/bin/bash
# Setup script for creating a demo jj repository with interesting state
# This repo is used for VHS screenshots and visual testing

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEMO_REPO="$SCRIPT_DIR/demo-repo"
FAKE_ORIGIN="$SCRIPT_DIR/fake-origin.git"

# Clean up any existing demo repo and fake origin
if [ -d "$DEMO_REPO" ]; then
    echo "Removing existing demo repo..."
    rm -rf "$DEMO_REPO"
fi
if [ -d "$FAKE_ORIGIN" ]; then
    echo "Removing existing fake origin..."
    rm -rf "$FAKE_ORIGIN"
fi

# Create a bare repository to serve as "origin"
echo "Creating fake origin repository..."
git init --bare "$FAKE_ORIGIN"

echo "Creating demo repository at $DEMO_REPO"
mkdir -p "$DEMO_REPO"
cd "$DEMO_REPO"

# Initialize git and jj
git init --initial-branch=main
jj git init --colocate

# Add the fake origin as a remote
git remote add origin "$FAKE_ORIGIN"

# Configure jj for demo
jj config set --repo user.name "Demo User"
jj config set --repo user.email "demo@example.com"

# Create initial commit (will become the "main" trunk)
echo "# Awesome Project" > README.md
echo "" >> README.md
echo "A demo project for jj-tui screenshots." >> README.md
jj describe -m "Initial commit"

# Create the main bookmark on this commit
jj bookmark create main

jj new

# Create some source files
mkdir -p src
cat > src/main.go << 'EOF'
package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
EOF

cat > src/utils.go << 'EOF'
package main

func add(a, b int) int {
    return a + b
}
EOF

jj describe -m "Add initial source files"
jj new

# Create a feature branch with bookmark
cat > src/feature.go << 'EOF'
package main

// DarkMode enables dark mode theme
var DarkMode = false

func toggleDarkMode() {
    DarkMode = !DarkMode
}
EOF

# Modify main.go
cat > src/main.go << 'EOF'
package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
    if DarkMode {
        fmt.Println("Dark mode enabled")
    }
}
EOF

jj describe -m "PROJ-142 Add dark mode support"
jj bookmark create feature/dark-mode

# Create another branch from main
jj new main

mkdir -p src
cat > src/pagination.go << 'EOF'
package main

// Page represents pagination state
type Page struct {
    Current int
    Total   int
}

func (p *Page) Next() {
    if p.Current < p.Total {
        p.Current++
    }
}
EOF

jj describe -m "PROJ-139 Fix pagination bug in search"
jj bookmark create fix/pagination

# Create a working copy with uncommitted changes
jj new main

mkdir -p src
cat > src/settings.go << 'EOF'
package main

// Settings holds user preferences
type Settings struct {
    Theme    string
    Language string
}
EOF

jj describe -m "PROJ-135 Implement settings page (WIP)"
jj bookmark create feature/settings

# Go back to working copy and make it the current edit
jj edit @

# Add some modified files to show in the files pane
# Create main.go if it doesn't exist (since we branched from main which doesn't have it)
if [ ! -f src/main.go ]; then
    cat > src/main.go << 'EOF'
package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
EOF
fi
echo "// TODO: Add more features" >> src/main.go
echo "" >> README.md
echo "## Features" >> README.md
echo "- Dark mode" >> README.md
echo "- User settings" >> README.md

# Push main and some branches to origin to create remote tracking branches
echo ""
echo "Pushing to fake origin to create remote tracking branches..."
jj git push --bookmark main --remote origin
jj git push --bookmark feature/dark-mode --remote origin
jj git push --bookmark fix/pagination --remote origin

# Fetch to ensure jj knows about the remote branches
jj git fetch --remote origin

echo ""
echo "Demo repository created successfully!"
echo "Location: $DEMO_REPO"
echo ""
echo "Repository state:"
jj log --no-pager -r 'all()' --template 'builtin_log_compact'
echo ""
echo "Bookmarks (including remote tracking):"
jj bookmark list --no-pager --all

