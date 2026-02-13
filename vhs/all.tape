# VHS tape for capturing a demo GIF of the TUI
# Run with: vhs vhs/all.tape
# 
# This tape shows a realistic workflow through the TUI:
# - Review current work in progress
# - Create a new commit
# - Find and select a ticket
# - Create a bookmark and PR
# - Review and merge a PR
# - Update ticket status
# - Return to commit graph

Output "screenshots/demo.gif"

Set FontSize 14
Set Width 1300
Set Height 650
Set Shell "bash"
Set Theme "Builtin Dark"
Set TypingSpeed 75ms

# Color support is forced in demo mode via lipgloss.SetColorProfile()

# Setup: Run in demo mode from the demo repo
Hide
Type "cd fixtures/demo-repo && ../../jj-tui --demo"
Enter
Sleep 3s
Show

# === SCENE 1: Review current work ===
# Graph view shows our work in progress
Sleep 2s

# Move down to see more commits
Down
Sleep 500ms
Down
Sleep 1.5s

# === SCENE 2: Create a new commit from current changes ===
# Press 'e' to create new commit
Type "e"
Sleep 1s
Tab
Sleep 1s
Type "]"
Sleep 1s
Tab
Sleep 1s
Up
Sleep 1s


# === SCENE 3: Describe the commit ===
# Press 'd' to describe
Type "d"
Sleep 1s
# Type a commit message
Backspace 7
Type "Extract utility functions to separate file"
Sleep 500ms
Ctrl+s
Sleep 1.5s

# === SCENE 4: Rebase on main ===
Type "r"
Sleep 1s
Down 5
Enter
Sleep 1.5s

# === SCENE 5: Find a ticket to work on ===
# Navigate to Tickets tab
Type "t"
Sleep 2s

# Scroll down through tickets
Down
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 1.5s
Enter
Sleep 2s
Enter

# === SCENE 6: Go to PRs tab ===
Type "p"
Sleep 2s

# === SCENE 7: Review PRs ===
# Move through PR list
Down
Sleep 500ms
Down
Sleep 1.5s

# === SCENE 9: Go back to Tickets to update status ===
Type "t"
Sleep 1.5s

# Change ticket status
Type "c"
Sleep 1s

# Select "Done" transition
Down
Sleep 500ms
Enter
Sleep 1.5s

# === SCENE 10: Check Branches view ===
Type "b"
Sleep 2s
Type "F"

# === SCENE 11: Return to Graph ===
Type "g"
Sleep 2s

# === END: Clean exit ===
Hide
Ctrl+q

