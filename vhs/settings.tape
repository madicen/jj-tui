# VHS tape for capturing the Settings view screenshot
# Run with: vhs vhs/settings.tape

Output "screenshots/settings.png"

Set FontSize 14
Set Width 1300
Set Height 600
Set Shell "bash"
Set Theme "Catppuccin Mocha"

# Setup: Run in demo mode from the demo repo
Type "cd fixtures/demo-repo && ../../jj-tui --demo"
Enter
Sleep 3s

# Navigate to Settings tab
Type ","
Sleep 2s

