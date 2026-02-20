Output demo-with.mp4

Set BorderRadius 25
Set WindowBar Colorful

Set FontSize 16
Set Width 800
Set Height 600
Set Padding 5
Set Theme 'Catppuccin Mocha'

Set TypingSpeed 100ms

# Open the editor
Hide
Type 'hx app.yaml'
Enter
Wait+Screen /NORMAL/
Show

# Go into Insert mode and type the trigger
Type 'i'
Wait+Screen /INSERT/
Type 'apiVersion: apps/v1'
Enter
Type 'kind: Deployment'
Enter

# Set timeout to let lsp fetch schema
Hide
Sleep 3s
Show


# Add metadata with autocomplete
Type 'm'
Ctrl+X
Wait+Screen /property/
Tab
Sleep 200ms

# Add name
Type 'n'
Ctrl+X
Wait+Screen /property/
Tab
Sleep 200ms
Type 'app'

# Add labels
Enter
Type 'l'
Ctrl+X
Wait+Screen /property/
Tab
Sleep 200ms
Type 'app: demo'

# Reset indent
Enter
Backspace@100ms 2

# Add spec
Type 's'
Ctrl+X
Wait+Screen /property/
Tab
Sleep 200ms

# Add matchLabels
Type 'm'
Ctrl+X
Wait+Screen /property/
Tab@100ms 2
Sleep 200ms
Type 'app: demo'

# Naviaget to template
Escape
Wait+Screen /NORMAL/
Type '2ji'
Wait+Screen /INSERT/

# Add metadata
Type 'm'
Ctrl+X
Wait+Screen /property/
Tab
Sleep 200ms
Type 'l'
Ctrl+X
Wait+Screen /property/
Tab
Sleep 200ms
Type 'app: demo'

# Reset indent
Enter
Backspace@100ms 2

# Add spec
Type 's'
Ctrl+X
Wait+Screen /property/
Tab
Sleep 200ms

# Add conatiner name
Type ' n'
Ctrl+X
Wait+Screen /property/
Tab
Sleep 200ms

Type 'web'
Enter

# Add conatiner image
Type 'i'
Ctrl+X
Wait+Screen /property/
Tab
Sleep 200ms

Type 'nginx:1.25-alpine'
Enter

# Add conatiner port
Type 'p'
Ctrl+X
Wait+Screen /property/
Tab
Sleep 200ms

Type 'c'
Ctrl+X
Wait+Screen /property/
Tab
Sleep 200ms

Type '80'

# Naviagate to root spec
Escape
Wait+Screen /NORMAL/
Type '13ko'
Wait+Screen /INSERT/

# Add replicas
Type 'r'
Ctrl+X
Wait+Screen /property/
Tab
Sleep 200ms
Type '"three'
Escape
Wait+Screen /NORMAL/
Sleep 3s
Type 'bma"c'
Wait+Screen /INSERT/
Type '3'
Escape

