# VHS tape for capturing the Pull Requests view screenshot
# Run with: vhs vhs/prs.tape

Output "screenshots/prs.png"

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

# Navigate to PRs tab
Type "p"
Sleep 2s

Screenshot "screenshots/prs.png"