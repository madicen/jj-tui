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

# Force color support for Lipgloss/Bubble Tea
Env TERM "xterm-256color"
Env COLORTERM "truecolor"

# Setup: Run in demo mode from the demo repo
Hide
Type "cd fixtures/demo-repo && CLICOLOR_FORCE=1 ../../jj-tui --demo"
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

Hide
# Exit Settings
Escape

# Navigate to Help
Type "h"
Show
Sleep 2s

Hide
# Exit cleanly
Ctrl+q

