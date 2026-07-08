# Revset search/filter: `/` overlay, text filter, Esc clears.
# Run: make revset-filter-gif
# Requires: VHS, bash, jj, git, fixtures/setup-revset-filter-vhs-repo.sh

Output "screenshots/revset-filter.gif"

Set FontSize 14
Set Width 1300
Set Height 700
Set Shell "bash"
Set Theme "Builtin Dark"
Set TypingSpeed 75ms

Hide
Type "( bash fixtures/setup-revset-filter-vhs-repo.sh && cd fixtures/revset-filter-vhs-repo && ../../jj-tui )"
Enter
Sleep 6s
Show

Sleep 2s

Type "/"
Sleep 1s
Type "golden"
Sleep 1s
Enter
Sleep 4s

Escape
Sleep 3s

Hide
Ctrl+q
Sleep 500ms
