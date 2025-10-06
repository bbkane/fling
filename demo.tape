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

Hide

Type 'setopt interactive_comments'
Enter

Type 'source $(brew --prefix)/share/zsh-syntax-highlighting/zsh-syntax-highlighting.zsh'
Enter

Type 'clear'
Enter

Show

Type "cd ~/Git-GH/dotfiles"
Sleep 200ms
Enter
Sleep 1s

Type "ls grabbit"
Sleep 200ms
Enter
Sleep 1s

Type "fling link --src-dir grabbit"
Sleep 200ms
Enter
Sleep 500ms
Type "yes"
Sleep 500ms
Enter
Sleep 1s

Type "fling unlink --src-dir grabbit --ask false"
Sleep 200ms
Enter
Sleep 1s

Sleep 3s
