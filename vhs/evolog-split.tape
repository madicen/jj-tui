# split (z): evolog picker + diff preview, choose parent, FAQ-style jj new/restore/abandon.
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
Sleep 3s
Show

# Graph: working copy is on demo/feature (jj edit); feature bookmark tip is selected.
Sleep 2s

# split (z) — loads evolog + file diff vs tip; pick a parent row with a real tree delta vs tip.
Type "z"
Sleep 3s

# Default row can be an intermediate evolution commit with an empty diff vs tip; move one step older (j).
Type "j"
Sleep 1s
Type "j"
Sleep 1s

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
