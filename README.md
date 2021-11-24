Basically a replacement to GNU Stow for my use cases:

fling
    --location_parent
    --link_to_parent
    --mode dry-run confirm force
    link
    unlink
    version

Ok, I have a potential linkPath and the srcPath it should refer to

see https://pkg.go.dev/io/fs#FileMode for differnt types of files - question: does a symlink to a directory have the directory modebit set? IT DOES NOT (on macos)!

states of srcPath (it must exist by construction)
- isSomethingElse: error - we only support directories and files in srcPath
- isSymlink
- isDir
- isFile

states of linkPath:
- notExists
- isSomethingElse
- isSymlinkToSrcpath
- isSymlinkToOther
- isDir
- isFile

srcPath isSomethingElse
    error
srcpath isSymlink
    error
srcPath isDir
    linkPath notExists: create a link, skip children of srcPath
    linkPath statErr: error
    linkPath isSomethingElse: error
    linkPath isSymlinkToSrcpath: log but continue (linkPath must also be a directory) (See question above)
    linkPath isSymlinkToOther: error - we only allow symlinks to corresponding dirs in srcPath
    linkPath isDir: error -  linkPath is existing Dir (so we don't want to erase it)
    linkPath isFile: error - linkPath is existing file
srcPath isFile
    linkPath notExists: create a link
    linkPath statErr: error
    linkPath isSomethingElse: error
    linkPath isSymlinkToSrcpath: log but continue
    linkPath isSymlinkToOther: error - we only allow symlinks to corresponding dirs in srcPath
    linkPath isDir: error - linkPath is existing Dir
    linkPath isFile: error - linkPath is existing file

With the caveat that I don't really need to skip children of srcPath, I think I have everything I need in the dir part... I can just not check if it's a directory or file


# https://pkg.go.dev/io/fs#FileMode


# TODO

- add unlink command - and use it to unsymlink ~/Gt/dotfiles/sqlite3
- add ignore pattern
- try it on my dotfiles
- add option to print out the equivalent `ln -s` commands
- Dear Lord this needs to be tested
- If you make a symlink where linkPath has folders that don't exist, ln -s will make the folder, then make the link in the folder. `$ ln -s /Users/bbkane/Git/dotfiles/nvim/.config/nvim /tmp/.config/nvim`. ln made /tmp/config a directory, then made nvim a link
- Is this why stow only deals with relative symlinks? - I think this is wrong actually. I think I made that directory earlier.