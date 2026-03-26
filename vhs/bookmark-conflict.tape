# Resolve diverged bookmark: Branches tab (c) opens local vs remote chooser; keep local or reset to remote.
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

# Commit graph: diverged bookmark shows warning on the feature row (and optional inline resolve).
# Open Branches to use Resolve Conflict (c).
Type "b"
Sleep 2s

# Local bookmark list order is main first; move to diverged vhs/conflict-feature.
Down
Sleep 1s

Type "c"
Sleep 3s

# Keep local revision (default); confirm.
Enter
Sleep 5s

Hide
Ctrl+q
Sleep 500ms
