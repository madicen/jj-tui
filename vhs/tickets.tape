# VHS tape for capturing the Tickets view screenshot
# Run with: vhs vhs/tickets.tape

Output "screenshots/tickets.png"

Set FontSize 14
Set Width 1300
Set Height 600
Set Shell "bash"
Set Theme "Catppuccin Mocha"

# Setup: Run in demo mode from the demo repo
Type "cd fixtures/demo-repo && ../../jj-tui --demo"
Enter
Sleep 3s

# Navigate to Tickets tab
Type "t"
Sleep 2s

