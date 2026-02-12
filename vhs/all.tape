# VHS tape for capturing a demo GIF of the TUI
# Run with: vhs vhs/all.tape
# 
# This tape generates a GIF showing navigation through all views.

Output "screenshots/demo.gif"

Set FontSize 14
Set Width 1300
Set Height 600
Set Shell "bash"
Set Theme "Catppuccin Mocha"

# Setup: Run in demo mode from the demo repo
Hide
Type "cd fixtures/demo-repo && ../../jj-tui --demo"
Enter
Sleep 3s
Show

# Graph view is default - pause to show it
Sleep 2s

# Navigate to PRs
Type "p"
Sleep 2s

# Navigate to Tickets
Type "t"
Sleep 2s

# Navigate to Branches
Type "b"
Sleep 2s

# Navigate to Settings
Type ","
Sleep 2s

# Exit Settings
Escape

# Navigate to Help
Type "h"
Sleep 2s

# Exit cleanly
Ctrl+q
Sleep 500ms

