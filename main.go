package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/bbkane/warg"
	"github.com/bbkane/warg/command"
	"github.com/bbkane/warg/flag"
	"github.com/bbkane/warg/section"
	"github.com/bbkane/warg/value"
	"github.com/karrick/godirwalk"
)

// This will be overriden by goreleaser
var version = "unkown version: error reading goreleaser info"

func getVersion() string {
	// If installed via `go install`, we'll be able to read runtime version info
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown version: error reading build info"
	}
	if info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	// If built via GoReleaser, we'll be able to read this from the linker flags
	return version
}

func printVersion(_ flag.PassedFlags) error {
	fmt.Println(getVersion())
	return nil
}

// checkMode for types of files we're not prepared to deal with :)
// It does not check for symlinks
func checkMode(mode os.FileMode) error {
	if mode&fs.ModeExclusive != 0 {
		return fmt.Errorf("ModeExclusive set")
	}
	if mode&fs.ModeTemporary != 0 {
		return fmt.Errorf("ModeTemporary set")
	}
	if mode&fs.ModeDevice != 0 {
		return fmt.Errorf("ModeDevice set")
	}
	if mode&fs.ModeNamedPipe != 0 {
		return fmt.Errorf("ModeNamedPipe set")
	}
	if mode&fs.ModeSocket != 0 {
		return fmt.Errorf("ModeSocket set")
	}
	if mode&fs.ModeCharDevice != 0 {
		return fmt.Errorf("ModeCharDevice set")
	}
	if mode&fs.ModeIrregular != 0 {
		return fmt.Errorf("ModeIrregular")
	}
	return nil
}

type linkToCreate struct {
	src  string
	link string
}
type existingLink struct {
	src  string
	link string
}

type pathErr struct {
	path string
	err  error
}

type pathsErr struct {
	src  string
	link string
	err  error
}

func link(pf flag.PassedFlags) error {
	linkDir := pf["--link-dir"].(string)
	srcDir := pf["--src-dir"].(string)
	// ignore := []string{}
	// if ignoreF, exists := pf["--ignore"]; exists {
	// 	ignore = ignoreF.([]string)
	// }

	linkDir, err := filepath.Abs(linkDir)
	if err != nil {
		return fmt.Errorf("couldn't get abs path for linkDir: %w", err)
	}

	srcDir, err = filepath.Abs(srcDir)
	if err != nil {
		return fmt.Errorf("couldn't get abs path for srcDir: %w", err)
	}

	linksToCreate := []linkToCreate{}
	existingLinks := []existingLink{}
	pathErrs := []pathErr{}
	pathsErrs := []pathsErr{}

	err = godirwalk.Walk(srcDir, &godirwalk.Options{

		Callback: func(srcPath string, srcDe *godirwalk.Dirent) error {
			// fmt.Printf("%s - %s\n", de.ModeType(), osPathname)
			if srcPath == srcDir {
				return nil // skip the first entry (toDir)
			}

			// determine linkPath
			relPath, err := filepath.Rel(srcDir, srcPath)
			if err != nil {
				p := pathErr{
					path: srcPath,
					err:  fmt.Errorf("can't get relative path: %s, %w", srcDir, err),
				}
				pathErrs = append(pathErrs, p)
				return godirwalk.SkipThis
			}
			linkPath := filepath.Join(linkDir, relPath)

			// fmt.Printf("srcPath: %s\nlinkPath: %s\n\n", srcPath, linkPath)

			if srcDe.IsSymlink() {
				// fmt.Printf("srcDe isSymlink: %s", srcPath)
				p := pathErr{
					path: srcPath,
					err:  errors.New("is symlink"),
				}
				pathErrs = append(pathErrs, p)
				return godirwalk.SkipThis
			}

			err = checkMode(srcDe.ModeType())
			if err != nil {
				// fmt.Printf("checkMode err: %s: %s", srcPath, err)
				p := pathErr{
					path: srcPath,
					err:  err,
				}
				pathErrs = append(pathErrs, p)
				return godirwalk.SkipThis
			}

			linkPathLstatRes, linkPathLstatErr := os.Lstat(linkPath)
			if errors.Is(linkPathLstatErr, fs.ErrNotExist) {
				// fmt.Printf("link: srcPath: %s , linkPath: %s\n", srcPath, linkPath)
				ltc := linkToCreate{
					src:  srcPath,
					link: linkPath,
				}
				linksToCreate = append(linksToCreate, ltc)
				return godirwalk.SkipThis // TODO: ensure this skips children.... It says "skip this directory entry", so probably...
			}
			if linkPathLstatErr != nil {
				// fmt.Printf("linkPathStatErr: %s: %s\n", linkPath, linkPathLstatErr)
				p := pathErr{
					path: linkPath,
					err:  linkPathLstatErr,
				}
				pathErrs = append(pathErrs, p)
				return godirwalk.SkipThis
			}
			err = checkMode(linkPathLstatRes.Mode())
			if err != nil {
				// fmt.Printf("checkMode err: %s: %s\n", linkPath, err)
				p := pathErr{
					path: linkPath,
					err:  err,
				}
				pathErrs = append(pathErrs, p)
				return godirwalk.SkipThis
			}
			// from my tests on MacOS, if the symlink bit is set, the directory mode will not be set
			// leaving this in here anyway, because, from the godirwalk docs, on Windows, if the symlink bit is set,
			// and it's a symlink to a directory, the directory bit will also be set
			// so it's easier to just keep this check in both branches
			if linkPathLstatRes.Mode()&fs.ModeSymlink != 0 {
				// it's a symlink, get target. We're already expecting an absolute link
				linkPathSymlinkTarget, err := os.Readlink(linkPath)
				if err != nil {
					// fmt.Printf("readlink Err: %s: %s\n", linkPath, err)
					p := pathErr{
						path: linkPath,
						err:  err,
					}
					pathErrs = append(pathErrs, p)
					return godirwalk.SkipThis
				}
				if linkPathSymlinkTarget == srcPath {
					// fmt.Printf("linkPath already points to target. No need to do more")
					el := existingLink{
						src:  srcPath,
						link: linkPath,
					}
					existingLinks = append(existingLinks, el)
					return godirwalk.SkipThis
				} else {
					// fmt.Printf("linkPath unrecognized symlink: %s -> %s , not %s\n", linkPath, linkPathSymlinkTarget, srcPath)
					pse := pathsErr{
						src:  srcPath,
						link: linkPath,
						err:  fmt.Errorf("unexpected symlink: target: %s", linkPathSymlinkTarget),
					}
					pathsErrs = append(pathsErrs, pse)
					return godirwalk.SkipThis
				}
			}

			if linkPathLstatRes.IsDir() {
				if srcDe.IsDir() {
					// I think this is ok and we don't need to report it :)
					// fmt.Printf("linkPath is already an existing dir. Continuing with children: %s\n", linkPath)
					return nil
				} else {
					// fmt.Printf("ERROR: linkPath is existing dir and srcPath is file: linkpath: %s , srcPath: %s\n", linkPath, srcPath)
					pse := pathsErr{
						src:  srcPath,
						link: linkPath,
						err:  errors.New("link is existing dir and src is file"),
					}
					pathsErrs = append(pathsErrs, pse)
					return nil
				}

			}
			// linkpath is an existing normal file
			p := pathErr{
				path: linkPath,
				err:  errors.New("linkPath is already an existing file"),
			}
			pathErrs = append(pathErrs, p)
			return godirwalk.SkipThis
		},
		// https://pkg.go.dev/github.com/karrick/godirwalk#Options
		Unsorted:            true,
		AllowNonDirectory:   false,
		FollowSymbolicLinks: false,
	})
	if err != nil {
		return fmt.Errorf("walking error: %w", err)
	}

	if len(linksToCreate) > 0 {
		fmt.Println("Links to create")
		for _, e := range linksToCreate {
			fmt.Printf("  %#v\n", e)
		}
	}

	if len(existingLinks) > 0 {
		fmt.Println("Existing Links (probably previously created)")
		for _, e := range existingLinks {
			fmt.Printf("  %#v\n", e)
		}
	}

	if len(pathErrs) > 0 {
		fmt.Println("Path errs")
		for _, e := range pathErrs {
			fmt.Printf("  %#v\n", e)
		}
	}

	if len(pathsErrs) > 0 {
		fmt.Println("Paths errs")
		for _, e := range pathsErrs {
			fmt.Printf("  %#v\n", e)
		}
	}

	return nil
}

func main() {

	linkDirFlag := flag.New(
		"Symlinks will be created in this directory pointing to files/directories in --to_dir",
		value.Path,
		flag.Alias("-l"),
		flag.Default("~"),
		flag.Required(),
	)

	srcDirFlag := flag.New(
		"Directory containing files and directories to link to",
		value.Path,
		flag.Alias("-s"),
		flag.Required(),
	)
	app := warg.New(
		"fling",
		section.New(
			"Create symlinks under location to link_to_parent",
			section.WithCommand(
				"version",
				"Print version",
				printVersion,
			),
			section.WithCommand(
				"link",
				"Link away!",
				link,
				command.AddFlag(
					"--link-dir",
					linkDirFlag,
				),
				command.AddFlag(
					"--src-dir",
					srcDirFlag,
				),
				command.WithFlag(
					"--ignore",
					"glob pattern to ignore. Does not expand ~.",
					value.StringSlice,
				),
			),
		),
	)
	app.MustRun(os.Args, os.LookupEnv)
}
