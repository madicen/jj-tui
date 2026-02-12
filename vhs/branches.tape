# VHS tape for capturing the Branches view screenshot
# Run with: vhs vhs/branches.tape

Output "screenshots/branches.png"

Set FontSize 14
Set Width 1300
Set Height 600
Set Shell "bash"
Set Theme "Catppuccin Mocha"

# Force color support for Lipgloss/Bubble Tea
Env TERM "xterm-256color"
Env COLORTERM "truecolor"

# Setup: Run in demo mode from the demo repo
Type "cd fixtures/demo-repo && CLICOLOR_FORCE=1 ../../jj-tui --demo"
Enter
Sleep 3s

# Navigate to Branches tab
Type "b"
Sleep 2s

Screenshot "screenshots/branches.png"