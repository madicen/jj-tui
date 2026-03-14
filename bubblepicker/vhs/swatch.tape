# VHS tape for the swatch example (2×2 grid; click opens modal)
# Run from bubblepicker dir: vhs vhs/swatch.tape

Output "screenshots/swatch.png"

Set FontSize 14
Set Width 900
Set Height 500
Set Shell "bash"
Set Theme "Builtin Dark"

Type "go run ./examples/swatch"
Enter
Sleep 3s

Screenshot "screenshots/swatch.png"

Type "q"
Enter
Sleep 500ms
