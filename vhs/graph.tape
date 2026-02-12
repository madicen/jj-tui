# VHS tape for capturing the Graph view screenshot
# Run with: vhs vhs/graph.tape

Output "screenshots/graph.png"

Set FontSize 14
Set Width 1300
Set Height 600
Set Shell "bash"
Set Theme "Catppuccin Mocha"

# Force color support for Lipgloss/Bubble Tea
Set Env "TERM" "xterm-256color"
Set Env "COLORTERM" "truecolor"

# Setup: Run the demo repo setup first, then navigate to it
Type "cd fixtures/demo-repo && CLICOLOR_FORCE=1 ../../jj-tui --demo"
Enter
Sleep 4s

Screenshot "screenshots/graph.png"
