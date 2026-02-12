# VHS tape for capturing the Graph view screenshot
# Run with: vhs vhs/graph.tape

Output "screenshots/graph.png"

Set FontSize 14
Set Width 1300
Set Height 600
Set Shell "bash"
Set Theme "Catppuccin Mocha"

# Setup: Run the demo repo setup first, then navigate to it
Type "cd fixtures/demo-repo && ../../jj-tui --demo"
Enter
Sleep 4s

# Graph view is the default - screenshot is captured here
# (VHS captures the last frame before recording ends)

