# Dedicated GIF for "Forgot New Commit?" (f): PR (demo), working-tree edits without a new jj commit, stack on bookmark@origin, then Update PR (u).
# Run: make after-origin-gif
# Requires: VHS, bash, jj, git

Output "screenshots/after-origin.gif"

Set FontSize 14
Set Width 1200
Set Height 600
Set Shell "bash"
Set Theme "Builtin Dark"
Set TypingSpeed 75ms

Hide
# Subshell so cd does not stick: after quit, cwd stays repo root and bash fixtures/... finds the script.
Type "( bash fixtures/setup-after-origin-vhs-repo.sh && cd fixtures/after-origin-vhs-repo && ../../jj-tui --demo )"
Enter
Sleep 3s
Show

# === Create PR (mock) from current feature bookmark tip ===
Sleep 2s
Type "c"
Sleep 1s
Tab
Type "Follow-up work after review"
Sleep 1s
Ctrl+S
Sleep 3s

# Exit to shell: append via script (VHS Type cannot embed nested quotes / long shell)
Hide
Ctrl+q
Sleep 1s
Type "bash fixtures/after-origin-vhs-append-and-tui.sh"
Enter
Sleep 3s
Show

# Graph: working copy now differs from vhs/feature@origin → Forgot New Commit? (f)
Sleep 2s
Type "f"
Sleep 4s

# Update PR (push vhs/feature to origin); demo includes open PR #901 for this branch after relaunch
Sleep 2s
Type "u"
Sleep 5s

Hide
Ctrl+q
