# Multi-select batch ops: Space toggles gutter markers; `a` opens batch-abandon confirm (cancel with n).
# Run: make multiselect-gif
# Requires: VHS, bash, jj, git, fixtures/setup-multiselect-vhs-repo.sh

Output "screenshots/multiselect.gif"

Set FontSize 14
Set Width 1300
Set Height 700
Set Shell "bash"
Set Theme "Builtin Dark"
Set TypingSpeed 75ms

# Color support is forced in demo mode via lipgloss.SetColorProfile()

Hide
Type "( bash fixtures/setup-multiselect-vhs-repo.sh && cd fixtures/multiselect-vhs-repo && ../../jj-tui --demo )"
Enter
Sleep 6s
Show

Sleep 2s

# Working copy (@) cannot be toggled; select three mutable ancestors.
Up
Sleep 500ms
Space
Sleep 600ms
Up
Sleep 500ms
Space
Sleep 600ms
Up
Sleep 500ms
Space
Sleep 2s

# Batch abandon confirm (cancel so the clip ends on the selection UI).
Type "a"
Sleep 2s
Type "n"
Sleep 1s
Escape
Sleep 2s

Hide
Ctrl+q
Sleep 500ms
