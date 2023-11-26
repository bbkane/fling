package main

import (
	"go.bbkane.com/warg"
	"go.bbkane.com/warg/command"
	"go.bbkane.com/warg/flag"
	"go.bbkane.com/warg/section"
	"go.bbkane.com/warg/value/scalar"
	"go.bbkane.com/warg/value/slice"
)

var version string

func app() *warg.App {
	linkUnlinkFlags := flag.FlagMap{
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
				scalar.Default("~"),
			),
			flag.Alias("-l"),
			flag.Required(),
		),
		"--src-dir": flag.New(
			"Directory containing files and directories to link to",
			scalar.Path(),
			flag.Alias("-s"),
			flag.Required(),
		),
	}

	app := warg.New(
		"fling",
		section.New(
			"Link and unlink directory heirarchies ",
			section.Command(
				"link",
				"Create links",
				link,
				command.ExistingFlags(linkUnlinkFlags),
			),
			section.Command(
				"unlink",
				"Unlink previously created links",
				unlink,
				command.ExistingFlags(linkUnlinkFlags),
			),
			section.ExistingCommand("version", warg.VersionCommand()),
			section.ExistingFlag("--color", warg.ColorFlag()),
			section.Footer("Homepage: https://github.com/bbkane/fling"),
		),
		warg.OverrideVersion(version),
		warg.SkipValidation(),
	)
	return &app
}

func main() {
	app().MustRun()
}
