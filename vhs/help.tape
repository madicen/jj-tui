# VHS tape for capturing the Help view screenshot
# Run with: vhs vhs/help.tape

Output "screenshots/help.png"

Set FontSize 14
Set Width 1300
Set Height 600
Set Shell "bash"
Set Theme "Builtin Dark"

# Color support is forced in demo mode via lipgloss.SetColorProfile()

# Setup: Run in demo mode from the demo repo
Type "cd fixtures/demo-repo && ../../jj-tui --demo"
Enter
Sleep 3s

# Navigate to Help tab
Type "h"
Sleep 2s

Screenshot "screenshots/help.png"
