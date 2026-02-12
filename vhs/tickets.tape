# VHS tape for capturing the Tickets view screenshot
# Run with: vhs vhs/tickets.tape

Output "screenshots/tickets.png"

Set FontSize 14
Set Width 1300
Set Height 600
Set Shell "bash"
Set Theme "Catppuccin Mocha"

# Force color support for Lipgloss/Bubble Tea
Set Env "TERM" "xterm-256color"
Set Env "COLORTERM" "truecolor"

# Setup: Run in demo mode from the demo repo
Type "cd fixtures/demo-repo && CLICOLOR_FORCE=1 ../../jj-tui --demo"
Enter
Sleep 3s

# Navigate to Tickets tab
Type "t"
Sleep 2s

Screenshot "screenshots/tickets.png"