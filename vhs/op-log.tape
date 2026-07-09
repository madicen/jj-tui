# Operation log browser: Ctrl+o lists jj ops, restore to an earlier op (y confirm).
# Run: make op-log-gif
# Requires: VHS, bash, jj, git, fixtures/setup-op-log-vhs-repo.sh

Output "screenshots/op-log.gif"

Set FontSize 14
Set Width 1300
Set Height 700
Set Shell "bash"
Set Theme "Builtin Dark"
Set TypingSpeed 75ms

# Color support is forced in demo mode via lipgloss.SetColorProfile()

Hide
Type "( bash fixtures/setup-op-log-vhs-repo.sh && cd fixtures/op-log-vhs-repo && ../../jj-tui --demo )"
Enter
Sleep 6s
Show

Sleep 2s

# Open operation log (real jj fixture repo; --demo only mocks GitHub/tickets).
Ctrl+o
Sleep 3s

# Newest op is at top; move down to an older entry and restore.
Down
Sleep 800ms
Down
Sleep 800ms
Down
Sleep 1s
Enter
Sleep 1s
Type "y"
Sleep 4s
Ctrl+r
Sleep 3s

Hide
Ctrl+q
Sleep 500ms
