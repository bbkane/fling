package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/bbkane/go-color"
	"github.com/bbkane/warg"
	"github.com/bbkane/warg/command"
	"github.com/bbkane/warg/flag"
	"github.com/bbkane/warg/help"
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

func (t linkToCreate) String() string {
	return fmt.Sprintf(
		"- %s: %s\n  %s: %s",
		color.Add(color.Bold, "src"),
		t.src,
		color.Add(color.Bold, "link"),
		t.link,
	)
}

type existingLink struct {
	src  string
	link string
}

func (t existingLink) String() string {
	return fmt.Sprintf(
		"- %s: %s\n  %s: %s",
		color.Add(color.Bold, "src"),
		t.src,
		color.Add(color.Bold, "link"),
		t.link,
	)
}

type pathErr struct {
	path string
	err  error
}

func (t pathErr) String() string {
	return fmt.Sprintf(
		"- %s: %s\n  %s: %s",
		color.Add(color.Bold, "path"),
		t.path,
		color.Add(color.Bold+color.ForegroundRed, "err"),
		t.err,
	)
}

type pathsErr struct {
	src  string
	link string
	err  error
}

func (t pathsErr) String() string {
	return fmt.Sprintf(
		"- %s: %s\n  %s: %s\n  %s: %s",
		color.Add(color.Bold, "src"),
		t.src,
		color.Add(color.Bold, "link"),
		t.link,
		color.Add(color.Bold+color.ForegroundRed, "err"),
		t.err,
	)
}

type fileInfo struct {
	linksToCreate []linkToCreate
	existingLinks []existingLink
	pathErrs      []pathErr
	pathsErrs     []pathsErr
}

func buildFileInfo(srcDir string, linkDir string) (*fileInfo, error) {
	linkDir, err := filepath.Abs(linkDir)
	if err != nil {
		return nil, fmt.Errorf("couldn't get abs path for linkDir: %w", err)
	}

	srcDir, err = filepath.Abs(srcDir)
	if err != nil {
		return nil, fmt.Errorf("couldn't get abs path for srcDir: %w", err)
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
						err:  fmt.Errorf("link is already a symlink to: %s", linkPathSymlinkTarget),
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
		return nil, fmt.Errorf("walking error: %w", err)
	}

	return &fileInfo{
		linksToCreate: linksToCreate,
		existingLinks: existingLinks,
		pathErrs:      pathErrs,
		pathsErrs:     pathsErrs,
	}, nil

}

// func unlink(pf flag.PassedFlags) error {
// 	linkDir := pf["--link-dir"].(string)
// 	srcDir := pf["--src-dir"].(string)

// 	help.ConditionallyEnableColor(pf, os.Stdout)
// 	fi, err := buildFileInfo(srcDir, linkDir)
// 	if err != nil {
// 		return err
// 	}
// }

func link(pf flag.PassedFlags) error {
	linkDir := pf["--link-dir"].(string)
	srcDir := pf["--src-dir"].(string)
	// ignore := []string{}
	// if ignoreF, exists := pf["--ignore"]; exists {
	// 	ignore = ignoreF.([]string)
	// }

	// help.CondionallyEnableColor(pf, os.Stdout)
	help.ConditionallyEnableColor(pf, os.Stdout)

	fi, err := buildFileInfo(srcDir, linkDir)
	if err != nil {
		return err
	}

	// Print fileInfo
	{
		f := bufio.NewWriter(os.Stdout)
		defer f.Flush()

		if len(fi.linksToCreate) > 0 {
			fmt.Fprint(
				f,
				color.Add(color.Bold+color.Underline, "Links to create:\n\n"),
			)
			for _, e := range fi.linksToCreate {
				fmt.Fprintf(f, "%s\n", e)
			}
			fmt.Fprintln(f)
		}

		if len(fi.existingLinks) > 0 {
			fmt.Fprint(
				f,
				color.Add(color.Underline+color.Bold, "Pre-existing Correct Links:\n\n"),
			)
			for _, e := range fi.existingLinks {
				fmt.Fprintf(f, "%s\n", e)
			}
			fmt.Fprintln(f)
		}

		if len(fi.pathErrs) > 0 {
			fmt.Fprint(
				f,
				color.Add(color.Bold+color.Underline+color.ForegroundRed, "Path errors:\n\n"),
			)
			for _, e := range fi.pathErrs {
				fmt.Fprintf(f, "%s\n", e)
			}
			fmt.Fprintln(f)
		}

		if len(fi.pathsErrs) > 0 {
			fmt.Fprint(
				f,
				color.Add(color.Bold+color.Underline+color.ForegroundRed, "Proposed link mismatch errors:\n\n"),
			)
			for _, e := range fi.pathsErrs {
				fmt.Fprintf(f, "%s\n", e)
			}
			fmt.Fprintln(f)
		}

		if len(fi.linksToCreate) == 0 {
			fmt.Fprint(
				f,
				color.Add(
					color.Bold+color.BrightForegroundGreen,
					"Nothing to do!\n",
				),
			)
			return nil // exit
		}
		fmt.Fprint(
			f,
			color.Add(
				color.Bold,
				"Create links?\n",
			),
		)
	}
	// confirmation prompt
	{
		fmt.Print("Type 'yes' to continue: ")
		reader := bufio.NewReader(os.Stdin)
		confirmation, err := reader.ReadString('\n')
		if err != nil {
			err = fmt.Errorf("confirmation ReadString error: %w", err)
			return err
		}
		confirmation = strings.TrimSpace(confirmation)
		if confirmation != "yes" {
			err := fmt.Errorf("confirmation not yes: %v", confirmation)
			return err
		}
	}

	for _, e := range fi.linksToCreate {
		err := os.Symlink(e.src, e.link)
		if err != nil {
			return err
		}
	}
	if len(fi.linksToCreate) == 0 {
		fmt.Print(
			color.Add(
				color.Bold+color.BrightForegroundGreen,
				"Done!\n",
			),
		)
		return nil // exit
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
					"Patterns to ignore. Only applied to the last element of the path (the name or base)",
					value.StringSlice,
				),
			),
			section.WithFlag(
				"--color",
				"Control color (including for --help)",
				value.StringEnum("true", "false", "auto"),
				flag.Alias("-c"),
				flag.Default("auto"),
			),
		),
	)
	app.MustRun(os.Args, os.LookupEnv)
}
