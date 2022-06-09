package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/karrick/godirwalk"
	"go.bbkane.com/gocolor"
	"go.bbkane.com/warg/command"
	"go.bbkane.com/warg/help"
)

// checkMode for types of files we're not prepared to deal with :)
// It does not check for symlinks.
// Also see https://pkg.go.dev/io/fs#FileMode
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

type linkT struct {
	src  string
	link string
}

func (t linkT) ColorString(color *gocolor.Color) string {
	return fmt.Sprintf(
		"- %s: %s\n  %s: %s",
		color.Add(color.Bold, "src"),
		t.src,
		color.Add(color.Bold, "link"),
		t.link,
	)
}

func fPrintLinkTs(f *bufio.Writer, color *gocolor.Color, lTs []linkT) {
	for _, e := range lTs {
		fmt.Fprintf(f, "%s\n", e.ColorString(color))
	}
}

// NOTE: making these type aliases instead of type definitions
// so I can use the String() method of linkT, even though
// it means I can assign instances of each types to instances of
// any other equivalent type...
type fileLinkToCreate = linkT
type dirLinkToCreate = linkT

type existingFileLink = linkT
type existingDirLink = linkT

type pathErr struct {
	path string
	err  error
}

func (t pathErr) ColorString(color *gocolor.Color) string {
	return fmt.Sprintf(
		"- %s: %s\n  %s: %s",
		color.Add(color.Bold, "path"),
		t.path,
		color.Add(color.Bold+color.FgRed, "err"),
		t.err,
	)
}

type pathsErr struct {
	src  string
	link string
	err  error
}

func (t pathsErr) ColorString(color *gocolor.Color) string {
	return fmt.Sprintf(
		"- %s: %s\n  %s: %s\n  %s: %s",
		color.Add(color.Bold, "src"),
		t.src,
		color.Add(color.Bold, "link"),
		t.link,
		color.Add(color.Bold+color.FgRed, "err"),
		t.err,
	)
}

type ignoredPath string

func (t ignoredPath) ColorString(color *gocolor.Color) string {
	return fmt.Sprintf(
		"- %s: %s",
		color.Add(color.Bold, "path"),
		string(t),
	)
}

type fileInfo struct {
	dirLinksToCreate  []dirLinkToCreate
	fileLinksToCreate []fileLinkToCreate
	existingDirLinks  []existingDirLink
	existingFileLinks []existingFileLink
	pathErrs          []pathErr
	pathsErrs         []pathsErr
	ignoredPaths      []ignoredPath
}

func fPrintHeader(f *bufio.Writer, color *gocolor.Color, header string) {
	fmt.Fprint(
		f,
		color.Add(color.Bold+color.Underline, header+"\n\n"),
	)
}

func fPrintErrorHeader(f *bufio.Writer, color *gocolor.Color, header string) {
	fmt.Fprint(
		f,
		color.Add(color.Bold+color.Underline+color.FgRed, header+"\n\n"),
	)
}

// replacePrefix return (s with prefixed replaced, true)
// if the string has the prefix, otherwise (s, false)
func replacePrefix(s string, prefix string, replacement string) (string, bool) {
	if strings.HasPrefix(s, prefix) {
		s = replacement + strings.TrimPrefix(s, prefix)
		return s, true
	}
	return s, false
}

func buildFileInfo(srcDir string, linkDir string, ignorePatterns []string, isDotfiles bool) (*fileInfo, error) {
	linkDir, err := filepath.Abs(linkDir)
	if err != nil {
		return nil, fmt.Errorf("couldn't get abs path for linkDir: %w", err)
	}

	srcDir, err = filepath.Abs(srcDir)
	if err != nil {
		return nil, fmt.Errorf("couldn't get abs path for srcDir: %w", err)
	}

	fi := fileInfo{}
	linkPathReplacements := make(map[string]string)

	err = godirwalk.Walk(srcDir, &godirwalk.Options{

		Callback: func(srcPath string, srcDe *godirwalk.Dirent) error {
			// fmt.Printf("%s - %s\n", de.ModeType(), osPathname)
			if srcPath == srcDir {
				return nil // skip the first entry (toDir)
			}

			// ignore srcPath name regexes
			for _, pattern := range ignorePatterns {
				// NOTE: can compile these regexes for speed
				match, err := regexp.Match(pattern, []byte(srcDe.Name()))
				if err != nil {
					err = fmt.Errorf("invalid --ignore pattern: %s: %w", pattern, err)
					return err // Exit immediately on a bad pattern.
				}
				if match {
					fi.ignoredPaths = append(fi.ignoredPaths, ignoredPath(srcPath))
					return godirwalk.SkipThis
				}
			}

			// determine linkPath
			relPath, err := filepath.Rel(srcDir, srcPath)
			if err != nil {
				p := pathErr{
					path: srcPath,
					err:  fmt.Errorf("can't get relative path: %s, %w", srcDir, err),
				}
				fi.pathErrs = append(fi.pathErrs, p)
				return godirwalk.SkipThis
			}
			linkPath := filepath.Join(linkDir, relPath)

			// Now that we have a linkPath, "correct" it if necessary by replacing dot- with .
			// because we're not changing srcPath, "errors" will keep popping up, so keep a list of
			// replacements around to "correct" parent directories
			if isDotfiles {
				// replace previous elements of the path from parents we've already seen
				// fmt.Printf("linkPathReplacements: %#v\n", linkPathReplacements)
				for path, replacement := range linkPathReplacements {
					// fmt.Printf(":%s: %s -> %s\n", linkPath, path, replacement)
					linkPath, _ = replacePrefix(linkPath, path, replacement)
				}

				linkPathName := filepath.Base(linkPath)
				linkPathDir := filepath.Dir(linkPath)
				// replace the last element of the path if necessary
				linkPathNameNew, replaced := replacePrefix(linkPathName, "dot-", ".")
				if replaced {
					// fmt.Printf("replaced: %s -> %s\n", linkPathName, linkPathNameNew)
					linkPathNew := filepath.Join(linkPathDir, linkPathNameNew)
					linkPathReplacements[linkPath] = linkPathNew
					linkPath = linkPathNew
				}
			}

			if srcDe.IsSymlink() {
				// fmt.Printf("srcDe isSymlink: %s", srcPath)
				p := pathErr{
					path: srcPath,
					err:  errors.New("is symlink"),
				}
				fi.pathErrs = append(fi.pathErrs, p)
				return godirwalk.SkipThis
			}

			err = checkMode(srcDe.ModeType())
			if err != nil {
				// fmt.Printf("checkMode err: %s: %s", srcPath, err)
				p := pathErr{
					path: srcPath,
					err:  err,
				}
				fi.pathErrs = append(fi.pathErrs, p)
				return godirwalk.SkipThis
			}

			linkPathLstatRes, linkPathLstatErr := os.Lstat(linkPath)
			if errors.Is(linkPathLstatErr, fs.ErrNotExist) {
				if srcDe.IsDir() {
					ltc := dirLinkToCreate{
						src:  srcPath,
						link: linkPath,
					}
					fi.dirLinksToCreate = append(fi.dirLinksToCreate, ltc)
					return godirwalk.SkipThis
				} else {
					ltc := fileLinkToCreate{
						src:  srcPath,
						link: linkPath,
					}
					fi.fileLinksToCreate = append(fi.fileLinksToCreate, ltc)
					return godirwalk.SkipThis
				}
			}

			// So linkPath does exist. Let's inspect it and see what we can do
			if linkPathLstatErr != nil {
				p := pathErr{
					path: linkPath,
					err:  linkPathLstatErr,
				}
				fi.pathErrs = append(fi.pathErrs, p)
				return godirwalk.SkipThis
			}
			err = checkMode(linkPathLstatRes.Mode())
			if err != nil {
				// fmt.Printf("checkMode err: %s: %s\n", linkPath, err)
				p := pathErr{
					path: linkPath,
					err:  err,
				}
				fi.pathErrs = append(fi.pathErrs, p)
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
					fi.pathErrs = append(fi.pathErrs, p)
					return godirwalk.SkipThis
				}
				if linkPathSymlinkTarget == srcPath {
					// fmt.Printf("linkPath already points to target. No need to do more")

					if srcDe.IsDir() {
						edl := existingDirLink{
							src:  srcPath,
							link: linkPath,
						}
						fi.existingDirLinks = append(fi.existingDirLinks, edl)
					} else {
						efl := existingFileLink{
							src:  srcPath,
							link: linkPath,
						}
						fi.existingFileLinks = append(fi.existingFileLinks, efl)
					}

					return godirwalk.SkipThis
				} else {
					// fmt.Printf("linkPath unrecognized symlink: %s -> %s , not %s\n", linkPath, linkPathSymlinkTarget, srcPath)
					pse := pathsErr{
						src:  srcPath,
						link: linkPath,
						err:  fmt.Errorf("link is already a symlink to src: %s", linkPathSymlinkTarget),
					}
					fi.pathsErrs = append(fi.pathsErrs, pse)
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
					fi.pathsErrs = append(fi.pathsErrs, pse)
					return nil
				}

			}
			// linkpath is an existing normal file
			p := pathErr{
				path: linkPath,
				err:  errors.New("linkPath is already an existing file"),
			}
			fi.pathErrs = append(fi.pathErrs, p)
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

	return &fi, nil

}

// askPrompt looks at the ask string and decides whether to continue
// the bool indicates whether to continue and the err indicates any errors
func askPrompt(ask string) (bool, error) {
	switch ask {
	case "true":
		fmt.Print("Type 'yes' to continue: ")
		reader := bufio.NewReader(os.Stdin)
		confirmation, err := reader.ReadString('\n')
		if err != nil {
			err = fmt.Errorf("confirmation ReadString error: %w", err)
			return false, err
		}
		confirmation = strings.TrimSpace(confirmation)
		if confirmation != "yes" {
			err := fmt.Errorf("confirmation not yes: %v", confirmation)
			return false, err
		}
		return true, nil
	case "false":
		return true, nil
	case "dry-run":
		return false, nil
	default:
		return false, fmt.Errorf("ask not valid: %s", ask)
	}
}

func unlink(ctx command.Context) error {
	ask := ctx.Flags["--ask"].(string)
	linkDir := ctx.Flags["--link-dir"].(string)
	srcDir := ctx.Flags["--src-dir"].(string)
	isDotfiles := ctx.Flags["--dotfiles"].(bool)
	ignorePatterns := []string{}
	if ignoreF, exists := ctx.Flags["--ignore"]; exists {
		ignorePatterns = ignoreF.([]string)
	}

	color, err := help.ConditionallyEnableColor(ctx.Flags, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error enabling color. Continuing without: %v\n", err)
	}

	fi, err := buildFileInfo(srcDir, linkDir, ignorePatterns, isDotfiles)
	if err != nil {
		return err
	}

	// Print fileInfo
	{
		f := bufio.NewWriter(os.Stdout)

		if len(fi.ignoredPaths) > 0 {
			fPrintHeader(f, &color, "Ignored paths:")
			for _, e := range fi.ignoredPaths {
				fmt.Fprintf(f, "%s\n", e.ColorString(&color))
			}
			fmt.Fprintln(f)
		}

		if len(fi.dirLinksToCreate) > 0 {
			fPrintHeader(f, &color, "Uncreated dir links:")
			fPrintLinkTs(f, &color, fi.dirLinksToCreate)
			fmt.Fprintln(f)
		}

		if len(fi.fileLinksToCreate) > 0 {
			fPrintHeader(f, &color, "Uncreated file links:")
			fPrintLinkTs(f, &color, fi.fileLinksToCreate)
			fmt.Fprintln(f)
		}

		if len(fi.existingDirLinks) > 0 {
			fPrintHeader(f, &color, "Dir links to delete:")
			fPrintLinkTs(f, &color, fi.existingDirLinks)
			fmt.Fprintln(f)
		}

		if len(fi.existingFileLinks) > 0 {
			fPrintHeader(f, &color, "File links to delete:")
			fPrintLinkTs(f, &color, fi.existingFileLinks)
			fmt.Fprintln(f)
		}

		if len(fi.pathErrs) > 0 {
			fPrintErrorHeader(f, &color, "Path errors:")
			for _, e := range fi.pathErrs {
				fmt.Fprintf(f, "%s\n", e.ColorString(&color))
			}
			fmt.Fprintln(f)
		}

		if len(fi.pathsErrs) > 0 {
			fPrintErrorHeader(f, &color, "Proposed link mismatch errors:")
			for _, e := range fi.pathsErrs {
				fmt.Fprintf(f, "%s\n", e.ColorString(&color))
			}
			fmt.Fprintln(f)
		}
		f.Flush()
	}
	if len(fi.existingFileLinks) == 0 && len(fi.existingDirLinks) == 0 {
		fmt.Print(
			color.Add(
				color.Bold+color.FgGreenBright,
				"Nothing to do!\n",
			),
		)
		return nil // exit
	}
	fmt.Print(
		color.Add(
			color.Bold,
			"Delete links?\n",
		),
	)

	keepGoing, err := askPrompt(ask)
	if !keepGoing {
		if err == nil {
			fmt.Print(
				color.Add(
					color.Bold+color.FgGreenBright,
					"Dry run - no changes made\n",
				),
			)
		}
		return err
	}

	for _, e := range fi.existingDirLinks {
		err := os.Remove(e.link)
		if err != nil {
			return err
		}
	}
	for _, e := range fi.existingFileLinks {
		err := os.Remove(e.link)
		if err != nil {
			return err
		}
	}
	fmt.Print(
		color.Add(
			color.Bold+color.FgGreenBright,
			"Done!\n",
		),
	)
	return nil
}

func link(ctx command.Context) error {
	ask := ctx.Flags["--ask"].(string)
	linkDir := ctx.Flags["--link-dir"].(string)
	srcDir := ctx.Flags["--src-dir"].(string)
	isDotfiles := ctx.Flags["--dotfiles"].(bool)
	ignorePatterns := []string{}
	if ignoreF, exists := ctx.Flags["--ignore"]; exists {
		ignorePatterns = ignoreF.([]string)
	}

	color, err := help.ConditionallyEnableColor(ctx.Flags, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error enabling color. Continuing without: %v\n", err)
	}

	fi, err := buildFileInfo(srcDir, linkDir, ignorePatterns, isDotfiles)
	if err != nil {
		return err
	}

	// Print fileInfo
	{
		f := bufio.NewWriter(os.Stdout)

		if len(fi.ignoredPaths) > 0 {
			fPrintHeader(f, &color, "Ignored paths:")
			for _, e := range fi.ignoredPaths {
				fmt.Fprintf(f, "%s\n", e.ColorString(&color))
			}
			fmt.Fprintln(f)
		}

		if len(fi.dirLinksToCreate) > 0 {
			fPrintHeader(f, &color, "Dir links to create:")
			fPrintLinkTs(f, &color, fi.dirLinksToCreate)
			fmt.Fprintln(f)
		}

		if len(fi.fileLinksToCreate) > 0 {
			fPrintHeader(f, &color, "File links to create:")
			fPrintLinkTs(f, &color, fi.fileLinksToCreate)
			fmt.Fprintln(f)
		}

		if len(fi.existingDirLinks) > 0 {
			fPrintHeader(f, &color, "Pre-existing correct dir links:")
			fPrintLinkTs(f, &color, fi.existingDirLinks)
			fmt.Fprintln(f)
		}
		if len(fi.existingFileLinks) > 0 {
			fPrintHeader(f, &color, "Pre-existing correct file links:")
			fPrintLinkTs(f, &color, fi.existingFileLinks)
			fmt.Fprintln(f)
		}

		if len(fi.pathErrs) > 0 {
			fPrintErrorHeader(f, &color, "Path errors:")
			for _, e := range fi.pathErrs {
				fmt.Fprintf(f, "%s\n", e.ColorString(&color))
			}
			fmt.Fprintln(f)
		}

		if len(fi.pathsErrs) > 0 {
			fPrintErrorHeader(f, &color, "Proposed link mismatch errors:")
			for _, e := range fi.pathsErrs {
				fmt.Fprintf(f, "%s\n", e.ColorString(&color))
			}
			fmt.Fprintln(f)
		}
		f.Flush()
	}

	if len(fi.fileLinksToCreate) == 0 && len(fi.dirLinksToCreate) == 0 {
		fmt.Print(
			color.Add(
				color.Bold+color.FgGreenBright,
				"Nothing to do!\n",
			),
		)
		return nil // exit
	}

	fmt.Print(
		color.Add(
			color.Bold,
			"Create links?\n",
		),
	)

	keepGoing, err := askPrompt(ask)
	if !keepGoing {
		if err == nil {
			fmt.Print(
				color.Add(
					color.Bold+color.FgGreenBright,
					"Dry run - no changed made\n",
				),
			)
		}
		return err
	}

	for _, e := range fi.dirLinksToCreate {
		err := os.Symlink(e.src, e.link)
		if err != nil {
			return err
		}
	}
	for _, e := range fi.fileLinksToCreate {
		err := os.Symlink(e.src, e.link)
		if err != nil {
			return err
		}
	}
	fmt.Print(
		color.Add(
			color.Bold+color.FgGreenBright,
			"Done!\n",
		),
	)
	return nil
}
