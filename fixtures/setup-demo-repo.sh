#!/bin/bash
# Setup script for creating a demo jj repository with interesting state
# This repo is used for VHS screenshots and visual testing

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEMO_REPO="$SCRIPT_DIR/demo-repo"

# Clean up any existing demo repo
if [ -d "$DEMO_REPO" ]; then
    echo "Removing existing demo repo..."
    rm -rf "$DEMO_REPO"
fi

echo "Creating demo repository at $DEMO_REPO"
mkdir -p "$DEMO_REPO"
cd "$DEMO_REPO"

# Initialize git and jj
git init --initial-branch=main
jj git init --colocate

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

# Create a conflicted state for demo (optional - can be enabled)
# This creates a branch that diverges from another

# Make main immutable by creating a fake remote tracking
# (In real usage, this would be from pushing to origin)
jj bookmark create main@origin -r main --allow-backwards 2>/dev/null || true

echo ""
echo "Demo repository created successfully!"
echo "Location: $DEMO_REPO"
echo ""
echo "Repository state:"
jj log --no-pager -r 'all()' --template 'builtin_log_compact'
echo ""
echo "Bookmarks:"
jj bookmark list --no-pager

