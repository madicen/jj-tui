# VHS tape for capturing the Pull Requests view screenshot
# Run with: vhs vhs/prs.tape

Output "screenshots/prs.png"

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

# Navigate to PRs tab
Type "p"
Sleep 2s

Screenshot "screenshots/prs.png"
