# split (z): evolog picker + step file list; o opens colored step diff overlay; Enter splits.
# Run: make evolog-split-gif
# Requires: VHS, bash, jj, git, fixtures/setup-evolog-split-vhs-repo.sh

Output "screenshots/evolog-split.gif"

Set FontSize 14
Set Width 1300
Set Height 620
Set Shell "bash"
Set Theme "Builtin Dark"
Set TypingSpeed 75ms

# Subshell so cd does not stick after quit (same pattern as vhs/after-origin.tape).
Hide
Type "( bash fixtures/setup-evolog-split-vhs-repo.sh && cd fixtures/evolog-split-vhs-repo && ../../jj-tui --demo )"
Enter
Sleep 4s
Show

# Graph: working copy is on demo/feature (jj edit); feature bookmark tip is selected.
Sleep 2s

# split (z) — loads evolog; default row (below @) has an empty step diff vs the row above — press j once
# to select the next row, whose step vs neighbor includes the rollout edits (feature-flag + ui-settings).
Type "z"
Sleep 3s

Type "j"
Sleep 1s

# Colored step diff overlay (same styling as graph file diff), scroll, then dismiss
Type "o"
Sleep 3s
Down
Sleep 500ms
Down
Sleep 500ms
Escape
Sleep 2s

# Key Enter (not Type "Enter", which would type the word "Enter").
Enter
Sleep 4s

Up
Sleep 1s
Up
Sleep 2s

Hide
Ctrl+q
Sleep 500ms
