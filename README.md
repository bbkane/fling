# fling

fling computes and creates/removes the minimal amount of symlinks needed in a directory to refer to files and directories in another directory, similar to [GNU Stow](https://www.gnu.org/software/stow/). I use fling to manage my [dotfiles](https://github.com/bbkane/dotfiles)

fling is much dumber than GNU Stow - it's missing several options, only considers one directory at a time, and deals with absolute paths only (in contrast, GNU Stow works exclusively with relative paths).

As a tradeoff, fling's codebase is simpler. fling compiles to a single binary and contains roughly 750 lines of Go code as of 2021-12-03. It's very easy to understand what fling will do for a particular invocation: fling prints out (in color!) what it plans to link/unlink and asks you (by default) before continuing. Finally, fling's `--dotfiles` option [works correctly](https://github.com/aspiers/stow/issues/33) (and is on by default).

## Install

- Homebrew: `brew install bbkane/tap/fling`
- Download Mac/Linux executable: [GitHub releases](https://github.com/bbkane/fling/releases)
- Go: `go install go.bbkane.com/fling@latest`
- Build with [goreleaser](https://goreleaser.com/) after cloning: `goreleaser --snapshot --skip-publish --rm-dist`

## Example

![demo](./demo.gif)

## Notes

fling was built to manage my dotfiles, and it's only tested on those dotfiles. Please feel free to contribute tests.

---

If you're migrating from GNU Stow, you may see an error similar to the following when you attempt to `fling link` something:

```
Proposed link mismatch errors:

- src: /Users/bbkane/Git/dotfiles/sqlite3/dot-sqliterc
  link: /Users/bbkane/.sqliterc
  err: link is already a symlink to src: Git/dotfiles/sqlite3/dot-sqliterc
```

This is because GNU Stow works with relative symlinks and fling works with absolute symlinks. Either unstow your directory (`stow -D`) or manually erase the symlink: `unlink /Users/bbkane/.sqliterc`
