package main

import (
	"go.bbkane.com/warg"
	"go.bbkane.com/warg/command"
	"go.bbkane.com/warg/completion"
	"go.bbkane.com/warg/flag"
	"go.bbkane.com/warg/path"
	"go.bbkane.com/warg/section"
	"go.bbkane.com/warg/value/scalar"
	"go.bbkane.com/warg/value/slice"
	"go.bbkane.com/warg/wargcore"
)

var version string

func app() *wargcore.App {
	linkUnlinkFlags := wargcore.FlagMap{
		"--ask": flag.New(
			"Whether to ask before making changes",
			scalar.String(
				scalar.Choices("true", "false", "dry-run"),
				scalar.Default("true"),
			),
			flag.Required(),
		),
		"--dotfiles": flag.New(
			"Files/dirs starting with 'dot-' will have links starting with '.'",
			scalar.Bool(
				scalar.Default(true),
			),
			flag.Required(),
		),
		"--ignore": flag.New(
			"Ignore file/dir if the name (not the whole path) matches passed regex",
			slice.String(
				slice.Default([]string{"README.*"}),
			),
			flag.Alias("-i"),
			flag.UnsetSentinel("UNSET"),
		),
		"--link-dir": flag.New(
			"Symlinks will be created in this directory pointing to files/directories in --src-dir",
			scalar.Path(
				scalar.Default(path.New("~")),
			),
			flag.Alias("-l"),
			flag.CompletionCandidates(func(ctx wargcore.Context) (*completion.Candidates, error) {
				return &completion.Candidates{
					Type:   completion.Type_Directories,
					Values: nil,
				}, nil
			}),
			flag.Required(),
		),
		"--src-dir": flag.New(
			"Directory containing files and directories to link to",
			scalar.Path(),
			flag.Alias("-s"),
			flag.CompletionCandidates(func(ctx wargcore.Context) (*completion.Candidates, error) {
				return &completion.Candidates{
					Type:   completion.Type_Directories,
					Values: nil,
				}, nil
			}),
			flag.Required(),
		),
	}

	app := warg.New(
		"fling",
		version,
		section.New(
			"Link and unlink directory heirarchies ",
			section.NewCommand(
				"link",
				"Create links",
				link,
				command.FlagMap(linkUnlinkFlags),
			),
			section.NewCommand(
				"unlink",
				"Unlink previously created links",
				unlink,
				command.FlagMap(linkUnlinkFlags),
			),
			section.CommandMap(warg.VersionCommandMap()),
			section.Footer("Homepage: https://github.com/bbkane/fling"),
		),
		warg.GlobalFlagMap(warg.ColorFlagMap()),
		warg.SkipValidation(),
	)
	return &app
}

func main() {
	app := app()
	app.MustRun()
}
