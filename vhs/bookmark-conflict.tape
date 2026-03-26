# Diverged bookmark: graph stays visible; centered popup (Branches c) shows local vs origin, side-by-side choices.
# Run: make bookmark-conflict-gif
# Requires: VHS, bash, jj, git, fixtures/setup-bookmark-conflict-vhs-repo.sh

Output "screenshots/bookmark-conflict.gif"

Set FontSize 14
Set Width 1300
Set Height 700
Set Shell "bash"
Set Theme "Builtin Dark"
Set TypingSpeed 75ms

Hide
Type "( bash fixtures/setup-bookmark-conflict-vhs-repo.sh && cd fixtures/bookmark-conflict-vhs-repo && ../../jj-tui --demo )"
Enter
Sleep 5s
Show

Sleep 2s

# Graph: diverged bookmark on feature row (dialog will overlay without hiding the graph).
Sleep 1s

# Branches tab → vhs/conflict-feature is first in list (LoadBranches sorts ahead-only locals first), so do NOT press Down (that selects main).
Type "b"
Sleep 2s
Type "c"
Sleep 3s

# Switch resolution with j / k (popup stays compact); confirm keep local.
Type "j"
Sleep 1s
Type "k"
Sleep 1s
Enter
Sleep 5s

Hide
Ctrl+q
Sleep 500ms
