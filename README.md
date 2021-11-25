Basically a replacement to GNU Stow for my use cases:

# TODO

- package for Homebrew
- add --dotfiles true|false flag
- add --script filename.sh flag that prints out an equivalent bash script to a file
- reconsider relative symlinks to be more stow compatible? Maybe with --type relative|absolute flag?
- README!

# goreleaser notes

## locally

Need to release with an unpushed new tag each time. Otherwise you get an error like:

```
   ⨯ release failed after 1.62s error=scm releases: failed to publish artifacts: failed to upload checksums.txt after 1 tries: POST https://uploads.github.com/repos/bbkane/fling/releases/54053584/assets?name=checksums.txt: 422 Validation Failed [{Resource:ReleaseAsset Field:name Code:already_exists Message:}]
```
Need to export GITHUB_TOKEN and KEY_GITHUB_GORELEASER_TO_HOMEBREW_TAP (see KeePassXC)

```
$ goreleaser release --rm-dist
```

Now that a local release is working, I'll try the CI/CD to avoid the following error?

```
    ⨯ release failed after 31.30s error=homebrew tap formula: failed to publish artifacts: PUT https://api.github.com/repos/bbkane/homebrew-tap/contents/Formula/fling.rb: 404 Not Found []
```

Still got it. I updated the repo secret to be obviously wrong. Let's see if the error message changes.

```
   ⨯ release failed after 15.99s error=homebrew tap formula: failed to publish artifacts: GET https://api.github.com/repos/bbkane/homebrew-tap/contents/Formula/fling.rb: 401 Bad credentials []
```

So that wasn't it...

Looking at the previous release, when I GET that file I don't get a 404 lile goreleaser says it does - https://api.github.com/repos/bbkane/homebrew-tap/contents/Formula/fling.rb

https://goreleaser.com/ci/actions/ - adding permissions to the job
