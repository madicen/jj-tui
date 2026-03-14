# VHS tape for the modal example (two color boxes; Tab+Enter or click)
# Run from bubblepicker dir: vhs vhs/modal.tape

Output "screenshots/modal.png"

Set FontSize 14
Set Width 900
Set Height 500
Set Shell "bash"
Set Theme "Builtin Dark"

Type "go run ./examples/modal"
Enter
Sleep 3s

Screenshot "screenshots/modal.png"

Type "q"
Enter
Sleep 500ms
