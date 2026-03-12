# VHS tape for the simple example (picker is the whole app)
# Run from bubblepicker dir: vhs vhs/simple.tape

Output "screenshots/simple.png"

Set FontSize 14
Set Width 900
Set Height 500
Set Shell "bash"
Set Theme "Builtin Dark"

Type "go run ./examples/simple"
Enter
Sleep 3s

Screenshot "screenshots/simple.png"

Type "q"
Enter
Sleep 500ms
