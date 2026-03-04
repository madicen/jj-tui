# VHS tape for capturing a demo GIF of the TUI with CPU and memory profiling
# Run with: make demo-gif-profile
#
# Same as all.tape but runs jj-tui with -cpuprofile and -memprofile.
# Profiles are written to screenshots/ when the TUI exits (Ctrl+q).

Output "screenshots/demo.gif"

Set FontSize 14
Set Width 1300
Set Height 650
Set Shell "bash"
Set Theme "Builtin Dark"
Set TypingSpeed 75ms

# Setup: Run in demo mode with profiling (profiles written to screenshots/ on exit)
Hide
Type "cd fixtures/demo-repo && ../../jj-tui --demo -cpuprofile=../../screenshots/cpu.prof -memprofile=../../screenshots/mem.prof"
Enter
Sleep 3s
Show

# === SCENE 1: Review current work ===
Sleep 2s
Down
Sleep 500ms
Down
Sleep 1.5s

# === SCENE 2: Create a new commit from current changes ===
Type "e"
Sleep 2s
Up
Sleep 500ms
Up
Sleep 500ms
Tab
Sleep 2s
Type "]"
Sleep 2s
Tab
Sleep 1s
Up
Sleep 2s

# === SCENE 3: Describe the commit ===
Type "d"
Sleep 1s
Backspace 7
Type "Extract utility functions to separate file"
Sleep 500ms
Ctrl+S
Sleep 1.5s

# === SCENE 4: Rebase on main ===
Type "r"
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 500ms
Enter
Sleep 2s

# === SCENE 5: Find a ticket to work on ===
Type "t"
Sleep 2s
Down
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 1.5s
Enter
Sleep 2s
Enter
Sleep 2s

# === SCENE 6: Create a PR ===
Up
Sleep 500ms
Up
Sleep 500ms
Up
Sleep 500ms
Up
Sleep 500ms
Up
Sleep 500ms
Type "c"
Sleep 1s
Tab
Type "Created new utility function for this"
Sleep 1s
Ctrl+S
Sleep 2s

# === SCENE 6: Go to PRs tab ===
Type "p"
Sleep 2s

# === SCENE 7: Review PRs ===
Down
Sleep 500ms
Down
Sleep 1.5s

# === SCENE 9: Go back to Tickets to update status ===
Type "t"
Sleep 1.5s
Type "c"
Sleep 2s
Type "D"
Sleep 2s

# === SCENE 10: Check Branches view ===
Type "b"
Sleep 2s

# === END: Clean exit (profiles written here) ===
Hide
Ctrl+q
