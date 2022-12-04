# https://github.com/charmbracelet/vhs

Require fling

Output demo.gif

Set Shell zsh
Set PlaybackSpeed 0.5
Set TypingSpeed 75ms

Set FontSize 22
Set Height 650
Set Width 1050
Set Padding 50

# https://github.com/charmbracelet/vhs/issues/39#issuecomment-1296334921
Hide
Sleep 600ms
Show

Type "cd ~/Git-GH/dotfiles"
Sleep 200ms
Enter
Sleep 1s

Type "ls grabbit"
Sleep 200ms
Enter
Sleep 1s

Type "fling -s grabbit -i 'README.*' link"
Sleep 200ms
Enter
Sleep 500ms
Type "yes"
Sleep 500ms
Enter
Sleep 1s

Type "fling -s grabbit -i 'README.*' --ask false unlink"
Sleep 200ms
Enter
Sleep 1s

Sleep 3s
