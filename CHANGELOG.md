# Changelog

All notable changes to this project will be documented in this file. The format
is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

Note the the latest version may be unreleased.

# v0.0.24

## Removed

- `arm` binary support (since warg doesn't support it. See https://github.com/bbkane/fling/pull/3

# v0.0.23

## Added

- `--src-dir` (`-s`) now accepts multiple values. Pass `-s` multiple times to symlink from several source directories at once (e.g., `fling link -s ~/dotfiles/nvim -s ~/dotfiles/tmux`). Conflicts between source directories are detected and reported before any links are created or deleted.

## Changed

- Updated warg to v0.40.2, adding bash/fish completion and changes to default command help output

# v0.0.22

## Added

- `zsh` tab completion via warg update!

# v0.0.21

Behind the scenes warg update, but nothing user-facing

# v0.0.20

## Fixed

- Fix `version` command

# v0.0.19

## Changed

- `--ignore` flag now ignores `README.*` by default. Pass `-i UNSET` to not ignore anything.

