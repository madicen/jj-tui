# Resolving a divergent change (d): pick which revision to keep; others are abandoned.
# Run: make divergent-gif
# Requires: VHS, bash, jj, git, fixtures/setup-divergent-vhs-repo.sh

Output "screenshots/divergent.gif"

Set FontSize 14
Set Width 1300
Set Height 700
Set Shell "bash"
Set Theme "Builtin Dark"
Set TypingSpeed 75ms

Hide
Type "( bash fixtures/setup-divergent-vhs-repo.sh && cd fixtures/divergent-vhs-repo && ../../jj-tui --demo )"
Enter
Sleep 4s
Show

Sleep 2s

# Newest-first graph: working copy row, then amended tip (bookmark), then sibling divergent revision.
# Move to the amended-tip row (already divergent; d opens the resolver).
Down
Sleep 1s

Type "d"
Sleep 3s

# Second modal option is the pre-amend commit (widget only vs widget + plan).
Down
Sleep 1.5s

Enter
Sleep 4s

Hide
Ctrl+q
Sleep 500ms
