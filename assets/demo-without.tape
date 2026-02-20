Output demo-without.mp4

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
Type 'metadata:'
Enter
Type 'name: demo-app'
Enter
Type 'labels:'
Enter
Type 'app: demo'
Enter
Backspace@100ms 2
Type 'spec:'
Enter
Type 'selector:'
Enter
Type 'matchLabels:'
Enter
Type 'app: demo'
Enter
Backspace@100ms 2
Type 'template:'
Enter
Type 'metadata:'
Enter
Type 'labels:'
Enter
Type 'app: demo'
Enter
Backspace@100ms 2
Type 'spec:'
Enter
Type 'conatiners:'
Enter
Type '- name: web'
Enter
Type 'image: nginx:1.25-alpine'
Enter
Type 'ports:'
Enter
Type '- containerPort: 80'
Escape
Wait+Screen /NORMAL/
Type '13ko'
Wait+Screen /INSERT/
Type 'replicas: "three'
Escape
Wait+Screen /NORMAL/
Sleep 3s
Type 'bma"c'
Wait+Screen /INSERT/
Type '3'
Escape


